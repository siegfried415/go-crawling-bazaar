package frontera

import (
	"context"
	"sync"
	"time"
	//"fmt"
	"sort"

        //wyong, 20210203
        "crypto/ecdsa"

	//wyong, 20210203
        "github.com/pkg/errors"

	//engine "github.com/siegfried415/gdf-rebuild/frontera/decision"

	//wyong, 20200827 
	//bsmsg "github.com/siegfried415/gdf-rebuild/frontera/message"

	//fnet "github.com/siegfried415/gdf-rebuild/frontera/network"
	//biddinglist "github.com/siegfried415/gdf-rebuild/frontera/biddinglist"

	cid "github.com/ipfs/go-cid"

	//wyong, 20200925
        //pstore "github.com/libp2p/go-libp2p-peerstore"
        //host "github.com/libp2p/go-libp2p-core/host"

	//go-libp2p-net -> go-libp2p-core/network,
	//wyong, 20201029 
	//inet "github.com/libp2p/go-libp2p-core/network"
	libp2phelpers "github.com/libp2p/go-libp2p-core/helpers"

	protocol "github.com/libp2p/go-libp2p-core/protocol" 
	//peer "github.com/libp2p/go-libp2p-core/peer" 


	//peer "github.com/libp2p/go-libp2p-peer"
	//metrics "github/ipfs/go-metrics-interface"

	//wyong, 20200730 
        //"github.com/siegfried415/gdf-rebuild/route"
        "github.com/siegfried415/gdf-rebuild/types"
        //"github.com/siegfried415/gdf-rebuild/rpc"
        //"github.com/siegfried415/gdf-rebuild/rpc/mux"
        "github.com/siegfried415/gdf-rebuild/proto"
	net "github.com/siegfried415/gdf-rebuild/net" 

	//wyong, 20210203 
        "github.com/siegfried415/gdf-rebuild/kms"

	//wyong, 20200928
	//"github.com/ugorji/go/codec"

	//wyong, 20201215 
	log "github.com/siegfried415/gdf-rebuild/utils/log" 

        //wyong, 20210118
        ecvrf "github.com/vechain/go-ecvrf"

        //wyong, 20200218
        "github.com/mfonda/simhash"

)

const (
	MainCycleInterval = 10 * time.Second 
	CrawlingInterval = time.Duration(10) * time.Second
) 

type BiddingClient struct {

	f *Frontera 	//wyong, 20210204 

	// sync channels for Run loop
	biddingIncoming     chan *biddingSet
	connectEvent chan peerStatus     // notification channel for peers connecting/disconnecting
	peerReqs     chan chan []proto.NodeID // channel to request connected peers on

	// synchronized by Run loop, only touch inside there
	peers map[proto.NodeID]*msgQueue
	bl    *ThreadSafe
	bcbl  *ThreadSafe

	//wyong, 20190115
	completed_bl *ThreadSafe 

	//network fnet.BiddingNetwork
	host net.RoutedHost 

	ctx     context.Context
	cancel  func()

	//wantlistGauge metrics.Gauge
	//sentHistogram metrics.Histogram

	bidIncoming     chan *bidSet	//wyong, 20190119 

	nodeID	proto.NodeID	//wyong, 20200826  

	//wyong, 20210204 
        tick          *time.Timer

}

type peerStatus struct {
	connect bool
	peer    proto.NodeID
}

func NewBiddingClient(ctx context.Context , /* network fnet.BiddingNetwork, */ host net.RoutedHost,   nodeID proto.NodeID  ) *BiddingClient {
	ctx, cancel := context.WithCancel(ctx)
	//wantlistGauge := metrics.NewCtx(ctx, "wantlist_total",
	//	"Number of items in wantlist.").Gauge()
	//sentHistogram := metrics.NewCtx(ctx, "sent_all_blocks_bytes", "Histogram of blocks sent by"+
	//	" this bitswap").Histogram(metricsBuckets)

	return &BiddingClient {
		biddingIncoming:      make(chan *biddingSet, 10),
		connectEvent:  make(chan peerStatus, 10),
		peerReqs:      make(chan chan []proto.NodeID),
		peers:         make(map[proto.NodeID]*msgQueue),
		bl:            NewThreadSafe(),
		bcbl:          NewThreadSafe(),

		//wyong, 20200831 
		completed_bl : NewThreadSafe(),

		//wyong, 20200925 
		//network:       network,
		host: 		host, 

		ctx:           ctx,
		cancel:        cancel,
		//wantlistGauge: wantlistGauge,
		//sentHistogram: sentHistogram,
		bidIncoming :  make(chan *bidSet, 10), 	//wyong, 20190119 
		nodeID   : nodeID, 	//wyong, 20200826 
	}
}


