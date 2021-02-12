// package biddinglist implements an object for bitswap that contains the keys
// that a given peer wants.
package frontera 

import (
	"sort"
	"time" 
	"sync"

	cid "github.com/ipfs/go-cid"
        //peer "github.com/libp2p/go-libp2p-peer"
	"github.com/siegfried415/gdf-rebuild/proto" 
)


//wyong, 20190114 
type BidEntry struct {
	Cid	cid.Cid 
	Hash	uint64	//wyong, 20210206 
	From	proto.NodeID 
}

//wyong, 20190118
func NewBidEntry(cid cid.Cid, hash uint64, from proto.NodeID ) *BidEntry {
	return &BidEntry{
		Cid:      cid,
		Hash:	  hash, //wyong, 20210206 
		From: 	  from,
	}
}

//wyong, 20190119 
func( bid *BidEntry)GetCid() cid.Cid {
	return bid.Cid
}

type BiddingEntry struct {
	Url	string 
	ParentUrl	string 	//wyong, 20210204

	Probability float64	//int, wyong, 20200827 

	//wyong, 20210203 
	//SesTrk map[proto.DomainID]struct{}

	// Trash in a book-keeping field
	Trash bool

	//wyong, 20190131
	Seen bool 

	//wyong, 20190114
	Bids map[proto.NodeID]*BidEntry  

	DomainID proto.DomainID		//wyong, 20210203 
	ExpectCrawlerCount	int	//wyong, 20210203 

	LastBroadcastTime	time.Time	//wyong, 20210204 

	Hash	[]byte 		//wyong, 20210205 
	Proof 	[]byte		//wyong, 20210205  

}

// NewRefEntry creates a new reference tracked biddinglist entry
func NewBiddingEntry(url string, parentUrl string, p float64, domainID proto.DomainID, expectCrawlerCount int , lastBroadcastTime time.Time  ) *BiddingEntry {
	return &BiddingEntry{
		Url:      url,
		ParentUrl : parentUrl, //wyong, 20210204 

		Probability: p,
		
		//wyong, 20210203 
		//SesTrk:   make(map[proto.DomainID]struct{}),

		Seen: false, //wyong, 20190131 
		Bids : 	  make(map[proto.NodeID]*BidEntry), //wyong, 20190125 

		DomainID : domainID,		//wyong, 20210203 
		ExpectCrawlerCount : expectCrawlerCount, //wyong, 20210203
		LastBroadcastTime  : lastBroadcastTime,  //wyong, 20210204 
	}
}


func (bidding *BiddingEntry) GetUrl() string {
	return bidding.Url
}


//wyong, 20190115 
func (bidding *BiddingEntry) AddBid(url string, cid cid.Cid, from proto.NodeID ) bool {
	bidding.Seen = false	//wyong, 20190131 

	if _, ok := bidding.Bids[from]; ok {
		return false
	}

	//todo, insert this commit to current revision (of url). 
	//wyong, 20210117

	bidding.Bids[from] = &BidEntry{
		Cid:  cid ,
		From: from, 
	}

	return true
}

//wyong, 20190115 
func (bidding *BiddingEntry) GetBids() []*BidEntry {
	bids := make([]*BidEntry, 0, len(bidding.Bids))
	for _, bid := range bidding.Bids{
		bids = append(bids, bid)
	}
	return bids
}

//wyong, 20190115 
func (bidding *BiddingEntry) CountOfBids() int {
	return len(bidding.Bids)
}

//todo, wyong, 20210204
func(bidding *BiddingEntry) NeedCrawlMore() int {
	return 0 
}

//wyong, 20200828 
func(bidding *BiddingEntry) GetCid() cid.Cid {
	c := cid.Cid{}
	for _, bid := range bidding.Bids{
		if c == cid.Undef { 
			c = bid.Cid 
		} else if c != bid.Cid {
			return cid.Cid{}
		}
	}
	return c
}

type entrySlice []*BiddingEntry
func (es entrySlice) Len() int           { return len(es) }
func (es entrySlice) Swap(i, j int)      { es[i], es[j] = es[j], es[i] }
func (es entrySlice) Less(i, j int) bool { return es[i].Probability > es[j].Probability }


type ThreadSafe struct {
	lk  sync.RWMutex
	set map[string]*BiddingEntry
}

func NewThreadSafe() *ThreadSafe {
	return &ThreadSafe{
		set: make(map[string]*BiddingEntry),
	}
}

