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

package frontera

import (
	"sync"
	"context"

	//"github.com/pkg/errors"

	"github.com/siegfried415/gdf-rebuild/kayak"
	kt "github.com/siegfried415/gdf-rebuild/kayak/types"
	"github.com/siegfried415/gdf-rebuild/proto"
	//rpc "github.com/siegfried415/gdf-rebuild/rpc/mux"
	net "github.com/siegfried415/gdf-rebuild/net"

	//wyong, 20201020
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/host"

	//wyong, 20200928 
	//"github.com/ugorji/go/codec"
)

const (
	// DomainKayakApplyMethodName defines the database kayak apply rpc method name.
	DomainKayakApplyMethodName = "Apply"
	// DomainKayakFetchMethodName defines the database kayak fetch rpc method name.
	DomainKayakFetchMethodName = "Fetch"

	//wyong, 20200925
	ProtocolKayakApply	= "/gdf/kayak/apply" 
	ProtocolKayakFetch	= "/gdf/kayak/fetch" 
)

// DomainKayakMuxService defines a mux service for sqlchain kayak.
type DomainKayakMuxService struct {
	serviceName string
	serviceMap  sync.Map
}

// NewDomainKayakMuxService returns a new kayak mux service.
func NewDomainKayakMuxService(serviceName string, /* server *rpc.Server */ h host.Host ) (s *DomainKayakMuxService, err error) {
	s = &DomainKayakMuxService{
		serviceName: serviceName,
	}

	//wyong, 20200925 
	//err = server.RegisterService(serviceName, s)
	h.SetStreamHandler(ProtocolKayakApply, s.ApplyHandler)
	h.SetStreamHandler(ProtocolKayakFetch, s.FetchHandler) 

	return
}

func (s *DomainKayakMuxService) register(id proto.DomainID, rt *kayak.Runtime) {
	s.serviceMap.Store(id, rt)

}

func (s *DomainKayakMuxService) unregister(id proto.DomainID) {
	s.serviceMap.Delete(id)
}

//wyong, 20201020 
// Apply handles kayak apply call.
//func (s *DomainKayakMuxService) ApplyHandler(req *kt.ApplyRequest,  _ *interface{}) (err error) {
func (ks *DomainKayakMuxService) ApplyHandler(s network.Stream ) {
	//decode req with Messagepack, wyong, 20200928 
	//var decReq kt.ApplyRequest 
        //dec := codec.NewDecoderBytes(req, new(codec.MsgpackHandle))
        //if err := dec.Decode(&decReq); err != nil {
        //        return err
        //}
        ctx := context.Background()
        var req kt.ApplyRequest  
        err := net.RecvMsg(ctx, s, &req)
        if err != nil {
                return
        }

	// call apply to specified kayak
	// treat req.Instance as DatabaseID
	id := proto.DomainID(req.Instance)

	v, ok := ks.serviceMap.Load(id)
	if ! ok {
		//return errors.Wrapf(ErrUnknownMuxRequest, "instance %v", decReq.Instance)
		return 
	}

	v.(*kayak.Runtime).FollowerApply(req.Log)
}

// Fetch handles kayak fetch call.
//func (s *DomainKayakMuxService) FetchHandler(req *kt.FetchRequest, resp *kt.FetchResponse) (err error) {
func (ks *DomainKayakMuxService) FetchHandler(s network.Stream) {
        ctx := context.Background()
        var req kt.FetchRequest  
        err := net.RecvMsg(ctx, s, &req)
        if err != nil {
                return
        }

	id := proto.DomainID(req.Instance)
	v, ok := ks.serviceMap.Load(id)
	if ! ok {
		//return errors.Wrapf(ErrUnknownMuxRequest, "instance %v", req.Instance)
		return
	}	
	
	var l *kt.Log
	l, err = v.(*kayak.Runtime).Fetch(req.GetContext(), req.Index)
	if err != nil {
		return	
	}

	resp := kt.FetchResponse {
		Log : l,
		Instance : req.Instance ,
	}

        net.SendMsg(ctx, s, &resp)
	//return

}
