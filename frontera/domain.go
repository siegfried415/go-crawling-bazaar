/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package frontera

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"fmt"

	"github.com/pkg/errors"

	//wyong, 20201018
	//"github.com/libp2p/go-libp2p-core/host"

	//wyong, 20200929 
	"github.com/siegfried415/gdf-rebuild/conf"


	//wyong, 20201002 
	"github.com/siegfried415/gdf-rebuild/crypto" 
	"github.com/siegfried415/gdf-rebuild/crypto/asymmetric"
	"github.com/siegfried415/gdf-rebuild/kms"

	"github.com/siegfried415/gdf-rebuild/kayak"
	kt "github.com/siegfried415/gdf-rebuild/kayak/types"
	kl "github.com/siegfried415/gdf-rebuild/kayak/wal"
	"github.com/siegfried415/gdf-rebuild/proto"
	urlchain "github.com/siegfried415/gdf-rebuild/urlchain"
	"github.com/siegfried415/gdf-rebuild/storage"
	"github.com/siegfried415/gdf-rebuild/types"
	"github.com/siegfried415/gdf-rebuild/utils/log"
	x "github.com/siegfried415/gdf-rebuild/xenomint"
	net "github.com/siegfried415/gdf-rebuild/net"

	//wyong, 20200805 
	lru "github.com/hashicorp/golang-lru"

	//wyong, 20200818
	cid "github.com/ipfs/go-cid"

)

const (
	// StorageFileName defines storage file name of database instance.
	StorageFileName = "storage.db3"

	// KayakWalFileName defines log pool name of database instance.
	KayakWalFileName = "kayak.ldb"

	// SQLChainFileName defines sqlchain storage file name.
	SQLChainFileName = "chain.db"

	// MaxRecordedConnectionSequences defines the max connection slots to anti reply attack.
	MaxRecordedConnectionSequences = 1000

	// PrepareThreshold defines the prepare complete threshold.
	PrepareThreshold = 1.0

	// CommitThreshold defines the commit complete threshold.
	CommitThreshold = 0.0

	// PrepareTimeout defines the prepare timeout config.
	PrepareTimeout = 10 * time.Second

	// CommitTimeout defines the commit timeout config.
	CommitTimeout = time.Minute

	// LogWaitTimeout defines the missing log wait timeout config.
	LogWaitTimeout = 10 * time.Second

	// SlowQuerySampleSize defines the maximum slow query log size (default: 1KB).
	SlowQuerySampleSize = 1 << 10

	//wyong, 20200805 
	activeWantsLimit = 16
)

// Database defines a single database instance in worker runtime.
type Domain struct {
	cfg            *DomainConfig
	domainID           proto.DomainID
	kayakWal       *kl.LevelDBWal
	kayakRuntime   *kayak.Runtime
	kayakConfig    *kt.RuntimeConfig
	connSeqs       sync.Map
	connSeqEvictCh chan uint64
	chain          *urlchain.Chain
	nodeID         proto.NodeID

	//wyong, 20201018
	host 		net.RoutedHost

	mux            *DomainKayakMuxService

	privateKey     *asymmetric.PrivateKey
	accountAddr    proto.AccountAddress

	//wyong, move session stuff here,  20200805 
	ctx            context.Context
	tofetch        *urlQueue
	activePeers    map[proto.NodeID]struct{}
	activePeersArr []proto.NodeID

	f *Frontera 
	incoming     chan bidRecv

	newReqs      chan []string 
	cancelUrls   chan []string
	interestReqs chan interestReq

	interest  *lru.Cache
	liveWants map[string]time.Time

	tick          *time.Timer
	baseTickDelay time.Duration

	latTotal time.Duration
	fetchcnt int

	//notif notifications.PubSub
	//uuid logging.Loggable
	//id  uint64
	//tag string
}

