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

package net

import (
	"fmt"
	"context" 

	"github.com/siegfried415/go-crawling-bazaar/net/consistent"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
        protocol "github.com/libp2p/go-libp2p-core/protocol"
)

// DHTService is server side RPC implementation.
type DHTService struct {
	Consistent *consistent.Consistent
}

// NewDHTServiceWithRing will return a new DHTService and set an existing hash ring.
func NewDHTServiceWithRing(c *consistent.Consistent) (s *DHTService, err error) {
	s = &DHTService{
		Consistent: c,
	}
	return
}

// NewDHTService will return a new DHTService.
func NewDHTService(h RoutedHost, DHTStorePath string, persistImpl consistent.Persistence, initPB bool) (s *DHTService, err error) {
	c, err := consistent.InitConsistent(DHTStorePath, persistImpl, initPB)
	if err != nil {
		log.WithError(err).Error("init DHT service failed")
		return
	}

	s, err = NewDHTServiceWithRing(c)

	h.SetStreamHandlerExt( protocol.ID("DHT.Nil"), s.NilHandler)
	h.SetStreamHandlerExt( protocol.ID("DHT.FindNode"), s.FindNodeHandler)
	h.SetStreamHandlerExt( protocol.ID("DHT.FindNeighbor"), s.FindNeighborHandler)
	h.SetStreamHandlerExt( protocol.ID("DHT.Ping"), s.PingHandler)
	
	return 
}

// Nil RPC does nothing just for probe.
func (DHT *DHTService) NilHandler( s Stream ) {
	return
}

//todo
//var permissionCheckFunc = IsPermitted

// FindNode RPC returns node with requested node id from DHT.
func (DHT *DHTService) FindNodeHandler( s Stream ) {

        ctx := context.Background()
        var req proto.FindNodeReq 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	//todo
	//if permissionCheckFunc != nil && !permissionCheckFunc(&req.Envelope, DHTFindNode) {
	//	err = fmt.Errorf("calling from node %s is not permitted", req.GetNodeID())
	//	log.Error(err)
	//	return
	//}

	node, err := DHT.Consistent.GetNode(string(req.ID))
	if err != nil {
		err = fmt.Errorf("get node %s from DHT failed: %s", req.NodeID, err)
		log.Error(err)
		return
	}

	resp := proto.FindNodeResp {
		Node : node ,
	}

        s.SendMsg(ctx, &resp)
	return
}

// FindNeighbor RPC returns FindNeighborReq.Count closest node from DHT.
func (DHT *DHTService) FindNeighborHandler( s Stream) {
        ctx := context.Background()
        var req proto.FindNeighborReq 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	//todo
	//if permissionCheckFunc != nil && !permissionCheckFunc(&req.Envelope, DHTFindNeighbor) {
	//	err = fmt.Errorf("calling from node %s is not permitted", req.GetNodeID())
	//	log.Error(err)
	//	return
	//}

	nodes, err := DHT.Consistent.GetNeighborsEx(string(req.ID), req.Count, req.Roles)
	if err != nil {
		err = fmt.Errorf("get nodes from DHT failed: %s", err)
		log.Error(err)
		return
	}

	resp := proto.FindNeighborResp {
		Nodes : nodes ,
	}

	log.WithFields(log.Fields{
		"neighCount": len(nodes),
		"req":        req,
	}).Debug("found nodes for find neighbor request")

        s.SendMsg(ctx, &resp)
	return
}

// Ping RPC adds PingReq.Node to DHT.
func (DHT *DHTService) PingHandler( s Stream) {
        ctx := context.Background()
        var req proto.PingReq 

        err := s.RecvMsg(ctx, &req)
        if err != nil {
              return
        }

	//todo
	//if permissionCheckFunc != nil && !permissionCheckFunc(&req.Envelope, DHTPing) {
	//	err = fmt.Errorf("calling Ping from node %s is not permitted", req.GetNodeID())
	//	log.Error(err)
	//	return
	//}

	// Presbyterian node is not permitted to set by RPC
	if req.Node.Role == proto.Leader || req.Node.Role == proto.Follower {
		err = fmt.Errorf("setting %s node is not permitted", req.Node.Role.String())
		log.Error(err)
		return
	}

	// todo, Checking MinNodeIDDifficulty
	//if req.Node.ID.Difficulty() < conf.GConf.MinNodeIDDifficulty {
	//	err = fmt.Errorf("node: %s difficulty too low", req.Node.ID)
	//	log.Error(err)
	//	return
	//}

	err = DHT.Consistent.Add(req.Node)
	if err != nil {
		err = fmt.Errorf("DHT.Consistent.Add %v failed: %s", req.Node, err)
	} else {
		resp := proto.PingResp {
			Msg: "Pong",
		}
        	s.SendMsg(ctx, &resp)
	}

	return
}
