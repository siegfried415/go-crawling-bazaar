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
	"time"

        cache "github.com/patrickmn/go-cache"
	cid "github.com/ipfs/go-cid"
	lru "github.com/hashicorp/golang-lru"

	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/crypto" 
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/kms"
	//"github.com/siegfried415/go-crawling-bazaar/utils/log"
	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/storage"
	"github.com/siegfried415/go-crawling-bazaar/types"
	urlchain "github.com/siegfried415/go-crawling-bazaar/urlchain"


)

const (
	// StorageFileName defines storage file name of database instance.
	StorageFileName = "storage.db3"

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

	activeBiddingsLimit = 16
	MinCrawlersExpected	= 2 
	IncrementalCrawlingInterval uint32 = 100
	MinFrowardProbability	float64 = 0.2 
)

// Database defines a single database instance in worker runtime.
type Domain struct {
	cfg            *DomainConfig
	domainID           proto.DomainID

	connSeqs       sync.Map
	chain          *urlchain.Chain
	nodeID         proto.NodeID
	host 		net.RoutedHost

	privateKey     *asymmetric.PrivateKey
	accountAddr    proto.AccountAddress

	//move session stuff here
	ctx            context.Context
	tofetch        *ProbBiddingQueue
	activePeers    map[proto.NodeID]struct{}
	activePeersArr []proto.NodeID

	waitQueue	*TimedBiddingQueue
	f *Frontera 
	bidIncoming	chan bidRecv 
	biddingReqs	chan[]types.UrlBidding	

	cancelBiddings   chan []string
	interestReqs chan interestReq

	interest  *lru.Cache
	liveBiddings map[string]time.Time

	tick          *time.Timer
	baseTickDelay time.Duration

	latTotal time.Duration
	fetchcnt int

	urlCidCache 	*cache.Cache 
	urlGraphCache	*cache.Cache
	urlNodeCache	*cache.Cache 
	
}

// NewDomain create a single domain instance using config.
func NewDomain(cfg *DomainConfig, f *Frontera, peers *proto.Peers, genesis *types.Block) (domain *Domain, err error) {
	// ensure dir exists
	if err = os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return
	}

	if peers == nil || genesis == nil {
		err = ErrInvalidDBConfig
		return
	}

	// get private key
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	var accountAddr proto.AccountAddress
	if accountAddr, err = crypto.PubKeyHash(privateKey.GetPublic()); err != nil {
		return
	}

	// init database
	domain = &Domain{
		cfg:            cfg,
		domainID:           cfg.DomainID,

		privateKey:     privateKey,
		accountAddr:    accountAddr,
		activePeers:   make(map[proto.NodeID]struct{}),
		host: cfg.Host,

		liveBiddings:     make(map[string]time.Time),
		biddingReqs:	make(chan []types.UrlBidding),
		cancelBiddings:    make(chan []string),
		tofetch:       newProbBiddingQueue(),
		interestReqs:  make(chan interestReq),

		waitQueue	: newTimedBiddingQueue(), 

		f:            f,
		bidIncoming:      make(chan bidRecv),
		baseTickDelay: time.Millisecond * 500,
	}

	//todo, get expiration time from urlchain miner time
	domain.urlCidCache = cache.New(5*time.Minute, 10*time.Minute)
	domain.urlGraphCache = cache.New(5*time.Minute, 10*time.Minute)
	domain.urlNodeCache = cache.New(5*time.Minute, 10*time.Minute)

	cache, _ := lru.New(2048)
	domain.interest = cache 

	for _, p := range peers.Servers {
		domain.addActivePeer(p)
	}

	defer func() {
		// on error recycle all resources
		if err != nil {
			// close chain
			if domain.chain != nil {
				domain.chain.Stop()
			}
		}
	}()

	// init storage
	storageFile := filepath.Join(cfg.DataDir, StorageFileName)
	storageDSN, err := storage.NewDSN(storageFile)
	if err != nil {
		return
	}

	if cfg.EncryptionKey != "" {
		storageDSN.AddParam("_crypto_key", cfg.EncryptionKey)
	}

	// init chain
	chainFile := filepath.Join(cfg.RootDir, SQLChainFileName)
	if domain.nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	chainCfg := &urlchain.Config{
		DomainID:      cfg.DomainID,
		ChainFilePrefix: chainFile,
		DataFile:        storageDSN.Format(),
		Genesis:         genesis,
		Peers:           peers,

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

	if err = domain.chain.Start(); err != nil {
		return
	}

	if err = domain.InitTables(); err != nil {
		return
	}
	
	go domain.run(context.Background())
	return
}

