/*
 * Copyright 2018 The CovenantSQL Authors.
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

package kayak

import (
	"context"
	"sync"
	"sync/atomic"

	//wyong, 20201018 
	"github.com/pkg/errors"
	"github.com/libp2p/go-libp2p-core/peer" 
	"github.com/libp2p/go-libp2p-core/protocol" 

	kt "github.com/siegfried415/gdf-rebuild/kayak/types"
	"github.com/siegfried415/gdf-rebuild/proto"

	//wyong, 20201008
	//rpc "github.com/siegfried415/gdf-rebuild/rpc/mux"
	net "github.com/siegfried415/gdf-rebuild/net"

	"github.com/siegfried415/gdf-rebuild/utils/trace"

        //wyong, 20200928
        //"github.com/ugorji/go/codec"

)

// rpcTracker defines the rpc call tracker
// support tracking the rpc result.
type rpcTracker struct {
	// related runtime
	r *Runtime
	// target nodes, a copy of current followers
	nodes []proto.NodeID
	// rpc method
	method string
	// rpc request
	req interface{}
	// minimum response count
	minCount int
	// responses
	errLock sync.RWMutex
	errors  map[proto.NodeID]error
	// scoreboard
	complete int
	sent     uint32
	doneOnce sync.Once
	doneCh   chan struct{}
	wg       sync.WaitGroup
	closed   uint32
}

func newTracker(r *Runtime, req interface{}, minCount int) (t *rpcTracker) {
	// copy nodes
	nodes := append([]proto.NodeID(nil), r.followers...)

	if minCount > len(nodes) {
		minCount = len(nodes)
	}
	if minCount < 0 {
		minCount = 0
	}

	t = &rpcTracker{
		r:        r,
		nodes:    nodes,
		method:   r.applyRPCMethod,
		req:      req,
		minCount: minCount,
		errors:   make(map[proto.NodeID]error, len(nodes)),
		doneCh:   make(chan struct{}),
	}

	return
}

func (t *rpcTracker) send() {
	if !atomic.CompareAndSwapUint32(&t.sent, 0, 1) {
		return
	}

	for i := range t.nodes {
		t.wg.Add(1)
		go t.callSingle(i)
	}

	if t.minCount == 0 {
		t.done()
	}
}

func (t *rpcTracker) callSingle(idx int) {

	// wyong, 20200925 
	//caller := t.r.TrackerNewCallerFunc(t.nodes[idx])
	//if pcaller, ok := caller.(*rpc.PersistentCaller); ok && pcaller != nil {
	//	defer pcaller.Close()
	//}
	//err := caller.Call(t.method, t.req, nil)

	//wyong, 20201018 
	ctx := context.Background()

	//todo, ProtoKayakApply = "/gdf/kayak/apply", wyong, 20200925 
	s, err := t.r.host.NewStream(ctx, peer.ID(t.nodes[idx]), protocol.ID("ProtoKayakApply"))
	if err != nil {
		//log.Debugf("error opening push stream to %s: %s", p, err.Error())
		return
	}

	rch := make(chan struct{}, 1)
	go func() {
		//payloadWriter(s)

		//todo, wyong, 20200925 
		//defer helpers.FullClose(s)

		//wyong, 20200928
                //var encReq []byte
                //enc := codec.NewEncoderBytes(&encReq, new(codec.MsgpackHandle))
                //if err := enc.Encode(&t.req); err != nil {
                //        log.Debugf("error: %s", err)
                //        return 
                //}

	        //w := bufio.NewWriter(s)
                //if _, err := w.Write(encReq); err != nil {
                //        log.Debugf("error: %s", err)
                //        return 
                //}

		//wyong, 20201008 
		_, err := s.(net.Stream).SendMsg(ctx, &t.req )
		if err != nil {
			err = errors.Wrap(err, "send DHT.Ping failed")
			return 
		}

		rch <- struct{}{}
	}()

	select {
	case <-rch:
	case <-ctx.Done():
		// this is taking too long, abort!
		s.Reset()
	}


	defer t.wg.Done()
	t.errLock.Lock()
	defer t.errLock.Unlock()
	t.errors[t.nodes[idx]] = err
	t.complete++

	if t.complete >= t.minCount {
		t.done()
	}
}

func (t *rpcTracker) done() {
	t.doneOnce.Do(func() {
		if t.doneCh != nil {
			select {
			case <-t.doneCh:
			default:
				close(t.doneCh)
			}
		}
	})
}

func (t *rpcTracker) get(ctx context.Context) (errors map[proto.NodeID]error, meets bool, finished bool) {
	if trace.IsEnabled() {
		// get request log type
		traceType := "rpcCall"

		if rawReq, ok := t.req.(*kt.ApplyRequest); ok {
			traceType += rawReq.Log.Type.String()
		}

		defer trace.StartRegion(ctx, traceType).End()
	}

	for {
		select {
		case <-t.doneCh:
			meets = true
		default:
		}

		select {
		case <-ctx.Done():
		case <-t.doneCh:
			meets = true
		}

		break
	}

	t.errLock.RLock()
	defer t.errLock.RUnlock()

	errors = make(map[proto.NodeID]error)

	for s, e := range t.errors {
		errors[s] = e
	}

	if !meets && len(errors) >= t.minCount {
		meets = true
	}

	if len(errors) == len(t.nodes) {
		finished = true
	}

	return
}

func (t *rpcTracker) close() {
	if !atomic.CompareAndSwapUint32(&t.closed, 0, 1) {
		return
	}

	t.wg.Wait()
	t.done()
}
