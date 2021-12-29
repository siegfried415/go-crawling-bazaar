package frontera

import (
	"context"

	process "github.com/jbenet/goprocess"
	log "github.com/siegfried415/go-crawling-bazaar/utils/log"
)

var TaskWorkerCount = 8

func (f *Frontera) startWorkers(px process.Process, ctx context.Context) {
	// Start up workers to handle requests from other nodes for the data on this node
	for i := 0; i < TaskWorkerCount; i++ {
		i := i
		px.Go(func(px process.Process) {
			f.taskWorker(ctx, i)
		})
	}
}

func (f *Frontera) taskWorker(ctx context.Context, id int) {
	defer log.Debug("bitswap task worker shutting down...")
	for {
		//log.Event(ctx, "Bitswap.TaskWorker.Loop", idmap)
		select {
		case bidMsg, ok := <-f.bs.Outbox():

			if !ok {
				continue
			}

			f.bc.SendBids(ctx, bidMsg )

		case <-ctx.Done():
			return
		}
	}
}
