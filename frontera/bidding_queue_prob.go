
package frontera

import (
	"errors" 
	"sort" 

        "github.com/siegfried415/go-crawling-bazaar/types"
)

//todo, biddingQueue->probBiddingQueue
type ProbBiddingQueue struct {
	elems []types.UrlBidding 
	eset map[string]struct{} 
}

func newProbBiddingQueue() *ProbBiddingQueue {
	return &ProbBiddingQueue{eset:  make(map[string]struct{})}
}

func (pbq *ProbBiddingQueue) Pop() (types.UrlBidding, error) {
	for {
		if len(pbq.elems) == 0 {
			return types.UrlBidding{}, errors.New("no item can be poped")
		}

		out := pbq.elems[0]
		pbq.elems = pbq.elems[1:]

		_, has := pbq.eset[out.Url]
		if has {
			delete(pbq.eset, out.Url)
			return out, nil 
		}
	}
}

//todo, Insert biddings into the queue accroding to probability. 
func (pbq *ProbBiddingQueue) Insert(bidding types.UrlBidding) int {
	return pbq.insert(bidding, 0 )
}

//todo, Insert biddings into the queue accroding to probability. 
func (pbq *ProbBiddingQueue) insert(bidding types.UrlBidding, from int ) int {
        rear1 :=  []types.UrlBidding{}
	if pbq.Has(bidding.Url) { 
		for i, b := range pbq.elems {
			if b.Url == bidding.Url { 
				if bidding.Probability <= b.Probability {
					return i + 1 
				}else {
					if i < ( pbq.Len() - 1 ) {
						rear1 = pbq.elems[i+1:]
					}
					pbq.elems = append(pbq.elems[0:i], rear1...)
					delete(pbq.eset, b.Url) 
					break 
				}
			}
		}
	}

	//put bidding into queue at some positon by score.
        rear2 :=  []types.UrlBidding{}
	i := 0 
	for i= from; i < pbq.Len(); i++ { 
		b := pbq.elems[i] 
		if bidding.Probability > b.Probability { 
			if i < ( pbq.Len() - 1 ) {
				rear2 = pbq.elems[i+1:]
			}
			break
		}
	}

	pbq.elems = append(pbq.elems[0:i], bidding )
	pbq.elems = append (pbq.elems, rear2...)
	pbq.eset[bidding.Url]=struct{}{}

	//return the position, so we have chance to skip elems[0:i+1]
	return i + 1

}

//Insert biddings into the queue accroding to probability. 
func (pbq *ProbBiddingQueue) Inserts(biddings []types.UrlBidding) {
	for _, bidding := range biddings {
		pbq.insert(bidding, 0 )
	}
}

//Insert biddings into the queue accroding to probability. 
func (pbq *ProbBiddingQueue) FastInserts(biddings []types.UrlBidding) {

	//Sort urls with probability. 
	sort.Slice(biddings, func(i, j int) bool {
		return biddings[i].Probability < biddings[j].Probability 
	})

	pos := 0 
	for _, bidding := range biddings {
		pos = pbq.insert(bidding, pos )
	}
}

func (pbq *ProbBiddingQueue) Push(b types.UrlBidding) {
	_, has := pbq.eset[b.Url] 
	if !has {
		//todo, get position by probability.  
		pbq.eset[b.Url]=struct{}{}
		pbq.elems = append(pbq.elems, b)
	}

}

func (pbq *ProbBiddingQueue) Remove(u string) {
	delete(pbq.eset, u) 
}

func (pbq *ProbBiddingQueue) Has(u string) bool {
	_, has := pbq.eset[u]
	return has 
}

func (pbq *ProbBiddingQueue) Len() int {
	return len(pbq.eset)
}
