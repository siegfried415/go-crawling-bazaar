/*
 * Copyright 2018 The CovenantSQL Authors.
 * Copyright 2022 https://github.com/siegfried415
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

	cid "github.com/ipfs/go-cid" 
	"github.com/pkg/errors"
	process "github.com/jbenet/goprocess" 
	procctx "github.com/jbenet/goprocess/context" 
        url "net/url"


	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/crypto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
	"github.com/siegfried415/go-crawling-bazaar/kms"
	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/types"
	"github.com/siegfried415/go-crawling-bazaar/urlchain"
	"github.com/siegfried415/go-crawling-bazaar/utils"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"

)

const (
	// DomainKayakRPCName defines rpc service name of database internal consensus.
	DomainKayakRPCName = "DBC" // aka. database consensus

	// DBMetaFileName defines frontera meta file name.
	DBMetaFileName = "db.meta"

	// DefaultSlowQueryTime defines the default slow query log time
	DefaultSlowQueryTime = time.Second * 5

	mwMinerDBCount = "service:miner:db:count"

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
	
	chainMux   *sqlchain.MuxService
	rpc        *FronteraRPCService
	busService *BusService
	address    proto.AccountAddress

	privKey    *asymmetric.PrivateKey

	nodeID	    proto.NodeID 

	// the peermanager manages sending messages to peers in a way that
	// wont block biddingsys operation
	bc *BiddingClient 

	// the BiddingServer is the bit of logic that decides who to send which blocks to
	//engine *decision.Engine
	bs *BiddingServer 

	// network delivers messages on behalf of the session
	//network fnet.BiddingNetwork
	host net.RoutedHost 

	process process.Process

	// Counters for various statistics
	counterLk sync.Mutex
	counters  *counters
}

// NewFrontera returns new database management instance.
func NewFrontera( cfg *FronteraConfig, peerHost net.RoutedHost  ) (f *Frontera, err error) {

	// important to use provided parent context (since it may include important
	// loggable data). It's probably not a good idea to allow biddingsys to be
	// coupled to the concerns of the ipfs daemon in this way.
	//
	// FIXME(btc) Now that biddingsys manages itself using a process, it probably
	// shouldn't accept a context anymore. Clients should probably use Close()
	// exclusively. We should probably find another way to share logging data
	//ctx, cancelFunc := context.WithCancel(parent)
	ctx := context.Background()

	px := process.WithTeardown(func() error {
		//notif.Shutdown()
		return nil
	})

	nodeID, err := kms.GetLocalNodeID()
	if err != nil {
                log.WithError(err).Error("get local node id failed")
                return nil, err  
        }

	f = &Frontera{
		cfg: cfg,
		nodeID: nodeID,

		//network.NodeId()? bstore? 
		bs :        NewBiddingServer (ctx, nodeID, peerHost ), 

		host : peerHost, 
		process:       px,
		bc :            NewBiddingClient(ctx, peerHost,  nodeID),
		counters:      new(counters),
	}

	f.bs.f  = f 
	f.bc.f  = f 


	// init sql-chain rpc mux
	if f.chainMux, err = sqlchain.NewMuxService("URLC" , peerHost ); err != nil {
		err = errors.Wrap(err, "register sqlchain mux service failed")
		return
	}

	// cache address of node
	var (
		pk   *asymmetric.PublicKey
		addr proto.AccountAddress
	)
	if pk, err = kms.GetLocalPublicKey(); err != nil {
		err = errors.Wrap(err, "failed to cache public key")
		return
	}
	if addr, err = crypto.PubKeyHash(pk); err != nil {
		err = errors.Wrap(err, "generate address failed")
		return
	}
	f.address = addr


	bs := NewBusService(ctx, peerHost,  addr, conf.GConf.ChainBusPeriod )
	f.busService = bs

	// private key cache
	f.privKey, err = kms.GetLocalPrivateKey()
	if err != nil {
		log.WithError(err).Warning("get private key failed")
		return
	}

	// init service
	f.rpc = NewFronteraRPCService( "FRT", peerHost,  f)

	return
}

func( f *Frontera) Start(ctx context.Context ) error {
	px := process.WithTeardown(func() error {
		//notif.Shutdown()
		return nil
	})

	//move the following code from builder to here
	if err := f.Init(); err != nil {
		err = errors.Wrap(err, "init Frontera failed")
		return err 
	}

	go f.bc.Run()

	// Start up biddingsyss async worker routines
	f.startWorkers(px, ctx)


	// bind the context and process.
	// do it over here to avoid closing before all setup is done.
	go func() {
		<-px.Closing() // process closes first
		//cancelFunc()
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
	// read meta
	var localMeta *FronteraMeta
	if localMeta, err = f.readMeta(); err != nil {
		err = errors.Wrap(err, "read frontera meta failed")
		return
	}

	// load current peers info from presbyterian network   
	var domainMapping = f.busService.GetCurrentDomainMapping()

	// init database
	if err = f.initDomains(localMeta, domainMapping); err != nil {
		err = errors.Wrap(err, "init databases with meta failed")
		return
	}

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


func (f *Frontera) createDomain(tx interfaces.Transaction, count uint32) {

	cd, ok := tx.(*types.CreateDomain)
	if !ok {
		log.WithError(ErrInvalidTransactionType).Warningf("invalid tx type in createDomain: %s",
			tx.GetTransactionType().String())
		return
	}

	var (
		domainID = proto.DomainID(cd.ResourceMeta.Domain)
		isTargetMiner = false
	)
	log.WithFields(log.Fields{
		"domain": domainID,
		"owner":      cd.Owner.String(),
		"nonce":      cd.Nonce,
	}).Debug("in createDomain")

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
	currentInstance := make(map[proto.DomainID]bool)
	wg := &sync.WaitGroup{}

	for id, profile := range profiles {
		log.Debugf("Frontera/initDomains, process domain %s", profile.ID ) 
		currentInstance[id] = true
		var instance *types.ServiceInstance
		if instance, err = f.buildSQLChainServiceInstance(profile); err != nil {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := f.Create(instance, false); err != nil {
				log.WithFields(log.Fields{
					"id": instance.DomainID,
				}).WithError(err).Error("failed to create domain instance")
			}
		}()
	}
	wg.Wait()


	// calculate to drop domains 
	toDropInstance := make(map[proto.DomainID]bool)

	for domainID := range meta.DOMAINS {
		if _, exists := currentInstance[domainID]; !exists {
			toDropInstance[domainID] = true
		}
	}

	// drop domain
	for domainID := range toDropInstance {
		log.Debugf("Frontera/initDomains, Drop domain %s", domainID ) 
		err = f.Drop(domainID)
		if err != nil {
			//if err != ErrNotExists {
			//	return
			//}
			log.Debugf("Frontera/initDomains, Drop domain err=%s ", err.Error()) 
		}
	}

	return nil 
}

// Create add new domain to the miner .
func (f *Frontera) Create(instance *types.ServiceInstance, cleanup bool) (err error) {

	log.Debugf("Frontera/Create begin ... ")
	if _, alreadyExists := f.getMeta(instance.DomainID); alreadyExists {
		log.Debugf("Frontera/Create, domain already exist")
		return ErrAlreadyExists
	}

	// set database root dir
	rootDir := filepath.Join(f.cfg.RootDir, string(instance.DomainID))

	// clear current data
	if cleanup {
		if err = os.RemoveAll(rootDir); err != nil {
			return
		}
	}

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

	// new db
	domainCfg := &DomainConfig{
		DomainID:             instance.DomainID,
		RootDir:                f.cfg.RootDir,
		DataDir:                rootDir,

		ChainMux:               f.chainMux,
		MaxWriteTimeGap:        f.cfg.MaxReqTimeGap,
		EncryptionKey:          instance.ResourceMeta.EncryptionKey,
		SpaceLimit:             instance.ResourceMeta.Space,

		//todo
		UpdateBlockCount:       60,	//conf.GConf.BillingBlockCount,

		UseEventualConsistency: instance.ResourceMeta.UseEventualConsistency,
		ConsistencyLevel:       instance.ResourceMeta.ConsistencyLevel,
		IsolationLevel:         instance.ResourceMeta.IsolationLevel,
		SlowQueryTime:          DefaultSlowQueryTime,

		Host:			f.host, 
 
	}

	// set last billing height
	if profile, ok := f.busService.RequestSQLProfile(domainCfg.DomainID); ok {
		domainCfg.LastBillingHeight = int32(profile.LastUpdatedHeight)
	}

	if domain, err = NewDomain(domainCfg, f, instance.Peers, instance.GenesisBlock); err != nil {
		log.Debugf("Frontera/Create, NewDomain failed, err = %s", err )

		return
	}

	// add to meta
	err = f.addMeta(instance.DomainID, domain)

	// update metrics
	domainCount.Add(1)

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

	//update frontera 's peer info which hold by bc and bs
	f.PeersConnected(instance.Peers )

	if domain, exists = f.getMeta(instance.DomainID); !exists {
		return ErrNotExists
	}

	// update peers
	return domain.UpdatePeers(instance.Peers)
}

// Query handles query request in frontera.
func (f *Frontera) Query(req *types.Request) (res *types.Response, err error) {
	var domain *Domain
	var exists bool

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
	if _, alreadyExists := f.domainMap.LoadOrStore(domainID, domain); alreadyExists {
		return ErrAlreadyExists
	}

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

	f.process.Close()

	return
}


type biddingRequest struct {
	Cid cid.Cid
	Ctx context.Context
}


// GetBlock attempts to retrieve a particular block from peers within the
// deadline enforced by the context.
func (f *Frontera) PutBidding(ctx context.Context, domainName string, requests []types.UrlRequest, parentUrlRequest types.UrlRequest ) (err error) {

	domain, exist := f.DomainForUrl(domainName ) 
	if exist != true {
		err = errors.New("domain not exist") 
		return 
	}

	ParentUrl := parentUrlRequest.Url
	ParentProbability := parentUrlRequest.Probability 

	for _, r := range requests  {
		err = domain.PutBidding(ctx, r.Url, r.Probability, ParentUrl, ParentProbability )
		if err != nil {
			return 
		}
	}

	return 
}


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


// CancelBiddings removes a given urls from the biddinglist
func (f *Frontera) CancelBiddings(urls []string,  domainID proto.DomainID ) bool {
	return f.bc.CancelBiddings(context.Background(), urls, nil, domainID )
}


func(f *Frontera) GetBidding(ctx context.Context) ([]*BiddingEntry,  error) {
	bl, _ := f.bs.GetBidding()
	if bl == nil {
		return []*BiddingEntry{}, nil
	}

	biddings := bl.BiddingEntries()
        out := make([]*BiddingEntry, 0, len(biddings))
        for _, bidding := range biddings {
		if (bidding.Seen == false ) {
                	out = append(out,bidding)
			bidding.Seen = true 
		}
        }

	return out, nil
}


func(f *Frontera) GetCompletedBiddings(ctx context.Context) ([]*BiddingEntry, error) {
	completedbiddings,_ := f.bc.GetCompletedBiddings()
        out := make([]*BiddingEntry, 0, len(completedbiddings))

        for _, bidding := range completedbiddings {
		if (bidding.Seen == false ) {
                	out = append(out,bidding)
			bidding.Seen = true 
		}
        }

	return out, nil
}

func(f *Frontera) GetUncompletedBiddings(ctx context.Context) ([]*BiddingEntry, error) {
	return f.bc.GetUncompletedBiddings()
}

func(f *Frontera) GetBids(ctx context.Context, url string ) ([]*BidEntry, error) {
	biddings, _ := f.bc.GetCompletedBiddings() 
	for _, bidding := range biddings {
		if bidding.GetUrl() == url {
			return bidding.GetBids(), nil 
		}
	}
	return nil, nil
}


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

                        //TODO
                        //bs.updateReceiveCounters(cid)

			cid, err  := cid.Decode(bid.Cid) 
			if err!= nil {
				return 
			}

                        if err := f.receiveBidFrom(ctx, p, bid.Url, cid, bid.Hash, bid.Proof, bid.SimHash ); err != nil {
                                log.Warningf("ReceiveMessage recvBidFrom error: %s", err)
                        }

                        //TODO, cannot call non-function bidding.BiddingEntry.Url (type string)
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

	// NOTE: There exists the possiblity for a race condition here.  If a user
	// creates a node, then adds it to the dagservice while another goroutine
	// is waiting on a GetBlock for that object, they will receive a reference
	// to the same node. We should address this soon, but i'm not going to do
	// it now as it requires more thought and isnt causing immediate problems.
	// bs.notifications.Publish(bid)

	d, exist := f.DomainForUrl(url) 
	if exist == true {
		if (f.bc.ReceiveBidForBiddings(ctx, url, c, from, d.domainID, hash, proof , simhash )) {
			//wait until bc have processed the incoming bid. 
			//if we have received two bids, remove bidding from domain.
			d.receiveBidFrom(from, url)
		}
	}

	return nil
}

func (f *Frontera) DomainForID(id proto.DomainID ) (domain *Domain, exists bool )  {
	//todo, just for debug, 20220103
	f.domainMap.Range(func(key , value interface{}) bool {
		domainID := key.(proto.DomainID)
		//d := rawDomain.(*Domain)
		log.Debugf("DomainForID, iter domain, %s", domainID )
		return true 
	})

	var rawDomain interface{}
	if rawDomain, exists = f.domainMap.Load(id); !exists {
		return
	}

	domain = rawDomain.(*Domain)
	return
}

// DomainForUrl returns a slice of all sessions that may be interested in the given cid
func (f *Frontera) DomainForUrl(urlstring string) (domain *Domain, exists bool )  {
        u, err := url.Parse(urlstring)
        if err != nil {
                return 
        }

        domainname := u.Scheme + "://" + u.Host
	var rawDomain interface{}
	if rawDomain, exists = f.domainMap.Load(proto.DomainID(domainname)); !exists {
		return
	}

	domain = rawDomain.(*Domain)
	return
}

func(f *Frontera) PutBid(ctx context.Context, url string, cid cid.Cid) error {
	f.bs.PutBid(ctx, url, cid )
	return nil 
}

func(f *Frontera) RetriveUrlCid(ctx context.Context, req *types.UrlCidRequestMessage ) (res *types.UrlCidResponse, err error) {
	log.Debugf("Frontera/RetriveUrlCid begin, domain = %s", req.Header.DomainID) 
	domain, exist := f.DomainForID(req.Header.DomainID) 
	if exist != true {
		err = errors.New("domain not exist") 
		return 
	}

	var cids []string
	for _, r := range req.Payload.Requests {
		var c cid.Cid 
		c, err = domain.RetriveUrlCid(ctx, r.ParentUrl,  r.Url) 
		if err != nil {
			break 	
		}

		//apend this cid to cids
		cids = append(cids, c.String()) 
	}

	res = &types.UrlCidResponse{
                Header: types.UrlCidSignedResponseHeader{
                        UrlCidResponseHeader: types.UrlCidResponseHeader{
                                Request:     req.Header.UrlCidRequestHeader,
                                RequestHash: req.Header.Hash(),
                                NodeID:      f.nodeID,
                                Timestamp:   getLocalTime(),
                                RowCount:    uint64(len(cids)),
                        },
                },
                Payload: types.UrlCidResponsePayload{
			Cids: cids, 
                },
        }

	return 
}

// Connected/Disconnected warns biddingsys about peer connections
func (f *Frontera) PeersConnected(peers *proto.Peers ) {
	for _, p := range peers.Servers {
		f.bc.Connected(p)
		f.bs.PeerConnected(p)
	}
}

// Connected/Disconnected warns biddingsys about peer connections
func (f *Frontera) PeersDisconnected(peers *proto.Peers ) {
	for _, p := range peers.Servers {
		f.bc.Disconnected(p)
		f.bs.PeerDisconnected(p)
	}
}

func (f *Frontera) GetBiddinglist() []string {
	entries := f.bc.bl.Entries()
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Url )
	}
	return out
}

