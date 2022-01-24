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
        log "github.com/siegfried415/go-crawling-bazaar/utils/log"
        "github.com/siegfried415/go-crawling-bazaar/proto"
        "github.com/siegfried415/go-crawling-bazaar/types"
)


func loadGenesis() (genesis *types.PBBlock, err error) {
        genesisInfo := conf.GConf.PB.PBGenesis
        log.WithField("config", genesisInfo).Info("load genesis config")

        genesis = &types.PBBlock{
                SignedHeader: types.PBSignedHeader{
                        PBHeader: types.PBHeader{
                                Version:   genesisInfo.Version,
                                Timestamp: genesisInfo.Timestamp,
                        },
                },
        }

        for _, ba := range genesisInfo.BaseAccounts {
                log.WithFields(log.Fields{
                        "address":             ba.Address.String(),
                        "stableCoinBalance":   ba.StableCoinBalance,
                        "covenantCoinBalance": ba.CovenantCoinBalance,
                }).Debug("setting one balance fixture in genesis block")

                genesis.Transactions = append(genesis.Transactions, types.NewBaseAccount(
                        &types.Account{
                                Address:      proto.AccountAddress(ba.Address),
                                TokenBalance: [types.SupportTokenNumber]uint64{ba.StableCoinBalance, ba.CovenantCoinBalance},
                        }))
        }

        // Rewrite genesis merkle and block hash
        if err = genesis.SetHash(); err != nil {
                return
        }
        return
}

