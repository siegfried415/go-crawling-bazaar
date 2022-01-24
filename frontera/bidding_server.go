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


// package decision implements the decision engine for the bitswap service.
package frontera 

import (
	"context"
	"sync"
	"time"
	"math/big" 
	"errors" 

	"crypto/ecdsa"

	ecvrf "github.com/vechain/go-ecvrf"
	cid "github.com/ipfs/go-cid"

	crypto "github.com/siegfried415/go-crawling-bazaar/crypto"
        "github.com/siegfried415/go-crawling-bazaar/kms"
        log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
        "github.com/siegfried415/go-crawling-bazaar/types"
        utils "github.com/siegfried415/go-crawling-bazaar/utils"
)


// TODO consider taking responsibility for other types of requests. For
// example, there could be a |cancelQueue| for all of the cancellation
// messages that need to go out. There could also be a |biddinglistQueue| for
// the local peer's biddinglists. Alternatively, these could all be bundled
// into a single, intelligent global queue that efficiently
// batches/combines and takes all of these into consideration.
//
// Right now, messages go onto the network for four reasons:
// 1. an initial `sendbiddinglist` message to a provider of the first key in a
//    request
// 2. a periodic full sweep of `sendbiddinglist` messages to all providers
// 3. upon receipt of blocks, a `cancel` message to all peers
// 4. draining the priority queue of `blockrequests` from peers
//
// Presently, only `blockrequests` are handled by the decision engine.
// However, there is an opportunity to give it more responsibility! If the
// decision engine is given responsibility for all of the others, it can
// intelligently decide how to combine requests efficiently.
//
// Some examples of what would be possible:
//
// * when sending out the biddinglists, include `cancel` requests
// * when handling `blockrequests`, include `sendbiddinglist` and `cancel` as
//   appropriate
// * when handling `cancel`, if we recently received a wanted block from a
//   peer, include a partial biddinglist that contains a few other high priority
//   blocks
//
// In a sense, if we treat the decision engine as a black box, it could do
// whatever it sees fit to produce desired outcomes (get wanted keys
// quickly, maintain good relationships with peers, etc).

const (
	// outboxChanBuffer must be 0 to prevent stale messages from being sent
	outboxChanBuffer = 0
	// maxMessageSize is the maximum size of the batched payload
	maxMessageSize = 512 * 1024
)

var (
        MaxChallengeTicket *big.Int
)

func init() {
        MaxChallengeTicket = &big.Int{}
        // The size of the challenge must equal the size of the Signature (ticket) generated.
        // Currently this is a Elliptic Curve VRF, which is 32 bytes. 
        MaxChallengeTicket.Exp(big.NewInt(2), big.NewInt(32*8), nil)
        MaxChallengeTicket.Sub(MaxChallengeTicket, big.NewInt(1))
}


type BiddingServer struct {
	f *Frontera 

	// peerRequestQueue is a priority queue of requests received from peers.
	// Requests are popped from the queue, packaged up, and placed in the
	// outbox.
	//peerRequestQueue *prq

	// FIXME it's a bit odd for the client and the worker to both share memory
	// (both modify the peerRequestQueue) and also to communicate over the
	// workSignal channel. consider sending requests over the channel and
	// allowing the worker to have exclusive access to the peerRequestQueue. In
	// that case, no lock would be required.
	workSignal chan struct{}

	// outbox contains outgoing messages to peers. This is owned by the
	// taskWorker goroutine
	outbox chan *types.UrlBidMessage

	lock sync.Mutex // protects the fields immediatly below
	// ledgerMap lists Ledgers by their Partner key.
	ledgerMap map[proto.NodeID]*ledger

	ticker *time.Ticker

	nodeid  proto.NodeID
	host net.RoutedHost 	
}

