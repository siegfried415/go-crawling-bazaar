package frontera

import (
	"context"
	"fmt"
	"time"

	//notifications "github.com/siegfried415/gdf-rebuild/frontera/notifications"

	//exchange "github.com/ipfs/go-ipfs-exchange-interface"
	lru "github.com/hashicorp/golang-lru"

	//cid "github.com/ipfs/go-cid"
	//blocks "github.com/ipfs/go-block-format"
	//loggables "github.com/ipfs/go-libp2p-loggables"
	//peer "github.com/libp2p/go-libp2p-peer"
	//logging "github/ipfs/go-log"

	"github.com/siegfried415/gdf-rebuild/proto" 
)

const activeWantsLimit = 16

// Session holds state for an individual bitswap transfer operation.
// This allows bitswap to make smarter decisions about who to send wantlist
// info to, and who to request blocks from
type Session struct {
	ctx            context.Context
	tofetch        *urlQueue
	activePeers    map[proto.NodeID]struct{}
	activePeersArr []proto.NodeID

	f *Frontera 
	incoming     chan bidRecv

	newReqs      chan []string 
	cancelUrls   chan []string
	interestReqs chan interestReq

	interest  *lru.Cache
	liveWants map[string]time.Time

	tick          *time.Timer
	baseTickDelay time.Duration

	latTotal time.Duration
	fetchcnt int

	//notif notifications.PubSub

	//uuid logging.Loggable

	id  uint64
	tag string
}

//TODO, search session accroding to url, if not found, create a new Session, 
//wyong, 20190113
func (f *Frontera) GetSession(ctx context.Context, url string) (*Session, error) {
	return f.NewSession(ctx), nil  
}

// NewSession creates a new bitswap session whose lifetime is bounded by the
// given context
func (f *Frontera) NewSession(ctx context.Context) /* exchange.Fetcher  wyong, 20190119 */ *Session {
	s := &Session{
		activePeers:   make(map[proto.NodeID]struct{}),
		liveWants:     make(map[string]time.Time),
		newReqs:       make(chan []string),
		cancelUrls:    make(chan []string),
		tofetch:       newUrlQueue(),
		interestReqs:  make(chan interestReq),
		ctx:           ctx,
		f:            f,
		incoming:      make(chan bidRecv),
		//notif:         notifications.New(),
		//uuid:          loggables.Uuid("GetBlockRequest"),
		baseTickDelay: time.Millisecond * 500,
		id:            f.getNextSessionID(),
	}

	s.tag = fmt.Sprint("bs-ses-", s.id)

	cache, _ := lru.New(2048)
	s.interest = cache

	f.sessLk.Lock()
	f.sessions = append(f.sessions, s)
	f.sessLk.Unlock()

	go s.run(ctx)

	return s
}

func (f *Frontera) removeSession(s *Session) {
	//s.notif.Shutdown()

	live := make([]string, 0, len(s.liveWants))
	for u := range s.liveWants {
		live = append(live, u)
	}
	f.CancelWants(live, s.id)

	f.sessLk.Lock()
	defer f.sessLk.Unlock()
	for i := 0; i < len(f.sessions); i++ {
		if f.sessions[i] == s {
			f.sessions[i] = f.sessions[len(f.sessions)-1]
			f.sessions = f.sessions[:len(f.sessions)-1]
			return
		}
	}
}

type bidRecv struct {
	from proto.NodeID
	url string 
}

func (s *Session) receiveBidFrom(from proto.NodeID,  url string  ) {
	//log.Debug("receiveBidFrom called...")
	select {
	case s.incoming <- bidRecv{from: from, url: url }:
	case <-s.ctx.Done():
	}
}

type interestReq struct {
	url   string  
	resp chan bool
}

// TODO: PERF: this is using a channel to guard a map access against race
// conditions. This is definitely much slower than a mutex, though its unclear
// if it will actually induce any noticeable slowness. This is implemented this
// way to avoid adding a more complex set of mutexes around the liveWants map.
// note that in the average case (where this session *is* interested in the
// block we received) this function will not be called, as the cid will likely
// still be in the interest cache.
func (s *Session) isLiveWant(url string) bool {
	resp := make(chan bool, 1)
	select {
	case s.interestReqs <- interestReq{
		url: url,
		resp: resp,
	}:
	case <-s.ctx.Done():
		return false
	}

	select {
	case want := <-resp:
		return want
	case <-s.ctx.Done():
		return false
	}
}

func (s *Session) interestedIn(url string ) bool {
	return s.interest.Contains(url) || s.isLiveWant(url)
}

const provSearchDelay = time.Second * 10

func (s *Session) addActivePeer(p proto.NodeID) {
	/* todo, wyong, 20200730 
	if _, ok := s.activePeers[p]; !ok {
		s.activePeers[p] = struct{}{}
		s.activePeersArr = append(s.activePeersArr, p)

		cmgr := s.f.network.ConnectionManager()
		cmgr.TagPeer(p, s.tag, 10)
	}
	*/
}

func (s *Session) resetTick() {
	if s.latTotal == 0 {
		s.tick.Reset(provSearchDelay)
	} else {
		avLat := s.latTotal / time.Duration(s.fetchcnt)
		s.tick.Reset(s.baseTickDelay + (3 * avLat))
	}
}

