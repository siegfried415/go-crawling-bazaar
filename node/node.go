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
	//"expvar"
	"fmt"
	//"syscall"
	"time"

	//"golang.org/x/crypto/ssh/terminal"

	"github.com/siegfried415/gdf-rebuild/conf"
	//"github.com/siegfried415/gdf-rebuild/kms"
	//"github.com/siegfried415/gdf-rebuild/route"
	//"github.com/siegfried415/gdf-rebuild/rpc"
	//"github.com/siegfried415/gdf-rebuild/rpc/mux"

	//wyong, 20201125 
	//"github.com/siegfried415/gdf-rebuild/utils"

	//"github.com/siegfried415/gdf-rebuild/utils/log"

	//wyong, 20200911 
	//"github.com/pkg/errors"

	//"github.com/ipfs/go-merkledag"
        //"github.com/ipfs/go-bitswap"
        //bsnet "github.com/ipfs/go-bitswap/network"
        //bserv "github.com/ipfs/go-blockservice"
        //"github.com/ipfs/go-datastore"
        //dss "github.com/ipfs/go-datastore/sync"
        //bstore "github.com/ipfs/go-ipfs-blockstore"

	//"github.com/libp2p/go-libp2p-core/crypto"
        //"github.com/libp2p/go-libp2p"
        //"github.com/libp2p/go-libp2p-core/host"
        //"github.com/libp2p/go-libp2p-core/routing"
        //dht "github.com/libp2p/go-libp2p-kad-dht"
        //dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"

	"context"
	//"fmt"
	//"os"
	//"reflect"
	//"runtime"
	//"time"

	//bserv "github.com/ipfs/go-blockservice"
	//"github.com/ipfs/go-hamt-ipld"
	logging "github.com/ipfs/go-log"
	//"github.com/libp2p/go-libp2p-core/host"
	//"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"

	//"github.com/siegfried415/gdf-rebuild/address"
	//"github.com/siegfried415/gdf-rebuild/chain"
	//"github.com/siegfried415/gdf-rebuild/clock"
	//"github.com/siegfried415/gdf-rebuild/config"
	//"github.com/siegfried415/gdf-rebuild/consensus"
	//"github.com/siegfried415/gdf-rebuild/message"
	//"github.com/siegfried415/gdf-rebuild/metrics"
	//"github.com/siegfried415/gdf-rebuild/mining"
	"github.com/siegfried415/gdf-rebuild/net"
	//"github.com/siegfried415/gdf-rebuild/net/pubsub"

	//wyong, 20200410 
	//"github.com/siegfried415/gdf-rebuild/paths"
	//"github.com/siegfried415/gdf-rebuild/porcelain"
	
	//wyong, 20200410 
	//"github.com/siegfried415/gdf-rebuild/proofs/sectorbuilder"
	//"github.com/siegfried415/gdf-rebuild/protocol/block"
	//"github.com/siegfried415/gdf-rebuild/protocol/hello"

	//wyong, 20200410 
	//"github.com/siegfried415/gdf-rebuild/protocol/retrieval"
	//"github.com/siegfried415/gdf-rebuild/protocol/storage"

	//wyong, 20201022 
	"github.com/siegfried415/gdf-rebuild/proto"

	//wyong, 20201027 
	//"github.com/siegfried415/gdf-rebuild/repo"

	//"github.com/siegfried415/gdf-rebuild/state"
	//"github.com/siegfried415/gdf-rebuild/types"
	//"github.com/siegfried415/gdf-rebuild/version"
	//vmerr "github.com/siegfried415/gdf-rebuild/vm/errors"

        //wyong, 20200212 
        //"github.com/siegfried415/callinfo"

	//wyong, 20200921 
	dag "github.com/siegfried415/gdf-rebuild/dag" 
	frontera "github.com/siegfried415/gdf-rebuild/frontera" 

	//wyong, 20201125 
	pb "github.com/siegfried415/gdf-rebuild/presbyterian" 
)

