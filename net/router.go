/*
 * Copyright (c) 2018 Filecoin Project
 * Copyright (c) 2022 https://github.com/siegfried415
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
	"context"
	"math/rand"
	"sync"

	"github.com/pkg/errors"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-core/network"
        protocol "github.com/libp2p/go-libp2p-core/protocol"

	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

// This struct wraps the filecoin nodes router.  This router is a
// go-libp2p-core/routing.Routing interface that provides both PeerRouting,
// ContentRouting and a Bootstrap init process. Filecoin nodes in online mode
// use a go-libp2p-kad-dht DHT to satisfy this interface. Nodes run the
// Bootstrap function to join the DHT on start up. The PeerRouting functionality
// enables filecoin nodes to lookup the network addresses of their peers given a
// peerID.  The ContentRouting functionality enables peers to provide and
// discover providers of network services. This is currently used by the
// auto-relay feature in the filecoin network to allow nodes to advertise
// themselves as relay nodes and discover other relay nodes.
//
// The Routing interface and its DHT instantiation also carries ValueStore
// functionality for using the DHT as a key value store.  Filecoin nodes do
// not currently use this functionality.

var (
	// ErrUnknownNodeID indicates we got unknown node id
	ErrUnknownNodeID = errors.New("unknown node id")

	// ErrNilNodeID indicates we got nil node id
	ErrNilNodeID = errors.New("nil node id")
)

// Router exposes the methods on the internal filecoin router that are needed
// by the system plumbing API.
type PBRouter struct {
	//for content routing
	routing routing.ContentRouting

        cache    map [proto.NodeID]string 
        pbNodeIDs map[proto.NodeID]string 
        pbNodes   map[proto.NodeID]proto.Node 

	host	 host.Host  
        sync.RWMutex
}

// NewRouter builds a new router.
func NewPBRouter(h host.Host, r routing.ContentRouting ) (*PBRouter, error) {
	pbr := &PBRouter {
		host: 	h, 
		routing: r, 
		cache:     make(map[proto.NodeID]string),
		pbNodeIDs: make(map[proto.NodeID]string),
        }

	_, err := pbr.initPBNodeIDs()
	return pbr, err 
}

// FindProvidersAsync searches for and returns peers who are able to provide a
// given key.
func (r *PBRouter) FindProvidersAsync(ctx context.Context, key cid.Cid, count int) <-chan peer.AddrInfo {
	return r.routing.FindProvidersAsync(ctx, key, count)
}


// NOTE,NOTE,NOTE, 
// Provide adds the given cid to the content routing system. If 'true' is
// passed, it also announces it, otherwise it is just kept in the local
// accounting of which objects are being provided.
func (r *PBRouter) Provide(context.Context, cid.Cid, bool) error {
	return nil 	
}

// FindPeer searches the libp2p router for a given peer id
func (r *PBRouter) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	nodeID := (proto.NodeID)(peerID.Pretty())
	rawNodeID := &nodeID 

	addr, err := r.GetNodeAddrCache(rawNodeID)
	if err != nil {
		//log.WithField("target", id.String()).WithError(err).Debug("get node addr from cache failed")
		if err == ErrUnknownNodeID {
			var node *proto.Node
			
			node, err = r.FindNodeInPB(rawNodeID)
			if err != nil {
				return peer.AddrInfo{}, err 
			}
			_ = r.SetNodeAddrCache(rawNodeID, node.Addr)
			addr = node.Addr 
		}
	}

	result, err := PeerAddrToAddrInfo(addr) 
	if err != nil {
		return peer.AddrInfo{}, err 
	}

	return *result, nil 
}


// GetNodeAddrCache gets node addr by node id, if cache missed try RPC.
func (r *PBRouter) GetNodeAddrCache(id *proto.NodeID) (addr string, err error) {
	if id == nil {
		return "", ErrNilNodeID
	}
	r.RLock()
	defer r.RUnlock()
	addr, ok := r.cache[*id]
	if !ok {
		return "", ErrUnknownNodeID
	}
	return
}

// setNodeAddrCache sets node id and addr.
func (r *PBRouter) SetNodeAddrCache(id *proto.NodeID, addr string) (err error) {
	if id == nil {
		return ErrNilNodeID
	}
	r.Lock()
	defer r.Unlock()
	r.cache[*id] = addr
	return
}


func (r *PBRouter) NewStreamExt(ctx context.Context, peerid proto.NodeID, pids ...protocol.ID) (Stream, error) {
        decPeerId, err := peer.IDB58Decode(peerid.String())
        if err != nil {
                err = errors.Wrap(err, "not valid Peer ID")
                return Stream{}, err
        }

        s, err := r.NewStream(ctx, decPeerId, pids...)
        if err!= nil {
                return Stream{}, err
        }

        return Stream{ Stream: s }, nil
}

func (r *PBRouter) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
        var result Stream
        if nodial, _ := network.GetNoDial(ctx); !nodial {
		err := r.host.Connect(ctx, peer.AddrInfo{ID: p})
                if err != nil {
                        return result, err
                }
        }

        s, err := r.host.NewStream(ctx, p, pids...)
        if err != nil {
                return result, err
        }

        return s, nil
}

// FindNodeInPB find node in presbyterian dht service.
func (r *PBRouter) FindNodeInPB(id *proto.NodeID) (*proto.Node, error) {
	pbs := r.GetPBs()
	if len(pbs) == 0 {
		err := errors.New("no available Presbyterian")
		return nil, err 
	}

	ctx := context.Background()
	req := &proto.FindNodeReq{
		ID: proto.NodeID(id.String()),
	}
	resp := new(proto.FindNodeResp)
	pbCount := len(pbs)
	offset := rand.Intn(pbCount)
	method := protocol.ID("DHT.FindNode")	//DHTFindNode.String()

	for i := 0; i != pbCount; i++ {
		pb := pbs[(offset+i)%pbCount]

		s, err := r.NewStreamExt(ctx, pb, method )
		if err != nil {
			log.Debugf("error opening push stream to %s: %s", pb, err.Error())
			continue 	
		}

		_, err = s.SendMsg(ctx, &req )
		if err != nil {
			log.WithFields(log.Fields{
				"method": method,
				"pb":     pb,
			}).WithError(err).Warning("call dht rpc failed")
			continue 	
		}

		err = s.RecvMsg(ctx, &resp)
		if err!= nil {
			log.WithFields(log.Fields{
				"method": method,
				"pb":     pb,
			}).WithError(err).Warning("call dht rpc failed")
			continue 	
		}

		s.Close()
		node := resp.Node
		return node, nil 
	}

	err := errors.New("could not find node in all presbyterians")
	return nil, err 
}


// initPBNodeIDs initializes Presbyterian route and map from config file and DNS Seed.
func (r *PBRouter) initPBNodeIDs() (pbNodeIDs map[proto.NodeID]string , err error ) {
	if conf.GConf == nil {
		log.Fatal("call conf.LoadConfig to init conf first")
	}

	// clear address map before init
	r.pbNodeIDs = make(map[proto.NodeID]string)
	pbNodeIDs = r.pbNodeIDs

	if r.pbNodes == nil {
		r.pbNodes = make(map[proto.NodeID]proto.Node)
	}

	if conf.GConf.KnownNodes != nil {
		for _, n := range conf.GConf.KnownNodes {
			rawID := &n.ID 
			if rawID != nil {
				if n.Role == proto.Leader || n.Role == proto.Follower {
					r.pbNodes[*rawID] = n
				}
				r.SetNodeAddrCache(rawID, n.Addr)
			}
		}
	}

	conf.GConf.SeedPBNodes = make([]proto.Node, 0, len(r.pbNodes))
	for _, n := range r.pbNodes {
		rawID := &n.ID 
		if rawID != nil {
			conf.GConf.SeedPBNodes = append(conf.GConf.SeedPBNodes, n)
			r.SetNodeAddrCache(rawID, n.Addr)
			r.pbNodeIDs[*rawID] = n.Addr
		}
	}

	return r.pbNodeIDs, nil 
}

// GetPBs returns the known PB node id list.
func (r *PBRouter) GetPBs() (pbAddrs []proto.NodeID) {
	pbAddrs = make([]proto.NodeID, 0, len(r.pbNodeIDs))
	for id := range r.pbNodeIDs {
		pbAddrs = append(pbAddrs, proto.NodeID(id.String()))
	}
	return
}

