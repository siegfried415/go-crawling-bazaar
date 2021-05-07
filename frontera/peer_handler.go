
package frontera

import (
	"context"
	"sync"
	"time"
	//"fmt"

	//engine "github.com/siegfried415/go-crawling-bazaar/frontera/decision"

	//wyong, 20200827 
	//bsmsg "github.com/siegfried415/go-crawling-bazaar/frontera/message"

	//fnet "github.com/siegfried415/go-crawling-bazaar/frontera/network"
	//biddinglist "github.com/siegfried415/go-crawling-bazaar/frontera/biddinglist"

	//cid "github.com/ipfs/go-cid"

	//wyong, 20200925
        //pstore "github.com/libp2p/go-libp2p-peerstore"
        //host "github.com/libp2p/go-libp2p-core/host"

	//go-libp2p-net -> go-libp2p-core/network,
	//wyong, 20201029 
	//inet "github.com/libp2p/go-libp2p-core/network"
	libp2phelpers "github.com/libp2p/go-libp2p-core/helpers"

	protocol "github.com/libp2p/go-libp2p-core/protocol" 
	//peer "github.com/libp2p/go-libp2p-core/peer" 


	//peer "github.com/libp2p/go-libp2p-peer"
	//metrics "github/ipfs/go-metrics-interface"

	//wyong, 20200730 
        //"github.com/siegfried415/go-crawling-bazaar/route"
        "github.com/siegfried415/go-crawling-bazaar/types"
        //"github.com/siegfried415/go-crawling-bazaar/rpc"
        //"github.com/siegfried415/go-crawling-bazaar/rpc/mux"
        "github.com/siegfried415/go-crawling-bazaar/proto"
	net "github.com/siegfried415/go-crawling-bazaar/net" 

	//wyong, 20200928
	//"github.com/ugorji/go/codec"

	//wyong, 20201215 
	log "github.com/siegfried415/go-crawling-bazaar/utils/log" 

        //wyong, 20210118
        //ecvrf "github.com/vechain/go-ecvrf"

)

type msgQueue struct {
	p proto.NodeID

	outlk   *sync.Mutex
	out     types.UrlBiddingMessage		//bsmsg.BiddingMessage

	//wyong, 20200925 
	//network fnet.BiddingNetwork
	host 	net.RoutedHost 
	
	bl      *ThreadSafe

	//wyong, 20200924 
	//sender fnet.MessageSender
        //caller rpc.PCaller
	sender *net.Stream 

	refcnt int

	work chan struct{}
	done chan struct{}
}


func (mq *msgQueue) runQueue(ctx context.Context) {
	//log.Debugf("runQueue called...")
	for {
		select {
		case <-mq.work: // there is work to be done
			//log.Debugf("runQueue, <-mq.work")
			mq.doWork(ctx)
		case <-mq.done:
			//log.Debugf("runQueue, <-mq.done")
			if mq.sender != nil  {
				//mq.sender.Close()

				//wyong, 20201029 
				//inet.FullClose(mq.sender) 
				libp2phelpers.FullClose(mq.sender) 
			}
			return
		case <-ctx.Done():
			//log.Debugf("runQueue, <-ctx.Done")
			if mq.sender != nil {
				mq.sender.Reset()
			}
			return
		}
	}
}

func (mq *msgQueue) doWork(ctx context.Context) {
	log.Debugf("msgQueue/doWork(10)\n")
	// grab outgoing message
	mq.outlk.Lock()
	blm := mq.out
	if /* blm == nil ||  wyong, 20200827 */  blm.Empty() { 
		mq.outlk.Unlock()
		return
	}
	mq.out = types.UrlBiddingMessage{} //nil, wyong, 20200827 
	mq.outlk.Unlock()

	log.Debugf("msgQueue/doWork(20)\n")

	// NB: only open a stream if we actually have data to send
	if mq.sender == nil {
		log.Debugf("msgQueue/doWork(30)\n")
		err := mq.openSender(ctx)
		if err != nil {
			log.Debugf("msgQueue/doWork(35)\n")
			log.Debugf("cant open message sender to peer %s: %s", mq.p, err)
			// TODO: cant connect, what now?
			return
		}
		log.Debugf("msgQueue/doWork(40)\n")
	}
	
	log.Debugf("msgQueue/doWork(50)\n")
	// send biddinglist updates
	for { // try to send this message until we fail.
		log.Debugf("msgQueue/doWork(60)\n")
		
		//wyong, 20200924 
		//err := mq.sender.SendMsg(ctx, blm)
        	//var response types.Response
		//err := mq.caller.Call(route.FronteraBidding.String(), blm, &response ) 
		_, err := (mq.sender).SendMsg(ctx, blm)
		if err == nil {
			log.Debugf("msgQueue/doWork(70)\n")
			return
		}

		// todo, wyong, 20200730 
		//log.Infof("bitswap send error: %s", err)
		mq.sender.Reset()
		mq.sender = nil

		log.Debugf("msgQueue/doWork(80)\n")

		select {
		case <-mq.done:
			log.Debugf("msgQueue/doWork(90)\n")
			return
		case <-ctx.Done():
			log.Debugf("msgQueue/doWork(100)\n")
			return
		case <-time.After(time.Millisecond * 100):
			log.Debugf("msgQueue/doWork(110)\n")
			// wait 100ms in case disconnect notifications are still propogating
			//log.Warning("SendMsg errored but neither 'done' nor context.Done() were set")
		}

		log.Debugf("msgQueue/doWork(120)\n")

		err = mq.openSender(ctx)
		if err != nil {
			log.Debugf("msgQueue/doWork(125)")
			log.Debugf("couldnt open sender again after SendMsg(%s) failed: %s\n", mq.p, err)
			// TODO(why): what do we do now?
			// I think the *right* answer is to probably put the message we're
			// trying to send back, and then return to waiting for new work or
			// a disconnect.
			return
		}

		log.Debugf("msgQueue/doWork(130)\n")

		// TODO: Is this the same instance for the remote peer?
		// If its not, we should resend our entire biddinglist to them
		//if mq.sender.InstanceID() != mq.lastSeenInstanceID {
		//	blm = mq.getFullWantlistMessage()
		//}
	}
}

