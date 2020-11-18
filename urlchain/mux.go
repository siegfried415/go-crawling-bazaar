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

	//wyong, 20201020
	"github.com/libp2p/go-libp2p-core/protocol" 
	"github.com/libp2p/go-libp2p-core/network" 
	"github.com/libp2p/go-libp2p-core/host" 

	"github.com/siegfried415/gdf-rebuild/proto"
	"github.com/siegfried415/gdf-rebuild/net"

	//wyong, 20201008
	//rpc "github.com/siegfried415/gdf-rebuild/rpc/mux"

        //wyong, 20200928
        //"github.com/ugorji/go/codec"

)

// MuxService defines multiplexing service of sql-chain.
type MuxService struct {
	ServiceName string
	serviceMap  sync.Map
}

// NewMuxService creates a new multiplexing service and registers it to rpc server.
func NewMuxService(serviceName string, /* server *rpc.Server */ h host.Host ) (service *MuxService, err error) {
	service = &MuxService{
		ServiceName: serviceName,
	}

	//wyong, 20200925 
	//err = server.RegisterService(serviceName, service)
        h.SetStreamHandler(protocol.ID("ProtocolUrlChainAdviseNewBlock"), service.AdviseNewBlockHandler)
        h.SetStreamHandler(protocol.ID("ProtocolUrlChainFetchBlock"), service.FetchBlockHandler)

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

//wyong, 20201020
// AdviseNewBlock is the RPC method to advise a new produced block to the target server.
//func (ms *MuxService) AdviseNewBlockHandler(req *MuxAdviseNewBlockReq, resp *MuxAdviseNewBlockResp) error {
func (ms *MuxService) AdviseNewBlockHandler(s network.Stream ) {
        //decode req with Messagepack, wyong, 20200928
        //var decReq MuxAdviseNewBlockReq 
        //dec := codec.NewDecoderBytes(req, new(codec.MsgpackHandle))
        //if err := dec.Decode(&decReq); err != nil {
        //        return err
        //}
        ctx := context.Background()
        var req MuxAdviseNewBlockReq 
        err := s.(net.Stream).RecvMsg(ctx, &req)
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
        //_, err = net.SendMsg(ctx, s, &resp)

}

//wyong, 20201020
// FetchBlock is the RPC method to fetch a known block from the target server.
//func (s *MuxService) FetchBlockHandler(req *MuxFetchBlockReq, resp *MuxFetchBlockResp) (err error) {
func (ms *MuxService) FetchBlockHandler(s network.Stream) {
        //decode req with Messagepack, wyong, 20200928
        //var decReq MuxFetchBlockReq 
        //dec := codec.NewDecoderBytes(req, new(codec.MsgpackHandle))
        //if err := dec.Decode(&decReq); err != nil {
        //        return err
        //}

        ctx := context.Background()
        var req MuxFetchBlockReq 
        err := s.(net.Stream).RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	v, ok := ms.serviceMap.Load(req.DomainID)
	if ! ok {
		//return ErrUnknownMuxRequest
		return 
	}

	var resp = MuxFetchBlockResp { 
		Envelope : req.Envelope,
		DomainID : req.DomainID,
	}

	v.(*ChainRPCService).FetchBlock(&req.FetchBlockReq, &resp.FetchBlockResp)

}
