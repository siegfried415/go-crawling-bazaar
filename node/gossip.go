/*
 * Copyright 2019 The CovenantSQL Authors.
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

package node

import (
        "context"

        protocol "github.com/libp2p/go-libp2p-core/protocol"

        net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

// GossipRequest defines the gossip request payload.
type GossipRequest struct {
	proto.Envelope
	Node *proto.Node
	TTL  uint32
}

// GossipService defines the gossip service instance.
type GossipService struct {
	s *KVServer
}

// NewGossipService returns new gossip service.
func NewGossipService(s *KVServer) (*GossipService, error) {
	gs := &GossipService{
		s: s,
	}

	s.host.SetStreamHandlerExt( protocol.ID("DHTG.SetNode"), gs.SetNodeHandler )
	return gs, nil 
}

// SetNode update current node info and broadcast node update request.
func (gs *GossipService) SetNodeHandler(/* req *GossipRequest, resp *interface{} */ s net.Stream) /* (err error) */  {
	//return s.s.SetNodeEx(req.Node, req.TTL, req.GetNodeID().ToNodeID())
        ctx := context.Background()
        var req GossipRequest  

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	gs.s.SetNodeEx(req.Node, req.TTL, req.GetNodeID().ToNodeID())
}
