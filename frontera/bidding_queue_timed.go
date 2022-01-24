/*
 * Copyright 2022 https://github.com/siegfried415
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */


package frontera

import (
	"errors" 
	"time"

        "github.com/siegfried415/go-crawling-bazaar/types"
)

//todo, biddingQueue->biddingQueue
type TimedBiddingQueue struct {
	elems []types.UrlBidding 
	eset map[string] time.Time 	
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
	if !has {
		tbq.eset[b.Url]= runat  
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
