// package decision implements the decision engine for the bitswap service.
package decision

import (
	"context"
	"sync"
	"time"
	"fmt" 

	//wyong, 20200827 
	//bsmsg "github.com/siegfried415/gdf-rebuild/frontera/message"

	wl "github.com/siegfried415/gdf-rebuild/frontera/wantlist"
	//bidlist "github.com/siegfried415/gdf-rebuild/frontera/bidlist"
	"github.com/siegfried415/gdf-rebuild/proto"

	//wyong, 20200827 
        "github.com/siegfried415/gdf-rebuild/types"

	cid "github.com/ipfs/go-cid"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	//blocks "github.com/ipfs/go-block-format"
	//peer "github.com/libp2p/go-libp2p-peer"
	logging "github.com/ipfs/go-log"
)


// TODO consider taking responsibility for other types of requests. For
// example, there could be a |cancelQueue| for all of the cancellation
// messages that need to go out. There could also be a |wantlistQueue| for
// the local peer's wantlists. Alternatively, these could all be bundled
// into a single, intelligent global queue that efficiently
// batches/combines and takes all of these into consideration.
//
// Right now, messages go onto the network for four reasons:
// 1. an initial `sendwantlist` message to a provider of the first key in a
//    request
// 2. a periodic full sweep of `sendwantlist` messages to all providers
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
// * when sending out the wantlists, include `cancel` requests
// * when handling `blockrequests`, include `sendwantlist` and `cancel` as
//   appropriate
// * when handling `cancel`, if we recently received a wanted block from a
//   peer, include a partial wantlist that contains a few other high priority
//   blocks
//
// In a sense, if we treat the decision engine as a black box, it could do
// whatever it sees fit to produce desired outcomes (get wanted keys
// quickly, maintain good relationships with peers, etc).

var log = logging.Logger("engine")
const (
	// outboxChanBuffer must be 0 to prevent stale messages from being sent
	outboxChanBuffer = 0
	// maxMessageSize is the maximum size of the batched payload
	maxMessageSize = 512 * 1024
)


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

type Engine struct {
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

	bs bstore.Blockstore

	lock sync.Mutex // protects the fields immediatly below
	// ledgerMap lists Ledgers by their Partner key.
	ledgerMap map[proto.NodeID]*ledger

	ticker *time.Ticker

	//wyong, 20190118
	nodeid  proto.NodeID
}

//add nodeid, wyong, 20190118
func NewEngine(ctx context.Context, nodeid proto.NodeID /* , bs bstore.Blockstore */ ) *Engine {
	e := &Engine{
		ledgerMap:        make(map[proto.NodeID]*ledger),
		nodeid:		  nodeid, 	//wyong, 20190118
		//bs:               bs,		//wyong, 20200813 
		//peerRequestQueue: newPRQ(),
		outbox:           make(chan /*  *Envelope */ *types.UrlBidMessage , outboxChanBuffer),
		workSignal:       make(chan struct{}, 1),
		ticker:           time.NewTicker(time.Millisecond * 100),
	}
	//go e.taskWorker(ctx)
	return e
}

func (e *Engine) WantlistForPeer(p proto.NodeID) (out []*wl.BiddingEntry) {
	partner := e.findOrCreate(p)
	partner.lk.Lock()
	defer partner.lk.Unlock()
	return partner.wantList.SortedBiddingEntries()
}

//wyong, 20190118
func(e *Engine) GetNodeId() proto.NodeID {
	return e.nodeid
}

