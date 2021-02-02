package frontera

import (
	"context"
	"sync"
	"time"
	//"fmt"

	//engine "github.com/siegfried415/gdf-rebuild/frontera/decision"

	//wyong, 20200827 
	//bsmsg "github.com/siegfried415/gdf-rebuild/frontera/message"

	//fnet "github.com/siegfried415/gdf-rebuild/frontera/network"
	biddinglist "github.com/siegfried415/gdf-rebuild/frontera/biddinglist"

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

	//wyong, 20200928
	//"github.com/ugorji/go/codec"

	//wyong, 20201215 
	log "github.com/siegfried415/gdf-rebuild/utils/log" 

        //wyong, 20210118
        ecvrf github.com/vechain/go-ecvrf

)

type BiddingManager struct {
	// sync channels for Run loop
	biddingIncoming     chan *biddingSet
	connectEvent chan peerStatus     // notification channel for peers connecting/disconnecting
	peerReqs     chan chan []proto.NodeID // channel to request connected peers on

	// synchronized by Run loop, only touch inside there
	peers map[proto.NodeID]*msgQueue
	bl    *biddinglist.ThreadSafe
	bcbl  *biddinglist.ThreadSafe

	//wyong, 20190115
	completed_bl *biddinglist.ThreadSafe 

	//network fnet.BiddingNetwork
	host net.RoutedHost 

	ctx     context.Context
	cancel  func()

	//wantlistGauge metrics.Gauge
	//sentHistogram metrics.Histogram

	bidIncoming     chan *bidSet	//wyong, 20190119 

	nodeID	proto.NodeID	//wyong, 20200826  
}

type peerStatus struct {
	connect bool
	peer    proto.NodeID
}

