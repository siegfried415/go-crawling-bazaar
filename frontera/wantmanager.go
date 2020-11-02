package frontera

import (
	"context"
	"sync"
	"time"
	"fmt"

	//engine "github.com/siegfried415/gdf-rebuild/frontera/decision"

	//wyong, 20200827 
	//bsmsg "github.com/siegfried415/gdf-rebuild/frontera/message"

	//fnet "github.com/siegfried415/gdf-rebuild/frontera/network"
	wantlist "github.com/siegfried415/gdf-rebuild/frontera/wantlist"

	cid "github.com/ipfs/go-cid"

	//wyong, 20200925
        pstore "github.com/libp2p/go-libp2p-peerstore"
        host "github.com/libp2p/go-libp2p-core/host"

	//go-libp2p-net -> go-libp2p-core/network,
	//wyong, 20201029 
	inet "github.com/libp2p/go-libp2p-core/network"
	libp2phelpers "github.com/libp2p/go-libp2p-core/helpers"

	protocol "github.com/libp2p/go-libp2p-core/protocol" 
	peer "github.com/libp2p/go-libp2p-core/peer" 


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
)

type WantManager struct {
	// sync channels for Run loop
	incoming     chan *wantSet
	connectEvent chan peerStatus     // notification channel for peers connecting/disconnecting
	peerReqs     chan chan []proto.NodeID // channel to request connected peers on

	// synchronized by Run loop, only touch inside there
	peers map[proto.NodeID]*msgQueue
	wl    *wantlist.ThreadSafe
	bcwl  *wantlist.ThreadSafe

	//wyong, 20190115
	completed_wantlist *wantlist.ThreadSafe 

	//network fnet.BiddingNetwork
	host host.Host 

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

func NewWantManager(ctx context.Context , /* network fnet.BiddingNetwork, */ host host.Host,   nodeID proto.NodeID  ) *WantManager {
	ctx, cancel := context.WithCancel(ctx)
	//wantlistGauge := metrics.NewCtx(ctx, "wantlist_total",
	//	"Number of items in wantlist.").Gauge()
	//sentHistogram := metrics.NewCtx(ctx, "sent_all_blocks_bytes", "Histogram of blocks sent by"+
	//	" this bitswap").Histogram(metricsBuckets)

	return &WantManager{
		incoming:      make(chan *wantSet, 10),
		connectEvent:  make(chan peerStatus, 10),
		peerReqs:      make(chan chan []proto.NodeID),
		peers:         make(map[proto.NodeID]*msgQueue),
		wl:            wantlist.NewThreadSafe(),
		bcwl:          wantlist.NewThreadSafe(),

		//wyong, 20200831 
		completed_wantlist : wantlist.NewThreadSafe(),

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

type msgQueue struct {
	p proto.NodeID

	outlk   *sync.Mutex
	out     types.UrlBiddingMessage		//bsmsg.BiddingMessage

	//wyong, 20200925 
	//network fnet.BiddingNetwork
	host 	host.Host 
	
	wl      *wantlist.ThreadSafe

	//wyong, 20200924 
	//sender fnet.MessageSender
        //caller rpc.PCaller
	sender inet.Stream 

	refcnt int

	work chan struct{}
	done chan struct{}
}

//wyong, 20200831
func (pm *WantManager) waitBidResult(ctx context.Context, dc chan bool ) bool {
	var result bool = false 
	select {
	case result = <-dc :
		fmt.Printf("WantManager/waitBidResult, result = <- dc\n")
	case <-pm.ctx.Done():
		fmt.Printf("WantManager/waitBidResult, <-pm.ctx.Done()\n")
	case <-ctx.Done():
		fmt.Printf("WantManager/waitBidResult, <-ctx.Done()\n")
	}
	return result 
}

//wyong, 20190119  
func (pm *WantManager) ReceiveBidForWants(ctx context.Context, url string, /* bids map[proto.NodeID]*wantlist.BidEntry, peers []proto.NodeID, wyong, 20200828 */ c cid.Cid,  from proto.NodeID,  domain proto.DomainID ) bool {
	//pm.addEntries(context.Background(), url,bids, peers, true, domain )
	var dc chan bool = make(chan bool) 	
	select {
	case pm.bidIncoming <- &bidSet{	url: url, 
					//bids: bids, wyong, 20200828  
					done: dc, 	//wyong, 20200831 
					cid : c,
					from : from, 
					domain: domain, 
				}:
		fmt.Printf("WantManager/receiveBidForWant, pm.bidIncoming <- &bidSet\n")
	case <-pm.ctx.Done():
		fmt.Printf("WantManager/receiveBidForWant, <-pm.ctx.Done()\n")
		return false 
	case <-ctx.Done():
		fmt.Printf("WantManager/receiveBidForWant, <-ctx.Done()\n")
		return false 
	}

	//wait for result, true if bidding sucessfully, false other. wyong, 20200831 
	return pm.waitBidResult(ctx, dc) 
}

type bidSet struct {
	url string  
	done chan bool	//wyong, 20200831 

	//wyong, 20200828 
	//bids map[proto.NodeID]*wantlist.BidEntry
	from proto.NodeID
	cid cid.Cid 
	
	domain    proto.DomainID 
}

// WantBlocks adds the given cids to the wantlist, tracked by the given domain 
func (pm *WantManager) WantUrls(ctx context.Context, urls []string, peers []proto.NodeID, domain proto.DomainID) {
	fmt.Printf("WantManager/WantUrls(10), want blocks: %s\n", urls)
	pm.addEntries(ctx, urls, peers, false, domain)
}

// CancelWants removes the given cids from the wantlist, tracked by the given domain 
func (pm *WantManager) CancelWants(ctx context.Context, urls []string,  peers []proto.NodeID, domain proto.DomainID) bool {
	pm.addEntries(context.Background(), urls, peers, true, domain)
	return true 
}

type wantSet struct {
	entries []types.UrlBidding 	//[]*bsmsg.BiddingEntry, wyong, 20200827 
	targets []proto.NodeID
	domain proto.DomainID  
}

func (pm *WantManager) addEntries(ctx context.Context, urls []string, targets []proto.NodeID, cancel bool, domain proto.DomainID) {
	fmt.Printf("WantManager/addEntries(0)\n")

	entries := make( /* []*bsmsg.BiddingEntry wyong, 20200827 */ []types.UrlBidding , 0, len(urls))
	for _, url := range urls {
		fmt.Printf("WantManager/addEntries(10), process url=%s\n", url )
		entries = append(entries,  /* &bsmsg.BiddingEntry */ types.UrlBidding {
			//wyong, 20200827 	
			//Cancel: cancel,
			//BiddingEntry:  wantlist.NewRefBiddingEntry(url, kMaxPriority-i),
			Url : url, 
			Probability : 1.0,	//todo, wyong, 2bility020082o 
		})
	}
	select {
	case pm.incoming <- &wantSet{entries: entries, targets: targets, domain: domain }:
		fmt.Printf("WantManager/addEntries(20), pm.incoming <- &wantSet\n")
	case <-pm.ctx.Done():
		fmt.Printf("WantManager/addEntries(30), <-pm.ctx.Done()\n")
	case <-ctx.Done():
		fmt.Printf("WantManager/addEntries(40), <-ctx.Done()")
	}
}

func (pm *WantManager) ConnectedPeers() []proto.NodeID {
	fmt.Printf("WantManager/ConnectedPeers (10)\n")
	resp := make(chan []proto.NodeID)
	pm.peerReqs <- resp
	return <-resp
}


func (pm *WantManager) SendBids(ctx context.Context, msg *types.UrlBidMessage ) {
	fmt.Printf("WantManager/SendBids(10)\n")
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
	//pm.sentHistogram.Observe(float64(msgSize))
	err := pm.network.SendMessage(ctx, env.Peer, env.Message )
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
	if target.IsEmpty() {
		fmt.Printf("WantManager/SendBids(15), target is empty\n")
		return
	}

	fmt.Printf("WantManager/SendBids(20), target=%s\n", target )
	//caller = mux.NewPersistentCaller(target) 
	s, err := pm.host.NewStream(ctx, peer.ID(target), protocol.ID("ProtocolFronteraBid"))
	if err != nil {
                return 
        }

        //var response types.Response
	//err := caller.Call(route.FronteraBid.String(), msg, &response ) 
	//if err == nil {
	//	fmt.Printf("WantManager/SendBids(25), err=%s\n", err.Error())
	//	return
	//}

        if _, err = net.SendMsg(ctx, s, msg ); err != nil {
                s.Reset()
                return 
        }

	fmt.Printf("WantManager/SendBids(30)\n")

	//todo, wyong, 20200925 

	//wyong, 20201029 
	//go inet.AwaitEOF(s)
	go libp2phelpers.AwaitEOF(s)

        s.Close()

}


func (pm *WantManager) startPeerHandler(p proto.NodeID) *msgQueue {
	fmt.Printf("WantManager/startPeerHandler(10)\n")

	mq, ok := pm.peers[p]
	if ok {
		mq.refcnt++
		return nil
	}

	mq = pm.newMsgQueue(p)

	//wyong, 20200827 
	// new peer, we will want to give them our full wantlist
	//fullwantlist := bsmsg.New(true, string(pm.nodeID))
	entries := make([]types.UrlBidding, 0, len(pm.bcwl.Entries()))
	for _, e := range pm.bcwl.Entries() {
		for k := range e.SesTrk {
			mq.wl.AddBiddingEntry(e, k)
		}
		//fullwantlist.AddBidding(e.Url, e.Priority, nil )
		entries = append(entries, types.UrlBidding{
						Url: e.Url,
						Probability : e.Probability, 	
					})
	}


	//build bidding message, wyong, 20200827 
	mq.out = types.UrlBiddingMessage{
		Header: types.SignedUrlBiddingHeader{
			UrlBiddingHeader: types.UrlBiddingHeader{
				QueryType:    types.WriteQuery,
				NodeID:       pm.nodeID,
				//DomainID:     pm.DomainID,

				//todo, wyong, 20200817
				//ConnectionID: connID,
				//SeqNo:        seqNo,
				//Timestamp:    getLocalTime(),
			},
		},
		Payload: types.UrlBiddingPayload{
			Requests: entries ,
		},
	}

	//mq.out = fullwantlist
	mq.work <- struct{}{}

	pm.peers[p] = mq
	go mq.runQueue(pm.ctx)
	return mq
}

func (pm *WantManager) stopPeerHandler(p proto.NodeID) {
	fmt.Printf("WantManager/stopPeerHandler(10)\n")
	pq, ok := pm.peers[p]
	if !ok {
		// TODO: log error?
		return
	}

	pq.refcnt--
	if pq.refcnt > 0 {
		return
	}

	close(pq.done)
	delete(pm.peers, p)
}

func (mq *msgQueue) runQueue(ctx context.Context) {
	//log.Debugf("runQueue called...")
	for {
		select {
		case <-mq.work: // there is work to be done
			//log.Debugf("runQueue, <-mq.work")
			mq.doWork(ctx)
		case <-mq.done:
			//log.Debugf("runQueue, <-mq.done")
			if mq.sender != nil {
				//mq.sender.Close()

				//wyong, 20201029 
				//inet.FullClose(mq.sender) 
				libp2phelpers.FullClose(mq.sender) 
			}
			return
		case <-ctx.Done():
			//log.Debugf("runQueue, <-ctx.Done")
			if mq.sender != nil {
				mq.sender.Reset()
			}
			return
		}
	}
}

func (mq *msgQueue) doWork(ctx context.Context) {
	fmt.Printf("msgQueue/doWork(10)\n")
	// grab outgoing message
	mq.outlk.Lock()
	wlm := mq.out
	if /* wlm == nil ||  wyong, 20200827 */  wlm.Empty() { 
		mq.outlk.Unlock()
		return
	}
	mq.out = types.UrlBiddingMessage{} //nil, wyong, 20200827 
	mq.outlk.Unlock()

	fmt.Printf("msgQueue/doWork(20)\n")

	// NB: only open a stream if we actually have data to send
	if mq.sender == nil {
		fmt.Printf("msgQueue/doWork(30)\n")
		err := mq.openSender(ctx)
		if err != nil {
			fmt.Printf("msgQueue/doWork(35)\n")
			fmt.Printf("cant open message sender to peer %s: %s", mq.p, err)
			// TODO: cant connect, what now?
			return
		}
		fmt.Printf("msgQueue/doWork(40)\n")
	}
	
	fmt.Printf("msgQueue/doWork(50)\n")
	// send wantlist updates
	for { // try to send this message until we fail.
		fmt.Printf("msgQueue/doWork(60)\n")
		
		//wyong, 20200924 
		//err := mq.sender.SendMsg(ctx, wlm)
        	//var response types.Response
		//err := mq.caller.Call(route.FronteraBidding.String(), wlm, &response ) 
		_, err := net.SendMsg(ctx, mq.sender, wlm)
		if err == nil {
			fmt.Printf("msgQueue/doWork(70)\n")
			return
		}

		// todo, wyong, 20200730 
		//log.Infof("bitswap send error: %s", err)
		mq.sender.Reset()
		mq.sender = nil

		fmt.Printf("msgQueue/doWork(80)\n")

		select {
		case <-mq.done:
			fmt.Printf("msgQueue/doWork(90)\n")
			return
		case <-ctx.Done():
			fmt.Printf("msgQueue/doWork(100)\n")
			return
		case <-time.After(time.Millisecond * 100):
			fmt.Printf("msgQueue/doWork(110)\n")
			// wait 100ms in case disconnect notifications are still propogating
			//log.Warning("SendMsg errored but neither 'done' nor context.Done() were set")
		}

		fmt.Printf("msgQueue/doWork(120)\n")

		err = mq.openSender(ctx)
		if err != nil {
			fmt.Printf("msgQueue/doWork(125)")
			fmt.Printf("couldnt open sender again after SendMsg(%s) failed: %s\n", mq.p, err)
			// TODO(why): what do we do now?
			// I think the *right* answer is to probably put the message we're
			// trying to send back, and then return to waiting for new work or
			// a disconnect.
			return
		}

		fmt.Printf("msgQueue/doWork(130)\n")

		// TODO: Is this the same instance for the remote peer?
		// If its not, we should resend our entire wantlist to them
		//if mq.sender.InstanceID() != mq.lastSeenInstanceID {
		//	wlm = mq.getFullWantlistMessage()
		//}
	}
}

// wyong, 20200924 
func (mq *msgQueue) openSender(ctx context.Context ) error {
	//log.Debugf("openSender called...")
	// allow ten minutes for connections this includes looking them up in the
	// dht dialing them, and handshaking
	//conctx, cancel := context.WithTimeout(ctx, time.Minute*10)
	//defer cancel()

	//log.Debugf("openSender(10)")
	//err := mq.network.ConnectTo(conctx, mq.p )
	err := mq.host.Connect(ctx, pstore.PeerInfo{ID: peer.ID(mq.p)})
	if err != nil {
		return err
	}

	//log.Debugf("openSender(20)")
	//todo, "gdf/frontera/bidding",  wyong, 20200924 
	//nsender, err := mq.network.NewMessageSender(ctx, mq.p)
	s, err := mq.host.NewStream(ctx, peer.ID(mq.p), protocol.ID("ProtocolFronteraBidding"))
	if err != nil {
		return err
	}

	//log.Debugf("openSender(30)")
	mq.sender = s 

	//if cfg.UseDirectRPC {
	//	caller = rpc.NewPersistentCaller(mq.p)
	//} else {
	//	caller = mux.NewPersistentCaller(mq.p)
	//}
	//caller := mux.NewPersistentCaller(mq.p)
	//
	//mq.caller = caller 

	return nil
}

func (pm *WantManager) Connected(p proto.NodeID) {
	select {
	case pm.connectEvent <- peerStatus{peer: p, connect: true}:
		fmt.Printf("WantManager/Connected, pm.connectEvent <- peerStatus\n")
	case <-pm.ctx.Done():
		fmt.Printf("WantManager/Connected, <-pm.ctx.Done()\n")
	}
}

func (pm *WantManager) Disconnected(p proto.NodeID) {
	select {
	case pm.connectEvent <- peerStatus{peer: p, connect: false}:
	case <-pm.ctx.Done():
	}
}

//wyong, 20190126 
func (pm *WantManager) GetCompletedBiddings() ([]*wantlist.BiddingEntry, error) {
	result := make([]*wantlist.BiddingEntry, 0, pm.wl.Len())

	//return pm.completed_wantlist.Entries(), nil 
	for _, bidding := range pm.wl.Entries() {
		if (bidding.CountOfBids() >= 2 ) {
			result = append(result, bidding) 
		}
	}

	return result, nil 
}

//wyong, 20190131
func (pm *WantManager) GetUncompletedBiddings() ([]*wantlist.BiddingEntry, error) {
	return pm.wl.Entries(), nil 
}

// TODO: use goprocess here once i trust it
func (pm *WantManager) Run() {
	fmt.Printf("WantManager/Run(10)\n")
	// NOTE: Do not open any streams or connections from anywhere in this
	// event loop. Really, just don't do anything likely to block.
	for {
		select {
		case bs := <-pm.bidIncoming:	//wyong, 20190119 
			fmt.Printf("WantManager/Run(20), bs : <- pm.bidIncoming\n")

			url := bs.url 
			bidding, _ := pm.wl.Contains(url)
			if bidding == nil {
				bs.done <- false //wyong, 20200831
				continue
			}

	
			/*wyong, 20200828 
			// add bids to wantlist
			//for _, bid := range bs.bids {
				//wyong, 20190115 
				//Store bid to bidding, and move the bidding to completed wantlist
				//if bidding has already two bids for it.  wyong, 20190119 

				if !bidding.AddBid(url, bid.Cid, bid.From ) {
					continue
				}

			}
			*/

			//wyong, 20200828 
			if !bidding.AddBid(url, bs.cid, bs.from ) {
				bs.done <- false //wyong, 20200831 
				continue 
			}

			if (bidding.CountOfBids() >= 2 ) {
				if pm.wl.Remove(url, bs.domain ) {
					//pm.wantlistGauge.Dec()
				}

				pm.completed_wantlist.AddBiddingEntry(bidding, bs.domain ) 
				bs.done <- true  //wyong, 20200831 
				continue 
			}
			
			
			bs.done <- false 	//wyong, 20200831 

			//TODO, don't forget send message to those peers to let them know we havn't 
			//need those url any more,  wyong, 20190119
			

		case ws := <-pm.incoming:
			fmt.Printf("WantManager/Run(30), ws : <- pm.incoming\n")
			// is this a broadcast or not?
			brdc := len(ws.targets) == 0

			// add changes to our wantlist
			for _, e := range ws.entries {
				if e.Cancel {
					if brdc {
						pm.bcwl.Remove(e.Url, ws.domain )
					}

					if pm.wl.Remove(e.Url, ws.domain) {
						//pm.wantlistGauge.Dec()
					}
					
				} else {
					if brdc {
						pm.bcwl.Add(e.Url, e.Probability,  ws.domain)
					}
					if pm.wl.Add(e.Url, e.Probability, ws.domain ) {
						//pm.wantlistGauge.Inc()
					}
				}
			}

			fmt.Printf("WantManager/Run(40)\n")
			// broadcast those wantlist changes
			if len(ws.targets) == 0 {
				fmt.Printf("WantManager/Run(50) ")
				for id, p := range pm.peers {
					fmt.Printf("WantManager/Run(60), peer=%s\n", id )
					p.addMessage(ws.entries, ws.domain , pm.nodeID )
				}
			} else {
				fmt.Printf("WantManager/Run(70)\n")
				for _, t := range ws.targets {
					fmt.Printf("WantManager/Run(80), target_node_id=%s\n", t)
					p, ok := pm.peers[t]
					if !ok {
						//todo, wyong, 20200825 
						fmt.Printf("WantManager/Run(90)\n")
						//fmt.Printf("tried sending wantlist change to non-partner peer: %s\n", t)
						//continue
						p = pm.startPeerHandler(t)
					}

					fmt.Printf("WantManager/Run(100)\n")
					p.addMessage(ws.entries, ws.domain, pm.nodeID )
				}
				fmt.Printf("WantManager/Run(110)\n")
			}

		case p := <-pm.connectEvent:
			fmt.Printf("WantManager/Run(120), P := <-pm.connectEvent\n")
			if p.connect {
				fmt.Printf("WantManager/Run(130)\n")
				pm.startPeerHandler(p.peer)
			} else {
				fmt.Printf("WantManager/Run(140)\n")
				pm.stopPeerHandler(p.peer)
			}
		case req := <-pm.peerReqs:
			fmt.Printf("WantManager/Run(150), req := <-pm.peerReqs\n")
			peers := make([]proto.NodeID, 0, len(pm.peers))
			for p := range pm.peers {
				fmt.Printf("WantManager/Run(160)\n")
				peers = append(peers, p)
			}
			req <- peers
		case <-pm.ctx.Done():
			fmt.Printf("WantManager/Run(170), <-pm.ctx.Done()\n")
			return
		}
	}
}

func (wm *WantManager) newMsgQueue(p proto.NodeID) *msgQueue {
	//log.Debugf("newMsgQueue called...")
	return &msgQueue{
		done:    make(chan struct{}),
		work:    make(chan struct{}, 1),
		wl:      wantlist.NewThreadSafe(),

		//wyong, 20200925 
		//network: wm.network,
		host 	: wm.host, 
		
		//wyong, 20200825 
		outlk:	new(sync.Mutex), 

		p:       p,
		refcnt:  1,
	}
}

func (mq *msgQueue) addMessage( entries /* []*bsmsg.BiddingEntry, */ []types.UrlBidding,  domain proto.DomainID, from proto.NodeID  ) {
	fmt.Printf("msgQueue/addMessage(10)\n")
	var work bool
	mq.outlk.Lock()
	defer func() {
		mq.outlk.Unlock()
		if !work {
			return
		}
		select {
		case mq.work <- struct{}{}:
		default:
		}
	}()

	fmt.Printf("msgQueue/addMessage (20)\n")
	// if we have no message held allocate a new one
	//if mq.out == nil {
	//	//wyong, 20200827 
	//	mq.out = bsmsg.New(false, string(from))
	//}

	//build bidding message, wyong, 20200827 
	mq.out = types.UrlBiddingMessage{
		Header: types.SignedUrlBiddingHeader{
			UrlBiddingHeader: types.UrlBiddingHeader{
				QueryType:    types.WriteQuery,
				NodeID:       from,
				DomainID:     domain,

				//todo, wyong, 20200817
				//ConnectionID: connID,
				//SeqNo:        seqNo,
				//Timestamp:    getLocalTime(),
			},
		},
		Payload: types.UrlBiddingPayload{
			Requests: entries ,
		},
	}


	fmt.Printf("msgQueue/addMessage (30)\n")

	/* 
	// TODO: add a msg.Combine(...) method
	// otherwise, combine the one we are holding with the
	// one passed in
	for _, e := range entries {
		fmt.Printf("msgQueue/addMessage (40)\n")
		if e.Cancel {
			fmt.Printf("msgQueue/addMessage(50)\n")
			if mq.wl.Remove(e.Url, domain ) {
				fmt.Printf("msgQueue/addMessage(60)")
				work = true
				mq.out.Cancel(e.Url)
			}
		} else {
			fmt.Printf("msgQueue/addMessage (70)\n")
			if mq.wl.Add(e.Url, e.Priority, domain ) {
				fmt.Printf("msgQueue/addMessage (80)\n")
				work = true

				//TODO, review if necessary send cids , wyong, 20190119 
				mq.out.AddBidding(e.Url, e.Priority, nil )
			}
		}
	}
	*/

}