const (
	mwMinerAddr         = "service:miner:addr"
	mwMinerExternalAddr = "service:miner:addr:external"
	mwMinerNodeID       = "service:miner:node"
	mwMinerWallet       = "service:miner:wallet"
	mwMinerDiskRoot     = "service:miner:disk:root"
)

var log = logging.Logger("node") // nolint: deadcode

var (
	// ErrNoMinerAddress is returned when the node is not configured to have any miner addresses.
	ErrNoMinerAddress = errors.New("no miner addresses configured")
)

// Node represents a full Filecoin node.
type Node struct {
        Host     net.RoutedHost

	//wyong, 20201119 
        //PeerHost host.Host

	//wyong, 20201022 
	Role proto.ServerRole 

	// OfflineMode, when true, disables libp2p.
	OfflineMode bool

	// Clock is a clock used by the node for time.
	//Clock clock.Clock

	//VersionTable version.ProtocolVersionTable

	//PorcelainAPI *porcelain.API

	//wyong, 20201027
	// Repo is the repo this node was created with.
	//
	// It contains all persistent artifacts of the filecoin node.
	//Repo repo.Repo

	//Blockstore BlockstoreSubmodule

	//Network NetworkSubmodule

	//Messaging MessagingSubmodule

	//Chain ChainSubmodule

	//BlockMining BlockMiningSubmodule

	//wyong, 20200410 
	//StorageProtocol StorageProtocolSubmodule

	//wyong, 20200410 
	//RetrievalProtocol RetrievalProtocolSubmodule

	//wyong, 20200412 
	//SectorStorage SectorBuilderSubmodule

	//wyong, 20200413 
	//FaultSlasher FaultSlasherSubmodule

	//HelloProtocol HelloProtocolSubmodule

	StorageNetworking StorageNetworkingSubmodule

	//todo, wyong, 20201021 
	//Wallet WalletSubmodule

	//wyong, 20201125 
	Chain *pb.Chain 

	//wyong, 20200921 
	DAG *dag.DAG 

	//wyong, 20201107 
	//Net *net.Network

	//wyong, 20200921 
	Frontera *frontera.Frontera 
}

/* wyong, 20201003 
// Start boots up the node.
func (node *Node) Start(ctx context.Context) error {

	if !node.OfflineMode {
		// Start bootstrapper.
		//context.Background()->ctx, wyong, 20201029 
		//node.Network.Bootstrapper.Start(ctx)


		// Register peer tracker disconnect function with network.
		net.TrackerRegisterDisconnect(node.Network.host.Network(), node.Network.PeerTracker)

		// todo, wyong, 20200921 
		//// Start up 'hello' handshake service
		//helloCallback := func(ci *types.ChainInfo) {
		//	node.Network.PeerTracker.Track(ci)
		//	// TODO Implement principled trusting of ChainInfo's
		//	// to address in #2674
		//	trusted := true
		//	err := node.Chain.Syncer.HandleNewTipSet(context.Background(), ci, trusted)
		//	if err != nil {
		//		log.Infof("error handling tipset from hello %s: %s", ci, err)
		//		return
		//	}
		//	// For now, consider the initial bootstrap done after the syncer has (synchronously)
		//	// processed the chain up to the head reported by the first peer to respond to hello.
		//	// This is an interim sequence until a secure network bootstrap is implemented:
		//	// https://github.com/siegfried415/go-decentralized-frontera/issues/2674.
		//	// For now, we trust that the first node to respond will be a configured bootstrap node
		//	// and that we trust that node to inform us of the chain head.
		//	// TODO: when the syncer rejects too-far-ahead blocks received over pubsub, don't consider
		//	// sync done until it's caught up enough that it will accept blocks from pubsub.
		//	// This might require additional rounds of hello.
		//	// See https://github.com/siegfried415/go-decentralized-frontera/issues/1105
		//	node.Chain.ChainSynced.Done()
		//}

		//node.HelloProtocol.HelloSvc = hello.New(node.Host(), node.Chain.ChainReader.GenesisCid(), helloCallback, node.PorcelainAPI.ChainHead, node.Network.NetworkName)



		//// register the update function on the peer tracker now that we have a hello service
		//node.Network.PeerTracker.SetUpdateFn(func(ctx context.Context, p peer.ID) (*types.ChainInfo, error) {
		//	hmsg, err := node.HelloProtocol.HelloSvc.ReceiveHello(ctx, p)
		//	if err != nil {
		//		return nil, err
		//	}
		//	return types.NewChainInfo(p, hmsg.HeaviestTipSetCids, hmsg.HeaviestTipSetHeight), nil
		//})


		//// Start heartbeats.
		//if err := node.setupHeartbeatServices(ctx); err != nil {
		//	return errors.Wrap(err, "failed to start heartbeat services")
		//}
	}

	return nil 

}
*/