// wyong, 20200924 
func (mq *msgQueue) openSender(ctx context.Context ) error {
	//log.Debugf("openSender called...")
	// allow ten minutes for connections this includes looking them up in the
	// dht dialing them, and handshaking
	//conctx, cancel := context.WithTimeout(ctx, time.Minute*10)
	//defer cancel()

	//log.Debugf("openSender(10)")
	log.Debugf("openSender(10)\n")
	//err := mq.network.ConnectTo(conctx, mq.p )

	//wyong, 20201008 
	//err := mq.host.Connect(ctx, pstore.PeerInfo{ID: peer.ID(mq.p)})
	//if err != nil {
	//	log.Debugf("openSender(15)\n")
	//	return err
	//}

	//log.Debugf("openSender(20)")
	log.Debugf("openSender(20)\n")
	//todo, "go-crawling-bazaar/frontera/bidding",  wyong, 20200924 
	//nsender, err := mq.network.NewMessageSender(ctx, mq.p)
	s, err := mq.host.NewStreamExt(ctx, mq.p, protocol.ID("FRT.Bidding"))
	if err != nil {
		log.Debugf("openSender(25)\n")
		return err
	}

	//log.Debugf("openSender(30)")
	log.Debugf("openSender(30)\n")
	mq.sender = &s 

	//if cfg.UseDirectRPC {
	//	caller = rpc.NewPersistentCaller(mq.p)
	//} else {
	//	caller = mux.NewPersistentCaller(mq.p)
	//}
	//caller := mux.NewPersistentCaller(mq.p)
	//
	//mq.caller = caller 

	return nil
}

func (mq *msgQueue) addBiddingMessage( biddings []types.UrlBidding,  domain proto.DomainID, from proto.NodeID  ) {
	log.Debugf("msgQueue/addBiddingMessage(10)\n")
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

	log.Debugf("msgQueue/addBiddingMessage (20)\n")
	// if we have no message held allocate a new one
	//if mq.out == nil {
	//	//wyong, 20200827 
	//	mq.out = bsmsg.New(false, string(from))
	//}

	//build bidding message, wyong, 20200827 
	mq.out = types.UrlBiddingMessage{
		Header: types.SignedUrlBiddingHeader{
			UrlBiddingHeader: types.UrlBiddingHeader{
				QueryType:    types.WriteQuery,
				NodeID:       from,
				DomainID:     domain,

				//todo, wyong, 20200817
				//ConnectionID: connID,
				//SeqNo:        seqNo,
				//Timestamp:    getLocalTime(),
			},
		},
		Payload: types.UrlBiddingPayload{
			Requests: biddings,
		},
	}


	log.Debugf("msgQueue/addBiddingMessage (30)\n")

	/* 
	// TODO: add a msg.Combine(...) method
	// otherwise, combine the one we are holding with the
	// one passed in
	for _, e := range biddings {
		log.Debugf("msgQueue/addBiddingMessage (40)\n")
		if e.Cancel {
			log.Debugf("msgQueue/addBiddingMessage(50)\n")
			if mq.bl.Remove(e.Url, domain ) {
				log.Debugf("msgQueue/addBiddingMessage(60)")
				work = true
				mq.out.Cancel(e.Url)
			}
		} else {
			log.Debugf("msgQueue/addBiddingMessage (70)\n")
			if mq.bl.Add(e.Url, e.Priority, domain ) {
				log.Debugf("msgQueue/addBiddingMessage (80)\n")
				work = true

				//TODO, review if necessary send cids , wyong, 20190119 
				mq.out.AddBidding(e.Url, e.Priority, nil )
			}
		}
	}
	*/

}
