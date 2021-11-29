// package decision implements the decision engine for the bitswap service.
package frontera 

import (
	"context"
	"sync"
	"time"
	//"fmt" 
	"math/big" 
	"errors" 

	//wyong, 20210202
	"crypto/ecdsa"

	//wyong, 20200827 
	//bsmsg "github.com/siegfried415/go-crawling-bazaar/frontera/message"

	//bl "github.com/siegfried415/go-crawling-bazaar/frontera/biddinglist"
	//bidlist "github.com/siegfried415/go-crawling-bazaar/frontera/bidlist"
	"github.com/siegfried415/go-crawling-bazaar/proto"

	//wyong, 20210202 
        //"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
        "github.com/siegfried415/go-crawling-bazaar/kms"

	//wyong, 20200827 
        "github.com/siegfried415/go-crawling-bazaar/types"

	cid "github.com/ipfs/go-cid"
	//bstore "github.com/ipfs/go-ipfs-blockstore"
	//blocks "github.com/ipfs/go-block-format"
	//peer "github.com/libp2p/go-libp2p-peer"
	//logging "github.com/ipfs/go-log"

	//wyong, 20201215
        log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	
	//wyong, 20210118 
	ecvrf "github.com/vechain/go-ecvrf"

	//wyong, 20210630
        //sortition "github.com/siegfried415/go-crawling-bazaar/frontera/sortition"

	//wyong, 20210706 
	net "github.com/siegfried415/go-crawling-bazaar/net"
	crypto "github.com/siegfried415/go-crawling-bazaar/crypto"

	//wyong, 20210719
	//pi "github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
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

//var log = logging.Logger("engine")
const (
	// outboxChanBuffer must be 0 to prevent stale messages from being sent
	outboxChanBuffer = 0
	// maxMessageSize is the maximum size of the batched payload
	maxMessageSize = 512 * 1024
)

var (
        MaxChallengeTicket *big.Int
)

//wyong, 20210202 
func init() {
        MaxChallengeTicket = &big.Int{}
        // The size of the challenge must equal the size of the Signature (ticket) generated.
        // Currently this is a Elliptic Curve VRF, which is 32 bytes. wyong, 20210219 
        MaxChallengeTicket.Exp(big.NewInt(2), big.NewInt(32*8), nil)
        MaxChallengeTicket.Sub(MaxChallengeTicket, big.NewInt(1))
}


/* no need anymore, wyong, 20200827 
// Envelope contains a message for a Peer
type Envelope struct {
	// Peer is the intended recipient
	Peer proto.NodeID

	// Message is the payload
	Message bsmsg.BiddingMessage

	// A callback to notify the decision queue that the task is complete
	Sent func()
}
*/

type BiddingServer struct {
	//wyong, 20210203 
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
	outbox chan *types.UrlBidMessage	//*Envelope, wyong, 20200827 

	//bs bstore.Blockstore

	lock sync.Mutex // protects the fields immediatly below
	// ledgerMap lists Ledgers by their Partner key.
	ledgerMap map[proto.NodeID]*ledger

	ticker *time.Ticker

	//wyong, 20190118
	nodeid  proto.NodeID

	//wyong, 20210706 
	host net.RoutedHost 	
}

//add nodeid, wyong, 20190118
func NewBiddingServer (ctx context.Context, nodeid proto.NodeID, host net.RoutedHost ) *BiddingServer {
	bs := &BiddingServer{
		ledgerMap:        make(map[proto.NodeID]*ledger),
		nodeid:		  nodeid, 	//wyong, 20190118
		//bs:               bs,		//wyong, 20200813 
		//peerRequestQueue: newPRQ(),
		outbox:           make(chan /*  *Envelope */ *types.UrlBidMessage , outboxChanBuffer),
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

//wyong, 20190118
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


/* wyong, 20190116 
//TODO, get bid from ledger, wyong, 20181222
func (bs *BiddingServer) getBid( p proto.NodeID) (*bidlist.Bidlist, error) {
	l := bs.findOrCreate(p)
	return l.GetBids()
}
*/

//TODO, wyong, 20181221
func(bs *BiddingServer) PutBid(ctx context.Context, url string, cid cid.Cid) {
	log.Debugf("BiddingServer/PutBid(10), url=%s, cid = %s\n", url, cid )

	//wyong, 20210220 
	//wyong, 20200827
        //d, exist := bs.f.DomainForUrl(url)
        //if exist != true {
	//	return 
        //}

	//write url<-->cid to domain.UrlCidsCache and url chain, wyong, 20210126  
	//move this to BiddingClient side, wyong, 20210220 
	//d.SetCid(url, cid )

	bidMsg, err := bs.createUrlBidMessage(ctx, /* d.domainID,*/  url, cid )
	if err != nil {
		log.Debugf("BiddingServer/PutBid(15), err = %s\n", err.Error())
		return // ctx cancelled
	}

	log.Debugf("BiddingServer/PutBid(20)\n")
	bs.outbox <- bidMsg  
	log.Debugf("BiddingServer/PutBid(30)\n")
}

/*TODO, taskWorker is unnecessary, wyong, 20181221
func (bs *BiddingServer) taskWorker(ctx context.Context) {
	log.Debugf("taskWorker called")
	defer close(bs.outbox) // because taskWorker uses the channel exclusively
	for {
		oneTimeUse := make(chan *Envelope, 1) // buffer to prevent blocking
		select {
		case <-ctx.Done():
			log.Debugf("taskWorker, ctx.Done fired")
			return
		case bs.outbox <- oneTimeUse:
			log.Debugf("taskWorker, oneTimeUse fired")
		}
		// receiver is ready for an outoing envelope. let's prepare one. first,
		// we must acquire a task from the PQ...
		envelope, err := bs.nextEnvelope(ctx)
		if err != nil {
			close(oneTimeUse)
			return // ctx cancelled
		}
		oneTimeUse <- envelope // buffered. won't block
		close(oneTimeUse)
	}
}
*/

//TODO,wyong, 20181221
func(bs *BiddingServer)createUrlBidMessage(ctx context.Context, url string, c cid.Cid) (*types.UrlBidMessage, error) {
	log.Debugf("BiddingServer/createUrlBidMessage(10), url=%s, cid = %s\n", url, c )
	var target proto.NodeID 	

	//wyong, 20210205 
	var hash []byte
	var proof []byte

	//todo, Need a better way to find bidding,  wyong, 20210205 
	for _, l := range bs.ledgerMap {
		//l.lk.Lock()
		if bidding, ok := l.BiddingListContains(url); ok {
			//bs.peerRequestQueue.Push(l.Partner, entry)
			//work = true
			//TODO, get a target 
			target = l.Partner
			log.Debugf("BiddingServer/createUrlBidMessage(20), found target %s\n", target ) 

			//todo, wyong, 20210205 
			hash = bidding.Hash
			proof = bidding.Proof 
			
			break
		}
		//l.lk.Unlock()
	}

	log.Debugf("BiddingServer/createUrlBidMessage(30)\n")
	// with a task in hand, we're ready to prepare the envelope...
	//msg := bsmsg.New(true,string(bs.nodeid))
	
	//wyong, 20190115 
	//bid, err := bsmsg.NewBid(url, cid)
	//if err == nil { 
	//	msg.AddBidEntry(bid)
	//}
	
	//TODO, don't forget set from by current node's id,  wyong, 20190118
	//cids := make(map[proto.NodeID]cid.Cid )
	//cids[from] = cid 
	from := bs.GetNodeId()
	//cids := map[proto.NodeID]cid.Cid{from : c} 
	//msg.AddBidding(url, 0, cids )

	log.Debugf("BiddingServer/createUrlBidMessage(40), from=%s\n", from )

	//return &Envelope{
	//	Peer:    target,
	//	Message: msg,
	//	Sent: func() {
	//		//TODO, wyong, 20181222
	//		//nextTask.Done(nextTask.Entries)
	//		//select {
	//		//case bs.workSignal <- struct{}{}:
	//		//	// work completing may mean that our queue will provide new
	//		//	// work to be done.
	//		//default:
	//		//}
	//	},
	//}, nil
	bid := types.UrlBid {
		Url  : url, 
		Cid  : c.String(), 
		Hash : hash, 
		Proof: proof, 

	}

	//build bidding message, wyong, 20200827
	bidMsg := &types.UrlBidMessage{
                 Header: types.UrlBidSignedHeader{
                         UrlBidHeader: types.UrlBidHeader{
                                 //QueryType:    types.WriteQuery,
				 Target : target, 
                                 NodeID:       from,
                                 //DomainID:     domain,  todo, wyong, 20200827 
 
                                 //todo, wyong, 20200817
                                 //ConnectionID: connID,
                                 //SeqNo:        seqNo,
                                 //Timestamp:    getLocalTime(),
                         },
                 },
                 Payload: types.UrlBidPayload{
                         Bids : []types.UrlBid { bid },
                 },
        }

	log.Debugf("BiddingServer/createUrlBidMessage(50), msg=%s\n", bidMsg )
	return bidMsg, nil 
}

/* nunecessary, wyong, 20181221
// nextEnvelope runs in the taskWorker goroutine. Returns an error if the
// context is cancelled before the next Envelope can be created.
func (bs *BiddingServer) nextEnvelope(ctx context.Context) (*Envelope, error) {
	log.Debugf("nextEnvelope called")
	for {
		nextTask := bs.peerRequestQueue.Pop()
		for nextTask == nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-bs.workSignal:
				nextTask = e.peerRequestQueue.Pop()
			case <-bs.ticker.C:
				bs.peerRequestQueue.thawRound()
				nextTask = bs.peerRequestQueue.Pop()
			}
		}

		// with a task in hand, we're ready to prepare the envelope...
		msg := bsmsg.New(true)
		for _, entry := range nextTask.Entries {
			block, err := bs.bs.Get(entry.Url)
			if err != nil {
				log.Errorf("tried to execute a task and errored fetching block: %s", err)
				continue
			}
			msg.AddBlock(block)
		}

		if msg.Empty() {
			// If we don't have the block, don't hold that against the peer
			// make sure to update that the task has been 'completed'
			nextTask.Done(nextTask.Entries)
			continue
		}

		return &Envelope{
			Peer:    nextTask.Target,
			Message: msg,
			Sent: func() {
				nextTask.Done(nextTask.Entries)
				select {
				case bs.workSignal <- struct{}{}:
					// work completing may mean that our queue will provide new
					// work to be done.
				default:
				}
			},
		}, nil
	}

	return &Envelope{}, nil
}
*/

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

//TODO, wyong, 20181222
func(bs *BiddingServer) GetBidding(/* wyong, 20181227 p proto.NodeID */ ) (*BiddingList, error) {
	log.Debugf("BiddingServer/GetBidding(10)\n")

	/* wyong, 20181227 
	//TODO, get biddings from ledger
	l := bs.findOrCreate(p)
	return l.GetBiddings()
	*/

	//TODO, wyong, 20181227 
	for _, ledger := range bs.ledgerMap {
		log.Debugf("BiddingServer/GetBidding(20)\n")
		return ledger.GetBiddings() 
	}

	log.Debugf("BiddingServer/GetBidding(30)\n")
	return nil, nil 
}

//wyong, 20210219
//func(bs *BiddingServer) getMaxChallengeTicket(challengeTicket []byte) *big.Int {
//	numTickets := len(challengeTicket)
//	log.Debugf("BiddingServer/getMaxChallengeTicket(10), numTickets=%d\n", numTickets )
//
//      MaxChallengeTicket := &big.Int{}
//      // The size of the challenge must equal the size of the Signature (ticket) generated.
//        // Currently this is a secp256k1.Sign signature, which is 65 bytes.
//        MaxChallengeTicket.Exp(big.NewInt(2), big.NewInt(int64(numTickets*8)), nil)
//        MaxChallengeTicket.Sub(MaxChallengeTicket, big.NewInt(1))
//
//	return MaxChallengeTicket 
//}



//split UrlBiddingMessageReceived to 2 functions, wyong, 20210117 
func (bs *BiddingServer) UrlBiddingMessageReceived( ctx context.Context,  m *types.UrlBiddingMessage ) error {
	log.Debugf("BiddingServer/UrlBiddingMessageReceived called(10)\n")
	if m.Empty() {
		log.Debugf("BiddingServer/UrlBiddingMessageReceived(15), received empty message\n")
	}

	p := m.Header.UrlBiddingHeader.NodeID 
	log.Debugf("BiddingServer/UrlBiddingMessageReceived called(20), from=%s\n", p )

	//wyong, 20210218 
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

	log.Debugf("BiddingServer/UrlBiddingReceived called(10)\n")
	/*todo, wyong, 20200827 
	if m.Full() {
		l.biddingList = bl.New()
	}
	*/

	//wyong, 20210205 
        //var privateKey *asymmetric.PrivateKey
        privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
                return err 
        }
	sk := (*ecdsa.PrivateKey)(privateKey)

	//wyong, 20210702 
	log.Debugf("BiddingServer/UrlBiddingReceived(20)\n" )
        addr, err := crypto.PubKeyHash(privateKey.PubKey())
	if err != nil {
                return err 
        }

	//wyong, 20210127 
	//IDs, err := kms.GetAllNodeID()
	//if err != nil {
	//	log.WithError(err).Error("get all node id failed")
	//	return err 
	//}
	//peersCount := int64(len(IDs)) 

	//wyong, 20210706 
        domain, exist := bs.f.DomainForID(domainID)
        if exist != true {
                err = errors.New("domain not exist")
                return err 
        }

	//wyong, 20210719 
	currentHeight := uint32(domain.chain.GetCurrentHeight())
	
	//wyong, 20210702 
	//peersCount := int64(len(domain.activePeers)) 
	//if _, ok := domain.activePeers[bs.nodeid]; ok {
	//	//does not take myself to account, wyong, 20210219 
	//	peersCount-- 
	//}
	balance, totalBalance , err := GetDomainTokenBalanceAndTotal(bs.host, domainID, addr, types.Particle )
	if err != nil {
                err = errors.New("can't get token balance of addr or total addrs in domain!")
                return err 
	}


	log.Debugf("BiddingServer/UrlBiddingReceived(30), balance=%d, totalBalance=%d\n", balance, totalBalance)

	//wyong, 20210719 
	//nonce, err := GetNextAccountNonce(bs.host, addr)
	//if err != nil {
        //        err = errors.New("can't get next nonce of addr !")
        //        return err 
	//}

	//var msgSize int
	//var activeEntries []*bl.BiddingEntry

	for _, bidding := range requests {
		log.Debugf("BiddingServer/UrlBiddingReceived called(40), bidding.Url=%s\n", bidding.Url )
		if bidding.Cancel {
			log.Debugf("BiddingServer/UrlBiddingReceived(50), cancel %s", bidding.Url)
			l.CancelBidding(bidding.Url)
			//bs.peerRequestQueue.Remove(entry.Url, p)
		} else {
			log.Debugf("BiddingServer/UrlBiddingReceived(60), wants %s with probability %f\n", bidding.Url, bidding.Probability)
			// `beta`: the VRF hash output
			// `pi`: the VRF proof
			// add nonce to prevent replay attack, wyong, 20210729 
			beta, pi, err := ecvrf.NewSecp256k1Sha256Tai().Prove(sk, 
					// []byte(bidding.Url)
					//  append(utils.UInt64ToBytes(nonce), []byte(bidding.Url)
					append(utils.Uint32ToBytes(currentHeight), []byte(bidding.Url) ... ))  
			if err != nil {
				// something wrong.
				// most likely sk is not properly loaded.
				//return err 
				continue 
			}

			log.Debugf("BiddingServer/UrlBiddingReceived(70)\n" )
			//todo, only node successfully in competeting has the right to crawl the url in bidding
			//wyong, 20210117 

			//add tokenBalance and totalTokenBalance, wyong, 20210702 
			if IsWinner(beta, int64(bidding.ExpectCrawlerCount), balance, totalBalance){
				//todo, save hash(beta) & proof(pi) with bidding, wyong, 20200118 
				//entry.Hash = beta 
				//entry.VRFProof = pi 
				//winBiddings = append (winBiddings, bidding) 

				log.Debugf("BiddingServer/UrlBiddingReceived(80)\n" )
				//wyong, 20210205 
				l.AddBidding(bidding.Url, bidding.ParentUrl, bidding.Probability, bidding.ExpectCrawlerCount, beta, pi )
			}

			log.Debugf("BiddingServer/UrlBiddingReceived(90)\n" )

			/*TODO,wyong, 20181221
			blockSize, err := bs.bs.GetSize(entry.Url)
			if err != nil {
				if err == bstore.ErrNotFound {
					continue
				}
				log.Error(err)
			} else {
				// we have the block
				newWorkExists = true
				if msgSize+blockSize > maxMessageSize {
					bs.peerRequestQueue.Push(p, activeEntries...)
					activeEntries = []*BiddingEntry{}
					msgSize = 0
				}
				activeEntries = append(activeEntries, entry.BiddingEntry)
				msgSize += blockSize
			}
			*/
		}

		log.Debugf("BiddingServer/UrlBiddingReceived(100)\n" )

	}

	/*
	if len(activeEntries) > 0 {
		bs.peerRequestQueue.Push(p, activeEntries...)
	}
	*/

	log.Debugf("BiddingServer/UrlBiddingReceived(110)\n" )

	/*wyong, 20190118
	for _, bid := range m.Bids() {
		//log.Debugf("got block %s %d bytes", block, len(block.RawData()))
		//l.ReceivedBytes(len(block.RawData()))
		l.AddBid(bid.Url, bid.Cid)
	}
	*/

	return nil
}

/*
func (bs *BiddingServer) addBid(bid bsmsg.BidEntry ) {
	log.Debugf("addBid called")
	work := false

	for _, l := range bs.ledgerMap {
		l.lk.Lock()
		if entry, ok := l.BiddingListContains(bid.Url); ok {
			bs.peerRequestQueue.Push(l.Partner, entry)
			work = true
		}
		l.lk.Unlock()
	}

	if work {
		bs.signalNewWork()
	}
}

func (bs *BiddingServer) AddBid(bid bsmsg.BidEntry ) {
	log.Debugf("AddBid called")
	bs.lock.Lock()
	defer bs.lock.Unlock()

	bs.addBid(bid)
}
*/


/*TODO, wyong, 20181220
// TODO add contents of m.BiddingList() to my local biddinglist? NB: could introduce
// race conditions where I send a message, but MessageSent gets handled after
// MessageReceived. The information in the local biddinglist could become
// inconsistent. Would need to ensure that Sends and acknowledgement of the
// send happen atomically

func (bs *BiddingServer) MessageSent(p proto.NodeID, m bsmsg.BiddingMessage) error {
	log.Debugf("MessageSend called")
	l := bs.findOrCreate(p)
	l.lk.Lock()
	defer l.lk.Unlock()

	for _, bid := range m.Bids() {
		//l.SentBytes(len(bid.RawData()))
		l.biddingList.Remove(bid.Url())
		bs.peerRequestQueue.Remove(bid.Url(), p)
	}

	return nil
}
*/


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
