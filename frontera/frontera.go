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
	"bytes"
	"context"
	"expvar"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
	"math"
	//"fmt"

	//wyong, 20200831 
        url "net/url"


	"github.com/pkg/errors"

	//wyong, 20201020 
	//procctx "github.com/ipfs/goprocess/context"
	//procctx "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess/context"
	procctx "github.com/jbenet/goprocess/context" 


	//wyong, 20210203 
	//decision "github.com/siegfried415/go-crawling-bazaar/frontera/decision"

	//biddinglist "github.com/siegfried415/go-crawling-bazaar/frontera/biddinglist" 
	//bsmsg "github.com/siegfried415/go-crawling-bazaar/frontera/message" 
	//fnet "github.com/siegfried415/go-crawling-bazaar/frontera/network" 

	"github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
	"github.com/siegfried415/go-crawling-bazaar/conf"


	"github.com/siegfried415/go-crawling-bazaar/crypto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"

	//wyong, 20201002 
	//"github.com/siegfried415/go-crawling-bazaar/address" 
	"github.com/siegfried415/go-crawling-bazaar/kms"

	"github.com/siegfried415/go-crawling-bazaar/proto"
	//"github.com/siegfried415/go-crawling-bazaar/route"
	"github.com/siegfried415/go-crawling-bazaar/urlchain"
	"github.com/siegfried415/go-crawling-bazaar/types"
	"github.com/siegfried415/go-crawling-bazaar/utils"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	net "github.com/siegfried415/go-crawling-bazaar/net"

	process "github.com/jbenet/goprocess" 
	cid "github.com/ipfs/go-cid" 
	//peer "github.com/libp2p/go-libp2p-peer" 

	//"github.com/libp2p/go-libp2p-host"
)

const (
	// DomainKayakRPCName defines rpc service name of database internal consensus.
	DomainKayakRPCName = "DBC" // aka. database consensus

	// DBMetaFileName defines frontera meta file name.
	DBMetaFileName = "db.meta"

	// DefaultSlowQueryTime defines the default slow query log time
	DefaultSlowQueryTime = time.Second * 5

	mwMinerDBCount = "service:miner:db:count"

	//wyong, 20200721 
	// maxProvidersPerRequest specifies the maximum number of providers desired
	// from the network. This value is specified because the network streams
	// results.
	// TODO: if a 'non-nice' strategy is implemented, consider increasing this value
	maxProvidersPerRequest = 3
	findProviderDelay      = 1 * time.Second
	providerRequestTimeout = time.Second * 10
	provideTimeout         = time.Second * 15
	sizeBatchRequestChan   = 32
	// kMaxPriority is the max priority as defined by the biddingsys protocol
	kMaxPriority = math.MaxInt32
)

var (
	domainCount = new(expvar.Int)

	//wyong, 20200721 
	HasBlockBufferSize    = 256
	provideKeysBufferSize = 2048
	provideWorkerMax      = 512

	// the 1<<18+15 is to observe old file chunks that are 1<<18 + 14 in size
	metricsBuckets = []float64{1 << 6, 1 << 10, 1 << 14, 1 << 18, 1<<18 + 15, 1 << 22}
)

func init() {
	expvar.Publish(mwMinerDBCount, domainCount)
}

type counters struct {
        blocksRecvd    uint64
        dupBlocksRecvd uint64
        dupDataRecvd   uint64
        blocksSent     uint64
        dataSent       uint64
        dataRecvd      uint64
        messagesRecvd  uint64
}

// DBMS defines a database management instance.
type Frontera struct {
	cfg        *FronteraConfig
	domainMap      sync.Map
	kayakMux   *DomainKayakMuxService
	chainMux   *sqlchain.MuxService
	rpc        *FronteraRPCService
	busService *BusService
	address    proto.AccountAddress

	privKey    *asymmetric.PrivateKey

	//wyong, 20200820
	nodeID	    proto.NodeID 

	//wyong, 20200721 
	// the peermanager manages sending messages to peers in a way that
	// wont block biddingsys operation
	bc *BiddingClient 

	// the BiddingServer is the bit of logic that decides who to send which blocks to
	//engine *decision.Engine
	bs *BiddingServer 

	// network delivers messages on behalf of the session
	//network fnet.BiddingNetwork
	host net.RoutedHost 

	// blockstore is the local database
	// NB: ensure threadsafety
	//blockstore blockstore.Blockstore

	// notifications engine for receiving new blocks and routing them to the
	// appropriate user requests
	//notifications notifications.PubSub

        /* TODO, wyong, 20181219
	// findKeys sends keys to a worker to find and connect to providers for them
	findKeys chan *biddingRequest

	// newBlocks is a channel for newly added blocks to be provided to the
	// network.  blocks pushed down this channel get buffered and fed to the
	// provideKeys channel later on to avoid too much network activity
	newBlocks chan cid.Cid
	// provideKeys directly feeds provide workers
	provideKeys chan cid.Cid
        */


	process process.Process

	// Counters for various statistics
	counterLk sync.Mutex
	counters  *counters

	// Metrics interface metrics
	//dupMetric metrics.Histogram
	//allMetric metrics.Histogram

	/* wyong, 20200805 
	// Sessions
	sessions []*Session
	sessLk   sync.Mutex

	sessID   uint64
	sessIDLk sync.Mutex
	*/
}

