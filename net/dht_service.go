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

	//wyong, 20201102 
        //host "github.com/libp2p/go-libp2p-core/host"
        protocol "github.com/libp2p/go-libp2p-core/protocol"
        network "github.com/libp2p/go-libp2p-core/network"

	//"github.com/siegfried415/gdf-rebuild/conf"
	"github.com/siegfried415/gdf-rebuild/net/consistent"
	//"github.com/siegfried415/gdf-rebuild/kms"
	"github.com/siegfried415/gdf-rebuild/proto"
	"github.com/siegfried415/gdf-rebuild/utils/log"
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
func NewDHTService(h RoutedHost, DHTStorePath string, persistImpl consistent.Persistence, initBP bool) (s *DHTService, err error) {
	c, err := consistent.InitConsistent(DHTStorePath, persistImpl, initBP)
	if err != nil {
		log.WithError(err).Error("init DHT service failed")
		return
	}

	//wyong, 20201102 
	//return NewDHTServiceWithRing(c)
	s, err = NewDHTServiceWithRing(c)

	//wyong, 20201102 
	h.SetStreamHandler( protocol.ID("DHT.Nil"), s.NilHandler)
	h.SetStreamHandler( protocol.ID("DHT.FindNode"), s.FindNodeHandler)
	h.SetStreamHandler( protocol.ID("DHT.FindNeighbor"), s.FindNeighborHandler)
	h.SetStreamHandler( protocol.ID("DHT.Ping"), s.PingHandler)
	
	return 
}

// Nil RPC does nothing just for probe.
func (DHT *DHTService) NilHandler(
	//req *interface{}, resp *interface{}) (err error
	s network.Stream, 
) {
	return
}

//todo, wyong, 20201110 
//var permissionCheckFunc = IsPermitted

// FindNode RPC returns node with requested node id from DHT.
func (DHT *DHTService) FindNodeHandler(
	//req *proto.FindNodeReq, resp *proto.FindNodeResp) (err error
	s network.Stream ,
) {

        ctx := context.Background()
        var req proto.FindNodeReq 

	//wyong, 20201118 
	ns := Stream{ Stream: s } 
        err := ns.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	//todo, wyong, 20201110 
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

	//resp.Node = node
	resp := proto.FindNodeResp {
		Node : node ,
	}

        ns.SendMsg(ctx, &resp)
	return
}

// FindNeighbor RPC returns FindNeighborReq.Count closest node from DHT.
func (DHT *DHTService) FindNeighborHandler(
	//req *proto.FindNeighborReq, resp *proto.FindNeighborResp) (err error
	s network.Stream,
) {
        ctx := context.Background()
        var req proto.FindNeighborReq 

	//wyong, 20201118 
	ns := Stream{ Stream: s } 
        err := ns.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	//todo, wyong, 20201110 
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

	//resp.Nodes = nodes
	resp := proto.FindNeighborResp {
		Nodes : nodes ,
	}

	log.WithFields(log.Fields{
		"neighCount": len(nodes),
		"req":        req,
	}).Debug("found nodes for find neighbor request")

        ns.SendMsg(ctx, &resp)
	return
}

// Ping RPC adds PingReq.Node to DHT.
func (DHT *DHTService) PingHandler(
	//req *proto.PingReq, resp *proto.PingResp) (err error
	s network.Stream,
) {
	//log.Debugf("got ping req, Yeah!")
	fmt.Printf("DHTService/PingHandler(10),got ping req, Yeah!\n")

        ctx := context.Background()
        var req proto.PingReq 

	//wyong, 20201118 
	ns := Stream{ Stream: s } 
        err := ns.RecvMsg(ctx, &req)
        //if err != nil {
	//	fmt.Printf("DHTService/PingHandler(15), err=%s\n", err )
        //      return
        //}

	fmt.Printf("DHTService/PingHandler(20)\n")

	//todo, wyong, 20201110 
	//if permissionCheckFunc != nil && !permissionCheckFunc(&req.Envelope, DHTPing) {
	//	err = fmt.Errorf("calling Ping from node %s is not permitted", req.GetNodeID())
	//	log.Error(err)
	//	return
	//}

	// BP node is not permitted to set by RPC
	if req.Node.Role == proto.Leader || req.Node.Role == proto.Follower {
		err = fmt.Errorf("setting %s node is not permitted", req.Node.Role.String())
		log.Error(err)
		return
	}

	fmt.Printf("DHTService/PingHandler(30)\n")
	// todo, wyong, 20201114 
	// Checking if ID Nonce Pubkey matched
	//if !kms.IsIDPubNonceValid(req.Node.ID.ToRawNodeID(), &req.Node.Nonce, req.Node.PublicKey) {
	//	err = fmt.Errorf("node: %s nonce public key not match", req.Node.ID)
	//	log.Error(err)
	//	return
	//}

	//todo, wyong, 20201118 
	// Checking MinNodeIDDifficulty
	//if req.Node.ID.Difficulty() < conf.GConf.MinNodeIDDifficulty {
	//	err = fmt.Errorf("node: %s difficulty too low", req.Node.ID)
	//	log.Error(err)
	//	return
	//}

	fmt.Printf("DHTService/PingHandler(40)\n")
	err = DHT.Consistent.Add(req.Node)
	if err != nil {
		err = fmt.Errorf("DHT.Consistent.Add %v failed: %s", req.Node, err)
		fmt.Printf("DHTService/PingHandler(45), err=%s\n", err)
	} else {
		fmt.Printf("DHTService/PingHandler(47)\n")
		//resp.Msg = "Pong"
		resp := proto.PingResp {
			Msg: "Pong",
		}
        	ns.SendMsg(ctx, &resp)
	}

	fmt.Printf("DHTService/PingHandler(50)\n")
	return
}
