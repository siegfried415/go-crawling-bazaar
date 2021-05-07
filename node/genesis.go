
package node 

import (
        //"fmt"
        //"syscall"
        //"time"

        //"github.com/pkg/errors"
        //"golang.org/x/crypto/ssh/terminal"

        //"github.com/siegfried415/go-crawling-bazaar/api"
        //pb "github.com/siegfried415/go-crawling-bazaar/presbyterian"
        "github.com/siegfried415/go-crawling-bazaar/conf"
        //"github.com/siegfried415/go-crawling-bazaar/crypto/kms"
        "github.com/siegfried415/go-crawling-bazaar/proto"
        //"github.com/siegfried415/go-crawling-bazaar/route"
        //rpc "github.com/siegfried415/go-crawling-bazaar/rpc/mux"
        "github.com/siegfried415/go-crawling-bazaar/types"
        //"github.com/siegfried415/go-crawling-bazaar/utils"
        //"github.com/siegfried415/go-crawling-bazaar/utils/log"
)


func loadGenesis() (genesis *types.BPBlock, err error) {
        genesisInfo := conf.GConf.BP.BPGenesis
        //log.WithField("config", genesisInfo).Info("load genesis config")

        genesis = &types.BPBlock{
                SignedHeader: types.BPSignedHeader{
                        BPHeader: types.BPHeader{
                                Version:   genesisInfo.Version,
                                Timestamp: genesisInfo.Timestamp,
                        },
                },
        }

        for _, ba := range genesisInfo.BaseAccounts {
                //log.WithFields(log.Fields{
                //        "address":             ba.Address.String(),
                //        "stableCoinBalance":   ba.StableCoinBalance,
                //        "covenantCoinBalance": ba.CovenantCoinBalance,
                //}).Debug("setting one balance fixture in genesis block")
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