// NewFrontera returns new database management instance.
func NewFrontera( cfg *FronteraConfig, peerHost net.RoutedHost  ) (f *Frontera, err error) {
	log.Infof("NewFrontera(10)") 

	//wyong, 20200721 
	// important to use provided parent context (since it may include important
	// loggable data). It's probably not a good idea to allow biddingsys to be
	// coupled to the concerns of the ipfs daemon in this way.
	//
	// FIXME(btc) Now that biddingsys manages itself using a process, it probably
	// shouldn't accept a context anymore. Clients should probably use Close()
	// exclusively. We should probably find another way to share logging data
	//ctx, cancelFunc := context.WithCancel(parent)
	ctx := context.Background()

	//ctx = metrics.CtxSubScope(ctx, "biddingsys")
	//dupHist := metrics.NewCtx(ctx, "recv_dup_blocks_bytes", "Summary of duplicate"+
	//	" data blocks recived").Histogram(metricsBuckets)
	//allHist := metrics.NewCtx(ctx, "recv_all_blocks_bytes", "Summary of all"+
	//	" data blocks recived").Histogram(metricsBuckets)

	//notif := notifications.New()

	//todo, wyong, 20201003 
	px := process.WithTeardown(func() error {
		//notif.Shutdown()
		return nil
	})

	log.Infof("NewFrontera(20)") 
	//wyong, 20200813
	nodeID, err := kms.GetLocalNodeID()
	if err != nil {
                log.WithError(err).Error("get local node id failed")
                return nil, err  
        }

	log.Infof("NewFrontera(30)") 
	f = &Frontera{
		cfg: cfg,

		//wyong, 20200721 
		//notifications: notif,

		//wyong, 20200820 
		nodeID: nodeID,

		//todo, network.NodeId()? bstore? wyong, 20200730 
		bs :        /* decision. */ NewBiddingServer (ctx, nodeID /*, bstore */ ), 

		//network:       network,
		host : peerHost, 

		//findKeys:      make(chan *biddingRequest, sizeBatchRequestChan),
		process:       px,
		//newBlocks:     make(chan cid.Cid, HasBlockBufferSize),
		//provideKeys:   make(chan cid.Cid, provideKeysBufferSize),

		//wyong, 20200924 
		bc :            NewBiddingClient(ctx, peerHost,  nodeID),

		counters:      new(counters),

		//dupMetric: dupHist,
		//allMetric: allHist,
	}

	//wyong, 20210203
	f.bs.f  = f 
	f.bc.f  = f 

	// init kayak rpc mux
	if f.kayakMux, err = NewDomainKayakMuxService(DomainKayakRPCName, peerHost ); err != nil {
		err = errors.Wrap(err, "register kayak mux service failed")
		return
	}

	log.Infof("NewFrontera(40)") 
	// init sql-chain rpc mux
	if f.chainMux, err = sqlchain.NewMuxService(/* route.SQLChainRPCName , wyong, 20201205 */ "URLC" , peerHost ); err != nil {
		err = errors.Wrap(err, "register sqlchain mux service failed")
		return
	}

	log.Infof("NewFrontera(50)") 
	// cache address of node
	var (
		pk   *asymmetric.PublicKey
		addr proto.AccountAddress
	)
	if pk, err = kms.GetLocalPublicKey(); err != nil {
		err = errors.Wrap(err, "failed to cache public key")
		return
	}
	log.Infof("NewFrontera(60)") 
	if addr, err = crypto.PubKeyHash(pk); err != nil {
		err = errors.Wrap(err, "generate address failed")
		return
	}
	f.address = addr


	log.Infof("NewFrontera(70)") 
	// init chain bus service
	//ctx := context.Background()

	bs := NewBusService(ctx, peerHost,  addr, conf.GConf.ChainBusPeriod )
	f.busService = bs

	log.Infof("NewFrontera(80)") 
	// private key cache
	f.privKey, err = kms.GetLocalPrivateKey()
	if err != nil {
		log.WithError(err).Warning("get private key failed")
		return
	}

	log.Infof("NewFrontera(90)") 

	//wyong, DBRPCName -> FronteraRPCName 
	// init service
	f.rpc = NewFronteraRPCService(/* route.FronteraRPCName wyong, 20201209 */ "FRT", peerHost,  f)


	//log.Infof("NewFrontera(100)") 
	//wyong, 20200721 
	//go f.bc.Run()

	//todo, wyong, 20201020 
	//network.SetDelegate(f)

	//log.Infof("NewFrontera(110)") 
	// Start up biddingsyss async worker routines
	//f.startWorkers(px, ctx)

	//log.Infof("NewFrontera(120)") 
	// bind the context and process.
	// do it over here to avoid closing before all setup is done.
	//go func() {
	//	<-px.Closing() // process closes first
	//	//cancelFunc()
	//	log.Infof("NewFrontera(130)") 
	//}()
	//procctx.CloseAfterContext(px, ctx) // parent cancelled first


	log.Infof("NewFrontera(140)") 
	return
}

func( f *Frontera) Start(ctx context.Context ) error {
	px := process.WithTeardown(func() error {
		//notif.Shutdown()
		return nil
	})

	//move the following code from builder to here, wyong, 20201215
	if err := f.Init(); err != nil {
		err = errors.Wrap(err, "init Frontera failed")
		return err 
	}

	log.Infof("NewFrontera(100)") 
	//wyong, 20200721 
	go f.bc.Run()

	log.Infof("NewFrontera(110)") 
	// Start up biddingsyss async worker routines
	f.startWorkers(px, ctx)

	log.Infof("NewFrontera(120)") 

	// bind the context and process.
	// do it over here to avoid closing before all setup is done.
	go func() {
		<-px.Closing() // process closes first
		//cancelFunc()
		log.Infof("NewFrontera(130)") 
	}()

	procctx.CloseAfterContext(px, ctx) // parent cancelled first
	return nil 
}