// NewDomain create a single domain instance using config.
func NewDomain(cfg *DomainConfig, f *Frontera, peers *proto.Peers, genesis *types.Block) (domain *Domain, err error) {
	log.Infof("NewDomain(10)")

	// ensure dir exists
	if err = os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return
	}

	log.Infof("NewDomain(20)")
	if peers == nil || genesis == nil {
		err = ErrInvalidDBConfig
		return
	}

	log.Infof("NewDomain(30)")
	// get private key
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	log.Infof("NewDomain(40)")
	var accountAddr proto.AccountAddress
	if accountAddr, err = crypto.PubKeyHash(privateKey.GetPublic()); err != nil {
		return
	}

	log.Infof("NewDomain(50)")
	// init database
	domain = &Domain{
		cfg:            cfg,
		domainID:           cfg.DomainID,
		mux:            cfg.KayakMux,
		connSeqEvictCh: make(chan uint64, 1),

		//todo, wyong, 20200930 
		privateKey:     privateKey,
		accountAddr:    accountAddr,

		//wyong, 20200805 
		activePeers:   make(map[proto.NodeID]struct{}),

		//todo, why 64? wyong, 20200825 
		//activePeersArr :make([]proto.NodeID, 64 ),  


		//wyong, 20201018 
		host: cfg.Host,

		liveWants:     make(map[string]time.Time),
		newReqs:       make(chan []string),
		cancelUrls:    make(chan []string),
		tofetch:       newUrlQueue(),
		interestReqs:  make(chan interestReq),
		//ctx:           ctx,

		//wyong, 20200825 
		f:            f,

		incoming:      make(chan bidRecv),
		//notif:         notifications.New(),
		//uuid:          loggables.Uuid("GetBlockRequest"),
		baseTickDelay: time.Millisecond * 500,
		//id:            f.getNextSessionID(),
	}

	//bugfix, wyong, 20200825 
	cache, _ := lru.New(2048)
	domain.interest = cache 

	//wyong, 20200825 
	for _, p := range peers.Servers {
		domain.addActivePeer(p)
	}

	defer func() {
		// on error recycle all resources
		if err != nil {
			// stop kayak runtime
			if domain.kayakRuntime != nil {
				domain.kayakRuntime.Shutdown()
			}

			// close chain
			if domain.chain != nil {
				domain.chain.Stop()
			}
		}
	}()

	log.Infof("NewDomain(60)")
	// init storage
	storageFile := filepath.Join(cfg.DataDir, StorageFileName)
	storageDSN, err := storage.NewDSN(storageFile)
	if err != nil {
		return
	}

	log.Infof("NewDomain(70)")
	if cfg.EncryptionKey != "" {
		storageDSN.AddParam("_crypto_key", cfg.EncryptionKey)
	}

	log.Infof("NewDomain(80)")
	// init chain
	chainFile := filepath.Join(cfg.RootDir, SQLChainFileName)
	if domain.nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	log.Infof("NewDomain(90)")
	chainCfg := &urlchain.Config{
		DomainID:      cfg.DomainID,
		ChainFilePrefix: chainFile,
		DataFile:        storageDSN.Format(),
		Genesis:         genesis,
		Peers:           peers,

		// currently sqlchain package only use Server.ID as node id
		MuxService: cfg.ChainMux,
		Server:     domain.nodeID,

		Period:            conf.GConf.SQLChainPeriod,
		Tick:              conf.GConf.SQLChainTick,
		QueryTTL:          conf.GConf.SQLChainTTL,

		LastBillingHeight: cfg.LastBillingHeight,
		UpdatePeriod:      cfg.UpdateBlockCount,
		IsolationLevel:    cfg.IsolationLevel,
	}
	if domain.chain, err = urlchain.NewChain(chainCfg); err != nil {
		return
	}

	log.Infof("NewDomain(100)")
	if err = domain.chain.Start(); err != nil {
		return
	}

	log.Infof("NewDomain(110)")
	//wyong, 20200817
	if err = domain.InitTable(); err != nil {
		return
	}
	

	/*
	//just for test, wyong, 20200821 
        hexcid := "f015512209d8453505bdc6f269678e16b3e56c2a2948a41f2c792617cc9611ed363c95b63"
        c, err := cid.Decode(hexcid)
        if err != nil {
		return 
        }
	log.Infof("NewDomain(116)")
	if err = domain.SetCid("http://127.0.0.1/index.html", c ); err != nil {
		return
	}
	*/	

	log.Infof("NewDomain(120)")
	// init kayak config
	kayakWalPath := filepath.Join(cfg.DataDir, KayakWalFileName)
	if domain.kayakWal, err = kl.NewLevelDBWal(kayakWalPath); err != nil {
		err = errors.Wrap(err, "init kayak log pool failed")
		return
	}

	log.Infof("NewDomain(130)")
	domain.kayakConfig = &kt.RuntimeConfig{
		Handler:          domain,
		PrepareThreshold: PrepareThreshold,
		CommitThreshold:  CommitThreshold,
		PrepareTimeout:   PrepareTimeout,
		CommitTimeout:    CommitTimeout,
		LogWaitTimeout:   LogWaitTimeout,
		Peers:            peers,
		Wal:              domain.kayakWal,
		NodeID:           domain.nodeID,

		//wyong, 20201018
		Host:		domain.host,

		InstanceID:       string(domain.domainID),
		ServiceName:      DomainKayakRPCName,
		ApplyMethodName:  DomainKayakApplyMethodName,
		FetchMethodName:  DomainKayakFetchMethodName,
	}

	// create kayak runtime
	if domain.kayakRuntime, err = kayak.NewRuntime(domain.kayakConfig); err != nil {
		return
	}

	log.Infof("NewDomain(140)")
	// register kayak runtime rpc
	domain.mux.register(domain.domainID, domain.kayakRuntime)

	log.Infof("NewDomain(150)")
	// start kayak runtime
	domain.kayakRuntime.Start()

	log.Infof("NewDomain(160)")
	// init sequence eviction processor
	go domain.evictSequences()

	log.Infof("NewDomain(170)")
	//todo, wyong, 20200806
	go domain.run(context.Background())
	
	log.Infof("NewDomain(180)")
	return
}

