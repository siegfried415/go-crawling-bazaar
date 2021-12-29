// package biddinglist implements an object for bitswap that contains the keys
// that a given peer wants.
package frontera 

import (
	"sort"
	"time" 
	"sync"

        "github.com/mfonda/simhash"

	cid "github.com/ipfs/go-cid"

        //log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto" 

)


type BidEntry struct {
	Cid	cid.Cid 
	Hash	uint64	
	From	proto.NodeID 

	VerifiedCount int 
	//Signature 	asymmetric.Signature	
}

func NewBidEntry(cid cid.Cid, hash uint64, from proto.NodeID ) *BidEntry {
	return &BidEntry{
		Cid:      cid,
		Hash:	  hash, 
		From: 	  from,
		VerifiedCount : 1 , 	
	}
}

func( bid *BidEntry)GetCid() cid.Cid {
	return bid.Cid
}

type BiddingEntry struct {
	Url	string 
	ParentUrl	string 	
	Probability float64	

	// Trash in a book-keeping field
	Trash bool

	Seen bool 
	Bids map[proto.NodeID]*BidEntry  
	DomainID proto.DomainID		
	ExpectCrawlerCount	int	
	LastBroadcastTime	time.Time	
	Hash	[]byte 		
	Proof 	[]byte		

}

// NewRefEntry creates a new reference tracked biddinglist entry
func NewBiddingEntry(url string, parentUrl string, p float64, domainID proto.DomainID, expectCrawlerCount int , lastBroadcastTime time.Time  ) *BiddingEntry {
	return &BiddingEntry{
		Url:      url,
		ParentUrl : parentUrl, 
		Probability: p,
		Seen: false, 
		Bids : 	  make(map[proto.NodeID]*BidEntry), 
		DomainID : domainID,		
		ExpectCrawlerCount : expectCrawlerCount, 
		LastBroadcastTime  : lastBroadcastTime, 
	}
}


func (bidding *BiddingEntry) GetUrl() string {
	return bidding.Url
}


func (bidding *BiddingEntry) AddBid(url string, cid cid.Cid, from proto.NodeID, hash uint64) bool {
	bidding.Seen = false	

	if _, ok := bidding.Bids[from]; ok {
		return false
	}

	verifiedCount := 1 
	//Update VerifiedCount of other bids
	if len(bidding.Bids) > 0 {
		for _, bid := range bidding.Bids{
			if simhash.Compare(hash,  bid.Hash) < 2 {
				verifiedCount ++
				bid.VerifiedCount++
			}
		}
	}

	bidding.Bids[from] = &BidEntry{
		Cid:  cid ,
		From: from, 
		VerifiedCount : verifiedCount, 
	}

	return true
}

func (bidding *BiddingEntry) GetMaxVerifiedBid() ( *BidEntry, error ) {
	if len(bidding.Bids) == 0 {
		return nil, nil 
	}

	maxVerifiedCount := 0 
	var maxVerifiedNodeID proto.NodeID
	for k, bid := range bidding.Bids{
		if bid.VerifiedCount > maxVerifiedCount {
			maxVerifiedCount = bid.VerifiedCount 
			maxVerifiedNodeID = k 	
		}
	}
	
	return bidding.Bids[maxVerifiedNodeID], nil 
}

func (bidding *BiddingEntry) NeedCrawlMore() (int, error ) {
	needCrawlCount := bidding.ExpectCrawlerCount 

	if len(bidding.Bids) > 0 {
		maxVerifiedBid, _ := bidding.GetMaxVerifiedBid() 
		needCrawlCount -= maxVerifiedBid.VerifiedCount 
		if needCrawlCount < 0 {
			needCrawlCount = 0 
		}
	}

	return needCrawlCount, nil 
}

func (bidding *BiddingEntry) GetBids() []*BidEntry {
	bids := make([]*BidEntry, 0, len(bidding.Bids))
	for _, bid := range bidding.Bids{
		bids = append(bids, bid)
	}
	return bids
}

func (bidding *BiddingEntry) CountOfBids() int {
	return len(bidding.Bids)
}

func(bidding *BiddingEntry) GetCid() (cid.Cid, error) {
	maxVerifiedBid, err := bidding.GetMaxVerifiedBid() 
	if err != nil {
		return cid.Cid{}, err 
	}

	return maxVerifiedBid.Cid, nil 
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
		e.DomainID = domain 	
		e.Seen = false	
		
		e.LastBroadcastTime = lastBroadcastTime

		return false
	}

	w.set[url] = &BiddingEntry{
		Url:      url,
		ParentUrl: parentUrl, 	
		Probability: probability,
		Seen: false, 

		DomainID : domain, 
		ExpectCrawlerCount : expectCrawlerCount,
		Bids : map[proto.NodeID]*BidEntry{}, 
		LastBroadcastTime : lastBroadcastTime, 
	}

	return true
}

// AddEntry adds given Entry to the biddinglist. For more information see Add method.
func (w *ThreadSafe) AddBiddingEntry(e *BiddingEntry, domain proto.DomainID ) bool {
	w.lk.Lock()
	defer w.lk.Unlock()
	if ex, ok := w.set[e.Url]; ok {
		ex.DomainID = domain 
		e.Seen = false	
		ex.LastBroadcastTime = e.LastBroadcastTime

		return false
	}
	w.set[e.Url] = e
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

	delete(w.set, url)
	return true
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
		be.Seen = false 	
		return false
	}

	w.set[url] = &BiddingEntry{
		Url:      url,
		ParentUrl : parentUrl, 	
		Probability: probability,
		Seen: false, 	
		ExpectCrawlerCount : expectCrawlerCount, 
		Bids : map[proto.NodeID]*BidEntry{}, 
		Hash : hash,		
		Proof : proof, 	
	}

	return true
}

func (w *BiddingList) AddBiddingEntry(e *BiddingEntry) bool {
	if be, ok := w.set[e.Url]; ok {
		be.Seen = false	
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


func (w *BiddingList) SortedBiddingEntries() []*BiddingEntry {
	es := w.BiddingEntries()
	sort.Sort(entrySlice(es))
	return es
}