func (f *Frontera) readMeta() (meta *FronteraMeta, err error) {
	filePath := filepath.Join(f.cfg.RootDir, DBMetaFileName)
	meta = NewFronteraMeta()

	var fileContent []byte

	if fileContent, err = ioutil.ReadFile(filePath); err != nil {
		// if not exists
		if os.IsNotExist(err) {
			// new without meta
			err = nil
			return
		}

		return
	}

	err = utils.DecodeMsgPack(fileContent, meta)

	// create empty meta if meta map is nil
	if err == nil && meta.DOMAINS == nil {
		meta = NewFronteraMeta()
	}

	return
}

func (f *Frontera) writeMeta() (err error) {
	meta := NewFronteraMeta()

	f.domainMap.Range(func(key, value interface{}) bool {
		domainID := key.(proto.DomainID)
		meta.DOMAINS[domainID] = true
		return true
	})

	var buf *bytes.Buffer
	if buf, err = utils.EncodeMsgPack(meta); err != nil {
		return
	}

	filePath := filepath.Join(f.cfg.RootDir, DBMetaFileName)
	err = ioutil.WriteFile(filePath, buf.Bytes(), 0644)

	return
}

// Init defines frontera init logic.
func (f *Frontera) Init() (err error) {

	log.Debugf("Frontera/Init(10)")

	// read meta
	var localMeta *FronteraMeta
	if localMeta, err = f.readMeta(); err != nil {
		err = errors.Wrap(err, "read frontera meta failed")
		return
	}

	//todo, wyong, 20200804 
	// load current peers info from block producer
	var domainMapping = f.busService.GetCurrentDomainMapping()

	//todo, wyong, 20200804 
	// init database
	if err = f.initDomains(localMeta, domainMapping); err != nil {
		err = errors.Wrap(err, "init databases with meta failed")
		return
	}

	//wyong, 20200729 
	if err = f.busService.Subscribe("/CreateDomain/", f.createDomain); err != nil {
		err = errors.Wrap(err, "init chain bus failed")
		return
	}
	if err = f.busService.Subscribe("/UpdateBilling/", f.updateBilling); err != nil {
		err = errors.Wrap(err, "init chain bus failed")
		return
	}
	f.busService.Start()

	return
}

func (f *Frontera) updateBilling(itx interfaces.Transaction, count uint32) {
	var (
		tx *types.UpdateBilling
		ok bool
	)
	if tx, ok = itx.(*types.UpdateBilling); !ok {
		log.WithFields(log.Fields{
			"type": itx.GetTransactionType(),
		}).WithError(ErrInvalidTransactionType).Warn("invalid tx type in update billing")
		return
	}
	// Get profile and database instance
	var (
		//id       = tx.Receiver.DomainID()
		profile  *types.SQLChainProfile
		domain *Domain
	)      

	//wyong, 20200806 
	id := tx.Receiver.DomainID()

	le := log.WithFields(log.Fields{
		"id": id,
	})
	if domain, ok = f.getMeta(id); !ok {
		le.Warn("cannot find database")
		return
	}
	if profile, ok = f.busService.RequestSQLProfile(id); !ok {
		le.Warn("cannot find profile")
		return
	}
	domain.chain.SetLastBillingHeight(int32(profile.LastUpdatedHeight))
}


//wyong, 20200729 
func (f *Frontera) createDomain(tx interfaces.Transaction, count uint32) {

	//wyong, 20200528
	log.Infof("frontera/frontera.go, createDomain called") 

	cd, ok := tx.(*types.CreateDomain)
	if !ok {
		log.WithError(ErrInvalidTransactionType).Warningf("invalid tx type in createDomain: %s",
			tx.GetTransactionType().String())
		return
	}

	var (
		//wyong, 20200820 
		//dbID          = proto.FromAccountAndNonce(cd.Owner, uint32(cd.Nonce))
		domainID = proto.DomainID(cd.ResourceMeta.Domain)
		isTargetMiner = false
	)
	log.WithFields(log.Fields{
		"domain": domainID,
		"owner":      cd.Owner.String(),
		"nonce":      cd.Nonce,
	}).Debug("in createDomain")

	//wyong, 20200820 
	p, ok := f.busService.RequestSQLProfile(domainID)
	if !ok {
		log.WithFields(log.Fields{
			"domain": domainID,
		}).Warning("domain profile not found")
		return
	}

	nodeIDs := make([]proto.NodeID, len(p.Miners))

	for i, mi := range p.Miners {
		if mi.Address == f.address {
			isTargetMiner = true
		}
		nodeIDs[i] = mi.NodeID
	}
	if !isTargetMiner {
		return
	}

	var si, err = f.buildSQLChainServiceInstance(p)
	if err != nil {
		log.WithError(err).Warn("failed to build sqlchain service instance from profile")
	}
	err = f.Create(si, true)
	if err != nil {
		log.WithError(err).Error("create database error")
	}

	//todo, wyong, 20201020 
	//if f.cfg.OnCreateDatabase != nil {
	//	go f.cfg.OnCreateDatabase()
	//}
}

