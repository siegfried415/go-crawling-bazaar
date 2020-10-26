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

package net

import (
	"math/rand"
	"sync"
	"time"
	"context" 

	"github.com/pkg/errors"

	//wyong, 20201008
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol" 

	"github.com/siegfried415/gdf-rebuild/kms"
	//"github.com/siegfried415/gdf-rebuild/naconn"
	"github.com/siegfried415/gdf-rebuild/proto"
	"github.com/siegfried415/gdf-rebuild/route"
	"github.com/siegfried415/gdf-rebuild/utils/log"
)

var (
	// ErrNoChiefBlockProducerAvailable defines failure on find chief block producer.
	ErrNoChiefBlockProducerAvailable = errors.New("no chief block producer found")

	//FIXME(auxten): remove currentPB stuff
	// currentPB represents current chief block producer node.
	currentPB proto.NodeID
	// currentPBLock represents the chief block producer access lock.
	currentPBLock sync.Mutex
)

/*
// Resolver implements the node ID resolver using PB network with mux-RPC protocol.
type Resolver struct {
	direct bool
}

// NewResolver returns a Resolver which resolves the mux-RPC server address of the
// target node ID.
func NewResolver() naconn.Resolver {
	return &Resolver{}
}

// NewDirectResolver returns a Resolver which resolves the direct RPC server address of the
// target node ID.
func NewDirectResolver() naconn.Resolver {
	return &Resolver{direct: true}
}

// Resolve implements the node ID resolver using the PB network with mux-RPC protocol.
func (r *Resolver) Resolve(id *proto.RawNodeID) (string, error) {
	if r.direct {
		node, err := GetNodeInfo(id)
		if err != nil {
			return "", err
		}
		if node.Role == proto.Miner {
			return node.DirectAddr, nil
		}
		return node.Addr, nil
	}
	return GetNodeAddr(id)
}

// ResolveEx implements the node ID resolver extended method using the PB network
// with mux-RPC protocol.
func (r *Resolver) ResolveEx(id *proto.RawNodeID) (*proto.Node, error) {
	return GetNodeInfo(id)
}

func init() {
	naconn.RegisterResolver(&Resolver{})
}

// GetNodeAddr tries best to get node addr.
func GetNodeAddr(id *proto.RawNodeID) (addr string, err error) {
	addr, err = route.GetNodeAddrCache(id)
	if err != nil {
		//log.WithField("target", id.String()).WithError(err).Debug("get node addr from cache failed")
		if err == route.ErrUnknownNodeID {
			var node *proto.Node
			node, err = FindNodeInPB(id)
			if err != nil {
				return
			}
			_ = route.SetNodeAddrCache(id, node.Addr)
			addr = node.Addr
		}
	}
	return
}

// GetNodeInfo tries best to get node info.
func GetNodeInfo(id *proto.RawNodeID) (nodeInfo *proto.Node, err error) {
	nodeInfo, err = kms.GetNodeInfo(proto.NodeID(id.String()))
	if err != nil {
		//log.WithField("target", id.String()).WithError(err).Info("get node info from KMS failed")
		if errors.Cause(err) == kms.ErrKeyNotFound {
			nodeInfo, err = FindNodeInPB(id)
			if err != nil {
				return
			}
			errSet := route.SetNodeAddrCache(id, nodeInfo.Addr)
			if errSet != nil {
				log.WithError(errSet).Warning("set node addr cache failed")
			}
			errSet = kms.SetNode(nodeInfo)
			if errSet != nil {
				log.WithError(errSet).Warning("set node to kms failed")
			}
		}
	}
	return
}

// FindNodeInPB find node in block producer dht service.
func FindNodeInPB(host host.Host, id *proto.RawNodeID) (*proto.Node, error) {
	bps := route.GetPBs()
	if len(bps) == 0 {
		err = errors.New("no available PB")
		return
	}

	//wyong, 20201008 
	//client := NewCaller()

	req := &proto.FindNodeReq{
		ID: proto.NodeID(id.String()),
	}
	resp := new(proto.FindNodeResp)
	bpCount := len(bps)
	offset := rand.Intn(bpCount)
	method := route.DHTFindNode.String()

	for i := 0; i != bpCount; i++ {
		bp := bps[(offset+i)%bpCount]

		//wyong, 20201008 
		//err = client.CallNode(bp, method, req, resp)
		s, err := host.NewStream(ctx, peer.ID(bp), method )
		if err != nil {
			log.Debugf("error opening push stream to %s: %s", p, err.Error())
			continue 	
		}

		_, err = net.MsgToStream(ctx, s, &req )
		if err != nil {
			log.WithFields(log.Fields{
				"method": method,
				"bp":     bp,
			}).WithError(err).Warning("call dht rpc failed")
			continue 	
		}

		resp, err := ioutil.ReadAll(s)
		if err!= nil {
			log.WithFields(log.Fields{
				"method": method,
				"bp":     bp,
			}).WithError(err).Warning("call dht rpc failed")
			continue 	
		}

		s.Close()
		//if err == nil {
		node = resp.Node
		return node, nil 
		//}


	}

	err = errors.Wrapf(err, "could not find node in all block producers")
	return nil, err 
}
*/

