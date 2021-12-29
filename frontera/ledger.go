package frontera

import (
	"sync"
	"time"

	log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)


func newLedger(p proto.NodeID) *ledger {
	log.Debugf("newLedger called...") 
	return &ledger{
		biddingList:   NewBiddingList(),
		Partner:    p,
		sentToPeer: make(map[string]time.Time),
	}
}

// ledger stores the data exchange relationship between two peers.
// NOT threadsafe
type ledger struct {
	// Partner is the remote Peer.
	Partner proto.NodeID

	// Accounting tracks bytes sent and received.
	Accounting debtRatio

	// lastExchange is the time of the last data exchange.
	lastExchange time.Time

	// exchangeCount is the number of exchanges with this peer
	exchangeCount uint64

	// biddingList is a (bounded, small) set of keys that Partner desires.
	biddingList *BiddingList

	// sentToPeer is a set of keys to ensure we dont send duplicate blocks
	// to a given peer
	sentToPeer map[string]time.Time

	// ref is the reference count for this ledger, its used to ensure we
	// don't drop the reference to this ledger in multi-connection scenarios
	ref int

	lk sync.Mutex
}

type Receipt struct {
	Peer      string
	Value     float64
	Sent      uint64
	Recv      uint64
	Exchanged uint64
}

type debtRatio struct {
	BytesSent uint64
	BytesRecv uint64
}

func (dr *debtRatio) Value() float64 {
	return float64(dr.BytesSent) / float64(dr.BytesRecv+1)
}

func (l *ledger) SentBytes(n int) {
	l.exchangeCount++
	l.lastExchange = time.Now()
	l.Accounting.BytesSent += uint64(n)
}

func (l *ledger) ReceivedBytes(n int) {
	l.exchangeCount++
	l.lastExchange = time.Now()
	l.Accounting.BytesRecv += uint64(n)
}

func (l *ledger) AddBidding(url string, parentUrl string, probability float64, expectCrawlerCount int, hash []byte, proof []byte ) {
	l.biddingList.Add(url, parentUrl, probability, expectCrawlerCount, hash, proof  )
}

func (l *ledger) GetBiddings() (*BiddingList, error) {
	return l.biddingList, nil
}

func (l *ledger) CancelBidding(url string) {
	l.biddingList.Remove(url)
}

func (l *ledger) BiddingListContains(url string) (*BiddingEntry, bool) {
	return l.biddingList.Contains(url)
}

func (l *ledger) ExchangeCount() uint64 {
	return l.exchangeCount
}

