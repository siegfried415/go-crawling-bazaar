
package node 

import (
        //"fmt"
        //"syscall"
        //"time"

        //"github.com/pkg/errors"
        //"golang.org/x/crypto/ssh/terminal"

        //"github.com/siegfried415/gdf/api"
        //pb "github.com/siegfried415/gdf/presbyterian"
        "github.com/siegfried415/gdf-rebuild/conf"
        //"github.com/siegfried415/gdf/crypto/kms"
        "github.com/siegfried415/gdf-rebuild/proto"
        //"github.com/siegfried415/gdf/route"
        //rpc "github.com/siegfried415/gdf/rpc/mux"
        "github.com/siegfried415/gdf-rebuild/types"
        //"github.com/siegfried415/gdf/utils"
        //"github.com/siegfried415/gdf/utils/log"
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