func NewBiddingServer (ctx context.Context, nodeid proto.NodeID, host net.RoutedHost ) *BiddingServer {
	bs := &BiddingServer{
		ledgerMap:        make(map[proto.NodeID]*ledger),
		nodeid:		  nodeid, 
		outbox:           make(chan *types.UrlBidMessage , outboxChanBuffer),
		workSignal:       make(chan struct{}, 1),
		ticker:           time.NewTicker(time.Millisecond * 100),
		host  : 	  host, 
	}
	//go e.taskWorker(ctx)
	return bs  
}

func (bs *BiddingServer) BiddinglistForPeer(p proto.NodeID) (out []*BiddingEntry) {
	partner := bs.findOrCreate(p)
	partner.lk.Lock()
	defer partner.lk.Unlock()
	return partner.biddingList.SortedBiddingEntries()
}

func(bs *BiddingServer) GetNodeId() proto.NodeID {
	return bs.nodeid
}

func (bs *BiddingServer) LedgerForPeer(p proto.NodeID) *Receipt {
	ledger := bs.findOrCreate(p)
	ledger.lk.Lock()
	defer ledger.lk.Unlock()

	return &Receipt{
		Peer:      ledger.Partner.String(),
		Value:     ledger.Accounting.Value(),
		Sent:      ledger.Accounting.BytesSent,
		Recv:      ledger.Accounting.BytesRecv,
		Exchanged: ledger.ExchangeCount(),
	}
}


func(bs *BiddingServer) PutBid(ctx context.Context, url string, cid cid.Cid) {
	bidMsg, err := bs.createUrlBidMessage(ctx, url, cid )
	if err != nil {
		return // ctx cancelled
	}

	bs.outbox <- bidMsg  
}

func(bs *BiddingServer)createUrlBidMessage(ctx context.Context, url string, c cid.Cid) (*types.UrlBidMessage, error) {
	var target proto.NodeID 	
	var hash []byte
	var proof []byte

	//todo, Need a better way to find bidding
	for _, l := range bs.ledgerMap {
		if bidding, ok := l.BiddingListContains(url); ok {
			target = l.Partner
			hash = bidding.Hash
			proof = bidding.Proof 
			
			break
		}
	}

	from := bs.GetNodeId()
	bid := types.UrlBid {
		Url  : url, 
		Cid  : c.String(), 
		Hash : hash, 
		Proof: proof, 

	}

	//build bidding message
	bidMsg := &types.UrlBidMessage{
                 Header: types.UrlBidSignedHeader{
                         UrlBidHeader: types.UrlBidHeader{
                                 //QueryType:    types.WriteQuery,
				 Target : target, 
                                 NodeID:       from,
                         },
                 },
                 Payload: types.UrlBidPayload{
                         Bids : []types.UrlBid { bid },
                 },
        }

	return bidMsg, nil 
}

// Outbox returns a channel of one-time use Envelope channels.
func (bs *BiddingServer) Outbox() <-chan *types.UrlBidMessage {
	return bs.outbox
}

// Returns a slice of Peers with whom the local node has active sessions
func (bs *BiddingServer) Peers() []proto.NodeID {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	response := make([]proto.NodeID, 0, len(bs.ledgerMap))

	for _, ledger := range bs.ledgerMap {
		response = append(response, ledger.Partner)
	}
	return response
}

//TODO
func(bs *BiddingServer) GetBidding() (*BiddingList, error) {
	log.Debugf("BiddingServer/GetBidding start ... ")
	for _, ledger := range bs.ledgerMap {
		log.Debugf("BiddingServer/GetBidding , call ledger GetBiddings()")
		return ledger.GetBiddings() 
	}

	return nil, nil 
}


//split UrlBiddingMessageReceived to 2 functions
func (bs *BiddingServer) UrlBiddingMessageReceived( ctx context.Context,  m *types.UrlBiddingMessage ) error {
	log.Debugf("BiddingServer/UrlBiddingMessageReceived start ... ")
	if m.Empty() {
		log.Debugf("received empty message\n")
	}

	p := m.Header.UrlBiddingHeader.NodeID 
	domainID := m.Header.DomainID

	return bs.UrlBiddingReceived(ctx, p, domainID,  m.Payload.Requests ) 
}