// UpdatePeers defines peers update query interface.
func (domain *Domain) UpdatePeers(peers *proto.Peers) (err error) {
	for _, p := range peers.Servers {
		domain.addActivePeer(p)
	}

	return domain.chain.UpdatePeers(peers)
}


// Shutdown stop database handles and stop service the database.
func (domain *Domain) Shutdown() (err error) {
	if domain.chain != nil {
		// stop chain
		if err = domain.chain.Stop(); err != nil {
			return
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

type bidRecv struct {
	from proto.NodeID
	url string 
}

func (domain *Domain) receiveBidFrom(from proto.NodeID,  url string  ) {
	select {
	case domain.bidIncoming <- bidRecv{from: from, url: url }:
	
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

	//case <-domain.ctx.Done():
	//	return false
	}

	select {
	case want := <-resp:
		return want

	//case <-domain.ctx.Done():
	//	return false
	}
}

func (domain *Domain) interestedIn(url string ) bool {
	return domain.interest.Contains(url) || domain.isLiveWant(url)
}

const provSearchDelay = time.Second * 10


func (domain *Domain) addActivePeer(p proto.NodeID) {
	if _, ok := domain.activePeers[p]; !ok {
		domain.activePeers[p] = struct{}{}
		domain.activePeersArr = append(domain.activePeersArr, p)

		//todo
		//cmgr := s.f.network.ConnectionManager()
		//cmgr.TagPeer(p, s.tag, 10)
	}
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
			for _, bidding := range biddings { 
				//when forwad probability of this url is zero, get parent of the node, 
				//caculate forward probability of this url, 

				if bidding.Probability == 0.0 { 
					bidding.Probability, _ = domain.GetProbability(bidding.ParentUrl, bidding.ParentProbability,  bidding.Url ) 
				}
			}


			//insert bidding in biddings accroding to their probability. 
			domain.tofetch.FastInserts(biddings) 

			addBiddings := make([]types.UrlBidding, 0, len(biddings))

			for {
				if (len(domain.liveBiddings) >= activeBiddingsLimit) {
					break 
				}

				bidding, err := domain.tofetch.Pop()
				if  err != nil  {
					break 
				}

				//currentHeight is int32
				currentHeight := uint32(domain.chain.GetCurrentHeight())

				//get or create url node from url chain
				urlNode, err := domain.GetUrlNode(bidding.Url)	
				if err != nil || urlNode == nil {
					urlNode, err = domain.CreateUrlNode(bidding.Url, currentHeight, 0, 0, 0 )	
					if err != nil || urlNode == nil {
						continue 
					}
				}

				//read last requested height, and filter out urls which crawled too fast, 
				if (urlNode.LastRequestedHeight + IncrementalCrawlingInterval) < currentHeight { 
					continue 
				} 

				if bidding.Probability <= MinFrowardProbability { 
					continue 
				}

				//todo, increment requested count and save to url chain
				domain.SetLastRequestedHeight(bidding.Url, currentHeight)

				addBiddings = append(addBiddings, bidding) 
			}

			domain.addBiddings(ctx, addBiddings )


		case bid := <-domain.bidIncoming:
			domain.tick.Stop()

			if bid.from != "" {
				domain.addActivePeer(bid.from)
			}

			bidding, exist := domain.f.bc.completed_bl.Contains(bid.url) 
			if exist == true { 
				
				domain.BiddingCompleted(ctx, bid.url )

				//Move back
				//this has been moved to crawler side 
				c, _ := bidding.GetCid() 
				domain.SetCid(bidding.GetUrl(), c )

			}

			//todo
			//domain.resetTick()

		case biddings := <-domain.cancelBiddings:
			domain.cancel(biddings)

		case <-domain.tick.C:
			biddings := make([]types.UrlBidding, 0 ) 

			//move request ready to go from wait queue to domain.tofetch
			for {
				_, bidding, err := domain.waitQueue.Pop() 
				if err != nil  {
					break 
				}

				//delete job from wait queue 
				biddings = append(biddings, bidding ) 
			}
			
			if len(biddings) > 0  {
				domain.PutBiddings(ctx, biddings)
			}

			//todo, make this tick for waitQueue
			domain.resetTick()

		case p := <-newpeers:
			domain.addActivePeer(p)

		case lwchk := <-domain.interestReqs:
			lwchk.resp <- domain.urlIsWanted(lwchk.url)

		case <-ctx.Done():
			domain.tick.Stop()
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
	if domain.urlIsWanted(url) {
		tval, ok := domain.liveBiddings[url]
		if ok {
			domain.latTotal += time.Since(tval)
			delete(domain.liveBiddings, url)
		} else {
			domain.tofetch.Remove(url)
		}

		urlNode, err := domain.GetUrlNode(url) 
		if err == nil {
			currentHeight := uint32(domain.chain.GetCurrentHeight()) 
			deltaHeight := currentHeight - urlNode.LastCrawledHeight
			newCrawlInterval := ( urlNode.CrawlInterval / 2 ) + ( deltaHeight / 2 )

			domain.SetCrawlInterval(url, newCrawlInterval) 

			//save last crawled height to url chain.
			domain.SetLastCrawledHeight(url, currentHeight) 

			//create a bidding in future for this url and push it to wait queue
			runtime := time.Now().Add(time.Duration(urlNode.CrawlInterval) * time.Second)
			domain.waitQueue.Push(runtime, types.UrlBidding {
							Url : url,
							Probability : 1.0,      //todo
							ExpectCrawlerCount : 1, //todo
						})
		}

		domain.fetchcnt++

		if next, err  := domain.tofetch.Pop();  err != nil   {
			domain.addBiddings(ctx, []types.UrlBidding{next})
		}
	}

}

//add parameter n, which indicate how many times the url should crawled at least, 
//if url was not blocked by this node,  move the url to local ledger, and crawl the other (n - 1) 
//times by remote peers
func (domain *Domain) addBiddings(ctx context.Context, biddings []types.UrlBidding) {
        for _, bidding := range biddings {
		now := time.Now()
		domain.liveBiddings[bidding.Url] = now
	}

	if len(biddings) > 0 {
		domain.f.bc.AddBiddings(ctx,  biddings, domain.activePeersArr, domain.domainID)
	}
}


func (domain *Domain) cancel(urls []string ) {
	for _, url := range urls {
		domain.tofetch.Remove(url)
	}
}

// GetBlock fetches a single block
func (domain *Domain) PutBidding(ctx context.Context, url string, probability float64, parentUrl string , parentProbability float64  ) (error) {
	bidding := types.UrlBidding {
		Url: url,
		Probability : probability, 
		ParentUrl : parentUrl, 
		ParentProbability : parentProbability, 
		ExpectCrawlerCount : MinCrawlersExpected, 
	}

	select {
		case domain.biddingReqs <- []types.UrlBidding{ bidding }:  
		case <-ctx.Done():
		//case <-domain.ctx.Done():
	}
	return nil 
}

func (domain *Domain) PutBiddings(ctx context.Context, biddings []types.UrlBidding ) error {
	select {
		case domain.biddingReqs <- biddings :  
		case <-ctx.Done():
		//case <-domain.ctx.Done():
	}
	return nil 
}

func (domain *Domain) RetriveUrlCid(ctx context.Context, parentUrl string, url string  ) (cid.Cid, error) {
	urlNode, err := domain.GetUrlNode(url)
	if err != nil {
		return cid.Cid{}, nil 
	}

	c, err := domain.GetCid(url) 
	if err != nil {
		return cid.Cid{}, nil 
	}

	err = domain.SetRetrivedCount(url, urlNode.RetrivedCount + 1 )
	if err != nil {
		return cid.Cid{}, nil 
	}
	
	// todo
	err = domain.AddUrlLinksCount(parentUrl, url ) 
	if err != nil {
		return cid.Cid{}, nil 
	}

	return c, nil 
}