// PingPB Send DHT.Ping Request with Anonymous ETLS session.
func PingPB(host host.Host, node *proto.Node, PBNodeID proto.NodeID) (err error) {
	//wyong, 20201020 
	ctx := context.Background()

	//client := NewCaller()
	req := &proto.PingReq{
		Node: *node,
	}

	resp := new(proto.PingResp)

	//wyong, 20201008 
	//err = client.CallNode(PBNodeID, "DHT.Ping", req, resp)
        s, err := host.NewStream(ctx, peer.ID(PBNodeID), protocol.ID("DHT.Ping"))
	if err != nil {
		err = errors.Wrap(err, "call DHT.Ping failed")
		return
	}
        defer s.Close()

	_, err = SendMsg(ctx, s, &req )
	if err != nil {
		err = errors.Wrap(err, "send DHT.Ping failed")
		return 
	}

	//resp, err = ioutil.ReadAll(s)
	err = RecvMsg(ctx, s, resp) 
	if err!= nil {
		err = errors.Wrap(err, "get DHT.Ping result failed")
		return 
	} 

	return
}

// GetCurrentPB returns nearest hash distance block producer as current node chief block producer.
func GetCurrentPB(host host.Host ) (bpNodeID proto.NodeID, err error) {
	//wyong, 20201020 
	ctx := context.Background()

	currentPBLock.Lock()
	defer currentPBLock.Unlock()

	if !currentPB.IsEmpty() {
		bpNodeID = currentPB
		return
	}

	var localNodeID proto.NodeID
	if localNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get random block producer first
	bpList := route.GetPBs()

	if len(bpList) == 0 {
		err = ErrNoChiefBlockProducerAvailable
		return
	}

	randomPB := bpList[rand.Intn(len(bpList))]

	// call random block producer for nearest block producer node
	req := &proto.FindNeighborReq{
		ID: localNodeID,
		Roles: []proto.ServerRole{
			proto.Leader,
			proto.Follower,
		},
		Count: 1,
	}
	res := new(proto.FindNeighborResp)
	//wyong, 20201008 
	//if err = NewCaller().CallNode(randomPB, "DHT.FindNeighbor", req, res); err != nil {
        s, err := host.NewStream(ctx, peer.ID(randomPB), protocol.ID("DHT.FindNeighbor"))
	if err != nil { 
		return
	}
        defer s.Close()

	_, err = SendMsg(ctx, s, &req )
	if err != nil {
		err = errors.Wrap(err, "send DHT.FindNeighbor failed")
		return 
	}

	//res, err = ioutil.ReadAll(s)
	err = RecvMsg(ctx, s, res) 
	if err!= nil {
		err = errors.Wrap(err, "get DHT.FindNeighbor result failed")
		return 
	} 


	if len(res.Nodes) <= 0 {
		// node not found
		err = errors.Wrap(ErrNoChiefBlockProducerAvailable,
			"get no hash nearest block producer nodes")
		return
	}

	if res.Nodes[0].Role != proto.Leader && res.Nodes[0].Role != proto.Follower {
		// not block producer
		err = errors.Wrap(ErrNoChiefBlockProducerAvailable,
			"no suitable nodes with proper block producer role")
		return
	}

	currentPB = res.Nodes[0].ID
	bpNodeID = currentPB

	return
}

// SetCurrentPB sets current node chief block producer.
func SetCurrentPB(bpNodeID proto.NodeID) {
	currentPBLock.Lock()
	defer currentPBLock.Unlock()
	currentPB = bpNodeID
}

// RequestPB sends request to main chain.
func RequestPB(host host.Host, method string, req interface{}, resp interface{}) (err error) {
	//wyong, 20201020
	ctx := context.Background()

	var bp proto.NodeID
	if bp, err = GetCurrentPB(host); err != nil {
		return err
	}

	//todo, method->protocol, wyong, 20201008 
	//return NewCaller().CallNode(bp, method, req, resp)
        s, err := host.NewStream(ctx, peer.ID(bp), protocol.ID(method))
        if err != nil {
                log.Debugf("error opening push stream to %s: %s", bp, err.Error())
                return
        }
        defer s.Close()

	_, err = SendMsg(ctx, s, &req )
	if err == nil {
		//log.WithFields(log.Fields{
		//	"index":    i.index,
		//	"instance": i.r.instanceID,
		//}).WithError(err).Debug("send fetch request failed")
		return err 
	}

	err = RecvMsg(ctx, s, resp )
	if err!= nil {
		//log.WithFields(log.Fields{
		//	"index":    i.index,
		//	"instance": i.r.instanceID,
		//}).WithError(err).Debug("get fetch result failed")
		return err 	
	} 

	return nil 

}

// RegisterNodeToPB registers the current node to bp network.
func RegisterNodeToPB(host host.Host, timeout time.Duration) (err error) {
	// get local node id
	localNodeID, err := kms.GetLocalNodeID()
	if err != nil {
		err = errors.Wrap(err, "register node to PB")
		return
	}

	// get local node info
	localNodeInfo, err := kms.GetNodeInfo(localNodeID)
	if err != nil {
		err = errors.Wrap(err, "register node to PB")
		return
	}

	log.WithField("node", localNodeInfo).Debug("construct local node info")

	pingWaitCh := make(chan proto.NodeID)
	bpNodeIDs := route.GetPBs()
	for _, bpNodeID := range bpNodeIDs {
		go func(ch chan proto.NodeID, id proto.NodeID) {
			for {
				//wyong, 20201020 
				err := PingPB(host, localNodeInfo, id)
				if err == nil {
					log.Infof("ping PB succeed: %v", localNodeInfo)
					select {
					case ch <- id:
					default:
					}
					return
				}

				log.Warnf("ping PB failed: %v", err)
				time.Sleep(3 * time.Second)
			}
		}(pingWaitCh, bpNodeID)
	}

	select {
	case bp := <-pingWaitCh:
		log.WithField("PB", bp).Infof("ping PB succeed")
	case <-time.After(timeout):
		return errors.New("ping PB timeout")
	}

	return
}