//wyong, 20200831
func (bc *BiddingClient) waitBidResult(ctx context.Context, dc chan bool ) bool {
	var result bool = false 
	select {
	case result = <-dc :
		log.Debugf("BiddingClient/waitBidResult, result = <- dc\n")
	case <-bc.ctx.Done():
		log.Debugf("BiddingClient/waitBidResult, <-bc.ctx.Done()\n")
	case <-ctx.Done():
		log.Debugf("BiddingClient/waitBidResult, <-ctx.Done()\n")
	}
	return result 
}

//wyong, 20190119  
func (bc *BiddingClient) ReceiveBidForBiddings(ctx context.Context, url string, c cid.Cid,  from proto.NodeID,  domain proto.DomainID , hash []byte, proof []byte ) bool {
	var dc chan bool = make(chan bool) 	
	select {
	case bc.bidIncoming <- &bidSet{	url: url, 
					//bids: bids, wyong, 20200828  
					done: dc, 	//wyong, 20200831 
					cid : c,
					from : from, 
					domain: domain, 
	
					hash: hash,	//wyong, 20210203 
					proof: proof, 	//wyong, 20210203 
				}:
		log.Debugf("BiddingClient/receiveBidForBiddings, bc.bidIncoming <- &bidSet\n")
	case <-bc.ctx.Done():
		log.Debugf("BiddingClient/receiveBidForBiddings, <-bc.ctx.Done()\n")
		return false 
	case <-ctx.Done():
		log.Debugf("BiddingClient/receiveBidForBiddings, <-ctx.Done()\n")
		return false 
	}

	//wait for result, true if bidding sucessfully, false other. wyong, 20200831 
	return bc.waitBidResult(ctx, dc) 
}

type bidSet struct {
	url string  
	done chan bool	//wyong, 20200831 

	//wyong, 20200828 
	//bids map[proto.NodeID]*BidEntry
	from proto.NodeID
	cid cid.Cid 
	domain    proto.DomainID 

	hash	[]byte 	//wyong, 20210203 	
	proof 	[]byte	//wyong, 20210203 
}

// AddBiddings adds the given cids to the biddinglist, tracked by the given domain 
func (bc *BiddingClient) AddBiddings(ctx context.Context, biddings []types.UrlBidding, peers []proto.NodeID, domain proto.DomainID) {
	log.Debugf("BiddingClient/AddBiddings(10), want blocks: %s\n", biddings)
	bc.addBiddings(ctx, biddings, peers, false, domain)
}

// CancelBiddings removes the given cids from the biddinglist, tracked by the given domain 
func (bc *BiddingClient) CancelBiddings(ctx context.Context, urls []string,  peers []proto.NodeID, domain proto.DomainID) bool {
	biddings := make([]types.UrlBidding, 0, len(urls)) 
	for _, url := range urls { 
		if _, exist := bc.bl.Contains(url); exist { 
			biddings = append(biddings, types.UrlBidding { 
				Url : url, 
				Cancel : true , 
			}) 
		}
	}

	if len(biddings) > 0 { 
		bc.addBiddings(ctx, biddings, peers, true, domain)
		return true 
	}

	return false  
}

type biddingSet struct {
	biddings []types.UrlBidding 	//[]*bsmsg.BiddingEntry, wyong, 20200827 
	targets []proto.NodeID
	domain proto.DomainID  
}