func (f *Frontera) buildSQLChainServiceInstance(
	profile *types.SQLChainProfile) (instance *types.ServiceInstance, err error,
) {
	var (
		nodeids = make([]proto.NodeID, len(profile.Miners))
		peers   *proto.Peers
		genesis = &types.Block{}
	)
	for i, v := range profile.Miners {
		nodeids[i] = v.NodeID
	}
	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Leader:  nodeids[0],
			Servers: nodeids[:],
		},
	}

	if f.privKey == nil {
		if f.privKey, err = kms.GetLocalPrivateKey(); err != nil {
			log.WithError(err).Warning("get private key failed in createDomain")
			return
		}
	}
	if err = peers.Sign(f.privKey); err != nil {
		return
	}


	if err = utils.DecodeMsgPack(profile.EncodedGenesis, genesis); err != nil {
		return
	}
	instance = &types.ServiceInstance{
		DomainID:   profile.ID,
		Peers:        peers,
		ResourceMeta: profile.Meta,
		GenesisBlock: genesis,
	}
	return
}

// UpdatePermission exports the update permission interface for test.
func (f *Frontera) UpdatePermission(domainID proto.DomainID, user proto.AccountAddress, permStat *types.PermStat) (err error) {
	f.busService.lock.Lock()
	defer f.busService.lock.Unlock()
	profile, ok := f.busService.sqlChainProfiles[domainID]
	if !ok {
		f.busService.sqlChainProfiles[domainID] = &types.SQLChainProfile{
			ID: domainID,
			Users: []*types.SQLChainUser{
				&types.SQLChainUser{
					Address:    user,
					Permission: permStat.Permission,
					Status:     permStat.Status,
				},
			},
		}
	} else {
		exist := false
		for _, u := range profile.Users {
			if u.Address == user {
				u.Permission = permStat.Permission
				u.Status = permStat.Status
				exist = true
				break
			}
		}
		if !exist {
			profile.Users = append(profile.Users, &types.SQLChainUser{
				Address:    user,
				Permission: permStat.Permission,
				Status:     permStat.Status,
			})
			f.busService.sqlChainProfiles[domainID] = profile
		}
	}

	_, ok = f.busService.sqlChainState[domainID]
	if !ok {
		f.busService.sqlChainState[domainID] = make(map[proto.AccountAddress]*types.PermStat)
	}
	f.busService.sqlChainState[domainID][user] = permStat

	return
}

func (f *Frontera) initDomains ( meta *FronteraMeta, profiles map[proto.DomainID]*types.SQLChainProfile) (err error,
) {
	log.Debugf("Frontera/initDomains(10)")

	currentInstance := make(map[proto.DomainID]bool)
	wg := &sync.WaitGroup{}

	for id, profile := range profiles {
		currentInstance[id] = true
		var instance *types.ServiceInstance
		if instance, err = f.buildSQLChainServiceInstance(profile); err != nil {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			log.Debugf("Frontera/initDomains(20)")
			if err := f.Create(instance, false); err != nil {
				log.WithFields(log.Fields{
					"id": instance.DomainID,
				}).WithError(err).Error("failed to create domain instance")
			}
			log.Debugf("Frontera/initDomains(30)")
		}()
	}
	wg.Wait()

	log.Debugf("Frontera/initDomains(40)")

	// calculate to drop domains 
	toDropInstance := make(map[proto.DomainID]bool)

	for domainID := range meta.DOMAINS {
		if _, exists := currentInstance[domainID]; !exists {
			toDropInstance[domainID] = true
		}
	}

	// drop domain
	for domainID := range toDropInstance {
		if err = f.Drop(domainID); err != nil {
			return
		}
	}

	log.Debugf("Frontera/initDomains(50)")
	return
}

// Create add new domain to the miner .
func (f *Frontera) Create(instance *types.ServiceInstance, cleanup bool) (err error) {
	//wyong, 20200818 
	log.Infof("Frontera/Create(10)") 

	if _, alreadyExists := f.getMeta(instance.DomainID); alreadyExists {
		return ErrAlreadyExists
	}

	// set database root dir
	rootDir := filepath.Join(f.cfg.RootDir, string(instance.DomainID))

	log.Infof("Frontera/Create(20)") 
	// clear current data
	if cleanup {
		if err = os.RemoveAll(rootDir); err != nil {
			return
		}
	}

	log.Infof("Frontera/Create(30)") 
	var domain *Domain

	defer func() {
		// close on failed
		if err != nil {
			if domain != nil {
				domain.Shutdown()
			}

			f.removeMeta(instance.DomainID)
		}
	}()

	log.Infof("Frontera/Create(40)") 
	// new db
	domainCfg := &DomainConfig{
		DomainID:             instance.DomainID,
		RootDir:                f.cfg.RootDir,
		DataDir:                rootDir,
		KayakMux:               f.kayakMux,
		ChainMux:               f.chainMux,
		MaxWriteTimeGap:        f.cfg.MaxReqTimeGap,
		EncryptionKey:          instance.ResourceMeta.EncryptionKey,
		SpaceLimit:             instance.ResourceMeta.Space,

		//todo, wyong, 20200929 
		UpdateBlockCount:       60,	//conf.GConf.BillingBlockCount,

		UseEventualConsistency: instance.ResourceMeta.UseEventualConsistency,
		ConsistencyLevel:       instance.ResourceMeta.ConsistencyLevel,
		IsolationLevel:         instance.ResourceMeta.IsolationLevel,
		SlowQueryTime:          DefaultSlowQueryTime,

		//wyong, 20200929
		//UrlChain : 		f.cfg.UrlChain, 

		//wyong, 20201018
		Host:			f.host, 
 
	}

	// set last billing height
	if profile, ok := f.busService.RequestSQLProfile(domainCfg.DomainID); ok {
		domainCfg.LastBillingHeight = int32(profile.LastUpdatedHeight)
	}

	log.Infof("Frontera/Create(50)") 
	if domain, err = NewDomain(domainCfg, f, instance.Peers, instance.GenesisBlock); err != nil {
		return
	}

	log.Infof("Frontera/Create(60)") 
	// add to meta
	err = f.addMeta(instance.DomainID, domain)

	log.Infof("Frontera/Create(70)") 
	// update metrics
	domainCount.Add(1)

	log.Infof("Frontera/Create(80)") 
	return
}

