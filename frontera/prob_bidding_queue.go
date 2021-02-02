
package frontera

import (
	"time"
	"fmt"
}

//todo, biddingQueue->probBiddingQueue, wyong, 20210126 
type probBiddingQueue struct {
	elems []*types.UrlBidding 
	//eset  *cid.Set
	eset map[string]struct{} 
}

func newProbBiddingQueue() *probBiddingQueue {
	//return &cidQueue{eset: cid.NewSet()}
	return &probBiddingQueue{eset:  make(map[string]struct{})}
}

func (pbq *probBiddingQueue) Pop() *types.UrlBidding{
	for {
		if len(pbq.elems) == 0 {
			return nil 
		}

		out := pbq.elems[0]
		pbq.elems = pbq.elems[1:]

		_, has := pbq.eset[out.Url]
		if has {
			delete(pbq.eset, out)
			return out
		}
	}
}

//todo, Insert biddings into the queue accroding to probability. 
func (pbq *probBiddingQueue) Insert(bs []*types.UrlBidding) {
	return pbq.insert(bidding, 0 )
}

//todo, Insert biddings into the queue accroding to probability. 
func (pbq *probBiddingQueue) insert(bidding *types.UrlBidding, from int ) {
        rear1 :=  []*types.UrlBidding{}
	if pbq.Has(bidding.Url) { 
		for i, b := range pbq.elems {
			if b.Url == bidding.Url { 
				if bidding.Probability <= b.Probability {
					return 
				}else {
					if i < ( pbp.Len() - 1 ) {
						rear1 = pbq.elems[i+1:]
					}
					pbq.elems = append(pbq.elems[0:i], rear1...)
					delete(pbq.eset, u) 
					break 
				}
			}
		}
	}

	//put bidding into queue at some positon by score.
        rear2 :=  []*types.UrlBidding{}
	for i:= from; i < pbp.Len(); i++ ) { 
		if bidding.Probability > b.Probability { 
			if i < ( bp.Len() - 1 ) {
				rear2 = pbq.elems[i+1:]
			}
			break
		}
	}

	pbq.elems = append(pbq.elems[0:i], bidding )
	pbq.elems = append (pbq.elems, rear2...)
	pbq.eset[b]=struct{}{}

	//return the position, so we have chance to skip elems[0:i+1]
	return i + 1

}

//Insert biddings into the queue accroding to probability. 
func (pbq *probBiddingQueue) Inserts(biddings []*types.UrlBidding) {
	for _, bidding := range biddings {
		pbq.insert(bidding, 0 )
	}
}

//Insert biddings into the queue accroding to probability. 
func (pbq *probBiddingQueue) FastInserts(biddings []*types.UrlBidding) {

	//Sort urls with probability. wyong, 20210119 
	sort.Slice(biddings, func(i, j int) bool {
		return biddings[i].Probability < biddings[j].Probability 
	})

	pos := 0 
	for _, bidding := range biddings {
		pos = pbq.insert(bidding, pos )
	}
}

func (pbq *probBiddingQueue) Push(b *types.UrlBidding) {
	_, has := pbq.eset[b.Url] 
	if !has {
		//todo, get position by probability.  
		pbq.eset[b]=struct{}{}
		pbq.elems = append(pbq.elems, b)
	}

}

func (pbq *probBiddingQueue) Remove(u string) {
	delete(pbq.eset, u) 
}

func (pbq *probBiddingQueue) Has(u string) bool {
	_, has := pbq.eset[u]
	return has 
}

func (pbq *probBiddingQueue) Len() int {
	return len(pbq.eset)
}
