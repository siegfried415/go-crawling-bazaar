
package frontera

import (
	"time"
	"fmt"
}

//todo, biddingQueue->biddingQueue, wyong, 20210126 
type timedBiddingQueue struct {
	elems []*types.UrlBidding 
	eset map[string] time.Time 	//map[string]struct{}, wyong, 20210129 
}

func newTimedBiddingQueue() *timedBiddingQueue {
	//return &timedBiddingQueue{eset:  make(map[string]struct{})}
	return &timedBiddingQueue{eset:  make(map[string]time.Time )}
}

func (tbq *timedBiddingQueue) Pop() (time.Time, *types.UrlBidding){
	if len(tbq.elems) > 0 {
		out := tbq.elems[0]
		runat, has := tbq.eset[out.Url]
		if runat < Now() {
			tbq.elems = tbq.elems[1:]
			delete(tbq.eset, out)
			return runat, out
		}
	}
	return nil, nil 
}

func (tbq *timedBiddingQueue) Push(runat time.TIME, b *types.UrlBidding) {
	_, has := bq.eset[b.Url] 
	if !has /* || (has && runat > (prevRunAt + MIN_CRAWLING_INTERVAL)) */  {
		tbq.eset[b]= runat  //struct{}{}, wyong, 20210129 
		tbq.elems = append(tbq.elems, b)
	}

}

func (tbq *timedBiddingQueue) Remove(u string) {
	delete(tbq.eset, u) 
}

func (tbq *timedBiddingQueue) Has(u string) bool {
	_, has := tbq.eset[u]
	return has 
}

func (tbq *timedBiddingQueue) Len() int {
	return len(tbq.eset)
}