// Drop remove database from the miner .
func (f *Frontera) Drop(domainID proto.DomainID) (err error) {
	var domain *Domain
	var exists bool

	if domain, exists = f.getMeta(domainID); !exists {
		return ErrNotExists
	}

	// shutdown database
	if err = domain.Destroy(); err != nil {
		return
	}

	// update metrics
	domainCount.Add(-1)

	// remove meta
	return f.removeMeta(domainID)
}

// Update apply the new peers config to frontera.
func (f *Frontera) Update(instance *types.ServiceInstance) (err error) {
	var domain *Domain
	var exists bool

	log.Debugf("Frontera/Update(10)\n") 
	//update frontera 's peer info which hold by bc and bs, wyong,20200825 
	f.PeersConnected(instance.Peers )

	log.Debugf("Frontera/Update(20)\n") 
	if domain, exists = f.getMeta(instance.DomainID); !exists {
		return ErrNotExists
	}

	log.Debugf("Frontera/Update(30)\n") 
	// update peers
	return domain.UpdatePeers(instance.Peers)
}

/*
//wyong, 20200723 
func (f *Frontera) Exec(req *types.Request) (res *types.Response, err error) {

}
*/

// Query handles query request in frontera.
func (f *Frontera) Query(req *types.Request) (res *types.Response, err error) {
	var domain *Domain
	var exists bool

	//wyong, 20200528
	log.Infof("frontera/frontera.go, Query called") 

	// check permission
	addr, err := crypto.PubKeyHash(req.Header.Signee)
	if err != nil {
		return
	}
	err = f.checkPermission(addr, req.Header.DomainID, req.Header.QueryType, req.Payload.Queries)
	if err != nil {
		return
	}

	// find domain
	if domain, exists = f.getMeta(req.Header.DomainID); !exists {
		err = ErrNotExists
		return
	}

	return domain.Query(req)
}

// Ack handles ack of previous response.
func (f *Frontera) Ack(ack *types.Ack) (err error) {
	var domain *Domain
	var exists bool

	// check permission
	addr, err := crypto.PubKeyHash(ack.Header.Signee)
	if err != nil {
		return
	}
	err = f.checkPermission(addr, ack.Header.Response.Request.DomainID, types.ReadQuery, nil)
	if err != nil {
		return
	}
	// find database
	if domain, exists = f.getMeta(ack.Header.Response.Request.DomainID); !exists {
		err = ErrNotExists
		return
	}

	// send query
	return domain.Ack(ack)
}

func (f *Frontera) getMeta(domainID proto.DomainID) (domain *Domain, exists bool) {
	var rawDomain interface{}

	if rawDomain, exists = f.domainMap.Load(domainID); !exists {
		return
	}

	domain = rawDomain.(*Domain)

	return
}

func (f *Frontera) addMeta(domainID proto.DomainID, domain *Domain) (err error) {
	log.Debugf("Frontera/addMeta(10), domainID =%s", domainID) 
	if _, alreadyExists := f.domainMap.LoadOrStore(domainID, domain); alreadyExists {
		log.Debugf("Frontera/addMeta(15)") 
		return ErrAlreadyExists
	}

	log.Debugf("Frontera/addMeta(20)") 
	return f.writeMeta()
}

func (f *Frontera) removeMeta(domainID proto.DomainID) (err error) {
	f.domainMap.Delete(domainID)
	return f.writeMeta()
}

func (f *Frontera) checkPermission(addr proto.AccountAddress, domainID proto.DomainID, queryType types.QueryType, queries []types.Query) (err error) {
	log.Debugf("in checkPermission, domain id: %s, user addr: %s", domainID, addr.String())

	var (
		permStat *types.PermStat
		ok       bool
	)

	// get database perm stat
	permStat, ok = f.busService.RequestPermStat(domainID, addr)

	// perm stat not exists
	if !ok {
		err = errors.Wrap(ErrPermissionDeny, "domain not exists")
		return
	}

	// check if query is enabled
	if !permStat.Status.EnableQuery() {
		err = errors.Wrapf(ErrPermissionDeny, "cannot query, status: %d", permStat.Status)
		return
	}

	// check query type permission
	switch queryType {
	case types.ReadQuery:
		if !permStat.Permission.HasReadPermission() {
			err = errors.Wrapf(ErrPermissionDeny, "cannot read, permission: %v", permStat.Permission)
			return
		}
	case types.WriteQuery:
		if !permStat.Permission.HasWritePermission() {
			err = errors.Wrapf(ErrPermissionDeny, "cannot write, permission: %v", permStat.Permission)
			return
		}
	default:
		err = errors.Wrapf(ErrInvalidPermission,
			"invalid permission, permission: %v", permStat.Permission)
		return
	}

	// check for query pattern
	var (
		disallowedQuery    string
		hasDisallowedQuery bool
	)

	if disallowedQuery, hasDisallowedQuery = permStat.Permission.HasDisallowedQueryPatterns(queries); hasDisallowedQuery {
		err = errors.Wrapf(ErrPermissionDeny, "disallowed query %s", disallowedQuery)
		log.WithError(err).WithFields(log.Fields{
			"permission": permStat.Permission,
			"query":      disallowedQuery,
		}).Debug("can not query")
		return
	}

	return
}

