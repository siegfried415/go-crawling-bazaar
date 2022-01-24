/*
 * Copyright 2018 The CovenantSQL Authors.
 * Copyright 2022 https://github.com/siegfried415
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
	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/kms"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

// InitKMS inits nasty stuff, only for testing.
func InitKMS(host RoutedHost, PubKeyStoreFile string) {
	//initResolver()
	kms.InitPublicKeyStore(PubKeyStoreFile, nil)
	if conf.GConf.KnownNodes != nil {
		for _, n := range conf.GConf.KnownNodes {
			rawNodeID := &n.ID 
			log.WithFields(log.Fields{
				"node": rawNodeID.String(),
				"addr": n.Addr,
			}).Debug("set node addr")

			router := host.Router()
			router.SetNodeAddrCache(rawNodeID, n.Addr)

			node := &proto.Node{
				ID:         n.ID,
				Addr:       n.Addr,
				DirectAddr: n.DirectAddr,
				PublicKey:  n.PublicKey,
				Role:       n.Role,
			}
			log.WithField("node", node).Debug("known node to set")
			err := kms.SetNode(node)
			if err != nil {
				log.WithField("node", node).WithError(err).Error("set node failed")
			}

			if n.ID == conf.GConf.ThisNodeID {
				kms.SetLocalNodeID([]byte(rawNodeID.String()))
			}
		}
	}
	log.Debugf("AllNodes:\n %#v\n", conf.GConf.KnownNodes)
}