func NewBiddingManager(ctx context.Context , /* network fnet.BiddingNetwork, */ host net.RoutedHost,   nodeID proto.NodeID  ) *BiddingManager {
	ctx, cancel := context.WithCancel(ctx)
	//wantlistGauge := metrics.NewCtx(ctx, "wantlist_total",
	//	"Number of items in wantlist.").Gauge()
	//sentHistogram := metrics.NewCtx(ctx, "sent_all_blocks_bytes", "Histogram of blocks sent by"+
	//	" this bitswap").Histogram(metricsBuckets)

	return &BiddingManager{
		biddingIncoming:      make(chan *biddingSet, 10),
		connectEvent:  make(chan peerStatus, 10),
		peerReqs:      make(chan chan []proto.NodeID),
		peers:         make(map[proto.NodeID]*msgQueue),
		bl:            biddinglist.NewThreadSafe(),
		bcbl:          biddinglist.NewThreadSafe(),

		//wyong, 20200831 
		completed_bl : biddinglist.NewThreadSafe(),

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
func (bm *BiddingManager) waitBidResult(ctx context.Context, dc chan bool ) bool {
	var result bool = false 
	select {
	case result = <-dc :
		log.Debugf("BiddingManager/waitBidResult, result = <- dc\n")
	case <-bm.ctx.Done():
		log.Debugf("BiddingManager/waitBidResult, <-bm.ctx.Done()\n")
	case <-ctx.Done():
		log.Debugf("BiddingManager/waitBidResult, <-ctx.Done()\n")
	}
	return result 
}

//wyong, 20190119  
func (bm *BiddingManager) ReceiveBidForBiddings(ctx context.Context, url string, c cid.Cid,  from proto.NodeID,  domain proto.DomainID ) bool {
	//bm.addBiddings(context.Background(), url,bids, peers, true, domain )
	var dc chan bool = make(chan bool) 	
	select {
	case bm.bidIncoming <- &bidSet{	url: url, 
					//bids: bids, wyong, 20200828  
					done: dc, 	//wyong, 20200831 
					cid : c,
					from : from, 
					domain: domain, 
				}:
		log.Debugf("BiddingManager/receiveBidForBiddings, bm.bidIncoming <- &bidSet\n")
	case <-bm.ctx.Done():
		log.Debugf("BiddingManager/receiveBidForBiddings, <-bm.ctx.Done()\n")
		return false 
	case <-ctx.Done():
		log.Debugf("BiddingManager/receiveBidForBiddings, <-ctx.Done()\n")
		return false 
	}

	//wait for result, true if bidding sucessfully, false other. wyong, 20200831 
	return bm.waitBidResult(ctx, dc) 
}

type bidSet struct {
	url string  
	done chan bool	//wyong, 20200831 

	//wyong, 20200828 
	//bids map[proto.NodeID]*biddinglist.BidEntry
	from proto.NodeID
	cid cid.Cid 
	
	domain    proto.DomainID 
}

// AddBiddings adds the given cids to the biddinglist, tracked by the given domain 
func (bm *BiddingManager) AddBiddings(ctx context.Context, urls []string, peers []proto.NodeID, domain proto.DomainID) {
	log.Debugf("BiddingManager/AddBiddings(10), want blocks: %s\n", urls)
	bm.addBiddings(ctx, urls, peers, false, domain)
}

// CancelBiddings removes the given cids from the biddinglist, tracked by the given domain 
func (bm *BiddingManager) CancelBiddings(ctx context.Context, urls []string,  peers []proto.NodeID, domain proto.DomainID) bool {
	bm.addBiddings(context.Background(), urls, peers, true, domain)
	return true 
}

type biddingSet struct {
	biddings []types.UrlBidding 	//[]*bsmsg.BiddingEntry, wyong, 20200827 
	targets []proto.NodeID
	domain proto.DomainID  
}

func (bm *BiddingManager) addBiddings(ctx context.Context, urls []string, targets []proto.NodeID, cancel bool, domain proto.DomainID) {
	log.Debugf("BiddingManager/addBiddings(0)\n")

	biddings := make([]types.UrlBidding , 0, len(urls))
	for _, url := range urls {
		log.Debugf("BiddingManager/addBiddings(10), process url=%s\n", url )
		biddings = append(biddings,  types.UrlBidding {
			//wyong, 20200827 	
			//Cancel: cancel,
			//BiddingEntry:  biddinglist.NewRefBiddingEntry(url, kMaxPriority-i),
			Url : url, 
			Probability : 1.0,	//todo, wyong, 2bility020082o 
		})
	}
	select {
	case bm.biddingIncoming <- &biddingSet{biddings: biddings, targets: targets, domain: domain }:
		log.Debugf("BiddingManager/addBiddings(20), bm.biddingIncoming <- &biddingSet\n")
	case <-bm.ctx.Done():
		log.Debugf("BiddingManager/addBiddings(30), <-bm.ctx.Done()\n")
	case <-ctx.Done():
		log.Debugf("BiddingManager/addBiddings(40), <-ctx.Done()")
	}
}

func (bm *BiddingManager) ConnectedPeers() []proto.NodeID {
	log.Debugf("BiddingManager/ConnectedPeers (10)\n")
	resp := make(chan []proto.NodeID)
	bm.peerReqs <- resp
	return <-resp
}


func (bm *BiddingManager) SendBids(ctx context.Context, msg *types.UrlBidMessage ) {
	log.Debugf("BiddingManager/SendBids(10)\n")
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
	//bm.sentHistogram.Observe(float64(msgSize))
	err := bm.network.SendMessage(ctx, env.Peer, env.Message )
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
	log.Debugf("BiddingManager/SendBids(12), target =%s\n", target )

	//bugfix, wyong, 20201208 
	//if target.IsEmpty() {
	if string(target) == "" { 
		log.Debugf("BiddingManager/SendBids(15), target is empty\n")
		return
	}

	log.Debugf("BiddingManager/SendBids(20), target=%s\n", target )
	//caller = mux.NewPersistentCaller(target) 
	s, err := bm.host.NewStreamExt(ctx, target, protocol.ID("FRT.Bid"))
	if err != nil {
                return 
        }

        //var response types.Response
	//err := caller.Call(route.FronteraBid.String(), msg, &response ) 
	//if err == nil {
	//	log.Debugf("BiddingManager/SendBids(25), err=%s\n", err.Error())
	//	return
	//}

        if _, err = s.SendMsg(ctx, msg ); err != nil {
                s.Reset()
                return 
        }

	log.Debugf("BiddingManager/SendBids(30)\n")

	//todo, wyong, 20200925 

	//wyong, 20201029 
	//go inet.AwaitEOF(s)
	go libp2phelpers.AwaitEOF(s)

        s.Close()

}


func (bm *BiddingManager) startPeerHandler(p proto.NodeID) *msgQueue {
	log.Debugf("BiddingManager/startPeerHandler(10)\n")

	mq, ok := bm.peers[p]
	if ok {
		mq.refcnt++
		return nil
	}

	mq = bm.newMsgQueue(p)

	//wyong, 20200827 
	// new peer, we will want to give them our full biddinglist
	//fullbiddinglist := bsmsg.New(true, string(bm.nodeID))
	biddings := make([]types.UrlBidding, 0, len(bm.bcbl.Entries()))
	for _, e := range bm.bcbl.Entries() {
		for k := range e.SesTrk {
			mq.bl.AddBiddingEntry(e, k)
		}
		//fullbiddinglist.AddBidding(e.Url, e.Priority, nil )
		biddings = append(biddings, types.UrlBidding{
						Url: e.Url,
						Probability : e.Probability, 	
					})
	}


	//build bidding message, wyong, 20200827 
	mq.out = types.UrlBiddingMessage{
		Header: types.SignedUrlBiddingHeader{
			UrlBiddingHeader: types.UrlBiddingHeader{
				QueryType:    types.WriteQuery,
				NodeID:       bm.nodeID,
				//DomainID:     bm.DomainID,

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

	bm.peers[p] = mq
	go mq.runQueue(bm.ctx)
	return mq
}

func (bm *BiddingManager) stopPeerHandler(p proto.NodeID) {
	log.Debugf("BiddingManager/stopPeerHandler(10)\n")
	pq, ok := bm.peers[p]
	if !ok {
		// TODO: log error?
		return
	}

	pq.refcnt--
	if pq.refcnt > 0 {
		return
	}

	close(pq.done)
	delete(bm.peers, p)
}


func (bm *BiddingManager) Connected(p proto.NodeID) {
	select {
	case bm.connectEvent <- peerStatus{peer: p, connect: true}:
		log.Debugf("BiddingManager/Connected, bm.connectEvent <- peerStatus\n")
	case <-bm.ctx.Done():
		log.Debugf("BiddingManager/Connected, <-bm.ctx.Done()\n")
	}
}

func (bm *BiddingManager) Disconnected(p proto.NodeID) {
	select {
	case bm.connectEvent <- peerStatus{peer: p, connect: false}:
	case <-bm.ctx.Done():
	}
}

//wyong, 20190126 
func (bm *BiddingManager) GetCompletedBiddings() ([]*biddinglist.BiddingEntry, error) {
	result := make([]*biddinglist.BiddingEntry, 0, bm.bl.Len())

	//return bm.completed_biddinglist.Entries(), nil 
	for _, bidding := range bm.bl.Entries() {
		if (bidding.CountOfBids() >= 2 ) {
			result = append(result, bidding) 
		}
	}

	return result, nil 
}

//wyong, 20190131
func (bm *BiddingManager) GetUncompletedBiddings() ([]*biddinglist.BiddingEntry, error) {
	return bm.bl.Entries(), nil 
}

// TODO: use goprocess here once i trust it
func (bm *BiddingManager) Run() {
	log.Debugf("BiddingManager/Run(10)\n")
	// NOTE: Do not open any streams or connections from anywhere in this
	// event loop. Really, just don't do anything likely to block.
	for {
		select {
		case ws := <-bm.biddingIncoming:
			log.Debugf("BiddingManager/Run(30), ws : <- bm.biddingIncoming\n")

			// todo, wyong, 20201217 
			//// is this a broadcast or not?
			//brdc := len(ws.targets) == 0

			//// add changes to our biddinglist
			//for _, e := range ws.biddings {
			//	if e.Cancel {
			//		if brdc {
			//			bm.bcbl.Remove(e.Url, ws.domain )
			//		}
			//
			//		if bm.bl.Remove(e.Url, ws.domain) {
			//			//bm.biddinglistGauge.Dec()
			//		}
			//					
			//	} else {
			//		if brdc {
			//			bm.bcbl.Add(e.Url, e.Probability,  ws.domain)
			//		}
			//		if bm.bl.Add(e.Url, e.Probability, ws.domain ) {
			//			//bm.biddinglistGauge.Inc()
			//		}
			//	}
			//}

			log.Debugf("BiddingManager/Run(40)\n")
			// broadcast those biddinglist changes
			if len(ws.targets) == 0 {
				log.Debugf("BiddingManager/Run(50) ")
				for id, p := range bm.peers {
					log.Debugf("BiddingManager/Run(60), peer=%s\n", id )
					p.addBiddingMessage(ws.biddings, ws.domain , bm.nodeID )
				}
			} else {
				log.Debugf("BiddingManager/Run(70)\n")
				for _, t := range ws.targets {
					log.Debugf("BiddingManager/Run(80), target_node_id=%s\n", t)
					p, ok := bm.peers[t]
					if !ok {
						//todo, wyong, 20200825 
						log.Debugf("BiddingManager/Run(90)\n")
						//log.Debugf("tried sending biddinglist change to non-partner peer: %s\n", t)
						//continue
						p = bm.startPeerHandler(t)
					}

					log.Debugf("BiddingManager/Run(100)\n")
					p.addBiddingMessage(ws.biddings, ws.domain, bm.nodeID )
				}
				log.Debugf("BiddingManager/Run(110)\n")
			}

		case bs := <-bm.bidIncoming:	//wyong, 20190119 
			log.Debugf("BiddingManager/Run(20), bs : <- bm.bidIncoming\n")

			url := bs.url 
			bidding, _ := bm.bl.Contains(url)
			if bidding == nil {
				bs.done <- false //wyong, 20200831
				continue
			}
	
			//wyong, 20210118
			// `pi` is the VRF proof
			// pk is publick key of bid.From 
			beta, err := ecvrf.NewSecp256k1Sha256Tai().Verify(pk, []byte(bid.url), bid.pi)
			if err != nil {
				// invalid proof
				continue	
			}

			//todo, verify if crawler has successfully in competition,  wyong, 20210118
			if verify(beta, expectedCrawler, minerPower, networkPower) != ok {
				continue 	
			} 

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

			if (bidding.CountOfBids() >= n ) {
				//todo, check if cids from every bid match or not. 
				//if not match, caculate how many more crawling are needed. 
				if need_crawl_count > 0 {
					//n = n + need_crawl_count 
					bm.AddBiddings(ctx, bidding.Url, need_crawl_count ) 
				}
			}

			//Crawled succeed 
			if bm.bl.Remove(url, bs.domain ) {
				//bm.biddinglistGauge.Dec()
			}
		
			bm.completed_bl.AddBiddingEntry(bidding, bs.domain ) 
			bs.done <- true  //wyong, 20200831 
			continue 
			
			//todo, wyong, 20201217
			//UrlCrawled()
			
			//wyong, 20210117 
			//bs.done <- false 	

			//TODO, don't forget send message to those peers to let them know we havn't 
			//need those url any more,  wyong, 20190119

			
		//wyong, 20210116
		case <-bm.tick.C:
			//todo, rebroadcast bidding when confirm timer or crawling timer timeout, 
			//wyong, 20210117 
			for _, bidding := range bm.bl.Entries() {
				//caculate confirmed count of current bidding.
				//for _, bid := range bidding.Bids{
				//	if bid.status >= STATUS_CONFIRMED {
				//		confirmed_count ++ 	
				//	}
				//}
				//
				//if confirmed_count < n {
				//	if (bidding.lastBroadcastTime + BroadcastInterval ) > Now() {
				//		bm.AddBiddings(ctx, bidding.Url, n - confirmed_count ) 
				//	}
				//	continue 
				//}

				//has received at least n confirmed from peers. 
				for _, bid := range bidding.Bids {
					if ((bid.status != STATUS_SUCCESS 
						&& bid.lastBroadcastTime + CrawlingInterval > Now())) {
						//todo, add bid.peer to blocklist.
						need_recrawl_count++ 	
					}
				}

				if need_recrawl_count > 0 {
					//n = n + need_recrawl_count 
					bm.AddBiddings(ctx, bidding.Url, need_recrawl_count ) 
				}

			}

		case p := <-bm.connectEvent:
			log.Debugf("BiddingManager/Run(120), P := <-bm.connectEvent\n")
			if p.connect {
				log.Debugf("BiddingManager/Run(130)\n")
				bm.startPeerHandler(p.peer)
			} else {
				log.Debugf("BiddingManager/Run(140)\n")
				bm.stopPeerHandler(p.peer)
			}
		case req := <-bm.peerReqs:
			log.Debugf("BiddingManager/Run(150), req := <-bm.peerReqs\n")
			peers := make([]proto.NodeID, 0, len(bm.peers))
			for p := range bm.peers {
				log.Debugf("BiddingManager/Run(160)\n")
				peers = append(peers, p)
			}
			req <- peers
		case <-bm.ctx.Done():
			log.Debugf("BiddingManager/Run(170), <-bm.ctx.Done()\n")
			return
		}
	}
}

func (bm *BiddingManager) newMsgQueue(p proto.NodeID) *msgQueue {
	//log.Debugf("newMsgQueue called...")
	return &msgQueue{
		done:    make(chan struct{}),
		work:    make(chan struct{}, 1),
		bl:      biddinglist.NewThreadSafe(),

		//wyong, 20200925 
		//network: wm.network,
		host 	: bm.host, 
		
		//wyong, 20200825 
		outlk:	new(sync.Mutex), 

		p:       p,
		refcnt:  1,
	}
}

