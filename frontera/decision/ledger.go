package decision

import (
	"sync"
	"time"
	"fmt"

	wl "github.com/siegfried415/gdf-rebuild/frontera/wantlist"
	//bidlist "github.com/siegfried415/gdf-rebuild/frontera/bidlist"
	"github.com/siegfried415/gdf-rebuild/proto"

	//cid "github.com/ipfs/go-cid"
	//peer "github.com/libp2p/go-libpproto2p-peer"
	//logging "github.com/ipfs/go-log"
)

//var log = logging.Logger("ledger")

func newLedger(p proto.NodeID) *ledger {
	log.Debugf("newLedger called...") 
	return &ledger{
		wantList:   wl.New(),
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

	// wantList is a (bounded, small) set of keys that Partner desires.
	wantList *wl.Wantlist

	//wyong, 20181223
	//bidList *bidlist.Bidlist

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

func (l *ledger) AddWant(url string, probability float64 ) {
	fmt.Printf("ledger/AddWant(10), peer %s wants %s with probability %f\n", l.Partner, url, probability )
	l.wantList.Add(url, probability )
}

func (l *ledger) GetWants() (*wl.Wantlist, error) {
	fmt.Printf("ledger/GetWants(10)\n") 
	return l.wantList, nil
}

func (l *ledger) CancelWant(url string) {
	l.wantList.Remove(url)
}

func (l *ledger) WantListContains(url string) (*wl.BiddingEntry, bool) {
	return l.wantList.Contains(url)
}

func (l *ledger) ExchangeCount() uint64 {
	return l.exchangeCount
}

/*
func (l *ledger) AddBid(url string, cid cid.Cid ) {
	l.bidList.Add(url, cid)
}

func(l *ledger) GetBids() (*bidlist.Bidlist, error) {
	return l.bidList, nil
}
*/