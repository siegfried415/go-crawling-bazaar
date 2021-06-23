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
	//"sync/atomic"
	"time"
	//"fmt"

	//wyong, 20210125 
	//"sort" 

	//wyong, 20210528 
	//"github.com/pkg/errors"

	//wyong, 20201018
	//"github.com/libp2p/go-libp2p-core/host"

	//wyong, 20200929 
	"github.com/siegfried415/go-crawling-bazaar/conf"


	//wyong, 20201002 
	"github.com/siegfried415/go-crawling-bazaar/crypto" 
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/kms"

	/* wyong, 20210528 
	"github.com/siegfried415/go-crawling-bazaar/kayak"
	kt "github.com/siegfried415/go-crawling-bazaar/kayak/types"
	kl "github.com/siegfried415/go-crawling-bazaar/kayak/wal"
	*/

	"github.com/siegfried415/go-crawling-bazaar/proto"
	urlchain "github.com/siegfried415/go-crawling-bazaar/urlchain"
	"github.com/siegfried415/go-crawling-bazaar/storage"
	"github.com/siegfried415/go-crawling-bazaar/types"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	//s "github.com/siegfried415/go-crawling-bazaar/state"
	net "github.com/siegfried415/go-crawling-bazaar/net"

	//wyong, 20200805 
	lru "github.com/hashicorp/golang-lru"

	//wyong, 20200818
	cid "github.com/ipfs/go-cid"

	//wyong, 20210122
        cache "github.com/patrickmn/go-cache"

)

const (
	// StorageFileName defines storage file name of database instance.
	StorageFileName = "storage.db3"

	//wyong, 20210528 
	// KayakWalFileName defines log pool name of database instance.
	//KayakWalFileName = "kayak.ldb"

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
	activeBiddingsLimit = 16

	//wyong, 20210205 
	MinCrawlersExpected	= 2 
	IncrementalCrawlingInterval uint32 = 100
	MinFrowardProbability	float64 = 0.2 
)

// Database defines a single database instance in worker runtime.
type Domain struct {
	cfg            *DomainConfig
	domainID           proto.DomainID

	//wyong, 20210528 
	//kayakWal       *kl.LevelDBWal
	//kayakRuntime   *kayak.Runtime
	//kayakConfig    *kt.RuntimeConfig

	connSeqs       sync.Map

	//wyong, 20210528 
	//connSeqEvictCh chan uint64

	chain          *urlchain.Chain
	nodeID         proto.NodeID

	//wyong, 20201018
	host 		net.RoutedHost

	//wyong, 20210528 
	//mux            *DomainKayakMuxService

	privateKey     *asymmetric.PrivateKey
	accountAddr    proto.AccountAddress

	//wyong, move session stuff here,  20200805 
	ctx            context.Context
	tofetch        *ProbBiddingQueue
	activePeers    map[proto.NodeID]struct{}
	activePeersArr []proto.NodeID

	//wyong, 20210129
	waitQueue	*TimedBiddingQueue

	f *Frontera 

	//wyong, 20210127 
	//incoming     chan bidRecv
	bidIncoming	chan bidRecv 

	//wyong, 20210125 
	//newReqs      chan []string 
	biddingReqs	chan[]types.UrlBidding	

	cancelBiddings   chan []string
	interestReqs chan interestReq

	interest  *lru.Cache
	liveBiddings map[string]time.Time

	tick          *time.Timer
	baseTickDelay time.Duration

	latTotal time.Duration
	fetchcnt int

	//notif notifications.PubSub
	//uuid logging.Loggable
	//id  uint64
	//tag string

	//wyong, 20210122 
	urlCidCache 	*cache.Cache 
	urlGraphCache	*cache.Cache
	urlNodeCache	*cache.Cache 
	
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

		//wyong, 20210528 
		//mux:            cfg.KayakMux,

		//wyong, 20210528 
		//connSeqEvictCh: make(chan uint64, 1),

		//todo, wyong, 20200930 
		privateKey:     privateKey,
		accountAddr:    accountAddr,

		//wyong, 20200805 
		activePeers:   make(map[proto.NodeID]struct{}),

		//todo, why 64? wyong, 20200825 
		//activePeersArr :make([]proto.NodeID, 64 ),  


		//wyong, 20201018 
		host: cfg.Host,

		liveBiddings:     make(map[string]time.Time),

		//wyong, 20210125 
		//newReqs:       make(chan []string),
		biddingReqs:	make(chan []types.UrlBidding),

		cancelBiddings:    make(chan []string),
		tofetch:       newProbBiddingQueue(),
		interestReqs:  make(chan interestReq),
		//ctx:           ctx,

		//wyong, 20210129
		waitQueue	: newTimedBiddingQueue(), 

		//wyong, 20200825 
		f:            f,

		//wyong, 20210127 
		//incoming:      make(chan bidRecv),
		bidIncoming:      make(chan bidRecv),

		//notif:         notifications.New(),
		//uuid:          loggables.Uuid("GetBlockRequest"),
		baseTickDelay: time.Millisecond * 500,
		//id:            f.getNextSessionID(),
	}

	//todo, get expiration time from urlchain miner time,  wyong, 20210122 
	domain.urlCidCache = cache.New(5*time.Minute, 10*time.Minute)
	domain.urlGraphCache = cache.New(5*time.Minute, 10*time.Minute)
	domain.urlNodeCache = cache.New(5*time.Minute, 10*time.Minute)

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
			/* wyong, 20210528 
			// stop kayak runtime
			if domain.kayakRuntime != nil {
				domain.kayakRuntime.Shutdown()
			}
			*/

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

		//bugfix, wyong, 20201203
		Host: 		cfg.Host, 

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
	if err = domain.InitTables(); err != nil {
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


	/* wyong, 20210528 
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
	*/


	/* wyong, 20210528 
	log.Infof("NewDomain(160)")
	// init sequence eviction processor
	go domain.evictSequences()
	*/


	log.Infof("NewDomain(170)")
	//todo, wyong, 20200806
	go domain.run(context.Background())
	
	log.Infof("NewDomain(180)")
	return
}