// Stop initiates the shutdown of the node.
func (node *Node) Stop(ctx context.Context) {
	//todo, wyong, 20201125 
        //node.Chain.ChainReader.HeadEvents().Unsub(node.Chain.HeaviestTipSetCh)
        //node.StopMining(ctx)

        //node.cancelSubscriptions()
        //node.Chain.ChainReader.Stop()

	switch node.Role  {
	case proto.Client:

	case proto.Leader: fallthrough 
	case proto.Follower: 
		node.Chain.Stop()

	case proto.Miner: 
		node.Frontera.Shutdown()

	case proto.Unknown:
                return 
	}	


        if err := node.Host.Close(); err != nil {
                fmt.Printf("error closing host: %s\n", err)
        }

	//wyong, 20201027 
        //if err := node.Repo.Close(); err != nil {
        //        fmt.Printf("error closing repo: %s\n", err)
        //}

        //node.Network.Bootstrapper.Stop()
        fmt.Println("stopping filecoin :(")
}


func (node *Node) Start(ctx context.Context) error {

	fmt.Printf("Node/Start(10)\n") 

	//todo, wyong, 20201021 
	// set generate key pair config
	//conf.GConf.GenerateKeyPair = genKeyPair

	if !node.OfflineMode {

		// Start bootstrapper.
		//context.Background()->ctx, wyong, 20201029 
		//node.Network.Bootstrapper.Start(ctx)

		//todo, wyong, 20201015 
		// Register peer tracker disconnect function with network.
		//net.TrackerRegisterDisconnect(node.Network.host.Network(), node.Network.PeerTracker)
	}

	fmt.Printf("Node/Start(20)\n") 

	//wyong, 20201125
	switch node.Role {
	case proto.Client: 
		fmt.Printf("Node/Start(30)\n") 
		return nil

	case proto.Leader: 
		fallthrough
	case proto.Follower:
		fmt.Printf("Node/Start(40)\n") 
		node.Chain.Start()

	case proto.Miner: 
		fmt.Printf("Node/Start(50)\n") 
		go func() {
			err := node.Host.RegisterNodeToPB(30 * time.Second)
			if err != nil {
				log.Fatalf("register node to BP failed: %v", err)
			}
		}()


		fmt.Printf("Node/Start(60)\n") 
		// start periodic provide service transaction generator
		go func() {
			tick := time.NewTicker(conf.GConf.Miner.ProvideServiceInterval)
			defer tick.Stop()

			for {
				//wyong, 20201021 
				//sendProvideService(reg)
				node.Host.SendProvideService()

				select {
				//case <-stopCh:
				//	return
				case <-tick.C:
				}
			}
		}()

		fmt.Printf("Node/Start(70)\n") 
		if err := node.Frontera.Start(ctx); err != nil {
			// FIXME(auxten): if restart all miners with the same db,
			// miners will fail to start
			time.Sleep(10 * time.Second)
			//log.WithError(err).Fatal("start dbms failed")
		}

		fmt.Printf("Node/Start(80)\n") 

	case proto.Unknown:
		fmt.Printf("Node/Start(90)\n") 
                return nil

	}


	//todo, wyong, 20201125 
	//<-utils.WaitForExit()
	//utils.StopProfile()

	fmt.Printf("Node/Start(100)\n") 
	return nil 
}