// UpdatePeers defines peers update query interface.
func (domain *Domain) UpdatePeers(peers *proto.Peers) (err error) {
	fmt.Printf("Domain/UpdatePeers(10)\n") 
	//wyong, 20200825 
	for _, p := range peers.Servers {
		fmt.Printf("Domain/UpdatePeers(20)\n") 
		domain.addActivePeer(p)
	}

	fmt.Printf("Domain/UpdatePeers(30)\n") 
	if err = domain.kayakRuntime.UpdatePeers(peers); err != nil {
		return
	}

	fmt.Printf("Domain/UpdatePeers(40)\n") 
	return domain.chain.UpdatePeers(peers)
}

//wyong, 20200817 
func (domain *Domain) InitTable() (err error) {

	// build query 
	query := types.Query {
		Pattern : "CREATE TABLE urlgraph (url TEXT PRIMARY KEY NOT NULL, cid CHAR(64))",
	}

        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,

				//todo, wyong, 20200817 
                                //ConnectionID: connID,
                                //SeqNo:        seqNo,
                                //Timestamp:    getLocalTime(),
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	request.SetContext(context.Background())
	if _, _, err := domain.chain.Query(request, true); err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
	}

	return 
}


//wyong, 20200821
func (domain *Domain) SetCid( url string, c cid.Cid) (err error) {

	fmt.Printf("Domain/SetCid(10), url=%s\n", url) 
	// build query 
	q := fmt.Sprintf( "INSERT INTO urlgraph VALUES('%s', '%s')", url, c.String()) 
	query := types.Query {
		Pattern : q ,
	}

	fmt.Printf("Domain/SetCid(20), q=%s\n", q ) 
        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,

				//todo, wyong, 20200817 
                                //ConnectionID: connID,
                                //SeqNo:        seqNo,
                                //Timestamp:    getLocalTime(),
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	fmt.Printf("Domain/SetCid(30)\n") 
	request.SetContext(context.Background())
	_, _, err = domain.chain.Query(request, true)
	if  err != nil {
		fmt.Printf("Domain/SetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	fmt.Printf("Domain/SetCid(40)\n" ) 
	return 
}

//wyong, 20200817
func (domain *Domain) GetCid( url string) (c cid.Cid, err error) {

	fmt.Printf("Domain/GetCid(10), url=%s\n", url) 
	// build query 
	q := fmt.Sprintf( "SELECT cid FROM urlgraph where url='%s'" , url ) 
	query := types.Query {
		Pattern : q ,
	}

	fmt.Printf("Domain/GetCid(20), q=%s\n", q ) 
        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.ReadQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,

				//todo, wyong, 20200817 
                                //ConnectionID: connID,
                                //SeqNo:        seqNo,
                                //Timestamp:    getLocalTime(),
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	fmt.Printf("Domain/GetCid(30)\n") 
	request.SetContext(context.Background())
	_, resp, err := domain.chain.Query(request, true)
	if  err != nil {
		fmt.Printf("Domain/GetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	fmt.Printf("Domain/GetCid(40)\n") 
	//result process , wyong, 20200817 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		return 
	}

	fmt.Printf("Domain/GetCid(50)\n") 
	fr := rows[0]		//first row
	if len(fr.Values) <=0 {
		return 
	} 

	fmt.Printf("Domain/GetCid(60)\n") 
	fcfr:= fr.Values[0].(string) 	//first column of first row 

	fmt.Printf("Domain/GetCid(70), raw cid =%s\n", fcfr ) 
	c, err  = cid.Decode(fcfr)	//convert it to cid 
	if  err != nil {
		fmt.Printf("Domain/GetCid(75)\n") 
		return 
	}

	fmt.Printf("Domain/GetCid(80), cid = %s\n", c.String()) 
	return 
}

// Query defines database query interface.
func (domain *Domain) Query(request *types.Request) (response *types.Response, err error) {
	// Just need to verify signature in domain.saveAck
	//if err = request.Verify(); err != nil {
	//	return
	//}

	var (
		isSlowQuery uint32
		tracker     *x.QueryTracker
		tmStart     = time.Now()
	)

	// log the query if the underlying storage layer take too long to response
	slowQueryTimer := time.AfterFunc(domain.cfg.SlowQueryTime, func() {
		// mark as slow query
		atomic.StoreUint32(&isSlowQuery, 1)
		domain.logSlow(request, false, tmStart)
	})
	defer slowQueryTimer.Stop()
	defer func() {
		if atomic.LoadUint32(&isSlowQuery) == 1 {
			// slow query
			domain.logSlow(request, true, tmStart)
		}
	}()

	switch request.Header.QueryType {
	case types.ReadQuery:
		if tracker, response, err = domain.chain.Query(request, false); err != nil {
			err = errors.Wrap(err, "failed to query read query")
			return
		}
	case types.WriteQuery:
		if domain.cfg.UseEventualConsistency {
			// reset context
			request.SetContext(context.Background())
			if tracker, response, err = domain.chain.Query(request, true); err != nil {
				err = errors.Wrap(err, "failed to execute with eventual consistency")
				return
			}
		} else {
			if tracker, response, err = domain.writeQuery(request); err != nil {
				err = errors.Wrap(err, "failed to execute")
				return
			}
		}
	default:
		// TODO(xq262144): verbose errors with custom error structure
		return nil, errors.Wrap(ErrInvalidRequest, "invalid query type")
	}

	//todo, wyong, 20200930 
	//response.Header.ResponseAccount = domain.accountAddr

	// build hash
	if err = response.BuildHash(); err != nil {
		err = errors.Wrap(err, "failed to build response hash")
		return
	}

	if err = domain.chain.AddResponse(&response.Header); err != nil {
		log.WithError(err).Debug("failed to add response to index")
		return
	}
	tracker.UpdateResp(response)

	return
}

func (domain *Domain) logSlow(request *types.Request, isFinished bool, tmStart time.Time) {
	if request == nil {
		return
	}

	// sample the queries
	querySample := ""

	for _, q := range request.Payload.Queries {
		if len(querySample) < SlowQuerySampleSize {
			querySample += "; "
			querySample += q.Pattern
		} else {
			break
		}
	}

	if len(querySample) >= SlowQuerySampleSize {
		querySample = querySample[:SlowQuerySampleSize-3]
		querySample += "..."
	}

	log.WithFields(log.Fields{
		"finished": isFinished,
		"domain":       request.Header.DomainID,
		"req_time": request.Header.Timestamp.String(),
		"req_node": request.Header.NodeID,
		"count":    request.Header.BatchCount,
		"type":     request.Header.QueryType.String(),
		"sample":   querySample,
		"start":    tmStart.String(),
		"elapsed":  time.Now().Sub(tmStart).String(),
	}).Error("slow query detected")
}

// Ack defines client response ack interface.
func (domain *Domain) Ack(ack *types.Ack) (err error) {
	// Just need to verify signature in domain.saveAck
	//if err = ack.Verify(); err != nil {
	//	return
	//}

	return domain.saveAck(&ack.Header)
}

// Shutdown stop database handles and stop service the database.
func (domain *Domain) Shutdown() (err error) {
	if domain.kayakRuntime != nil {
		// shutdown, stop kayak
		if err = domain.kayakRuntime.Shutdown(); err != nil {
			return
		}

		// unregister
		domain.mux.unregister(domain.domainID)
	}

	if domain.kayakWal != nil {
		// shutdown, stop kayak
		domain.kayakWal.Close()
	}

	if domain.chain != nil {
		// stop chain
		if err = domain.chain.Stop(); err != nil {
			return
		}
	}

	if domain.connSeqEvictCh != nil {
		// stop connection sequence evictions
		select {
		case _, ok := <-domain.connSeqEvictCh:
			if ok {
				close(domain.connSeqEvictCh)
			}
		default:
			close(domain.connSeqEvictCh)
		}
	}

	return
}

// Destroy stop database instance and destroy all data/meta.
func (domain *Domain) Destroy() (err error) {
	if err = domain.Shutdown(); err != nil {
		return
	}

	// TODO(xq262144): remove database files, now simply remove whole root dir
	os.RemoveAll(domain.cfg.DataDir)

	return
}

func (domain *Domain) writeQuery(request *types.Request) (tracker *x.QueryTracker, response *types.Response, err error) {
	// check database size first, wal/kayak/chain database size is not included
	if domain.cfg.SpaceLimit > 0 {
		path := filepath.Join(domain.cfg.DataDir, StorageFileName)
		var statInfo os.FileInfo
		if statInfo, err = os.Stat(path); err != nil {
			if !os.IsNotExist(err) {
				return
			}
		} else {
			if uint64(statInfo.Size()) > domain.cfg.SpaceLimit {
				// rejected
				err = ErrSpaceLimitExceeded
				return
			}
		}
	}

	// call kayak runtime Process
	var result interface{}
	if result, _, err = domain.kayakRuntime.Apply(request.GetContext(), request); err != nil {
		err = errors.Wrap(err, "apply failed")
		return
	}

	var (
		tr *TrackerAndResponse
		ok bool
	)
	if tr, ok = (result).(*TrackerAndResponse); !ok {
		err = errors.Wrap(err, "invalid response type")
		return
	}
	tracker = tr.Tracker
	response = tr.Response
	return
}

func (domain *Domain) saveAck(ackHeader *types.SignedAckHeader) (err error) {
	return domain.chain.VerifyAndPushAckedQuery(ackHeader)
}

func getLocalTime() time.Time {
	return time.Now().UTC()
}

type bidRecv struct {
	from proto.NodeID
	url string 
}

func (domain *Domain) receiveBidFrom(from proto.NodeID,  url string  ) {
	//log.Debug("receiveBidFrom called...")
	select {
	case domain.incoming <- bidRecv{from: from, url: url }:
	
	//wyong, 20200824 
	//case <-domain.ctx.Done():
	}
}

type interestReq struct {
	url   string  
	resp chan bool
}

// TODO: PERF: this is using a channel to guard a map access against race
// conditions. This is definitely much slower than a mutex, though its unclear
// if it will actually induce any noticeable slowness. This is implemented this
// way to avoid adding a more complex set of mutexes around the liveWants map.
// note that in the average case (where this session *is* interested in the
// block we received) this function will not be called, as the cid will likely
// still be in the interest cache.
func (domain *Domain) isLiveWant(url string) bool {
	resp := make(chan bool, 1)
	select {
	case domain.interestReqs <- interestReq{
		url: url,
		resp: resp,
	}:

	//wyong, 20200824 
	//case <-domain.ctx.Done():
	//	return false
	}

	select {
	case want := <-resp:
		return want

	//wyong, 20200824 
	//case <-domain.ctx.Done():
	//	return false
	}
}

func (domain *Domain) interestedIn(url string ) bool {
	return domain.interest.Contains(url) || domain.isLiveWant(url)
}

const provSearchDelay = time.Second * 10


//wyong, 20200825 
func (domain *Domain) addActivePeer(p proto.NodeID) {
	fmt.Printf("Domain/addActivePeer(10), p=%s\n", p) 
	if _, ok := domain.activePeers[p]; !ok {
		fmt.Printf("Domain/addActivePeer(20)\n") 
		domain.activePeers[p] = struct{}{}
		domain.activePeersArr = append(domain.activePeersArr, p)

		// todo, wyong, 20200730 
		//cmgr := s.f.network.ConnectionManager()
		//cmgr.TagPeer(p, s.tag, 10)
	}
	fmt.Printf("Domain/addActivePeer(30)\n") 
}

func (domain *Domain) resetTick() {
	if domain.latTotal == 0 {
		domain.tick.Reset(provSearchDelay)
	} else {
		avLat := domain.latTotal / time.Duration(domain.fetchcnt)
		domain.tick.Reset(domain.baseTickDelay + (3 * avLat))
	}
}

func (domain *Domain) run(ctx context.Context) {
	domain.tick = time.NewTimer(provSearchDelay)
	newpeers := make(chan proto.NodeID, 16)
	for {
		select {
		case bid := <-domain.incoming:
			fmt.Printf("Domain/run(10), bid := <- s.incoming, bid.url = %s\n", bid.url )
			domain.tick.Stop()

			if bid.from != "" {
				domain.addActivePeer(bid.from)
			}

			//wyong, 20200828 
			bidding, exist := domain.f.wm.completed_wantlist.Contains(bid.url) 
			if exist == true { 
				domain.SetCid(bidding.GetUrl(), bidding.GetCid())
			}

			domain.receiveBid(ctx, bid.url )
			domain.resetTick()


		case urls := <-domain.newReqs:
			fmt.Printf("Domain/run(20), urls := <- domain.newReqs\n" )

			for _, url := range urls {
				fmt.Printf("Domain/run(30), add url(%s) into interest\n", url)  
				domain.interest.Add(url, nil)
			}

			fmt.Printf("Domain/run(40), domain.liveWants=%d, activeWantsLimit=%d\n", domain.liveWants, activeWantsLimit )  
			if len(domain.liveWants) < activeWantsLimit {
				toadd := activeWantsLimit - len(domain.liveWants)
				fmt.Printf("Domain/run(50), toadd = %d\n", toadd )  
				if toadd > len(urls) {
					toadd = len(urls)
				}

				now := urls[:toadd]
				urls = urls[toadd:]

				fmt.Printf("Domain/run(60), now set = %s\n", now )  
				domain.wantUrls(ctx, now)
			}

			fmt.Printf("Domain/run(70)\n")  
			for _, url := range urls {
				fmt.Printf("Domain/run(80), add url(%s) into tofetch \n", url)  
				domain.tofetch.Push(url)
			}
			fmt.Printf("Domain/run(80)\n")  

		case urls := <-domain.cancelUrls:
			fmt.Printf("domain/run, urls := <- domain.cancelUrls \n" )
			domain.cancel(urls)

		case <-domain.tick.C:
			fmt.Printf("domain/run,<- domain.tick.C\n" )
			live := make([]string, 0, len(domain.liveWants))
			now := time.Now()
			for url := range domain.liveWants {
				live = append(live, url)
				domain.liveWants[url] = now
			}

			//todo, wyong, 20200813 
			// Broadcast these keys to everyone we're connected to
			//domain.f.wm.WantUrls(ctx, live, nil, domain.domainID)

			/*TODO, review the following code is necessary, wyong, 20190119
			if len(live) > 0 {
				go func(k string ) {
					// TODO: have a task queue setup for this to:
					// - rate limit
					// - manage timeouts
					// - ensure two 'findprovs' calls for the same block don't run concurrently
					// - share peers between sessions based on interest set
					for p := range domain.f.network.FindProvidersAsync(ctx, k, 10) {
						newpeers <- p
					}
				}(live[0])
			}
			*/
			domain.resetTick()
		case p := <-newpeers:
			//log.Debug("session/run, p := <-newpeers " )
			domain.addActivePeer(p)

		case lwchk := <-domain.interestReqs:
			//log.Debug("session/run, lwchk := <-domain.interestReqs" )
			lwchk.resp <- domain.urlIsWanted(lwchk.url)

		case <-ctx.Done():
			//log.Debug("session/run, <-ctx.Done()" )
			domain.tick.Stop()

			/*TODO, hold the session for later use, wyong, 20190124
			domain.f.removeSession(s)

			cmgr := domain.f.network.ConnectionManager()
			for _, p := range domain.activePeersArr {
				cmgr.UntagPeer(p, domain.tag)
			}
			*/

			return
		}
	}
}

func (domain *Domain) urlIsWanted(url string) bool {
	_, ok := domain.liveWants[url]
	if !ok {
		ok = domain.tofetch.Has(url)
	}

	return ok
}

func (domain *Domain) receiveBid(ctx context.Context, url string ) {
	//log.Debug("receiveBid called..." )
	//url := bid.Url()

	if domain.urlIsWanted(url) {
		tval, ok := domain.liveWants[url]
		if ok {
			domain.latTotal += time.Since(tval)
			delete(domain.liveWants, url)
		} else {
			domain.tofetch.Remove(url)
		}
		domain.fetchcnt++
		//domain.notif.Publish(url)

		if next := domain.tofetch.Pop(); /* next.Defined() */ next != ""  {
			domain.wantUrls(ctx, []string{next})
		}
	}
}

func (domain *Domain) wantUrls(ctx context.Context, urls []string ) {
	fmt.Printf("Domain/wantUrls(10)\n") 

	now := time.Now()
	for _, url := range urls {
		domain.liveWants[url] = now
	}

	//todo, select 2~3 random selected peers, wyong, 20200805 
	domain.f.wm.WantUrls(ctx, urls, domain.activePeersArr, /* todo, wyong, 20200825 []proto.NodeID{},*/ domain.domainID)
}

func (domain *Domain) cancel(urls []string ) {
	for _, url := range urls {
		domain.tofetch.Remove(url)
	}
}

func (domain *Domain) cancelWants(urls []string ) {
	select {
	case domain.cancelUrls <- urls:

	//wyong, 20200824 
	//case <-domain.ctx.Done():
	}
}

/* wyong, 20200806 
func (domain *Domain) put(ctx context.Context, urls []string ) {
	select {
	case domain.newReqs <- urls:
	case <-ctx.Done():
	case <-domain.ctx.Done():
	}
}
*/

// GetBlock fetches a single block
func (domain *Domain) PutBidding(ctx context.Context, url string, probability float64 ) (/* blocks.Block, */  error) {
	//log.Debug("PutBidding called")
	//return putBidding(ctx, url, domain.PutBiddings)
	//return domain.put(ctx, []string{url}) 
	select {
		case domain.newReqs <- []string{url}:
		case <-ctx.Done():
		
		//wyong, 20200824 
		//case <-domain.ctx.Done():
	}
	return nil 
}

type urlQueue struct {
	elems []string 
	//eset  *cid.Set
	eset map[string]struct{} 
}

func newUrlQueue() *urlQueue {
	//return &cidQueue{eset: cid.NewSet()}
	return &urlQueue{eset:  make(map[string]struct{})}
}

func (uq *urlQueue) Pop() string {
	for {
		if len(uq.elems) == 0 {
			return "" 
		}

		out := uq.elems[0]
		uq.elems = uq.elems[1:]

		//if cq.eset.Has(out) {
		_, has := uq.eset[out]
		if has {
			//cq.eset.Remove(out)
			delete(uq.eset, out)
			return out
		}
	}
}


func (uq *urlQueue) Push(u string) {
	//if uq.eset.Visit(u) {
	//	uq.elems = append(uq.elems, u)
	//}

	_, has := uq.eset[u] 
	if !has {
		uq.eset[u]=struct{}{}
		uq.elems = append(uq.elems, u)
	}

}

func (uq *urlQueue) Remove(u string) {
	//cq.eset.Remove(c)
	delete(uq.eset, u) 
}

func (uq *urlQueue) Has(u string) bool {
	_, has := uq.eset[u]
	return has 
}

func (uq *urlQueue) Len() int {
	//return cq.eset.Len()
	return len(uq.eset)
}