// UpdatePeers defines peers update query interface.
func (domain *Domain) UpdatePeers(peers *proto.Peers) (err error) {
	log.Debugf("Domain/UpdatePeers(10)\n") 
	//wyong, 20200825 
	for _, p := range peers.Servers {
		log.Debugf("Domain/UpdatePeers(20)\n") 
		domain.addActivePeer(p)
	}

	/* wyong, 20210528 
	log.Debugf("Domain/UpdatePeers(30)\n") 
	if err = domain.kayakRuntime.UpdatePeers(peers); err != nil {
		return
	}
	*/

	log.Debugf("Domain/UpdatePeers(40)\n") 
	return domain.chain.UpdatePeers(peers)
}


// Shutdown stop database handles and stop service the database.
func (domain *Domain) Shutdown() (err error) {
	/* wyong, 20210528 
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
	*/

	if domain.chain != nil {
		// stop chain
		if err = domain.chain.Stop(); err != nil {
			return
		}
	}

	/* wyong, 20210528 
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
	*/

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

type bidRecv struct {
	from proto.NodeID
	url string 
}

func (domain *Domain) receiveBidFrom(from proto.NodeID,  url string  ) {
	//log.Debug("receiveBidFrom called...")
	select {
	case domain.bidIncoming <- bidRecv{from: from, url: url }:
	
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
// way to avoid adding a more complex set of mutexes around the liveBiddings map.
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
	log.Debugf("Domain/addActivePeer(10), p=%s\n", p) 
	if _, ok := domain.activePeers[p]; !ok {
		log.Debugf("Domain/addActivePeer(20)\n") 
		domain.activePeers[p] = struct{}{}
		domain.activePeersArr = append(domain.activePeersArr, p)

		// todo, wyong, 20200730 
		//cmgr := s.f.network.ConnectionManager()
		//cmgr.TagPeer(p, s.tag, 10)
	}
	log.Debugf("Domain/addActivePeer(30)\n") 
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
		case biddings := <-domain.biddingReqs:	// <-domain.newReqs: 
			log.Debugf("Domain/run(10), urls := <- domain.biddingReqs\n" )

			//for _, url := range urls {
			//	log.Debugf("Domain/run(30), add url(%s) into interest\n", url)  
			//	domain.interest.Add(url, nil)
			//}

			log.Debugf("Domain/run(20), domain.liveBiddings=%d, activeBiddingsLimit=%d\n", domain.liveBiddings, activeBiddingsLimit )  

			for _, bidding := range biddings { 
				//when forwad probability of this url is zero, get parent of the node, 
				//caculate forward probability of this url, 

				log.Debugf("Domain/run(30), check bidding(%s) 's probability \n", bidding.Url )

				if bidding.Probability == 0.0 { 
					log.Debugf("Domain/run(31), bidding(%s) 's probability is zero\n", bidding.Url )
					bidding.Probability, _ = domain.GetProbability(bidding.ParentUrl, bidding.ParentProbability,  bidding.Url ) 
					log.Debugf("Domain/run(32), bidding(%s) 's probability is set to %d\n", bidding.Url, bidding.Probability )
				}
			}

			log.Debugf("Domain/run(40)\n")

			//insert bidding in biddings accroding to their probability. 
			domain.tofetch.FastInserts(biddings) 

			log.Debugf("Domain/run(50)\n")

			//wyong, 20210203 
			addBiddings := make([]types.UrlBidding, 0, len(biddings))

			log.Debugf("Domain/run(60)\n")
			for {
				log.Debugf("Domain/run(70)\n")
				if (len(domain.liveBiddings) >= activeBiddingsLimit) {
					break 
				}

				log.Debugf("Domain/run(80)\n")
				bidding, err := domain.tofetch.Pop()
				if  err != nil  {
					break 
				}

				log.Debugf("Domain/run(90), Pop bidding(%s)\n", bidding.Url )
				//currentHeight is int32, wyong, 20210205 	
				currentHeight := uint32(domain.chain.GetCurrentHeight())

				log.Debugf("Domain/run(100), currentHeigh = %d\n", currentHeight )
				//get or create url node from url chain
				urlNode, err := domain.GetUrlNode(bidding.Url)	
				if err != nil || urlNode == nil {
					urlNode, err = domain.CreateUrlNode(/* parentUrl, */ bidding.Url, currentHeight, 0, 0, 0 )	
					if err != nil || urlNode == nil {
						continue 
					}
				}

				log.Debugf("Domain/run(110), urlNode =%s \n", urlNode )
				//read last requested height, and filter out urls which crawled too fast, 
				if (urlNode.LastRequestedHeight + IncrementalCrawlingInterval) < currentHeight { 
					continue 
				} 

				log.Debugf("Domain/run(120)\n")
				if bidding.Probability <= MinFrowardProbability { 
					continue 
				}

				log.Debugf("Domain/run(130)\n")
				//todo, increment requested count and save to url chain,wyong, 20210126  
				domain.SetLastRequestedHeight(bidding.Url, currentHeight)

				log.Debugf("Domain/run(130)\n")

				addBiddings = append(addBiddings, bidding) 
				log.Debugf("Domain/run(140)\n")
			}

			log.Debugf("Domain/run(150)\n")  
			domain.addBiddings(ctx, addBiddings )

			log.Debugf("Domain/run(160)\n")  

		case bid := <-domain.bidIncoming:
			log.Debugf("Domain/run(200), bid := <- s.bidIncoming, bid.url = %s\n", bid.url )
			domain.tick.Stop()

			if bid.from != "" {
				domain.addActivePeer(bid.from)
			}

			log.Debugf("Domain/run(210)\n")
			//wyong, 20200828 
			bidding, exist := domain.f.bc.completed_bl.Contains(bid.url) 
			if exist == true { 
				
				log.Debugf("Domain/run(220)\n")
				domain.BiddingCompleted(ctx, bid.url )

				//Move back, wyong, 20210220 
				//this has been moved to crawler side , wyong, 20210129 
				c, _ := bidding.GetCid() 
				domain.SetCid(bidding.GetUrl(), c )

				log.Debugf("Domain/run(230)\n")

			}

			//todo, wyong, 20210129 
			//domain.resetTick()

		case biddings := <-domain.cancelBiddings:
			log.Debugf("domain/run(300), urls := <- domain.cancelBiddings \n" )
			domain.cancel(biddings)

		case <-domain.tick.C:
			
			//todo, wyong, 20210205 
			biddings := make([]types.UrlBidding, 0 ) 

			//todo, move request ready to go from wait queue to domain.tofetch, wyong, 20210116 
			//for job := range domain.waitQueue {
			//while _, bidding := domain.waitQueue.Pop() { 
			for {
				_, bidding, err := domain.waitQueue.Pop() 
				if err != nil  {
					break 
				}

				//if job.runTime > now() {
				//	//wait queue is sorted , so break as soon as possible. 
				//	break
				//}

				//delete job from wait queue 
				biddings = append(biddings, bidding ) 
			}
			
			if len(biddings) > 0  {
				domain.PutBiddings(ctx, biddings)
			}

			//move re-broadcast to want manager, wyong, 20210117 
			//log.Debugf("domain/run,<- domain.tick.C\n" )
			//live := make([]string, 0, len(domain.liveBiddings))
			//now := time.Now()
			//for url := range domain.liveBiddings {
			//	live = append(live, url)
			//	domain.liveBiddings[url] = now
			//}
			//
			// Broadcast these keys to everyone we're connected to
			//domain.f.wm.WantUrls(ctx, live, nil, domain.domainID)

			//TODO, review the following code is necessary, wyong, 20190119
			//if len(live) > 0 {
			//	go func(k string ) {
			//		// TODO: have a task queue setup for this to:
			//		// - rate limit
			//		// - manage timeouts
			//		// - ensure two 'findprovs' calls for the same block don't run concurrently
			//		// - share peers between sessions based on interest set
			//		for p := range domain.f.network.FindProvidersAsync(ctx, k, 10) {
			//			newpeers <- p
			//		}
			//	}(live[0])
			//}

			//todo, make this tick for waitQueue, wyong, 20210117  
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
	_, ok := domain.liveBiddings[url]
	if !ok {
		ok = domain.tofetch.Has(url)
	}

	return ok
}

func (domain *Domain) BiddingCompleted(ctx context.Context, url string ) {
	//log.Debug("receiveBid called..." )
	log.Debugf("Domain/BiddingCompleted(10), url=%s\n", url )
	//url := bid.Url()

	if domain.urlIsWanted(url) {
		log.Debugf("Domain/BiddingCompleted(20)\n")
		tval, ok := domain.liveBiddings[url]
		if ok {
			log.Debugf("Domain/BiddingCompleted(30)\n")
			domain.latTotal += time.Since(tval)
			delete(domain.liveBiddings, url)
		} else {
			log.Debugf("Domain/BiddingCompleted(40)\n")
			domain.tofetch.Remove(url)
		}

		log.Debugf("Domain/BiddingCompleted(50)\n")
		urlNode, err := domain.GetUrlNode(url) 
		if err == nil {
			currentHeight := uint32(domain.chain.GetCurrentHeight()) 
			deltaHeight := currentHeight - urlNode.LastCrawledHeight
			newCrawlInterval := ( urlNode.CrawlInterval / 2 ) + ( deltaHeight / 2 )

			log.Debugf("Domain/BiddingCompleted(60), currentHeight=%d, deltaHeight=%d, newCrawlInterval=%d\n", currentHeight, deltaHeight, newCrawlInterval )
			domain.SetCrawlInterval(url, newCrawlInterval) 

			log.Debugf("Domain/BiddingCompleted(70)\n")
			//save last crawled height to url chain.
			domain.SetLastCrawledHeight(url, currentHeight) 

			log.Debugf("Domain/BiddingCompleted(80)\n")
			//create a bidding in future for this url and push it to wait queue
			runtime := time.Now().Add(time.Duration(urlNode.CrawlInterval) * time.Second)
			domain.waitQueue.Push(runtime, types.UrlBidding {
							Url : url,
							Probability : 1.0,      //todo
							ExpectCrawlerCount : 1, //todo
						})
			log.Debugf("Domain/BiddingCompleted(90)\n")

		}

		log.Debugf("Domain/BiddingCompleted(100)\n")
		domain.fetchcnt++
		//domain.notif.Publish(url)

		if next, err  := domain.tofetch.Pop();  err != nil   {
			log.Debugf("Domain/BiddingCompleted(110)\n")
			domain.addBiddings(ctx, []types.UrlBidding{next})
		}
		log.Debugf("Domain/BiddingCompleted(120)\n")

	}

}

//add parameter n, which indicate how many times the url should crawled at least, 
//if url was not blocked by this node,  move the url to local ledger, and crawl the other (n - 1) 
//times by remote peers , wyong, 20210117  
func (domain *Domain) addBiddings(ctx context.Context, /* urls []string */ biddings []types.UrlBidding) {

	/* wyong, 20210204 
        localBiddings := make([]types.UrlBidding , 0, len(urls))
        remoteBiddings := make([]types.UrlBidding , 0, len(urls))
        for _, bidding := range biddings {
		if !isBlockedUrl(bidding.Url) {
			localBiddings = append(localBiddings,  types.UrlBidding {
				//wyong, 20200827
				//Cancel: cancel,
				//BiddingEntry:  NewBiddingEntry(url, kMaxPriority-i),
				Url : bidding.Url,
				Probability : bidding.Probability,      
				ParentUrl   : bidding.ParentUrl, 
				ExpectCrawlerCount : 1,	//wyong, 20210117 
			})

			url.ExpectCrawlerCount = url.ExpectCrawlerCount - 1 
		}

		if url.ExpectCrawlerCount > 0 {
			remoteBiddings = append (remoteBiddings, types.UrlBidding{ 
				Url : bidding.Url,
				Probability : bidding.Probability,      
				ParentUrl   : bidding.ParentUrl,
				ExpectCrawlerCount : bidding.ExpectCrawlerCount,
			})
		}

		now := time.Now()
		domain.liveBiddings[url] = now
		
        }

	//put local biddings to engine.ledger to notify spider sub-system there are jobs to run. 
	if len(localBiddings) > 0 {
		domain.f.bs.UrlBiddingReceived(ctx, domain.nodeID, localBiddings )
	}
	
	//broadcast remoteBiddings to all active peers. 
	if len(remoteBiddings) > 0 {
		//now := time.Now()
		//for _, remoteBidding := range remoteBiddings {
		//	domain.liveBiddings[remoteBidding.Url] = now
		//}

		domain.f.bc.AddBiddings(ctx,  remoteBiddings, 
						domain.activePeersArr, 
						domain.domainID)
	}

	*/

        for _, bidding := range biddings {
		now := time.Now()
		domain.liveBiddings[bidding.Url] = now
	}

	if len(biddings) > 0 {
		domain.f.bc.AddBiddings(ctx,  biddings, domain.activePeersArr, domain.domainID)
	}

	//return 
}


func (domain *Domain) cancel(urls []string ) {
	for _, url := range urls {
		domain.tofetch.Remove(url)
	}
}

/* wyong, 20210126 
func (domain *Domain) cancelBiddings(urls []string ) {
	select {
	case domain.cancelUrls <- urls:

	//wyong, 20200824 
	//case <-domain.ctx.Done():
	}
}
*/

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
func (domain *Domain) PutBidding(ctx context.Context, url string, probability float64, parentUrl string , parentProbability float64  ) (/* blocks.Block, */  error) {
	//log.Debug("PutBidding called")
	//return putBidding(ctx, url, domain.PutBiddings)
	//return domain.put(ctx, []string{url}) 
	bidding := types.UrlBidding {
		Url: url,
		Probability : probability, 

		//wyong, 20210205 
		ParentUrl : parentUrl, 
		ParentProbability : parentProbability, 
		
		//wyong, 20210203 
		ExpectCrawlerCount : MinCrawlersExpected, 
	}

	select {
		case domain.biddingReqs <- []types.UrlBidding{ bidding }:  
		case <-ctx.Done():
		
		//wyong, 20200824 
		//case <-domain.ctx.Done():
	}
	return nil 
}

func (domain *Domain) PutBiddings(ctx context.Context, biddings []types.UrlBidding ) error {
	select {
		case domain.biddingReqs <- biddings :  
		case <-ctx.Done():
		//wyong, 20200824 
		//case <-domain.ctx.Done():
	}
	return nil 
}

func (domain *Domain) RetriveUrlCid(ctx context.Context, parentUrl string, url string  ) (cid.Cid, error) {
	log.Debugf("Domain/RetriveUrlCid(10), parentUrl=%s, url=%s\n", parentUrl, url ) 
	urlNode, err := domain.GetUrlNode(url)
	if err != nil {
		log.Debugf("Domain/RetriveUrlCid(15), err=%s\n", err.Error()) 
		return cid.Cid{}, nil 
	}

	log.Debugf("Domain/RetriveUrlCid(20), urlNode=%s\n", urlNode ) 
	c, err := domain.GetCid(url) 
	if err != nil {
		log.Debugf("Domain/RetriveUrlCid(25), err=%s\n", err.Error()) 
		return cid.Cid{}, nil 
	}

	log.Debugf("Domain/RetriveUrlCid(30), cid =%s\n", c.String() ) 
	err = domain.SetRetrivedCount(url, urlNode.RetrivedCount + 1 )
	if err != nil {
		log.Debugf("Domain/RetriveUrlCid(35), err=%s\n", err.Error()) 
		return cid.Cid{}, nil 
	}
	
	log.Debugf("Domain/RetriveUrlCid(40)\n") 
	// todo, wyong, 20210224 
	err = domain.AddUrlLinksCount(parentUrl, url ) 
	if err != nil {
		log.Debugf("Domain/RetriveUrlCid(45), err=%s\n", err.Error()) 
		return cid.Cid{}, nil 
	}

	log.Debugf("Domain/RetriveUrlCid(50)\n") 
	return c, nil 
}
