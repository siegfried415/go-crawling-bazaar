package frontera

import (
	"context"
	"sync"
	"time"
	//"fmt"
	//"sort"

        //wyong, 20210203
        "crypto/ecdsa"

	//wyong, 20210203
        "github.com/pkg/errors"

	//engine "github.com/siegfried415/go-crawling-bazaar/frontera/decision"

	//wyong, 20200827 
	//bsmsg "github.com/siegfried415/go-crawling-bazaar/frontera/message"

	//fnet "github.com/siegfried415/go-crawling-bazaar/frontera/network"
	//biddinglist "github.com/siegfried415/go-crawling-bazaar/frontera/biddinglist"

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
        //"github.com/siegfried415/go-crawling-bazaar/route"
        "github.com/siegfried415/go-crawling-bazaar/types"
        //"github.com/siegfried415/go-crawling-bazaar/rpc"
        //"github.com/siegfried415/go-crawling-bazaar/rpc/mux"
        "github.com/siegfried415/go-crawling-bazaar/proto"
	net "github.com/siegfried415/go-crawling-bazaar/net" 

	//wyong, 20210203 
        "github.com/siegfried415/go-crawling-bazaar/kms"

	//wyong, 20200928
	//"github.com/ugorji/go/codec"

	//wyong, 20201215 
	log "github.com/siegfried415/go-crawling-bazaar/utils/log" 

        //wyong, 20210118
        ecvrf "github.com/vechain/go-ecvrf"

        //wyong, 20200218
        //"github.com/mfonda/simhash"

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
func (bc *BiddingClient) ReceiveBidForBiddings(ctx context.Context, url string, c cid.Cid,  from proto.NodeID,  domain proto.DomainID , hash []byte, proof []byte , simhash uint64 ) bool {
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

					simhash: simhash, //wyong, 20210220 
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

	simhash uint64	//wyong, 20210220 
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
	log.Debugf("BiddingClient/addBiddings(10), biddings = %s, targets = %s, cancel =%b, domain=%s\n", biddings, targets, cancel, string(domain))

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

	log.Debugf("BiddingClient/SendBids(20)\n")
	//caller = mux.NewPersistentCaller(target) 
	s, err := bc.host.NewStreamExt(ctx, target, protocol.ID("FRT.Bid"))
	if err != nil {
                return 
        }

	log.Debugf("BiddingClient/SendBids(30)\n")
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

	log.Debugf("BiddingClient/SendBids(40)\n")

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


/* move this code to BiddingEntry, wyong, 20210220 
//wyong, 20210206 
func (bc *BiddingClient) NeedCrawlMore(bidding *BiddingEntry) (int, error ) {
	log.Debugf("BiddingClient/NeedCrawlMore(10), bidding=%s, bidding.ExpectCrawlerCount=%d\n", bidding, bidding.ExpectCrawlerCount )
	needCrawlCount := bidding.ExpectCrawlerCount  
	bids := bidding.GetBids() 
	if len(bids) == 0 {
		return needCrawlCount, nil 
	}

	log.Debugf("BiddingClient/NeedCrawlMore(20), len(bids)=%d\n", len(bids))
	bidCounts := make ([]int, len(bids))
	for k :=0; k<len(bidCounts); k++ {
		//initialize every count to 1, wyong, 20210220 
		bidCounts[k] = 1 
	}
 
	for i :=0; i < len(bids) ; i++ {
		log.Debugf("BiddingClient/NeedCrawlMore(30), i=%d\n", i )
		for j := i + 1 ; j< len(bids); j++ {
			log.Debugf("BiddingClient/NeedCrawlMore(40), j=%d\n", j )
			if simhash.Compare(bids[i].Hash,  bids[j].Hash) < 2 {
				log.Debugf("BiddingClient/NeedCrawlMore(50)\n")
				bidCounts[i]++
				bidCounts[j]++
			}
		}
	}

	log.Debugf("BiddingClient/NeedCrawlMore(60)\n")
        sort.Slice(bidCounts, func(i, j int) bool {
                return bidCounts[i] < bidCounts[j]
        })

	log.Debugf("BiddingClient/NeedCrawlMore(70), bidCounts=%s\n", bidCounts[0] )
	needCrawlCount = bidding.ExpectCrawlerCount - bidCounts[0]
	if needCrawlCount < 0 {
		needCrawlCount = 0 
	}

	log.Debugf("BiddingClient/NeedCrawlMore(80), needCrawlCount=%d\n", needCrawlCount)
	return needCrawlCount, nil 
}
*/

//wyong, 20210203 
func (bc *BiddingClient) GetUnAckedPeers (bidding *BiddingEntry) ([]proto.NodeID, error ) {
	log.Debugf("BiddingClient/GetUnAckedPeers(10), bidding=%s\n", bidding)
	domain, exist := bc.f.DomainForID(bidding.DomainID)
	if exist != true {
		err := errors.New("domain not exist")
		return []proto.NodeID{}, err 	
	}

	log.Debugf("BiddingClient/GetUnAckedPeers(20)\n")
	bids := bidding.GetBids()
	if bids == nil || len(bids)==0 {
		log.Debugf("BiddingClient/GetUnAckedPeers(25), domain.activePeersArr=%s\n", domain.activePeersArr )
		return domain.activePeersArr, nil 
	}

	log.Debugf("BiddingClient/GetUnAckedPeers(30)\n")
	peers := make([]proto.NodeID, 0, len(domain.activePeers)) 
	for _, peer := range domain.activePeersArr {
		log.Debugf("BiddingClient/GetUnAckedPeers(40), peer=%s\n", peer )
		acked := false  
		for _, bid := range bids {
			log.Debugf("BiddingClient/GetUnAckedPeers(50), bid.From=%s\n", bid.From)
			if peer == bid.From {
				log.Debugf("BiddingClient/GetUnAckedPeers(60), acked = true\n")
				acked = true  
				break 
			}
		}
		log.Debugf("BiddingClient/GetUnAckedPeers(70), acked =%b\n", acked )
		if !acked  {
			log.Debugf("BiddingClient/GetUnAckedPeers(80), acked = true\n")
			peers = append(peers, peer) 
		}
	}
	
	log.Debugf("BiddingClient/GetUnAckedPeers(90), peers = %s\n", peers )
	return peers, nil 
}

//wyong, 20210204
func (bc *BiddingClient) recrawlBidding(bidding *BiddingEntry, recrawlCount int ) error {
	log.Debugf("BiddingClient/recrawlBidding(10), bidding=%s, recrrawlCount = %d\n", bidding, recrawlCount )
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

	log.Debugf("BiddingClient/recrawlBidding(20), peers =%s\n", peers )
	bc.addBiddings(context.Background(), 
			[]types.UrlBidding { newBidding }, 
				peers,	//wyong, 20210203 
				false, 		//wyong, 20210203 
			bidding.DomainID ) 

	log.Debugf("BiddingClient/recrawlBidding(30)\n")
	return nil 

}

// TODO: use goprocess here once i trust it
func (bc *BiddingClient) Run() {
	log.Debugf("BiddingClient/Run start...\n")

	//wyong, 20210204 
        bc.tick = time.NewTimer(MainCycleInterval )

	// NOTE: Do not open any streams or connections from anywhere in this
	// event loop. Really, just don't do anything likely to block.
	for {
		select {
		case ws := <-bc.biddingIncoming:
			log.Debugf("BiddingClient/Run(10), ws : <- bc.biddingIncoming\n")

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

			log.Debugf("BiddingClient/Run(20)\n")
			// broadcast those biddinglist changes
			//if len(ws.targets) == 0 {
			//	log.Debugf("BiddingClient/Run(30) ")
			//	for id, p := range bc.peers {
			//		log.Debugf("BiddingClient/Run(40), peer=%s\n", id )
			//		p.addBiddingMessage(ws.biddings, ws.domain , bc.nodeID )
			//	}
			//} else {
				log.Debugf("BiddingClient/Run(50)\n")
				for _, t := range ws.targets {
					log.Debugf("BiddingClient/Run(60), target_node_id=%s\n", t)
					p, ok := bc.peers[t]
					if !ok {
						//todo, wyong, 20200825 
						log.Debugf("BiddingClient/Run(70)\n")
						//log.Debugf("tried sending biddinglist change to non-partner peer: %s\n", t)
						//continue
						p = bc.startPeerHandler(t)
					}

					log.Debugf("BiddingClient/Run(80)\n")
					p.addBiddingMessage(ws.biddings, ws.domain, bc.nodeID )
				}
			//}

			log.Debugf("BiddingClient/Run(90)\n")

		case bs := <-bc.bidIncoming:	//wyong, 20190119 
			log.Debugf("BiddingClient/Run(100), bs : <- bc.bidIncoming\n")

			url := bs.url 
			bidding, _ := bc.bl.Contains(url)
			if bidding == nil {
				bs.done <- false //wyong, 20200831
				continue
			}

			log.Debugf("BiddingClient/Run(110), bidding = %s\n", bidding )

			//wyong, 20210203 
			pk, err := kms.GetPublicKey(bs.from)
			if err != nil {
				err = errors.Wrap(err, "failed to cache public key")
				continue 
			}

			log.Debugf("BiddingClient/Run(120)\n")
			//wyong, 20210118
			// `pi` is the VRF proof
			// pk is publick key of bid.From 
			_, err = ecvrf.NewSecp256k1Sha256Tai().Verify((*ecdsa.PublicKey)(pk), []byte(bs.url), bs.proof)
			if err != nil {
				// invalid proof
				continue	
			}

			log.Debugf("BiddingClient/Run(130)\n")
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
			if !bidding.AddBid(url, bs.cid, bs.from, bs.simhash ) {
				bs.done <- false //wyong, 20200831 
				continue 
			}

			log.Debugf("BiddingClient/Run(140)\n")
			if (bidding.CountOfBids() < bidding.ExpectCrawlerCount ) {
				continue 
			}

			log.Debugf("BiddingClient/Run(150)\n")
			//todo, check if cids from every bid match or not. 
			//if not match, caculate how many more crawling are needed. 
			//bids := bidding.GetBids() 
			need_crawl_more, err := bidding.NeedCrawlMore() 
			if need_crawl_more > 0 {
				//add need_crawl_count to let bidding (in bidding list) can handle more bid, 
				//this is subtle, wyong, 20210204 
				bidding.ExpectCrawlerCount +=  need_crawl_more

				bc.recrawlBidding(bidding, need_crawl_more) 
				continue 
			}

			log.Debugf("BiddingClient/Run(160)\n")
			//Crawled succeed 
			if bc.bl.Remove(url, bs.domain ) {
				//bc.biddinglistGauge.Dec()
			}
		
			log.Debugf("BiddingClient/Run(170)\n")
			bc.completed_bl.AddBiddingEntry(bidding, bs.domain ) 
			bs.done <- true  //wyong, 20200831 

			log.Debugf("BiddingClient/Run(180)\n")
			continue 
			
			//todo, wyong, 20201217
			//UrlCrawled()
			
			//wyong, 20210117 
			//bs.done <- false 	

			//TODO, don't forget send message to those peers to let them know we havn't 
			//need those url any more,  wyong, 20190119

		//wyong, 20210116
		case <-bc.tick.C:
			log.Debugf("BiddingClient/Run(200), <-bc.tick.C \n")
			//todo, rebroadcast bidding when crawling timer timeout, 
			//wyong, 20210117 
			for _, bidding := range bc.bl.Entries() {

				log.Debugf("BiddingClient/Run(210), bidding=%s \n", bidding )
				bidCount := len(bidding.GetBids())
				if (bidCount >= bidding.ExpectCrawlerCount ) || 
					(bidCount < bidding.ExpectCrawlerCount && time.Since(bidding.LastBroadcastTime) < CrawlingInterval ) {
					continue 
				}
				
				log.Debugf("BiddingClient/Run(220)\n")
				need_recrawl_more := bidding.ExpectCrawlerCount - bidCount 
				if need_recrawl_more > 0 {
					//don't add need_recrawl_more of bidding,
					//because we don't need bids more then bidding.ExpectCrawlerCount. 
					//wyong, 20210204 
					log.Debugf("BiddingClient/Run(230), need_recrawl_more = %d\n", need_recrawl_more )
					bc.recrawlBidding(bidding, need_recrawl_more ) 
					log.Debugf("BiddingClient/Run(240)\n")
				}
				log.Debugf("BiddingClient/Run(250)\n")
			}

			log.Debugf("BiddingClient/Run(260)\n")
			bc.tick.Reset(MainCycleInterval)

		case p := <-bc.connectEvent:
			log.Debugf("BiddingClient/Run(300), P := <-bc.connectEvent\n")
			if p.connect {
				log.Debugf("BiddingClient/Run(310)\n")
				bc.startPeerHandler(p.peer)
			} else {
				log.Debugf("BiddingClient/Run(320)\n")
				bc.stopPeerHandler(p.peer)
			}
		case req := <-bc.peerReqs:
			log.Debugf("BiddingClient/Run(400), req := <-bc.peerReqs\n")
			peers := make([]proto.NodeID, 0, len(bc.peers))
			for p := range bc.peers {
				log.Debugf("BiddingClient/Run(410)\n")
				peers = append(peers, p)
			}
			req <- peers
		case <-bc.ctx.Done():
			log.Debugf("BiddingClient/Run(500), <-bc.ctx.Done()\n")
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