// MessageReceived performs book-keeping. Returns error if passed invalid arguments.
func (bs *BiddingServer) UrlBiddingReceived( ctx context.Context, p proto.NodeID, domainID proto.DomainID,  requests []types.UrlBidding) error {

	newWorkExists := false
	defer func() {
		if newWorkExists {
			bs.signalNewWork()
		}
	}()

	l := bs.findOrCreate(p)
	l.lk.Lock()
	defer l.lk.Unlock()

        privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
                return err 
        }
	sk := (*ecdsa.PrivateKey)(privateKey)

        addr, err := crypto.PubKeyHash(privateKey.PubKey())
	if err != nil {
                return err 
        }

        domain, exist := bs.f.DomainForID(domainID)
        if exist != true {
                err = errors.New("domain not exist")
                return err 
        }

	currentHeight := uint32(domain.chain.GetCurrentHeight())
	balance, totalBalance , err := GetDomainTokenBalanceAndTotal(bs.host, domainID, addr, types.Particle )
	if err != nil {
                err = errors.New("can't get token balance of addr or total addrs in domain!")
                return err 
	}

	for _, bidding := range requests {
		if bidding.Cancel {
			l.CancelBidding(bidding.Url)
		} else {
			// `beta`: the VRF hash output
			// `pi`: the VRF proof
			// add nonce to prevent replay attack
			beta, pi, err := ecvrf.NewSecp256k1Sha256Tai().Prove(sk, 
				append(utils.Uint32ToBytes(currentHeight), []byte(bidding.Url) ... ))  
			if err != nil {
				// something wrong.
				// most likely sk is not properly loaded.
				//return err 
				continue 
			}

			//todo, only node successfully in competeting has the right to crawl the url in bidding
			//add tokenBalance and totalTokenBalance
			if IsWinner(beta, int64(bidding.ExpectCrawlerCount), balance, totalBalance){
				//todo, save hash(beta) & proof(pi) with bidding
				//entry.Hash = beta 
				//entry.VRFProof = pi 
				//winBiddings = append (winBiddings, bidding) 

				l.AddBidding(bidding.Url, bidding.ParentUrl, bidding.Probability, bidding.ExpectCrawlerCount, beta, pi )
			}

		}


	}

	return nil
}

func (bs *BiddingServer) PeerConnected(p proto.NodeID) {
	bs.lock.Lock()
	defer bs.lock.Unlock()
	l, ok := bs.ledgerMap[p]
	if !ok {
		l = newLedger(p)
		bs.ledgerMap[p] = l
	}
	l.lk.Lock()
	defer l.lk.Unlock()
	l.ref++
}

func (bs *BiddingServer) PeerDisconnected(p proto.NodeID) {
	bs.lock.Lock()
	defer bs.lock.Unlock()
	l, ok := bs.ledgerMap[p]
	if !ok {
		return
	}
	l.lk.Lock()
	defer l.lk.Unlock()
	l.ref--
	if l.ref <= 0 {
		delete(bs.ledgerMap, p)
	}
}

func (bs *BiddingServer) numBytesSentTo(p proto.NodeID) uint64 {
	// NB not threadsafe
	return bs.findOrCreate(p).Accounting.BytesSent
}

func (bs *BiddingServer) numBytesReceivedFrom(p proto.NodeID) uint64 {
	// NB not threadsafe
	return bs.findOrCreate(p).Accounting.BytesRecv
}

// ledger lazily instantiates a ledger
func (bs *BiddingServer) findOrCreate(p proto.NodeID) *ledger {
	log.Debugf("findOrCreate called") 
	bs.lock.Lock()
	defer bs.lock.Unlock()
	l, ok := bs.ledgerMap[p]
	if !ok {
		log.Debugf("findOrCreate , before call newLedger...") 
		l = newLedger(p)
		bs.ledgerMap[p] = l
	}
	return l
}

func (bs *BiddingServer) signalNewWork() {
	// Signal task generation to restart (if stopped!)
	select {
	case bs.workSignal <- struct{}{}:
	default:
	}
}
