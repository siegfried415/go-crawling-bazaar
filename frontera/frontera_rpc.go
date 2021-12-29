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
	"context" 

	metrics "github.com/rcrowley/go-metrics"
	protocol "github.com/libp2p/go-libp2p-core/protocol"

	//log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/types"
)

var (
	dbQuerySuccCounter metrics.Meter
	dbQueryFailCounter metrics.Meter
)

// ObserverFetchBlockReq defines the request for observer to fetch block.
type ObserverFetchBlockReq struct {
	proto.Envelope
	proto.DatabaseID
	Count int32 // sqlchain block serial number since genesis block (0)
}

// ObserverFetchBlockResp defines the response for observer to fetch block.
type ObserverFetchBlockResp struct {
	Count int32 // sqlchain block serial number since genesis block (0)
	Block *types.Block
}

// FronteraRPCService is the rpc endpoint of database management.
type FronteraRPCService struct {
	frontera *Frontera
}

// NewFronteraRPCService returns new frontera rpc service endpoint.
func NewFronteraRPCService( serviceName string, h net.RoutedHost,  f *Frontera) ( service *FronteraRPCService) {
	service = &FronteraRPCService{
		frontera: f,
	}

	h.SetStreamHandlerExt( protocol.ID("FRT.UrlRequest"), service.UrlRequestHandler)
	h.SetStreamHandlerExt( protocol.ID("FRT.UrlCidRequest"), service.UrlCidRequestHandler) 
	h.SetStreamHandlerExt( protocol.ID("FRT.Bidding"), service.BiddingMessageHandler) 
	h.SetStreamHandlerExt( protocol.ID("FRT.Bid"), service.BidMessageHandler) 
	
	dbQuerySuccCounter = metrics.NewMeter()
	metrics.Register("db-query-succ", dbQuerySuccCounter)
	dbQueryFailCounter = metrics.NewMeter()
	metrics.Register("db-query-fail", dbQueryFailCounter)

	return
}


func (rpc *FronteraRPCService) UrlRequestHandler (s net.Stream) {
        ctx := context.Background()
        var req types.UrlRequestMessage 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }


	var res *types.UrlResponse
	if err = rpc.frontera.PutBidding(ctx, 
			string(req.Header.DomainID), 
			req.Payload.Requests, 
			req.Payload.ParentUrlRequest); err!=nil{
		return
	}

	res = &types.UrlResponse{
                Header: types.UrlSignedResponseHeader{
                        UrlResponseHeader: types.UrlResponseHeader{
                                Request:     req.Header.UrlRequestHeader,
                                RequestHash: req.Header.Hash(),
                                NodeID:      rpc.frontera.nodeID, //f.nodeID,
                                Timestamp:   getLocalTime(),
                        },
                },
        }

        s.SendMsg(ctx, res)

}

func (rpc *FronteraRPCService) UrlCidRequestHandler ( s net.Stream) {
        ctx := context.Background()
        var req types.UrlCidRequestMessage 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	var res *types.UrlCidResponse
	if res, err = rpc.frontera.RetriveUrlCid(ctx, &req); err != nil {
		return
	}

        s.SendMsg(ctx, res)
}

func (rpc *FronteraRPCService) BiddingMessageHandler ( s net.Stream) {
        ctx := context.Background()
        var req types.UrlBiddingMessage 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

        //This call records changes to wantlists, blocks received,
        // and number of bytes transfered. 
        rpc.frontera.bs.UrlBiddingMessageReceived(ctx, &req )
}

func (rpc *FronteraRPCService) BidMessageHandler ( s net.Stream) {
        ctx := context.Background()
        var req types.UrlBidMessage 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

        rpc.frontera.UrlBidMessageReceived(ctx, &req )
}

