/*
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
	"context"
	//"fmt"
	"sync"
	"time"

        "crypto/ecdsa"
        ecvrf "github.com/vechain/go-ecvrf"
        "github.com/pkg/errors"

	cid "github.com/ipfs/go-cid"
	libp2phelpers "github.com/libp2p/go-libp2p-core/helpers"
	protocol "github.com/libp2p/go-libp2p-core/protocol" 

	crypto "github.com/siegfried415/go-crawling-bazaar/crypto"
        "github.com/siegfried415/go-crawling-bazaar/kms"
	log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
	net "github.com/siegfried415/go-crawling-bazaar/net" 
        "github.com/siegfried415/go-crawling-bazaar/proto"
        "github.com/siegfried415/go-crawling-bazaar/types"
        utils "github.com/siegfried415/go-crawling-bazaar/utils"
)

const (
	MainCycleInterval = 10 * time.Second 
	CrawlingInterval = time.Duration(10) * time.Second
) 

type BiddingClient struct {

	f *Frontera 	

	// sync channels for Run loop
	biddingIncoming     chan *biddingSet
	connectEvent chan peerStatus     // notification channel for peers connecting/disconnecting
	peerReqs     chan chan []proto.NodeID // channel to request connected peers on

	// synchronized by Run loop, only touch inside there
	peers map[proto.NodeID]*msgQueue
	bl    *ThreadSafe
	bcbl  *ThreadSafe
	completed_bl *ThreadSafe 

	host net.RoutedHost 

	ctx     context.Context
	cancel  func()

	bidIncoming     chan *bidSet	
	nodeID	proto.NodeID	

        tick          *time.Timer
}

type peerStatus struct {
	connect bool
	peer    proto.NodeID
}

func NewBiddingClient(ctx context.Context, host net.RoutedHost, nodeID proto.NodeID) *BiddingClient {
	ctx, cancel := context.WithCancel(ctx)
	return &BiddingClient {
		biddingIncoming:      make(chan *biddingSet, 10),
		connectEvent:  make(chan peerStatus, 10),
		peerReqs:      make(chan chan []proto.NodeID),
		peers:         make(map[proto.NodeID]*msgQueue),
		bl:            NewThreadSafe(),
		bcbl:          NewThreadSafe(),
		completed_bl : NewThreadSafe(),
		host: 		host, 
		ctx:           ctx,
		cancel:        cancel,
		bidIncoming :  make(chan *bidSet, 10), 	
		nodeID   : nodeID, 	
	}
}


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

func (bc *BiddingClient) ReceiveBidForBiddings(ctx context.Context, url string, c cid.Cid,  from proto.NodeID,  domain proto.DomainID , hash []byte, proof []byte , simhash uint64 ) bool {
	var dc chan bool = make(chan bool) 	
	select {
	case bc.bidIncoming <- &bidSet{	url: url, 
					done: dc, 	
					cid : c,
					from : from, 
					domain: domain, 
					hash: hash,
					proof: proof,
					simhash: simhash, 
				}:
		log.Debugf("BiddingClient/receiveBidForBiddings, bc.bidIncoming <- &bidSet\n")
	case <-bc.ctx.Done():
		log.Debugf("BiddingClient/receiveBidForBiddings, <-bc.ctx.Done()\n")
		return false 
	case <-ctx.Done():
		log.Debugf("BiddingClient/receiveBidForBiddings, <-ctx.Done()\n")
		return false 
	}

	//wait for result, true if bidding sucessfully, false other. 
	return bc.waitBidResult(ctx, dc) 
}

type bidSet struct {
	url string  
	done chan bool	

	from proto.NodeID
	cid cid.Cid 
	domain    proto.DomainID 

	hash	[]byte 	
	proof 	[]byte	
	simhash uint64	
}

// AddBiddings adds the given cids to the biddinglist, tracked by the given domain 
func (bc *BiddingClient) AddBiddings(ctx context.Context, biddings []types.UrlBidding, peers []proto.NodeID, domain proto.DomainID) {
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
	biddings []types.UrlBidding 
	targets []proto.NodeID
	domain proto.DomainID  
}

func (bc *BiddingClient) addBiddings(ctx context.Context, biddings []types.UrlBidding, targets []proto.NodeID, cancel bool, domain proto.DomainID) {
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
	resp := make(chan []proto.NodeID)
	bc.peerReqs <- resp
	return <-resp
}


func (bc *BiddingClient) SendBids(ctx context.Context, msg *types.UrlBidMessage ) {
	target := msg.Header.UrlBidHeader.Target 
	if string(target) == "" { 
		return
	}

	s, err := bc.host.NewStreamExt(ctx, target, protocol.ID("FRT.Bid"))
	if err != nil {
                return 
        }

        if _, err = s.SendMsg(ctx, msg ); err != nil {
                s.Reset()
                return 
        }

	go libp2phelpers.AwaitEOF(s)
        s.Close()
}


func (bc *BiddingClient) startPeerHandler(p proto.NodeID) *msgQueue {
	mq, ok := bc.peers[p]
	if ok {
		mq.refcnt++
		return nil
	}

	mq = bc.newMsgQueue(p)

	// new peer, we will want to give them our full biddinglist
	biddings := make([]types.UrlBidding, 0, len(bc.bcbl.Entries()))
	for _, b := range bc.bcbl.Entries() {
		mq.bl.AddBiddingEntry(b, b.DomainID) 
		biddings = append(biddings, types.UrlBidding{
						Url: b.Url,
						Probability : b.Probability, 	
					})
	}


	//build bidding message
	mq.out = types.UrlBiddingMessage{
		Header: types.SignedUrlBiddingHeader{
			UrlBiddingHeader: types.UrlBiddingHeader{
				QueryType:    types.WriteQuery,
				NodeID:       bc.nodeID,
			},
		},
		Payload: types.UrlBiddingPayload{
			Requests: biddings,
		},
	}

	mq.work <- struct{}{}

	bc.peers[p] = mq
	go mq.runQueue(bc.ctx)
	return mq
}

func (bc *BiddingClient) stopPeerHandler(p proto.NodeID) {
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

func (bc *BiddingClient) GetUncompletedBiddings() ([]*BiddingEntry, error) {
	return bc.bl.Entries(), nil 
}

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

	peers := make([]proto.NodeID, 0, len(domain.activePeers)) 
	for _, peer := range domain.activePeersArr {
		acked := false  
		for _, bid := range bids {
			if peer == bid.From {
				acked = true  
				break 
			}
		}
		if !acked  {
			peers = append(peers, peer) 
		}
	}
	
	return peers, nil 
}

func (bc *BiddingClient) recrawlBidding(bidding *BiddingEntry, recrawlCount int ) error {
	newBidding := types.UrlBidding {
		Url : bidding.Url, 
		Probability : bidding.Probability,
		ParentUrl   : bidding.ParentUrl, 
		ExpectCrawlerCount : recrawlCount , 
	}

	//filter out those peers that have ack our
	peers , err := bc.GetUnAckedPeers(bidding ) 
	if err != nil {
		err = errors.New("domain not exist")
		return err 	
	}

	bc.addBiddings(context.Background(), 
			[]types.UrlBidding { newBidding }, 
				peers,	
				false, 
			bidding.DomainID ) 

	return nil 

}

// TODO: use goprocess here once i trust it
func (bc *BiddingClient) Run() {

        bc.tick = time.NewTimer(MainCycleInterval )

	// NOTE: Do not open any streams or connections from anywhere in this
	// event loop. Really, just don't do anything likely to block.
	for {
		select {
		case ws := <-bc.biddingIncoming:
			for _, b := range ws.biddings {
				if bc.bl.Add(b.Url, b.ParentUrl, b.Probability, ws.domain, b.ExpectCrawlerCount , time.Now()) {
					//bc.biddinglistGauge.Inc()
				}
			}

			for _, t := range ws.targets {
				p, ok := bc.peers[t]
				if !ok {
					//log.Debugf("tried sending biddinglist change to non-partner peer: %s\n", t)
					p = bc.startPeerHandler(t)
				}

				p.addBiddingMessage(ws.biddings, ws.domain, bc.nodeID )
			}


		case bs := <-bc.bidIncoming:
			url := bs.url 
			bidding, _ := bc.bl.Contains(url)
			if bidding == nil {
				bs.done <- false 
				continue
			}

			//get domain by bidding.DomainID
			domain, exist := bc.f.DomainForID(bidding.DomainID)
			if exist != true {
				//err = errors.New("domain not exist")
				continue 	
			}

			currentHeight := uint32(domain.chain.GetCurrentHeightFromTime( bidding.LastBroadcastTime)) 
			pk, err := kms.GetPublicKey(bs.from)
			if err != nil {
				err = errors.Wrap(err, "failed to cache public key")
				continue 
			}

			// `pi` is the VRF proof
			// pk is publick key of bid.From 
			_, err = ecvrf.NewSecp256k1Sha256Tai().Verify((*ecdsa.PublicKey)(pk), 
				append(utils.Uint32ToBytes(currentHeight), []byte(bs.url)...), 
				bs.proof)
			if err != nil {
				// invalid proof
				continue	
			}

			//get addr by pubkey
			addr, err := crypto.PubKeyHash(pk)
			if err != nil {
				continue 
			}

			balance, totalBalance , err := GetDomainTokenBalanceAndTotal(bc.host, bs.domain, addr, types.Particle )
			if err != nil {
				//err = errors.New("can't get token balance of addr or total addrs in domain!")
				continue 	
			}

			if IsWinner(bs.hash, int64(bidding.ExpectCrawlerCount), balance, totalBalance) != true {
				continue 
			}

			if !bidding.AddBid(url, bs.cid, bs.from, bs.simhash ) {
				bs.done <- false 
				continue 
			}

			if (bidding.CountOfBids() < bidding.ExpectCrawlerCount ) {
				continue 
			}

			//check if cids from every bid match or not. 
			//if not match, caculate how many more crawling are needed. 
			need_crawl_more, err := bidding.NeedCrawlMore() 
			if need_crawl_more > 0 {
				//add need_crawl_count to let bidding (in bidding list) can handle more
				//bid, this is subtle
				bidding.ExpectCrawlerCount +=  need_crawl_more

				bc.recrawlBidding(bidding, need_crawl_more) 
				continue 
			}

			//Crawled succeed 
			if bc.bl.Remove(url, bs.domain ) {
				//bc.biddinglistGauge.Dec()
			}
		
			bc.completed_bl.AddBiddingEntry(bidding, bs.domain ) 
			bs.done <- true  

			continue 
			
		case <-bc.tick.C:
			//todo, rebroadcast bidding when crawling timer timeout, 
			for _, bidding := range bc.bl.Entries() {

				bidCount := len(bidding.GetBids())
				if (bidCount >= bidding.ExpectCrawlerCount ) || 
				   	(bidCount < bidding.ExpectCrawlerCount && 
				    	time.Since(bidding.LastBroadcastTime) < CrawlingInterval ) {
					continue 
				}
				
				need_recrawl_more := bidding.ExpectCrawlerCount - bidCount 
				if need_recrawl_more > 0 {
					//don't add need_recrawl_more of bidding,
					//because we don't need bids more then bidding.ExpectCrawlerCount. 
					bc.recrawlBidding(bidding, need_recrawl_more ) 
				}
			}

			bc.tick.Reset(MainCycleInterval)

		case p := <-bc.connectEvent:
			if p.connect {
				bc.startPeerHandler(p.peer)
			} else {
				bc.stopPeerHandler(p.peer)
			}

		case req := <-bc.peerReqs:
			peers := make([]proto.NodeID, 0, len(bc.peers))
			for p := range bc.peers {
				peers = append(peers, p)
			}
			req <- peers

		case <-bc.ctx.Done():
			return
		}

	}
}

func (bc *BiddingClient) newMsgQueue(p proto.NodeID) *msgQueue {
	return &msgQueue{
		done:    make(chan struct{}),
		work:    make(chan struct{}, 1),
		bl:      NewThreadSafe(),
		host 	: bc.host, 
		outlk:	new(sync.Mutex), 
		p:       p,
		refcnt:  1,
	}
}

