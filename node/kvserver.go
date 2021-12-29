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

package node

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	//"github.com/siegfried415/go-crawling-bazaar/utils/log"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/storage"
	"github.com/siegfried415/go-crawling-bazaar/utils"
)

// KVServer holds LocalStorage instance and implements consistent persistence interface.
type KVServer struct {
	current   proto.NodeID
	host	net.RoutedHost
	peers     *proto.Peers
	storage   *LocalStorage
	ctx       context.Context
	cancelCtx context.CancelFunc
	timeout   time.Duration
	wg        sync.WaitGroup
}

// NewKVServer returns the kv server instance.
func NewKVServer(host net.RoutedHost, currentNode proto.NodeID, peers *proto.Peers, storage *LocalStorage, timeout time.Duration) (s *KVServer) {
	ctx, cancelCtx := context.WithCancel(context.Background())

	return &KVServer{
		current:   currentNode,
		host : 	   host,
		peers:     peers,
		storage:   storage,
		ctx:       ctx,
		cancelCtx: cancelCtx,
		timeout:   timeout,
	}
}

// Init implements consistent.Persistence
func (s *KVServer) Init(storePath string, initNodes []proto.Node) (err error) {
	for _, n := range initNodes {
		err = s.storage.SetNode(&n)
		if err != nil {
			//log.WithError(err).Error("init dht kv server failed")
			return
		}
	}
	return
}

// SetNode implements consistent.Persistence
func (s *KVServer) SetNode(node *proto.Node) (err error) {
	return s.SetNodeEx(node, 1, s.current)
}

// SetNodeEx is used by gossip service to broadcast to other nodes.
func (s *KVServer) SetNodeEx(node *proto.Node, ttl uint32, origin proto.NodeID) (err error) {
	//log.WithFields(log.Fields{
	//	"node":   node.ID,
	//	"ttl":    ttl,
	//	"origin": origin,
	//}).Debug("update node to kv storage")

	router := s.host.Router()
	
	//node.ID.ToRawNodeID() -> &node.ID
        err = router.SetNodeAddrCache(&node.ID, node.Addr)
        if err != nil {
                //log.WithFields(log.Fields{
                //      "id":   node.ID,
                //      "addr": node.Addr,
                //}).WithError(err).Error("set node addr cache failed")
        }

	// set local
	if err = s.storage.SetNode(node); err != nil {
		err = errors.Wrap(err, "set node failed")
		return
	}

	if ttl > 0 {
		s.nonBlockingSync(node, origin, ttl-1)
	}

	return
}

// DelNode implements consistent.Persistence.
func (s *KVServer) DelNode(nodeID proto.NodeID) (err error) {
	// no need to del node currently
	return
}

// Reset implements consistent.Persistence.
func (s *KVServer) Reset() (err error) {
	return
}

// GetAllNodeInfo implements consistent.Persistence.
func (s *KVServer) GetAllNodeInfo() (nodes []proto.Node, err error) {
	var result [][]interface{}
	query := "SELECT `node` FROM `dht`;"
	_, _, result, err = s.storage.Query(context.Background(), []storage.Query{
		{
			Pattern: query,
		},
	})
	if err != nil {
		//log.WithField("query", query).WithError(err).Error("query failed")
		return
	}
	//log.Debugf("SQL: %#v\nResults: %#v", query, result)

	nodes = make([]proto.Node, 0, len(result))

	for _, r := range result {
		if len(r) == 0 {
			continue
		}
		nodeBytes, ok := r[0].([]byte)
		//log.Debugf("nodeBytes: %#v, %#v", nodeBytes, ok)
		if !ok {
			continue
		}

		nodeDec := proto.NewNode()
		err = utils.DecodeMsgPack(nodeBytes, nodeDec)
		if err != nil {
			//log.WithError(err).Error("unmarshal node info failed")
			continue
		}
		nodes = append(nodes, *nodeDec)
	}

	if len(nodes) > 0 {
		err = nil
	}

	return
}

// Stop stops the dht node server and wait for inflight gossip requests.
func (s *KVServer) Stop() {
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
	s.wg.Wait()
}

func (s *KVServer) nonBlockingSync(node *proto.Node, origin proto.NodeID, ttl uint32) {
	if s.peers == nil {
		return
	}

	c, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()
	for _, n := range s.peers.Servers {
		if n != s.current && n != origin {
			// sync
			req := &GossipRequest{
				Node: node,
				TTL:  ttl,
			}

			s.wg.Add(1)
			go func(node proto.NodeID) {
				defer s.wg.Done()

				s, err := s.host.NewStreamExt(c, node, protocol.ID("DHTG.SetNode"))
				if err != nil {
					return
				}
				
				if _, err = s.SendMsg(c, req ); err != nil {
					s.Reset()
					return
				}

			}(n)
		}
	}
}