// Shutdown defines frontera shutdown logic.
func (f *Frontera) Shutdown() (err error) {
	f.domainMap.Range(func(_, rawDomain interface{}) bool {
		domain := rawDomain.(*Domain)

		if err = domain.Shutdown(); err != nil {
			log.WithError(err).Error("shutdown domain failed")
		}

		return true
	})

	// persist meta
	err = f.writeMeta()

	f.busService.Stop()

	//wyong, 20200721 
	f.process.Close()

	return
}


type biddingRequest struct {
	Cid cid.Cid
	Ctx context.Context
}


// GetBlock attempts to retrieve a particular block from peers within the
// deadline enforced by the context.
func (f *Frontera) PutBidding(ctx context.Context, req *types.UrlRequestMessage ) (res *types.UrlResponse, err error) {
	log.Debugf("Frontera/PutBidding(10)\n")
       	
	//wyong, 20200820 
	domain, exist := f.DomainForID(req.Header.DomainID) 
	if exist != true {
		err = errors.New("domain not exist") 
		return 
	}

	log.Debugf("Frontera/PutBidding(20), domain=%s\n", domain.domainID )
	for _, r := range req.Payload.Requests {
		log.Debugf("Frontera/PutBidding(30)\n")

		//todo, add parameter 'ParentUrl', wyong, 20210125 
		err = domain.PutBidding(ctx, r.Url, r.Probability, r.ParentUrl, r.ParentProbability  )
		if err != nil {
			log.Debugf("Frontera/PutBidding(35), err=%s\n", err.Error())
			return 
		}
		log.Debugf("Frontera/PutBidding(40)\n")
	}

	log.Debugf("Frontera/PutBidding(50)\n")
	res = &types.UrlResponse{
                Header: types.UrlSignedResponseHeader{
                        UrlResponseHeader: types.UrlResponseHeader{
                                Request:     req.Header.UrlRequestHeader,
                                RequestHash: req.Header.Hash(),
                                NodeID:      f.nodeID,
                                Timestamp:   getLocalTime(),
                                //RowCount:    uint64(len(cids)),
                                //LogOffset:   id,
                        },
                },
        }

	log.Debugf("Frontera/PutBidding(60)\n")
	return 
}


//wyong, 20190131 
func (f *Frontera) DelBidding(ctx context.Context, url string) error {
	d, exist := f.DomainForUrl(url) 
	if exist != true {
		f.CancelBiddings([]string{url,} , d.domainID) 
	}

	return nil 
}


func (f *Frontera) LedgerForPeer(p proto.NodeID) * /* decision. */ Receipt {
	return f.bs.LedgerForPeer(p)
}

/*wyong, 20200805 
func (f *Frontera) getNextSessionID() uint64 {
	f.sessIDLk.Lock()
	defer f.sessIDLk.Unlock()
	f.sessID++
	return f.sessID
}
*/


// CancelBiddings removes a given urls from the biddinglist
func (f *Frontera) CancelBiddings(urls []string,  domainID proto.DomainID ) bool {
	return f.bc.CancelBiddings(context.Background(), urls, nil, domainID )
}


func(f *Frontera) GetBidding(ctx context.Context) ([]*BiddingEntry,  error) {
	log.Debugf("Frontera/GetBidding(10)\n ")
	//wyong, 20190131
	bl, _ := f.bs.GetBidding()
	if bl == nil {
		log.Debugf("Frontera/GetBidding(15)\n ")
		return []*BiddingEntry{}, nil
	}

	log.Debugf("Frontera/GetBidding(20)\n ")
	biddings := bl.BiddingEntries()
        out := make([]*BiddingEntry, 0, len(biddings))
        for _, bidding := range biddings {
                log.Debugf("Frontera/GetBidding(30), %s\n", bidding.Url )
		if (bidding.Seen == false ) {
                	out = append(out,bidding)
			bidding.Seen = true 
		}
        }

	log.Debugf("Frontera/GetBidding(40)\n")
	return out, nil
}


//wyong, 20190116
func(f *Frontera) GetCompletedBiddings(ctx context.Context) ([]*BiddingEntry, error) {
	log.Debugf("GetCompletedBiddings called...")

	//wyong, 20190131
	//return bs.bc.GetCompletedBiddings() 

	completedbiddings,_ := f.bc.GetCompletedBiddings()
        out := make([]*BiddingEntry, 0, len(completedbiddings))

        for _, bidding := range completedbiddings {
                //log.Debugf("GetBidding (30), %s", e.GetUrl())
		if (bidding.Seen == false ) {
                	out = append(out,bidding)
			bidding.Seen = true 
		}
        }

	return out, nil
}

//wyong, 20190116
func(f *Frontera) GetUncompletedBiddings(ctx context.Context) ([]*BiddingEntry, error) {
	log.Debugf("GetUncompletedBiddings called...")
	return f.bc.GetUncompletedBiddings()
}

//wyong, 20190120
func(f *Frontera) GetBids(ctx context.Context, url string ) ([]*BidEntry, error) {
	log.Debugf("GetBids called...")
	biddings, _ := f.bc.GetCompletedBiddings() 
	for _, bidding := range biddings {
		if bidding.GetUrl() == url {
			return bidding.GetBids(), nil 
		}
	}
	return nil, nil
}


