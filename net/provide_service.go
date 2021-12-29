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
	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/crypto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/kms"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/types"
)

const (
	defaultGasPrice = 1
	maxUint64       = 1<<64 - 1
)

var (
	metricKeyMemory   = "node_memory_MemAvailable_bytes"
	metricKeyLoadAvg  = "node_load15"
	metricKeyCPUCount = "node_cpu_count"
	metricKeySpace    = "node_filesystem_free_bytes"
)

func (rh RoutedHost) SendProvideService( /* reg *prometheus.Registry */ ) {

	var (
		nodeID      proto.NodeID
		privateKey  *asymmetric.PrivateKey
		err         error
		minerAddr   proto.AccountAddress
	)

	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		log.WithError(err).Error("get local node id failed")
		return
	}

	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		log.WithError(err).Error("get local private key failed")
		return
	}

	if minerAddr, err = crypto.PubKeyHash(privateKey.PubKey()); err != nil {
		log.WithError(err).Error("get miner account address failed")
		return
	}

	var (
		nonceReq  = new(types.NextAccountNonceReq)
		nonceResp = new(types.NextAccountNonceResp)
		req       = new(types.AddTxReq)
		resp      = new(types.AddTxResp)
	)

	nonceReq.Addr = minerAddr

	if err = rh.RequestPB("MCC.NextAccountNonce", &nonceReq, &nonceResp); err != nil {
		// allocate nonce failed
		log.WithError(err).Error("allocate nonce for transaction failed")
		return
	}

	tx := types.NewProvideService(
		&types.ProvideServiceHeader{
			GasPrice:      defaultGasPrice,
			TokenType:     types.Particle,
			NodeID:        nodeID,
		},
	)

	if conf.GConf.Miner != nil && len(conf.GConf.Miner.TargetUsers) > 0 {
		tx.ProvideServiceHeader.TargetUser = conf.GConf.Miner.TargetUsers
	}

	//TODO
	//tx.Nonce = nonceResp.Nonce

	if err = tx.Sign(privateKey); err != nil {
		log.WithError(err).Error("sign provide service transaction failed")
		return
	}

	req.TTL = 1
	req.Tx = tx

	if err = rh.RequestPB("MCC.AddTx", &req, &resp); err != nil {
		// add transaction failed
		log.WithError(err).Error("send provide service transaction failed")
		return
	}
}