// Add adds the given cid to the biddinglist with the specified priority, governed
// by the session ID 'ses'.  if a cid is added under multiple session IDs, then
// it must be removed by each of those sessions before it is no longer 'in the
// biddinglist'. Calls to Add are idempotent given the same arguments. Subsequent
// calls with different values for priority will not update the priority
// TODO: think through priority changes here
// Add returns true if the cid did not exist in the biddinglist before this call
// (even if it was under a different session)
func (w *ThreadSafe) Add(url string, parentUrl string,  probability float64, domain proto.DomainID, expectCrawlerCount int, lastBroadcastTime time.Time ) bool {
	w.lk.Lock()
	defer w.lk.Unlock()
	if e, ok := w.set[url]; ok {
		//e.SesTrk[domain] = struct{}{}
		e.DomainID = domain 	//wyong, 20210203 
		e.Seen = false		//wyong, 20190131 
		
		//wyong, 20210204
		e.LastBroadcastTime = lastBroadcastTime

		return false
	}

	w.set[url] = &BiddingEntry{
		Url:      url,
		ParentUrl: parentUrl, 	//wyong, 20210204 
		Probability: probability,
		Seen: false, //wyong, 20190131 

		//wyong, 20210203 
		//SesTrk:   map[proto.DomainID]struct{}{domain: struct{}{}},
		DomainID : domain, 

		//wyong, 20210204 
		ExpectCrawlerCount : expectCrawlerCount,

		Bids : map[proto.NodeID]*BidEntry{}, //wyong, 20200831 

		//wyong, 20210204
		LastBroadcastTime : lastBroadcastTime, 
	}

	return true
}

// AddEntry adds given Entry to the biddinglist. For more information see Add method.
func (w *ThreadSafe) AddBiddingEntry(e *BiddingEntry, domain proto.DomainID ) bool {
	w.lk.Lock()
	defer w.lk.Unlock()
	if ex, ok := w.set[e.Url]; ok {
		//wyong, 20210203 
		//ex.SesTrk[domain] = struct{}{}
		ex.DomainID = domain 
		e.Seen = false	//wyong, 20190131 

		//wyong, 20210204
		ex.LastBroadcastTime = e.LastBroadcastTime

		return false
	}
	w.set[e.Url] = e

	//wyong, 20210203 
	//e.SesTrk[domain] = struct{}{}
	e.DomainID = domain 
	
	return true
}

// Remove removes the given cid from being tracked by the given session.
// 'true' is returned if this call to Remove removed the final session ID
// tracking the cid. (meaning true will be returned iff this call caused the
// value of 'Contains(c)' to change from true to false)
func (w *ThreadSafe) Remove(url string, domain proto.DomainID ) bool {
	w.lk.Lock()
	defer w.lk.Unlock()
	_, ok := w.set[url]
	if !ok {
		return false
	}

	//wyong, 20210203 
	//delete(e.SesTrk, domain )
	//if len(e.SesTrk) == 0 {
		delete(w.set, url)
		return true
	//}
	//return false
}

// Contains returns true if the given cid is in the biddinglist tracked by one or
// more sessions
func (w *ThreadSafe) Contains(url string) (*BiddingEntry, bool) {
	w.lk.RLock()
	defer w.lk.RUnlock()
	e, ok := w.set[url]
	return e, ok
}

func (w *ThreadSafe) Entries() []*BiddingEntry {
	w.lk.RLock()
	defer w.lk.RUnlock()
	es := make([]*BiddingEntry, 0, len(w.set))
	for _, e := range w.set {
		es = append(es, e)
	}
	return es
}

func (w *ThreadSafe) SortedEntries() []*BiddingEntry {
	es := w.Entries()
	sort.Sort(entrySlice(es))
	return es
}

func (w *ThreadSafe) Len() int {
	w.lk.RLock()
	defer w.lk.RUnlock()
	return len(w.set)
}

// not threadsafe
type BiddingList struct {
	set map[string]*BiddingEntry
}

func NewBiddingList() *BiddingList {
	return &BiddingList{
		set: make(map[string]*BiddingEntry),
	}
}

func (w *BiddingList) Len() int {
	return len(w.set)
}

func (w *BiddingList) Add(url string, parentUrl string, probability float64, expectCrawlerCount int,  hash []byte, proof []byte  ) bool {
	if be, ok := w.set[url]; ok {
		be.Seen = false 	//wyong, 20190131 
		return false
	}

	w.set[url] = &BiddingEntry{
		Url:      url,
		ParentUrl : parentUrl, 	//wyong, 20210204 
		Probability: probability,
		Seen: false, 	//wyong, 20190131 
		ExpectCrawlerCount : expectCrawlerCount, //wyong, 20210204 
		Bids : map[proto.NodeID]*BidEntry{}, //wyong, 20200831 
		Hash : hash,		//wyong, 20210205  
		Proof : proof, 		//wyong, 20210205 
	}

	return true
}

func (w *BiddingList) AddBiddingEntry(e *BiddingEntry) bool {
	if be, ok := w.set[e.Url]; ok {
		be.Seen = false		//wyong, 20190131 
		return false
	}
	w.set[e.Url] = e
	return true
}

func (w *BiddingList) Remove(url string) bool {
	_, ok := w.set[url]
	if !ok {
		return false
	}

	delete(w.set, url)
	return true
}

func (w *BiddingList) Contains(url string) (*BiddingEntry, bool) {
	e, ok := w.set[url]
	return e, ok
}

func (w *BiddingList) BiddingEntries() []*BiddingEntry {
	es := make([]*BiddingEntry, 0, len(w.set))
	for _, e := range w.set {
		es = append(es, e)
	}
	return es
}

//wyong, 20190131
//func (w *BiddingList) SeenEntries() bool {
//	for _, e := range w.set {
//		e.Seen = true 	
//	}
//	return true  
//}


func (w *BiddingList) SortedBiddingEntries() []*BiddingEntry {
	es := w.BiddingEntries()
	sort.Sort(entrySlice(es))
	return es
}
