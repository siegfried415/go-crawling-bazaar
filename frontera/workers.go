package frontera

import (
	"context"
	//"math/rand"
	//"sync"
	//"time"
	//"fmt"

	//bsmsg "github.com/siegfried415/gdf-rebuild/frontera/message"

	//cid "github.com/ipfs/go-cid"
	process "github.com/jbenet/goprocess"
	//procctx "github.com/ipfs/goprocess/context"
	//peer "github.com/libp2p/go-libp2p-peer"
	//logging "github.com/ipfs/go-log"

	//wyong, 20201215
	log "github.com/siegfried415/gdf-rebuild/utils/log"
)

var TaskWorkerCount = 8

func (f *Frontera) startWorkers(px process.Process, ctx context.Context) {
	/*TODO, wyong, 20181220
	// Start up a worker to handle block requests this node is making
	px.Go(func(px process.Process) {
		bs.providerQueryManager(ctx)
	})
	*/

	// Start up workers to handle requests from other nodes for the data on this node
	for i := 0; i < TaskWorkerCount; i++ {
		i := i
		px.Go(func(px process.Process) {
			f.taskWorker(ctx, i)
		})
	}

	/*TODO, wyong, 20181220
	// Start up a worker to manage periodically resending our wantlist out to peers
	px.Go(func(px process.Process) {
		bs.rebroadcastWorker(ctx)
	})

	// Start up a worker to manage sending out provides messages
	px.Go(func(px process.Process) {
		bs.provideCollector(ctx)
	})

	// Spawn up multiple workers to handle incoming blocks
	// consider increasing number if providing blocks bottlenecks
	// file transfers
	px.Go(bs.provideWorker)
	*/
}

func (f *Frontera) taskWorker(ctx context.Context, id int) {
	//idmap := logging.LoggableMap{"ID": id}
	defer log.Debug("bitswap task worker shutting down...")
	for {
		//log.Event(ctx, "Bitswap.TaskWorker.Loop", idmap)
		log.Debugf("Frontera/taskWorker(10), enter loop\n") 
		select {
		case bidMsg, ok := <-f.engine.Outbox():
			log.Debugf("Frontera/taksWorker(20), f.engine.Outbix fired!\n")

			//select {
			//case envelope, ok := <-nextEnvelope:
				//log.Debugf("taksWorker, nextEnvelope fired!")
				if !ok {
					log.Debugf("Frontera/taksWorker(25)\n")
					continue
				}

				/*TODO,need review the following code,  wyong, 20190115 
				// update the BS ledger to reflect sent message
				// TODO: Should only track *useful* messages in ledger
				outgoing := bsmsg.New(false)
				for _, bid := range envelope.Message.Bids() {
					log.Event(ctx, "Bitswap.TaskWorker.Work", logging.LoggableF(func() map[string]interface{} {
						return logging.LoggableMap{
							"ID":     id,
							"Target": envelope.Peer.Pretty(),
							"Url": bid.BidEntry.Url,
							"Bid":  bid.BidEntry.Cid.String(),
						}
					}))
					outgoing.AddBidEntry(&bid)
				}

				//TODO,wyong, 20181220
				//bs.engine.MessageSent(envelope.Peer, outgoing)
				*/

				log.Debugf("Frontera/taksWorker(30)\n")
				f.wm.SendBids(ctx, bidMsg )
				log.Debugf("Frontera/taksWorker(40)\n")

				/*
				//bs.counterLk.Lock()
				//for _, block := range envelope.Message.Blocks() {
				//	bs.counters.blocksSent++
				//	bs.counters.dataSent += uint64(len(block.RawData()))
				//}
				//bs.counterLk.Unlock()
				*/

			//case <-ctx.Done():
            //    log.Debugf("taksWorker, ctx.Done fired!")
			//	return
			//}
		case <-ctx.Done():
			log.Debugf("Frontera/taksWorker(50), ctx.Done fired!")
			return
		}
	}
}