func (s *Session) run(ctx context.Context) {
	s.tick = time.NewTimer(provSearchDelay)
	newpeers := make(chan proto.NodeID, 16)
	for {
		select {
		case bid := <-s.incoming:
			//log.Debug("session/run, bid := <- s.incoming, bid.url = %s", bid.url )
			s.tick.Stop()

			if bid.from != "" {
				s.addActivePeer(bid.from)
			}

			s.receiveBid(ctx, bid.url )
			s.resetTick()

		case urls := <-s.newReqs:
			//log.Debug("session/run, urls := <- s.newReqs " )
			for _, url := range urls {
				s.interest.Add(url, nil)
			}
			if len(s.liveWants) < activeWantsLimit {
				toadd := activeWantsLimit - len(s.liveWants)
				if toadd > len(urls) {
					toadd = len(urls)
				}

				now := urls[:toadd]
				urls = urls[toadd:]

				s.wantUrls(ctx, now)
			}
			for _, url := range urls {
				s.tofetch.Push(url)
			}

		case urls := <-s.cancelUrls:
			//log.Debug("session/run, urls := <- s.cancelUrls " )
			s.cancel(urls)

		case <-s.tick.C:
			//log.Debug("session/run,<- s.tick.C" )
			live := make([]string, 0, len(s.liveWants))
			now := time.Now()
			for url := range s.liveWants {
				live = append(live, url)
				s.liveWants[url] = now
			}

			// Broadcast these keys to everyone we're connected to
			s.f.wm.WantUrls(ctx, live, nil, s.id)

			/*TODO, review the following code is necessary, wyong, 20190119
			if len(live) > 0 {
				go func(k string ) {
					// TODO: have a task queue setup for this to:
					// - rate limit
					// - manage timeouts
					// - ensure two 'findprovs' calls for the same block don't run concurrently
					// - share peers between sessions based on interest set
					for p := range s.bs.network.FindProvidersAsync(ctx, k, 10) {
						newpeers <- p
					}
				}(live[0])
			}
			*/
			s.resetTick()
		case p := <-newpeers:
			//log.Debug("session/run, p := <-newpeers " )
			s.addActivePeer(p)

		case lwchk := <-s.interestReqs:
			//log.Debug("session/run, lwchk := <-s.interestReqs" )
			lwchk.resp <- s.urlIsWanted(lwchk.url)

		case <-ctx.Done():
			//log.Debug("session/run, <-ctx.Done()" )
			s.tick.Stop()

			/*TODO, hold the session for later use, wyong, 20190124
			s.bs.removeSession(s)

			cmgr := s.bs.network.ConnectionManager()
			for _, p := range s.activePeersArr {
				cmgr.UntagPeer(p, s.tag)
			}
			*/

			return
		}
	}
}

func (s *Session) urlIsWanted(url string) bool {
	_, ok := s.liveWants[url]
	if !ok {
		ok = s.tofetch.Has(url)
	}

	return ok
}

func (s *Session) receiveBid(ctx context.Context, url string ) {
	//log.Debug("receiveBid called..." )
	//url := bid.Url()

	if s.urlIsWanted(url) {
		tval, ok := s.liveWants[url]
		if ok {
			s.latTotal += time.Since(tval)
			delete(s.liveWants, url)
		} else {
			s.tofetch.Remove(url)
		}
		s.fetchcnt++
		//s.notif.Publish(url)

		if next := s.tofetch.Pop(); /* next.Defined() */ next != ""  {
			s.wantUrls(ctx, []string{next})
		}
	}
}

func (s *Session) wantUrls(ctx context.Context, urls []string ) {
	now := time.Now()
	for _, url := range urls {
		s.liveWants[url] = now
	}
	s.f.wm.WantUrls(ctx, urls, s.activePeersArr, s.id)
}

func (s *Session) cancel(urls []string ) {
	for _, url := range urls {
		s.tofetch.Remove(url)
	}
}

func (s *Session) cancelWants(urls []string ) {
	select {
	case s.cancelUrls <- urls:
	case <-s.ctx.Done():
	}
}

func (s *Session) fetch(ctx context.Context, urls []string ) {
	select {
	case s.newReqs <- urls:
	case <-ctx.Done():
	case <-s.ctx.Done():
	}
}

// GetBlocks fetches a set of blocks within the context of this session and
// returns a channel that found blocks will be returned on. No order is
// guaranteed on the returned blocks.
func (s *Session) PutBiddings(ctx context.Context, urls []string) ( error) {
	//log.Debug("PutBiddings called")
	//ctx = logging.ContextWithLoggable(ctx, s.uuid)
	return putBiddingsImpl(ctx, urls, /* s.notif,*/  s.fetch, s.cancelWants)
}

// GetBlock fetches a single block
func (s *Session) PutBidding(parent context.Context, url string) (/* blocks.Block, */  error) {
	//log.Debug("PutBidding called")
	return putBidding(parent, url, s.PutBiddings)
}

type urlQueue struct {
	elems []string 
	//eset  *cid.Set
	eset map[string]struct{} 
}

func newUrlQueue() *urlQueue {
	//return &cidQueue{eset: cid.NewSet()}
	return &urlQueue{eset:  make(map[string]struct{})}
}

func (uq *urlQueue) Pop() string {
	for {
		if len(uq.elems) == 0 {
			return "" 
		}

		out := uq.elems[0]
		uq.elems = uq.elems[1:]

		//if cq.eset.Has(out) {
		_, has := uq.eset[out]
		if has {
			//cq.eset.Remove(out)
			delete(uq.eset, out)
			return out
		}
	}
}


func (uq *urlQueue) Push(u string) {
	//if uq.eset.Visit(u) {
	//	uq.elems = append(uq.elems, u)
	//}

	_, has := uq.eset[u] 
	if !has {
		uq.eset[u]=struct{}{}
		uq.elems = append(uq.elems, u)
	}

}

func (uq *urlQueue) Remove(u string) {
	//cq.eset.Remove(c)
	delete(uq.eset, u) 
}

func (uq *urlQueue) Has(u string) bool {
	_, has := uq.eset[u]
	return has 
}

func (uq *urlQueue) Len() int {
	//return cq.eset.Len()
	return len(uq.eset)
}
