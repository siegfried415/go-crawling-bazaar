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

package sqlchain

import (
	"sync"
	"context" 

	"github.com/libp2p/go-libp2p-core/protocol" 

	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/net"

)

// MuxService defines multiplexing service of sql-chain.
type MuxService struct {
	ServiceName string
	serviceMap  sync.Map
}

// NewMuxService creates a new multiplexing service and registers it to rpc server.
func NewMuxService(serviceName string, h net.RoutedHost ) (service *MuxService, err error) {
	service = &MuxService{
		ServiceName: serviceName,
	}

        h.SetStreamHandlerExt(protocol.ID("URLC.AdviseNewBlock"), service.AdviseNewBlockHandler)
        h.SetStreamHandlerExt(protocol.ID("URLC.FetchBlock"), service.FetchBlockHandler)

	return
}

func (s *MuxService) register(id proto.DomainID, service *ChainRPCService) {
	s.serviceMap.Store(id, service)
}

func (s *MuxService) unregister(id proto.DomainID) {
	s.serviceMap.Delete(id)
}

// MuxAdviseNewBlockReq defines a request of the AdviseNewBlock RPC method.
type MuxAdviseNewBlockReq struct {
	proto.Envelope
	proto.DomainID
	AdviseNewBlockReq
}

// MuxAdviseNewBlockResp defines a response of the AdviseNewBlock RPC method.
type MuxAdviseNewBlockResp struct {
	proto.Envelope
	proto.DomainID
	AdviseNewBlockResp
}

// MuxFetchBlockReq defines a request of the FetchBlock RPC method.
type MuxFetchBlockReq struct {
	proto.Envelope
	proto.DomainID
	FetchBlockReq
}

// MuxFetchBlockResp defines a response of the FetchBlock RPC method.
type MuxFetchBlockResp struct {
	proto.Envelope
	proto.DomainID
	FetchBlockResp
}

// AdviseNewBlock is the RPC method to advise a new produced block to the target server.
func (ms *MuxService) AdviseNewBlockHandler(s net.Stream ) {
        ctx := context.Background()
        var req MuxAdviseNewBlockReq 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	v, ok := ms.serviceMap.Load(req.DomainID)
	if !ok {
		//return ErrUnknownMuxRequest
		return 
	}

        var resp = MuxAdviseNewBlockResp{
		Envelope : req.Envelope, 
		DomainID : req.DomainID,
        }

	v.(*ChainRPCService).AdviseNewBlock(&req.AdviseNewBlockReq, &resp.AdviseNewBlockResp)
        _, err = s.SendMsg(ctx, &resp)

}

// FetchBlock is the RPC method to fetch a known block from the target server.
func (ms *MuxService) FetchBlockHandler(s net.Stream) {
        ctx := context.Background()
        var req MuxFetchBlockReq 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	var resp = MuxFetchBlockResp { 
		Envelope : req.Envelope,
		DomainID : req.DomainID,
	}

	v, ok := ms.serviceMap.Load(req.DomainID)
	if ! ok {
        	_, err = s.SendMsg(ctx, &resp)
		return 
	}

	v.(*ChainRPCService).FetchBlock(&req.FetchBlockReq, &resp.FetchBlockResp)
        _, err = s.SendMsg(ctx, &resp)
}