//wyong, 20200828 
func (f *Frontera) UrlBidMessageReceived(ctx context.Context, req *types.UrlBidMessage) (err error) {

	from := req.Header.UrlBidHeader.NodeID

        // TODO: this is bad, and could be easily abused.
        // Should only track *useful* messages in ledger
        bids := req.Payload.Bids
        if len(bids) == 0 {
                return
        }

        wg := sync.WaitGroup{}
        for _, bid := range bids {
                wg.Add(1)
                go func(p proto.NodeID, bid types.UrlBid ) {
                        // TODO: this probably doesnt need to be a goroutine...
                        defer wg.Done()

                        //TODO,wyong, 20181220
                        //bs.updateReceiveCounters(cid)

                        log.Debugf("Frontera/UrlBidMessageReceived(10), got cid(%s) for url(%s) from %s", bid.Cid, bid.Url, p)
			cid, err  := cid.Decode(bid.Cid) 
			if err!= nil {
				//todo, wyong, 20200907 
				return 
			}

                        if err := f.receiveBidFrom(ctx, p, bid.Url, cid, bid.Hash, bid.Proof, bid.SimHash ); err != nil {
                                log.Warningf("ReceiveMessage recvBidFrom error: %s", err)
                        }

                        //TODO, cannot call non-function bidding.BiddingEntry.Url (type string)
                        //wyong, 20190119
                        //log.Event(ctx, "Biddingsys.PutBiddingRequest.End", bidding.Url())

                }(from, bid)
        }
        wg.Wait()
	
	return 
}


// TODO: Some of this stuff really only needs to be done when adding a block
// from the user, not when receiving it from the network.
// In case you run `git blame` on this comment, I'll save you some time: ask
// @whyrusleeping, I don't know the answers you seek.
func (f *Frontera) receiveBidFrom(ctx context.Context,  from proto.NodeID, url string, c cid.Cid, hash []byte, proof []byte , simhash uint64 ) error {
	select {
	case <-f.process.Closing():
		return errors.New("biddingsys is closed")
	default:
	}

	//err := bs.blockstore.Put(blk)
	//if err != nil {
	//	log.Errorf("Error writing block to datastore: %s", err)
	//	return err
	//}

	// NOTE: There exists the possiblity for a race condition here.  If a user
	// creates a node, then adds it to the dagservice while another goroutine
	// is waiting on a GetBlock for that object, they will receive a reference
	// to the same node. We should address this soon, but i'm not going to do
	// it now as it requires more thought and isnt causing immediate problems.
	// bs.notifications.Publish(bid)


	//bidding := biddingmsg.BiddingEntry
	//url := bidding.Url

	//ks := []string{url}
	//get bids from message, wyong, 20190115
	//bids := bidding.Bids
	
	log.Debugf("Frontera/receiveBidFrom(10)\n")
	d, exist := f.DomainForUrl(url) 
	if exist == true {
		log.Debugf("Frontera/receiveBidFrom(20)\n")
		if (f.bc.ReceiveBidForBiddings(ctx, url, c, from, d.domainID, hash, proof , simhash )) {
			//wait until bc have processed the incoming bid. 
			//if we have received two bids, remove bidding from domain.
			//wyong, 20200831
			log.Debugf("Frontera/receiveBidFrom(30)\n")
			d.receiveBidFrom(from, url)
			log.Debugf("Frontera/receiveBidFrom(40)\n")
		}
		log.Debugf("Frontera/receiveBidFrom(50)\n")

	}

	//f.bs.AddBid(bid)
	
	//select {
	//case bs.newBids <- bid.Cid():
	//	// send block off to be reprovided
	//case <-bs.process.Closing():
	//	return bs.process.Close()
	//}

	log.Debugf("Frontera/receiveBidFrom(60)\n")
	return nil
}

//wyong, 20200820 
func (f *Frontera) DomainForID(id proto.DomainID ) (domain *Domain, exists bool )  {

	log.Debugf("Frontera/DomainForID(10), id =%s\n", id) 

	//just for debug, wyong, 20200820 
	f.domainMap.Range(func(_, rawDomain interface{}) bool {
		domain := rawDomain.(*Domain)
		log.Debugf("Frontera/DomainForID(20), domainMap, domain.domainID=%s\n", domain.domainID) 
		return true 
	})

	log.Debugf("Frontera/DomainForID(30)\n") 
	var rawDomain interface{}
	if rawDomain, exists = f.domainMap.Load(id); !exists {
		log.Debugf("Frontera/DomainForID(35)\n") 
		return
	}

	domain = rawDomain.(*Domain)
	log.Debugf("Frontera/DomainForID(40), found domain=%s\n", domain.domainID ) 
	return
}

// DomainForUrl returns a slice of all sessions that may be interested in the given cid
func (f *Frontera) DomainForUrl(urlstring string) (domain *Domain, exists bool )  {
	/* todo, wyong, 20200805 
	f.sessLk.Lock()
	defer f.sessLk.Unlock()

	//var out []*Domain
	for _, s := range f.sessions {
		if s.interestedIn(url) {
			out = append(out, s)
		}
	}
	*/

	log.Debugf("Frontera/DomainForUrl(10), urlstring=%s\n", urlstring) 
        u, err := url.Parse(urlstring)
        if err != nil {
                return 
        }

        domainname := u.Scheme + "://" + u.Host
	log.Debugf("Frontera/DomainForUrl(20), domainname=%s\n", domainname ) 

	var rawDomain interface{}
	if rawDomain, exists = f.domainMap.Load(proto.DomainID(domainname)); !exists {
		return
	}

	domain = rawDomain.(*Domain)
	//log.Debugf("Frontera/DomainForUrl(30), domain=%s\n", domain) 

	return
}