func (bc *BiddingClient) addBiddings(ctx context.Context, biddings []types.UrlBidding, targets []proto.NodeID, cancel bool, domain proto.DomainID) {
	log.Debugf("BiddingClient/addBiddings(0)\n")

	//biddings := make([]types.UrlBidding , 0, len(urls))
	//for _, url := range urls {
	//	log.Debugf("BiddingClient/addBiddings(10), process url=%s\n", url )
	//	biddings = append(biddings,  types.UrlBidding {
	//		//wyong, 20200827 	
	//		//Cancel: cancel,
	//		//BiddingEntry:  NewBiddingEntry(url, kMaxPriority-i),
	//		Url : url, 
	//		Probability : 1.0,	//todo, wyong, 2bility020082o 
	//	})
	//}

	select {
	case bc.biddingIncoming <- &biddingSet{biddings: biddings, targets: targets, domain: domain }:
		log.Debugf("BiddingClient/addBiddings(20), bc.biddingIncoming <- &biddingSet\n")
	case <-bc.ctx.Done():
		log.Debugf("BiddingClient/addBiddings(30), <-bc.ctx.Done()\n")
	case <-ctx.Done():
		log.Debugf("BiddingClient/addBiddings(40), <-ctx.Done()")
	}
}

func (bc *BiddingClient) ConnectedPeers() []proto.NodeID {
	log.Debugf("BiddingClient/ConnectedPeers (10)\n")
	resp := make(chan []proto.NodeID)
	bc.peerReqs <- resp
	return <-resp
}


func (bc *BiddingClient) SendBids(ctx context.Context, msg *types.UrlBidMessage ) {
	log.Debugf("BiddingClient/SendBids(10)\n")
	// Blocks need to be sent synchronously to maintain proper backpressure
	// throughout the network stack
	//defer env.Sent()

	/* TODO!!! create msg from envelope accrodding to new schema, wyong, 20190115 
	//msgSize := 0
	msg := bsmsg.New(false)
	for _, bid := range env.Message.Bids() {
		//msgSize += len(block.RawData())
		msg.AddBidEntry(&bid)
		//log.Infof("Sending block %s to %s", bid, env.Peer)
	}
	*/


	/* wyong, 20200730 
	//bc.sentHistogram.Observe(float64(msgSize))
	err := bc.network.SendMessage(ctx, env.Peer, env.Message )
	if err != nil {
		//log.Infof("sendBids error: %s", err)
	}
	*/

	//var caller rpc.PCaller

	//todo, wyong, 20200730 
	//if cfg.UseDirectRPC {
	//	caller = rpc.NewPersistentCaller(env.Peer)
	//} else {
	//	caller = mux.NewPersistentCaller(env.Peer)
	//}

	//wyong, 20200827 
	target := msg.Header.UrlBidHeader.Target 
	log.Debugf("BiddingClient/SendBids(12), target =%s\n", target )

	//bugfix, wyong, 20201208 
	//if target.IsEmpty() {
	if string(target) == "" { 
		log.Debugf("BiddingClient/SendBids(15), target is empty\n")
		return
	}

	log.Debugf("BiddingClient/SendBids(20), target=%s\n", target )
	//caller = mux.NewPersistentCaller(target) 
	s, err := bc.host.NewStreamExt(ctx, target, protocol.ID("FRT.Bid"))
	if err != nil {
                return 
        }

        //var response types.Response
	//err := caller.Call(route.FronteraBid.String(), msg, &response ) 
	//if err == nil {
	//	log.Debugf("BiddingClient/SendBids(25), err=%s\n", err.Error())
	//	return
	//}

        if _, err = s.SendMsg(ctx, msg ); err != nil {
                s.Reset()
                return 
        }

	log.Debugf("BiddingClient/SendBids(30)\n")

	//todo, wyong, 20200925 

	//wyong, 20201029 
	//go inet.AwaitEOF(s)
	go libp2phelpers.AwaitEOF(s)

        s.Close()

}


