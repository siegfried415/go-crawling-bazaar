
package frontera

import (
	"errors" 
	"time"
	//"fmt"

        "github.com/siegfried415/gdf-rebuild/types"
)

//todo, biddingQueue->biddingQueue, wyong, 20210126 
type TimedBiddingQueue struct {
	elems []types.UrlBidding 
	eset map[string] time.Time 	//map[string]struct{}, wyong, 20210129 
}

func newTimedBiddingQueue() *TimedBiddingQueue {
	return &TimedBiddingQueue{eset:  make(map[string]time.Time )}
}

func (tbq *TimedBiddingQueue) Pop() (time.Time, types.UrlBidding, error ){
	if len(tbq.elems) > 0 {
		out := tbq.elems[0]
		runat, _ := tbq.eset[out.Url]
		if runat.Before(time.Now()) {
			tbq.elems = tbq.elems[1:]
			delete(tbq.eset, out.Url)
			return runat, out, nil 
		}
	}

	return time.Time{}, 
		types.UrlBidding{}, 
	errors.New("no bidding")

}

func (tbq *TimedBiddingQueue) Push(runat time.Time, b types.UrlBidding) {
	_, has := tbq.eset[b.Url] 
	if !has /* || (has && runat > (prevRunAt + MIN_CRAWLING_INTERVAL)) */  {
		tbq.eset[b.Url]= runat  //struct{}{}, wyong, 20210129 
		tbq.elems = append(tbq.elems, b)
	}

}

func (tbq *TimedBiddingQueue) Remove(u string) {
	delete(tbq.eset, u) 
}

func (tbq *TimedBiddingQueue) Has(u string) bool {
	_, has := tbq.eset[u]
	return has 
}

func (tbq *TimedBiddingQueue) Len() int {
	return len(tbq.eset)
}
