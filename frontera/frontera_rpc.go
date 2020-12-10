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
	"fmt"

	//"github.com/pkg/errors"
	metrics "github.com/rcrowley/go-metrics"

	//wyong, 20201020
	//host "github.com/libp2p/go-libp2p-core/host"
	//network "github.com/libp2p/go-libp2p-core/network"
	protocol "github.com/libp2p/go-libp2p-core/protocol"

	"github.com/siegfried415/gdf-rebuild/proto"
	//"github.com/siegfried415/gdf-rebuild/route"

	//wyong, 20201008 
	//"github.com/siegfried415/gdf-rebuild/rpc"
	//"github.com/siegfried415/gdf-rebuild/rpc/mux"
	net "github.com/siegfried415/gdf-rebuild/net"

	"github.com/siegfried415/gdf-rebuild/types"

	//wyong, 20200928 
	//"github.com/ugorji/go/codec"

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
func NewFronteraRPCService( serviceName string, /* server *mux.Server, direct *rpc.Server, wyong, 20200925 */ h net.RoutedHost,  f *Frontera) ( service *FronteraRPCService) {
	service = &FronteraRPCService{
		frontera: f,
	}

	/* wyong, 20200925 
	server.RegisterService(serviceName, service)
	if direct != nil {
		direct.RegisterService(serviceName, service)
	}
	*/

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


//wyong, 20201020 
func (rpc *FronteraRPCService) UrlRequestHandler (
	//req *types.UrlRequestMessage , res *types.UrlResponse) (err error
	s net.Stream, 
) {
	//todo, turn urlrequest from client to bidding whose target are random selected crawlers. 
	fmt.Printf("FronteraRPCService/ReceiveUrlRequest called\n") 

	//decode req with Messagepack, wyong, 20200928 
	//var decReq types.UrlRequestMessage
        //dec := codec.NewDecoderBytes(req, new(codec.MsgpackHandle))
        //if err := dec.Decode(&decReq); err != nil {
        //        return err
        //}

        ctx := context.Background()
        var req types.UrlRequestMessage 

	//wyong, 20201203
	//ns := net.Stream{ Stream: s }

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	//rpc.frontera.PutBidding(req.Payload.request )

	//ctx := context.Background() 
	var res *types.UrlResponse
	if res, err = rpc.frontera.PutBidding(ctx, &req); err != nil {
		return
	}

	//*res = *r
        s.SendMsg(ctx, res)

	//return nil 
}

//wyong, 20201020 
func (rpc *FronteraRPCService) UrlCidRequestHandler ( 
	//req *types.UrlCidRequestMessage, res *types.UrlCidResponse) (err error
	s net.Stream, 
) {

	//todo, turn urlrequest from client to bidding whose target are random selected crawlers. 
	fmt.Printf("FronteraRPCService/ReceiveUrlCidRequest called\n") 

	//decode req with Messagepack, wyong, 20200928 
	//var decReq types.UrlCidRequestMessage
        //dec := codec.NewDecoderBytes(req, new(codec.MsgpackHandle))
        //if err := dec.Decode(&decReq); err != nil {
        //        return err
        //}

        ctx := context.Background()
        var req types.UrlCidRequestMessage 

	//wyong, 20201203
	//ns := net.Stream{ Stream: s }

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	var res *types.UrlCidResponse
	//ctx := context.Background() 
	if res, err = rpc.frontera.GetCid(ctx, &req); err != nil {
		return
	}

	//*res = *r

        s.SendMsg(ctx, res)
	//return
}

//wyong, 20201020 
func (rpc *FronteraRPCService) BiddingMessageHandler (
	//req *types.UrlBiddingMessage, res *types.Response) (err error
	s net.Stream, 
) {
        fmt.Printf("FronteraRPCService/ReceiveBiddingMessage called\n")
        //atomic.AddUint64(&bs.counters.messagesRecvd, 1)

	//decode req with Messagepack, wyong, 20200928 
	//var decReq types.UrlBiddingMessage
        //dec := codec.NewDecoderBytes(req, new(codec.MsgpackHandle))
        //if err := dec.Decode(&decReq); err != nil {
        //        return err
        //}

        ctx := context.Background()
        var req types.UrlBiddingMessage 

	//wyong, 20201203
	//ns := net.Stream{ Stream: s }

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

        //todo, This call records changes to wantlists, blocks received,
        // and number of bytes transfered. wyong, 20200730 
	//ctx := context.Background() 
        rpc.frontera.engine.UrlBiddingMessageReceived(ctx, &req )
	
	//return nil 
}

//wyong, 20201020 
func (rpc *FronteraRPCService) BidMessageHandler ( 
	//req *types.UrlBidMessage, res *types.Response) (err error
	s net.Stream, 
) {
        fmt.Printf("FronteraRPCService/ReceiveBidMessage called\n")

	//decode req with Messagepack, wyong, 20200928 
	//var decReq types.UrlBidMessage
        //dec := codec.NewDecoderBytes(req, new(codec.MsgpackHandle))
        //if err := dec.Decode(&decReq); err != nil {
        //        return err
        //}

        ctx := context.Background()
        var req types.UrlBidMessage 

	//wyong, 20201203
	//ns := net.Stream{ Stream: s }

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	//ctx := context.Background() 
        rpc.frontera.UrlBidMessageReceived(ctx, &req )

	//return nil 
}

/*
// Query rpc, called by client to issue read/write query.
func (rpc *FronteraRPCService) Query(req *types.Request, res *types.Response) (err error) {
	// Just need to verify signature in db.saveAck
	//if err = req.Verify(); err != nil {
	//	dbQueryFailCounter.Mark(1)
	//	return
	//}
	// verify query is sent from the request node
	if req.Envelope.NodeID.String() != string(req.Header.NodeID) {
		// node id mismatch
		err = errors.Wrap(ErrInvalidRequest, "request node id mismatch in query")
		dbQueryFailCounter.Mark(1)
		return
	}

	var r *types.Response
	if r, err = rpc.frontera.Query(req); err != nil {
		dbQueryFailCounter.Mark(1)
		return
	}

	*res = *r
	dbQuerySuccCounter.Mark(1)

	return
}

// Ack rpc, called by client to confirm read request.
func (rpc *FronteraRPCService) Ack(ack *types.Ack, _ *types.AckResponse) (err error) {
	// Just need to verify signature in db.saveAck
	//if err = ack.Verify(); err != nil {
	//	return
	//}
	// verify if ack node is the original ack node
	if ack.Envelope.NodeID.String() != string(ack.Header.Response.Request.NodeID) {
		err = errors.Wrap(ErrInvalidRequest, "request node id mismatch in ack")
		return
	}

	// verification
	err = rpc.frontera.Ack(ack)

	return
}

// Deploy rpc, called by BP to create/drop database and update peers.
func (rpc *FronteraRPCService) Deploy(req *types.UpdateService, _ *types.UpdateServiceResponse) (err error) {
	// verify request node is block producer
	if !route.IsPermitted(&req.Envelope, route.DBSDeploy) {
		err = errors.Wrap(ErrInvalidRequest, "node not permitted for deploy request")
		return
	}

	// verify signature
	if err = req.Verify(); err != nil {
		return
	}

	// create/drop/update
	switch req.Header.Op {
	case types.CreateDB:
		err = rpc.frontera.Create(&req.Header.Instance, true)
	case types.UpdateDB:
		err = rpc.frontera.Update(&req.Header.Instance)
	case types.DropDB:
		err = rpc.frontera.Drop(req.Header.Instance.DatabaseID)
	}

	return
}
*/