func (e *Engine) LedgerForPeer(p proto.NodeID) *Receipt {
	ledger := e.findOrCreate(p)

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
func (e *Engine) getBid( p proto.NodeID) (*bidlist.Bidlist, error) {
	l := e.findOrCreate(p)
	return l.GetBids()
}
*/

//TODO, wyong, 20181221
func(e *Engine) CreateBid(ctx context.Context, url string, cid cid.Cid) {
	fmt.Printf("Engine/CreateBid(10), url=%s, cid = %s\n", url, cid )
	/*todo, wyong, 20200827
        d, exist := e.f.DomainForUrl(url)
        if exist != true {
		return 
        }
	*/

	bidMsg, err := e.createUrlBidMessage(ctx, /* d.domainID,*/  url, cid )
	if err != nil {
		fmt.Printf("Engine/CreateBid(15), err = %s\n", err.Error())
		return // ctx cancelled
	}

	fmt.Printf("Engine/CreateBid(20)\n")
	e.outbox <- bidMsg  
	fmt.Printf("Engine/CreateBid(30)\n")
}

/*TODO, taskWorker is unnecessary, wyong, 20181221
func (e *Engine) taskWorker(ctx context.Context) {
	log.Debugf("taskWorker called")
	defer close(e.outbox) // because taskWorker uses the channel exclusively
	for {
		oneTimeUse := make(chan *Envelope, 1) // buffer to prevent blocking
		select {
		case <-ctx.Done():
			log.Debugf("taskWorker, ctx.Done fired")
			return
		case e.outbox <- oneTimeUse:
			log.Debugf("taskWorker, oneTimeUse fired")
		}
		// receiver is ready for an outoing envelope. let's prepare one. first,
		// we must acquire a task from the PQ...
		envelope, err := e.nextEnvelope(ctx)
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
func(e *Engine)createUrlBidMessage(ctx context.Context, url string, c cid.Cid) (*types.UrlBidMessage, error) {
	fmt.Printf("Engine/createUrlBidMessage(10), url=%s, cid = %s\n", url, c )
	var target proto.NodeID 	
	for _, l := range e.ledgerMap {
		//l.lk.Lock()
		if _, ok := l.WantListContains(url); ok {
			//e.peerRequestQueue.Push(l.Partner, entry)
			//work = true
			//TODO, get a target 
			target = l.Partner
			fmt.Printf("Engine/createUrlBidMessage(20), found target %s\n", target ) 
			break
		}
		//l.lk.Unlock()
	}

	fmt.Printf("Engine/createUrlBidMessage(30)\n")
	// with a task in hand, we're ready to prepare the envelope...
	//msg := bsmsg.New(true,string(e.nodeid))
	
	//wyong, 20190115 
	//bid, err := bsmsg.NewBid(url, cid)
	//if err == nil { 
	//	msg.AddBidEntry(bid)
	//}
	
	//TODO, don't forget set from by current node's id,  wyong, 20190118
	//cids := make(map[proto.NodeID]cid.Cid )
	//cids[from] = cid 
	from := e.GetNodeId()
	//cids := map[proto.NodeID]cid.Cid{from : c} 
	//msg.AddBidding(url, 0, cids )

	fmt.Printf("Engine/createUrlBidMessage(40), from=%s\n", from )

	//return &Envelope{
	//	Peer:    target,
	//	Message: msg,
	//	Sent: func() {
	//		//TODO, wyong, 20181222
	//		//nextTask.Done(nextTask.Entries)
	//		//select {
	//		//case e.workSignal <- struct{}{}:
	//		//	// work completing may mean that our queue will provide new
	//		//	// work to be done.
	//		//default:
	//		//}
	//	},
	//}, nil

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
                         Cids : map[string]string { url : c.String() },
                 },
        }

	fmt.Printf("Engine/createUrlBidMessage(50), msg=%s\n", bidMsg )
	return bidMsg, nil 
}

/* nunecessary, wyong, 20181221
// nextEnvelope runs in the taskWorker goroutine. Returns an error if the
// context is cancelled before the next Envelope can be created.
func (e *Engine) nextEnvelope(ctx context.Context) (*Envelope, error) {
	log.Debugf("nextEnvelope called")
	for {
		nextTask := e.peerRequestQueue.Pop()
		for nextTask == nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-e.workSignal:
				nextTask = e.peerRequestQueue.Pop()
			case <-e.ticker.C:
				e.peerRequestQueue.thawRound()
				nextTask = e.peerRequestQueue.Pop()
			}
		}

		// with a task in hand, we're ready to prepare the envelope...
		msg := bsmsg.New(true)
		for _, entry := range nextTask.Entries {
			block, err := e.bs.Get(entry.Url)
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
				case e.workSignal <- struct{}{}:
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
func (e *Engine) Outbox() <-chan *types.UrlBidMessage {
	return e.outbox
}

// Returns a slice of Peers with whom the local node has active sessions
func (e *Engine) Peers() []proto.NodeID {
	e.lock.Lock()
	defer e.lock.Unlock()

	response := make([]proto.NodeID, 0, len(e.ledgerMap))

	for _, ledger := range e.ledgerMap {
		response = append(response, ledger.Partner)
	}
	return response
}

//TODO, wyong, 20181222
func(e *Engine) GetBidding(/* wyong, 20181227 p proto.NodeID */ ) (*wl.Wantlist, error) {
	fmt.Printf("Engine/GetBidding(10)\n")

	/* wyong, 20181227 
	//TODO, get biddings from ledger
	l := e.findOrCreate(p)
	return l.GetWants()
	*/

	//TODO, wyong, 20181227 
	for _, ledger := range e.ledgerMap {
		fmt.Printf("Engine/GetBidding(20)\n")
		return ledger.GetWants() 
	}

	fmt.Printf("Engine/GetBidding(30)\n")
	return nil, nil 
}