func (bc *BiddingClient) startPeerHandler(p proto.NodeID) *msgQueue {
	log.Debugf("BiddingClient/startPeerHandler(10)\n")

	mq, ok := bc.peers[p]
	if ok {
		mq.refcnt++
		return nil
	}

	mq = bc.newMsgQueue(p)

	//wyong, 20200827 
	// new peer, we will want to give them our full biddinglist
	//fullbiddinglist := bsmsg.New(true, string(bc.nodeID))
	biddings := make([]types.UrlBidding, 0, len(bc.bcbl.Entries()))
	for _, b := range bc.bcbl.Entries() {
		//todo, wyong, 20210203 
		//for k := range e.SesTrk {
		//	mq.bl.AddBiddingEntry(e, k)
		//}
		mq.bl.AddBiddingEntry(b, b.DomainID) 

		//fullbiddinglist.AddBidding(e.Url, e.Priority, nil )
		biddings = append(biddings, types.UrlBidding{
						Url: b.Url,
						Probability : b.Probability, 	
					})
	}


	//build bidding message, wyong, 20200827 
	mq.out = types.UrlBiddingMessage{
		Header: types.SignedUrlBiddingHeader{
			UrlBiddingHeader: types.UrlBiddingHeader{
				QueryType:    types.WriteQuery,
				NodeID:       bc.nodeID,
				//DomainID:     bc.DomainID,

				//todo, wyong, 20200817
				//ConnectionID: connID,
				//SeqNo:        seqNo,
				//Timestamp:    getLocalTime(),
			},
		},
		Payload: types.UrlBiddingPayload{
			Requests: biddings,
		},
	}

	//mq.out = fullbiddinglist
	mq.work <- struct{}{}

	bc.peers[p] = mq
	go mq.runQueue(bc.ctx)
	return mq
}

func (bc *BiddingClient) stopPeerHandler(p proto.NodeID) {
	log.Debugf("BiddingClient/stopPeerHandler(10)\n")
	pq, ok := bc.peers[p]
	if !ok {
		// TODO: log error?
		return
	}

	pq.refcnt--
	if pq.refcnt > 0 {
		return
	}

	close(pq.done)
	delete(bc.peers, p)
}


func (bc *BiddingClient) Connected(p proto.NodeID) {
	select {
	case bc.connectEvent <- peerStatus{peer: p, connect: true}:
		log.Debugf("BiddingClient/Connected, bc.connectEvent <- peerStatus\n")
	case <-bc.ctx.Done():
		log.Debugf("BiddingClient/Connected, <-bc.ctx.Done()\n")
	}
}

func (bc *BiddingClient) Disconnected(p proto.NodeID) {
	select {
	case bc.connectEvent <- peerStatus{peer: p, connect: false}:
	case <-bc.ctx.Done():
	}
}

//wyong, 20190126 
func (bc *BiddingClient) GetCompletedBiddings() ([]*BiddingEntry, error) {
	result := make([]*BiddingEntry, 0, bc.bl.Len())

	//return bc.completed_biddinglist.Entries(), nil 
	for _, bidding := range bc.bl.Entries() {
		if (bidding.CountOfBids() >= 2 ) {
			result = append(result, bidding) 
		}
	}

	return result, nil 
}

//wyong, 20190131
func (bc *BiddingClient) GetUncompletedBiddings() ([]*BiddingEntry, error) {
	return bc.bl.Entries(), nil 
}

//wyong, 20210206 
func (bc *BiddingClient) NeedCrawlMore(bidding *BiddingEntry) (int, error ) {
	needCrawlCount := bidding.ExpectCrawlerCount  
	bids := bidding.GetBids() 
	if len(bids) == 0 {
		return needCrawlCount, nil 
	}

	bidCounts := make ([]int, 0, len(bids))
	for i :=0; i < len(bids) ; i++ {
		for j := i+1 ; j< len(bids); j++ {
			if simhash.Compare(bids[i].Hash,  bids[j].Hash) < 2 {
				bidCounts[i]++
				bidCounts[j]++
			}
		}
	}

        sort.Slice(bidCounts, func(i, j int) bool {
                return bidCounts[i] < bidCounts[j]
        })

	if MinCrawlersExpected > bidCounts[0]{
		needCrawlCount = MinCrawlersExpected - bidCounts[0]
	}

	return needCrawlCount, nil 
}

//wyong, 20210203 
func (bc *BiddingClient) GetUnAckedPeers (bidding *BiddingEntry) ([]proto.NodeID, error ) {
	domain, exist := bc.f.DomainForID(bidding.DomainID)
	if exist != true {
		err := errors.New("domain not exist")
		return []proto.NodeID{}, err 	
	}

	bids := bidding.GetBids()
	if bids == nil || len(bids)==0 {
		return domain.activePeersArr, nil 
	}

	peers := make([]proto.NodeID, 0, len(domain.activePeersArr)) 
	for _, peer := range domain.activePeersArr {
		acked := true 
		for _, bid := range bids {
			if peer == bid.From {
				acked = true  
				break 
			}
		}
		if !acked {
			peers = append(peers, peer) 
		}
	}
	
	return peers, nil 
}