/*
func (f *Frontera) ReceiveMessage(ctx context.Context, p proto.NodeID, incoming bsmsg.BiddingMessage) {
	log.Debugf("receiveMessage called")
	atomic.AddUint64(&bs.counters.messagesRecvd, 1)

	// This call records changes to biddinglists, blocks received,
	// and number of bytes transfered.
	f.bs.MessageReceived(p, incoming)

	// TODO: this is bad, and could be easily abused.
	// Should only track *useful* messages in ledger
	biddings := incoming.Biddings()
	if len(biddings) == 0 {
		return
	}

	wg := sync.WaitGroup{}
	for _, bidding := range biddings {
		wg.Add(1)
		go func(bidding bsmsg.BiddingEntry) { 
			// TODO: this probably doesnt need to be a goroutine...
			defer wg.Done()

			//TODO,wyong, 20181220
			//bs.updateReceiveCounters(cid)

			log.Debugf("got cid for url %s from %s", bidding.GetUrl(),  p)
			if err := bs.receiveBidFrom(bidding, p); err != nil {
				log.Warningf("ReceiveMessage recvBidFrom error: %s", err)
			}

			//TODO, cannot call non-function bidding.BiddingEntry.Url (type string)
			//wyong, 20190119
			//log.Event(ctx, "Biddingsys.PutBiddingRequest.End", bidding.Url())

		}(bidding )
	}
	wg.Wait()

}
*/


//wyong, 20181221
func(f *Frontera) PutBid(ctx context.Context, url string, cid cid.Cid) error {
	log.Debugf("Frontera/PutBid(10), url = %s, cid = %s\n", url, cid)
	f.bs.PutBid(ctx, url, cid )
	return nil 
}

//wyong, 20200817
func(f *Frontera) RetriveUrlCid(ctx context.Context, req *types.UrlCidRequestMessage ) (res *types.UrlCidResponse, err error) {
	log.Debugf("Frontera/RetriveUrlCid(10), domainID =%s\n", req.Header.DomainID ) 

	//wyong, 20200820 
	domain, exist := f.DomainForID(req.Header.DomainID) 
	if exist != true {
		err = errors.New("domain not exist") 
		return 
	}

	log.Debugf("Frontera/RetriveUrlCid(20), domain.domainID =%s\n", string(domain.domainID)) 
	var cids []string
	for _, r := range req.Payload.Requests {
		var c cid.Cid 
		c, err = domain.RetriveUrlCid(ctx, r.ParentUrl,  r.Url) 
		log.Debugf("Frontera/RetriveUrlCid(30), r.Url=%s\n", r.Url ) 
		if err != nil {
			break 	
		}
		log.Debugf("Frontera/RetriveUrlCid(40), cid=%s\n", c.String()) 

		//apend this cid to cids, wyong, 20200820
		cids = append(cids, c.String()) 
	}

	log.Debugf("Frontera/RetriveUrlCid(50)\n") 
	res = &types.UrlCidResponse{
                Header: types.UrlCidSignedResponseHeader{
                        UrlCidResponseHeader: types.UrlCidResponseHeader{
                                Request:     req.Header.UrlCidRequestHeader,
                                RequestHash: req.Header.Hash(),
                                NodeID:      f.nodeID,
                                Timestamp:   getLocalTime(),
                                RowCount:    uint64(len(cids)),
                                //LogOffset:   id,
                        },
                },
                Payload: types.UrlCidResponsePayload{
                        //Columns:   cnames,
                        //DeclTypes: ctypes,
                        //Rows:      buildRowsFromNativeData(data),
			Cids: cids, 
                },
        }

	log.Debugf("Frontera/RetriveUrlCid(60)\n") 
	return 
}

//wyong, 20200825 
// Connected/Disconnected warns biddingsys about peer connections
func (f *Frontera) PeersConnected(peers *proto.Peers ) {
	log.Debugf("Frontera/PeersConnected(10)\n")
	for _, p := range peers.Servers {
		log.Debugf("Frontera/PeerConnected(20), p=%s\n", p )
		f.bc.Connected(p)
		f.bs.PeerConnected(p)
	}
}

// Connected/Disconnected warns biddingsys about peer connections
func (f *Frontera) PeersDisconnected(peers *proto.Peers ) {
	log.Debugf("Frontera/PeerDisconnected(10)\n")
	for _, p := range peers.Servers {
		log.Debugf("Frontera/PeerDisconnected(20), p=%s\n", p )
		f.bc.Disconnected(p)
		f.bs.PeerDisconnected(p)
	}
}

/*
func (bs *Biddingsys) ReceiveError(err error) {
	log.Infof("Biddingsys ReceiveError: %s", err)
	// TODO log the network error
	// TODO bubble the network error up to the parent context/error logger
}


func (bs *Biddingsys) Close() error {
	return bs.process.Close()
}
*/


func (f *Frontera) GetBiddinglist() []string {
	entries := f.bc.bl.Entries()
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Url )
	}
	return out
}

/*
func (bs *Biddingsys) IsOnline() bool {
	return true
}
*/

//TODO, search domain accroding to url. wyong, 20190113
func (f *Frontera) GetDomain(ctx context.Context, url string) (*Domain, error) {
	//todo, wyong, 20200805 
	return nil, nil   
}