// MessageReceived performs book-keeping. Returns error if passed invalid
// arguments.
func (e *Engine) UrlBiddingMessageReceived( ctx context.Context,  m *types.UrlBiddingMessage ) error {
	fmt.Printf("Engine/UrlBiddingMessageReceived called(10)\n")
	if m.Empty() {
		fmt.Printf("Engine/UrlBiddingMessageReceived(15), received empty message\n")
	}

	p := m.Header.UrlBiddingHeader.NodeID 
	fmt.Printf("Engine/UrlBiddingMessageReceived called(20), from=%s\n", p )

	newWorkExists := false
	defer func() {
		if newWorkExists {
			e.signalNewWork()
		}
	}()

	l := e.findOrCreate(p)
	l.lk.Lock()
	defer l.lk.Unlock()

	fmt.Printf("Engine/UrlBiddingMessageReceived called(30)\n")
	/*todo, wyong, 20200827 
	if m.Full() {
		l.wantList = wl.New()
	}
	*/

	//var msgSize int
	//var activeEntries []*wl.BiddingEntry

	//for _, entry := range m.Wantlist() {
	for _, entry := range m.Payload.Requests {
		fmt.Printf("Engine/UrlBiddingMessageReceived called(40), entry.Url=%s\n", entry.Url )
		if entry.Cancel {
			fmt.Printf("Engine/UrlBiddingMessageReceived(50), cancel %s", entry.Url)
			l.CancelWant(entry.Url)
			//e.peerRequestQueue.Remove(entry.Url, p)
		} else {
			fmt.Printf("Engine/UrlBiddingMessageReceived(60), wants %s with probability %f\n", entry.Url, entry.Probability)
			l.AddWant(entry.Url, entry.Probability)

			/*TODO,wyong, 20181221
			blockSize, err := e.bs.GetSize(entry.Url)
			if err != nil {
				if err == bstore.ErrNotFound {
					continue
				}
				log.Error(err)
			} else {
				// we have the block
				newWorkExists = true
				if msgSize+blockSize > maxMessageSize {
					e.peerRequestQueue.Push(p, activeEntries...)
					activeEntries = []*wl.BiddingEntry{}
					msgSize = 0
				}
				activeEntries = append(activeEntries, entry.BiddingEntry)
				msgSize += blockSize
			}
			*/
		}
	}

	/*
	if len(activeEntries) > 0 {
		e.peerRequestQueue.Push(p, activeEntries...)
	}
	*/

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
func (e *Engine) addBid(bid bsmsg.BidEntry ) {
	log.Debugf("addBid called")
	work := false

	for _, l := range e.ledgerMap {
		l.lk.Lock()
		if entry, ok := l.WantListContains(bid.Url); ok {
			e.peerRequestQueue.Push(l.Partner, entry)
			work = true
		}
		l.lk.Unlock()
	}

	if work {
		e.signalNewWork()
	}
}

func (e *Engine) AddBid(bid bsmsg.BidEntry ) {
	log.Debugf("AddBid called")
	e.lock.Lock()
	defer e.lock.Unlock()

	e.addBid(bid)
}
*/


/*TODO, wyong, 20181220
// TODO add contents of m.WantList() to my local wantlist? NB: could introduce
// race conditions where I send a message, but MessageSent gets handled after
// MessageReceived. The information in the local wantlist could become
// inconsistent. Would need to ensure that Sends and acknowledgement of the
// send happen atomically

func (e *Engine) MessageSent(p proto.NodeID, m bsmsg.BiddingMessage) error {
	log.Debugf("MessageSend called")
	l := e.findOrCreate(p)
	l.lk.Lock()
	defer l.lk.Unlock()

	for _, bid := range m.Bids() {
		//l.SentBytes(len(bid.RawData()))
		l.wantList.Remove(bid.Url())
		e.peerRequestQueue.Remove(bid.Url(), p)
	}

	return nil
}
*/


func (e *Engine) PeerConnected(p proto.NodeID) {
	e.lock.Lock()
	defer e.lock.Unlock()
	l, ok := e.ledgerMap[p]
	if !ok {
		l = newLedger(p)
		e.ledgerMap[p] = l
	}
	l.lk.Lock()
	defer l.lk.Unlock()
	l.ref++
}

func (e *Engine) PeerDisconnected(p proto.NodeID) {
	e.lock.Lock()
	defer e.lock.Unlock()
	l, ok := e.ledgerMap[p]
	if !ok {
		return
	}
	l.lk.Lock()
	defer l.lk.Unlock()
	l.ref--
	if l.ref <= 0 {
		delete(e.ledgerMap, p)
	}
}

func (e *Engine) numBytesSentTo(p proto.NodeID) uint64 {
	// NB not threadsafe
	return e.findOrCreate(p).Accounting.BytesSent
}

func (e *Engine) numBytesReceivedFrom(p proto.NodeID) uint64 {
	// NB not threadsafe
	return e.findOrCreate(p).Accounting.BytesRecv
}

// ledger lazily instantiates a ledger
func (e *Engine) findOrCreate(p proto.NodeID) *ledger {
	log.Debugf("findOrCreate called") 
	e.lock.Lock()
	defer e.lock.Unlock()
	l, ok := e.ledgerMap[p]
	if !ok {
		log.Debugf("findOrCreate , before call newLedger...") 
		l = newLedger(p)
		e.ledgerMap[p] = l
	}
	return l
}

func (e *Engine) signalNewWork() {
	// Signal task generation to restart (if stopped!)
	select {
	case e.workSignal <- struct{}{}:
	default:
	}
}