//wyong, 20210204
func (bc *BiddingClient) recrawlBidding(bidding *BiddingEntry, recrawlCount int ) error {
	newBidding := types.UrlBidding {
		Url : bidding.Url, 
		Probability : bidding.Probability,
		ParentUrl   : bidding.ParentUrl, 
		ExpectCrawlerCount : recrawlCount , 
	}

	//filter out those peers that have ack our, wyong, 20210203 
	peers , err := bc.GetUnAckedPeers(bidding ) 
	if err != nil {
		err = errors.New("domain not exist")
		return err 	
	}

	bc.addBiddings(context.Background(), 
			[]types.UrlBidding { newBidding }, 
				peers,	//wyong, 20210203 
				false, 		//wyong, 20210203 
			bidding.DomainID ) 

	return nil 

}

// TODO: use goprocess here once i trust it
func (bc *BiddingClient) Run() {
	log.Debugf("BiddingClient/Run(10)\n")

	//wyong, 20210204 
        bc.tick = time.NewTimer(MainCycleInterval )

	// NOTE: Do not open any streams or connections from anywhere in this
	// event loop. Really, just don't do anything likely to block.
	for {
		select {
		case ws := <-bc.biddingIncoming:
			log.Debugf("BiddingClient/Run(30), ws : <- bc.biddingIncoming\n")

			// todo, wyong, 20201217 
			//// is this a broadcast or not?
			//brdc := len(ws.targets) == 0

			// add changes to our biddinglist
			//for _, e := range ws.biddings {
			//	if e.Cancel {
			//		if brdc {
			//			bc.bcbl.Remove(e.Url, ws.domain )
			//		}
			//
			//		if bc.bl.Remove(e.Url, ws.domain) {
			//			//bc.biddinglistGauge.Dec()
			//		}
			//					
			//	} else {
			//		if brdc {
			//			bc.bcbl.Add(e.Url, e.Probability,  ws.domain)
			//		}
			//		if bc.bl.Add(e.Url, e.Probability, ws.domain ) {
			//			//bc.biddinglistGauge.Inc()
			//		}
			//	}
			//}

			//wyong, 20210203 
			for _, b := range ws.biddings {
				if bc.bl.Add(b.Url, b.ParentUrl, b.Probability, ws.domain, b.ExpectCrawlerCount , time.Now()) {
					//bc.biddinglistGauge.Inc()
				}
			}

			log.Debugf("BiddingClient/Run(40)\n")
			// broadcast those biddinglist changes
			if len(ws.targets) == 0 {
				log.Debugf("BiddingClient/Run(50) ")
				for id, p := range bc.peers {
					log.Debugf("BiddingClient/Run(60), peer=%s\n", id )
					p.addBiddingMessage(ws.biddings, ws.domain , bc.nodeID )
				}
			} else {
				log.Debugf("BiddingClient/Run(70)\n")
				for _, t := range ws.targets {
					log.Debugf("BiddingClient/Run(80), target_node_id=%s\n", t)
					p, ok := bc.peers[t]
					if !ok {
						//todo, wyong, 20200825 
						log.Debugf("BiddingClient/Run(90)\n")
						//log.Debugf("tried sending biddinglist change to non-partner peer: %s\n", t)
						//continue
						p = bc.startPeerHandler(t)
					}

					log.Debugf("BiddingClient/Run(100)\n")
					p.addBiddingMessage(ws.biddings, ws.domain, bc.nodeID )
				}
				log.Debugf("BiddingClient/Run(110)\n")
			}

		case bs := <-bc.bidIncoming:	//wyong, 20190119 
			log.Debugf("BiddingClient/Run(20), bs : <- bc.bidIncoming\n")

			url := bs.url 
			bidding, _ := bc.bl.Contains(url)
			if bidding == nil {
				bs.done <- false //wyong, 20200831
				continue
			}

			//wyong, 20210203 
			pk, err := kms.GetPublicKey(bs.from)
			if err != nil {
				err = errors.Wrap(err, "failed to cache public key")
				continue 
			}

			//wyong, 20210118
			// `pi` is the VRF proof
			// pk is publick key of bid.From 
			_, err = ecvrf.NewSecp256k1Sha256Tai().Verify((*ecdsa.PublicKey)(pk), []byte(bs.url), bs.proof)
			if err != nil {
				// invalid proof
				continue	
			}

			//todo, verify if crawler has successfully in competition,  wyong, 20210118
			//if verify(bs.beta, expectedCrawler, minerPower, networkPower) != ok {
			//	continue 	
			//} 

			/*wyong, 20200828 
			// add bids to biddinglist
			//for _, bid := range bs.bids {
				//wyong, 20190115 
				//Store bid to bidding, and move the bidding to completed biddinglist
				//if bidding has already two bids for it.  wyong, 20190119 

				if !bidding.AddBid(url, bid.Cid, bid.From ) {
					continue
				}

			}
			*/

			//todo, wyong, 20201217 
			//wyong, 20200828 
			if !bidding.AddBid(url, bs.cid, bs.from ) {
				bs.done <- false //wyong, 20200831 
				continue 
			}

			if (bidding.CountOfBids() < bidding.ExpectCrawlerCount ) {
				continue 
			}

			//todo, check if cids from every bid match or not. 
			//if not match, caculate how many more crawling are needed. 
			//bids := bidding.GetBids() 
			need_crawl_more, err := bc.NeedCrawlMore(bidding) 
			if need_crawl_more > 0 {
				//add need_crawl_count to let bidding (in bidding list) can handle more bid, 
				//this is subtle, wyong, 20210204 
				bidding.ExpectCrawlerCount +=  need_crawl_more

				bc.recrawlBidding(bidding, need_crawl_more) 
				continue 
			}

			//Crawled succeed 
			if bc.bl.Remove(url, bs.domain ) {
				//bc.biddinglistGauge.Dec()
			}
		
			bc.completed_bl.AddBiddingEntry(bidding, bs.domain ) 
			bs.done <- true  //wyong, 20200831 
			continue 
			
			//todo, wyong, 20201217
			//UrlCrawled()
			
			//wyong, 20210117 
			//bs.done <- false 	

			//TODO, don't forget send message to those peers to let them know we havn't 
			//need those url any more,  wyong, 20190119

		//wyong, 20210116
		case <-bc.tick.C:
			//todo, rebroadcast bidding when crawling timer timeout, 
			//wyong, 20210117 
			for _, bidding := range bc.bl.Entries() {

				bidCount := len(bidding.GetBids())
				if (bidCount >= bidding.ExpectCrawlerCount ) || 
					(bidCount < bidding.ExpectCrawlerCount && time.Since(bidding.LastBroadcastTime) < CrawlingInterval ) {
					continue 
				}
				
				need_recrawl_more := bidding.ExpectCrawlerCount - bidCount 
				if need_recrawl_more > 0 {
					//don't add need_recrawl_more of bidding,
					//because we don't need bids more then bidding.ExpectCrawlerCount. 
					//wyong, 20210204 
					bc.recrawlBidding(bidding, need_recrawl_more ) 
				}
			}

			bc.tick.Reset(MainCycleInterval)

		case p := <-bc.connectEvent:
			log.Debugf("BiddingClient/Run(120), P := <-bc.connectEvent\n")
			if p.connect {
				log.Debugf("BiddingClient/Run(130)\n")
				bc.startPeerHandler(p.peer)
			} else {
				log.Debugf("BiddingClient/Run(140)\n")
				bc.stopPeerHandler(p.peer)
			}
		case req := <-bc.peerReqs:
			log.Debugf("BiddingClient/Run(150), req := <-bc.peerReqs\n")
			peers := make([]proto.NodeID, 0, len(bc.peers))
			for p := range bc.peers {
				log.Debugf("BiddingClient/Run(160)\n")
				peers = append(peers, p)
			}
			req <- peers
		case <-bc.ctx.Done():
			log.Debugf("BiddingClient/Run(170), <-bc.ctx.Done()\n")
			return
		}
	}
}

func (bc *BiddingClient) newMsgQueue(p proto.NodeID) *msgQueue {
	//log.Debugf("newMsgQueue called...")
	return &msgQueue{
		done:    make(chan struct{}),
		work:    make(chan struct{}, 1),
		bl:      NewThreadSafe(),

		//wyong, 20200925 
		//network: wm.network,
		host 	: bc.host, 
		
		//wyong, 20200825 
		outlk:	new(sync.Mutex), 

		p:       p,
		refcnt:  1,
	}
}