/* wyong, 20181224
func (bs *Biddingsys) provideWorker(px process.Process) {

	limit := make(chan struct{}, provideWorkerMax)

	limitedGoProvide := func(k cid.Cid, wid int) {
                log.Debugf("limitedGoProvide called !")
		defer func() {
			// replace token when done
			<-limit
		}()
		ev := logging.LoggableMap{"ID": wid}

		ctx := procctx.OnClosingContext(px) // derive ctx from px
		defer log.EventBegin(ctx, "Bitswap.ProvideWorker.Work", ev, k).Done()

		ctx, cancel := context.WithTimeout(ctx, provideTimeout) // timeout ctx
		defer cancel()

		if err := bs.network.Provide(ctx, k); err != nil {
			log.Warning(err)
		}
	}

	// worker spawner, reads from bs.provideKeys until it closes, spawning a
	// _ratelimited_ number of workers to handle each key.
	for wid := 2; ; wid++ {
		ev := logging.LoggableMap{"ID": 1}
		log.Event(procctx.OnClosingContext(px), "Bitswap.ProvideWorker.Loop", ev)

		select {
		case <-px.Closing():	
                        log.Debugf("provideWorker, px.Closing() fired!")
			return
		case k, ok := <-bs.provideKeys:
                        log.Debugf("provideWorker, bs.provideKeys fired!")
			if !ok {
				log.Debug("provideKeys channel closed")
				return
			}
			select {
			case <-px.Closing():
                        	log.Debugf("provideWorker, px.Closing() fired!")
				return
			case limit <- struct{}{}:
                        	log.Debugf("provideWorker, struct{}{} fired!")
				go limitedGoProvide(k, wid)
			}
		}
	}
}
*/

/* wyong, 20181224
func (bs *Biddingsys) provideCollector(ctx context.Context) {
	defer close(bs.provideKeys)
	var toProvide []cid.Cid
	var nextKey cid.Cid
	var keysOut chan cid.Cid

	for {
		select {
		case blkey, ok := <-bs.newBlocks:
                        log.Debugf("provideCollector, bs.newBlocks fired!")
			if !ok {
				log.Debug("newBlocks channel closed")
				return
			}

			if keysOut == nil {
				nextKey = blkey
				keysOut = bs.provideKeys
			} else {
				toProvide = append(toProvide, blkey)
			}
		case keysOut <- nextKey:
                        log.Debugf("provideCollector, nextKey fired!")
			if len(toProvide) > 0 {
				nextKey = toProvide[0]
				toProvide = toProvide[1:]
			} else {
				keysOut = nil
			}
		case <-ctx.Done():
                        log.Debugf("provideCollector, ctx.Donw fired!")
			return
		}
	}
}
*/

/* wyong, 20181224
func (bs *Biddingsys) rebroadcastWorker(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	broadcastSignal := time.NewTicker(rebroadcastDelay.Get())
	defer broadcastSignal.Stop()

	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()

	for {
		log.Event(ctx, "Bitswap.Rebroadcast.idle")
		select {
		case <-tick.C:
                        log.Debugf("rebroadcastWorker, tick.C fired!")
			n := bs.wm.wl.Len()
			if n > 0 {
				log.Debug(n, " keys in bitswap wantlist")
			}
		case <-broadcastSignal.C: // resend unfulfilled wantlist keys
                        log.Debugf("rebroadcastWorker, broadcastSignal.C fired!")
			log.Event(ctx, "Bitswap.Rebroadcast.active")
			entries := bs.wm.wl.Entries()
			if len(entries) == 0 {
				continue
			}

			// TODO: come up with a better strategy for determining when to search
			// for new providers for blocks.
			i := rand.Intn(len(entries))
			bs.findKeys <- &biddingRequest{
				Cid: entries[i].Cid,
				Ctx: ctx,
			}
		case <-parent.Done():
                        log.Debugf("rebroadcastWorker, parent.Donw() fired!")
			return
		}
	}
}
*/

/* wyong, 20181224
func (bs *Biddingsys) providerQueryManager(ctx context.Context) {
	var activeLk sync.Mutex
	kset := cid.NewSet()

	for {
		select {
		case e := <-bs.findKeys:
                        log.Debugf("providerQueryManager, bs.findKeys fired!")
			select { // make sure its not already cancelled
			case <-e.Ctx.Done():
				continue
			default:
			}

			activeLk.Lock()
			if kset.Has(e.Cid) {
				activeLk.Unlock()
				continue
			}
			kset.Add(e.Cid)
			activeLk.Unlock()

			go func(e *biddingRequest) {
                        	log.Debugf("providerQueryManager, go func called!")
				child, cancel := context.WithTimeout(e.Ctx, providerRequestTimeout)
				defer cancel()
				providers := bs.network.FindProvidersAsync(child, e.Cid, maxProvidersPerRequest)
				wg := &sync.WaitGroup{}
				for p := range providers {
					wg.Add(1)
					go func(p peer.ID) {
						defer wg.Done()
						err := bs.network.ConnectTo(child, p)
						if err != nil {
							log.Debug("failed to connect to provider %s: %s", p, err)
						}
					}(p)
				}
				wg.Wait()
				activeLk.Lock()
				kset.Remove(e.Cid)
				activeLk.Unlock()
			}(e)

		case <-ctx.Done():
                        log.Debugf("providerQueryManager, ctx.Donw() fired!")
			return
		}
	}
}
*/
