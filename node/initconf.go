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
	"github.com/siegfried415/go-crawling-bazaar/conf"
	//log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/kms"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

func GetPeersFromConf( publicKeystorePath string) (peers *proto.Peers, err error) {
	privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		//log.WithError(err).Fatal("get local private key failed")
		return nil, err 
	}

	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:   1,
			Leader: conf.GConf.PB.NodeID,
		},
	}

	if conf.GConf.KnownNodes != nil {
		for _, n := range conf.GConf.KnownNodes {
			if n.Role == proto.Leader || n.Role == proto.Follower {
				//todo
				//FIXME all KnownNodes
				//conf.GConf.KnownNodes[i].PublicKey = kms.PB.PublicKey
				peers.Servers = append(peers.Servers, n.ID)
			}
		}
	}

	//log.Debugf("AllNodes:\n %#v\n", conf.GConf.KnownNodes)
	err = peers.Sign(privateKey)
	if err != nil {
		//log.WithError(err).Error("sign peers failed")
		return nil, err
	}

	//log.Debugf("peers:\n %#v\n", peers)
	return peers, nil 
}

func (node *Node) InitNodePeers( publicKeystorePath string) (nodes *[]proto.Node, thisNode *proto.Node, err error) {
	//route.initResolver()
	kms.InitPublicKeyStore(publicKeystorePath, nil)

	// set p route and public keystore
	if conf.GConf.KnownNodes != nil {
		for i, p := range conf.GConf.KnownNodes {
			//rawNodeIDHash, err := hash.NewHashFromStr(string(p.ID))
			//if err != nil {
			//	//log.WithError(err).Error("load hash from node id failed")
			//	return nil, nil, err
			//}

			//log.WithFields(log.Fields{
			//	"node": rawNodeIDHash.String(),
			//	"addr": p.Addr,
			//}).Debug("set node addr")
			//rawNodeID := &proto.RawNodeID{Hash: *rawNodeIDHash}

			//rawNodeID -> &p.ID
			router := node.Host.Router()
			router.SetNodeAddrCache(&p.ID, p.Addr)

			nd := &proto.Node{
				ID:         p.ID,
				Addr:       p.Addr,
				DirectAddr: p.DirectAddr,
				PublicKey:  p.PublicKey,
				//Nonce:      p.Nonce,
				Role:       p.Role,
			}
			err = kms.SetNode(nd)
			if err != nil {
				//log.WithField("node", node).WithError(err).Error("set node failed")
			}
			if p.ID == proto.NodeID(node.Host.ID()) {
				//kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &p.Nonce)
				kms.SetLocalNodeID([]byte(p.ID.String()))
				thisNode = &conf.GConf.KnownNodes[i]
			}
		}
	}

	return
}
