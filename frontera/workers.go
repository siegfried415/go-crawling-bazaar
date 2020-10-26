/*
 * Copyright (c) 2018 Filecoin Project
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
