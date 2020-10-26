/*
 * Copyright (c) 2018 Filecoin Project
 * Copyright (c) 2022 https://github.com/siegfried415
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
	"context"
	"sync"
	"time"

	libp2phelpers "github.com/libp2p/go-libp2p-core/helpers"
	protocol "github.com/libp2p/go-libp2p-core/protocol" 

	log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
	net "github.com/siegfried415/go-crawling-bazaar/net" 
        "github.com/siegfried415/go-crawling-bazaar/proto"
        "github.com/siegfried415/go-crawling-bazaar/types"
)

type msgQueue struct {
	p proto.NodeID

	outlk   *sync.Mutex
	out     types.UrlBiddingMessage		//bsmsg.BiddingMessage

	host 	net.RoutedHost 
	
	bl      *ThreadSafe
	sender *net.Stream 

	refcnt int

	work chan struct{}
	done chan struct{}
}


func (mq *msgQueue) runQueue(ctx context.Context) {
	for {
		select {
		case <-mq.work: // there is work to be done
			mq.doWork(ctx)
		case <-mq.done:
			if mq.sender != nil  {
				libp2phelpers.FullClose(mq.sender) 
			}
			return
		case <-ctx.Done():
			if mq.sender != nil {
				mq.sender.Reset()
			}
			return
		}
	}
}

func (mq *msgQueue) doWork(ctx context.Context) {
	// grab outgoing message
	mq.outlk.Lock()
	blm := mq.out
	if blm.Empty() { 
		mq.outlk.Unlock()
		return
	}
	mq.out = types.UrlBiddingMessage{} 
	mq.outlk.Unlock()


	// NB: only open a stream if we actually have data to send
	if mq.sender == nil {
		err := mq.openSender(ctx)
		if err != nil {
			log.Debugf("cant open message sender to peer %s: %s", mq.p, err)
			// TODO: cant connect, what now?
			return
		}
	}
	
	// send biddinglist updates
	for { // try to send this message until we fail.
		
		_, err := (mq.sender).SendMsg(ctx, blm)
		if err == nil {
			return
		}

		mq.sender.Reset()
		mq.sender = nil

		select {
		case <-mq.done:
			return
		case <-ctx.Done():
			return
		case <-time.After(time.Millisecond * 100):
			// wait 100ms in case disconnect notifications are still propogating
			//log.Warning("SendMsg errored but neither 'done' nor context.Done() were set")
		}

		err = mq.openSender(ctx)
		if err != nil {
			log.Debugf("couldnt open sender again after SendMsg(%s) failed: %s\n", mq.p, err)
			// TODO(why): what do we do now?
			// I think the *right* answer is to probably put the message we're
			// trying to send back, and then return to waiting for new work or
			// a disconnect.
			return
		}

		// TODO: Is this the same instance for the remote peer?
		// If its not, we should resend our entire biddinglist to them
		//if mq.sender.InstanceID() != mq.lastSeenInstanceID {
		//	blm = mq.getFullWantlistMessage()
		//}
	}
}

func (mq *msgQueue) openSender(ctx context.Context ) error {
	// allow ten minutes for connections this includes looking them up in the
	// dht dialing them, and handshaking
	//conctx, cancel := context.WithTimeout(ctx, time.Minute*10)
	//defer cancel()

	s, err := mq.host.NewStreamExt(ctx, mq.p, protocol.ID("FRT.Bidding"))
	if err != nil {
		return err
	}

	mq.sender = &s 
	return nil
}

func (mq *msgQueue) addBiddingMessage( biddings []types.UrlBidding,  domain proto.DomainID, from proto.NodeID  ) {
	var work bool
	mq.outlk.Lock()
	defer func() {
		mq.outlk.Unlock()
		if !work {
			return
		}
		select {
		case mq.work <- struct{}{}:
		default:
		}
	}()

	//build bidding message
	mq.out = types.UrlBiddingMessage{
		Header: types.SignedUrlBiddingHeader{
			UrlBiddingHeader: types.UrlBiddingHeader{
				QueryType:    types.WriteQuery,
				NodeID:       from,
				DomainID:     domain,
			},
		},
		Payload: types.UrlBiddingPayload{
			Requests: biddings,
		},
	}

}
