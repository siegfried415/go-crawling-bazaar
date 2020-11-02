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
	"github.com/siegfried415/gdf-rebuild/utils"
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
        "github.com/libp2p/go-libp2p-core/host"
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
        Host     host.Host

        PeerHost host.Host

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

	Network NetworkSubmodule

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

	//wyong, 20200921 
	DAG *dag.DAG 
	Net *net.Network

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
        //node.Chain.ChainReader.HeadEvents().Unsub(node.Chain.HeaviestTipSetCh)
        //node.StopMining(ctx)

        //node.cancelSubscriptions()
        //node.Chain.ChainReader.Stop()

        // wyong, 20200410
        //if node.SectorBuilder() != nil {
        //        if err := node.SectorBuilder().Close(); err != nil {
        //                fmt.Printf("error closing sector builder: %s\n", err)
        //        }
        //        node.SectorStorage.sectorBuilder = nil
        //}

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

/*
// Host returns the nodes host.
func (node *Node) Host() host.Host {
        return node.Host
}
*/

func (node *Node) Start(ctx context.Context) error {

	// init profile, if cpuProfile, memProfile length is 0, nothing will be done
	//_ = utils.StartProfile(cpuProfile, memProfile)

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


	// start rpc
	//var (
	//	server *mux.Server
	//	direct *rpc.Server
	//)
	
	//if server, direct, err = initNode(); err != nil {
	//	log.WithError(err).Fatal("init node failed")
	//}

	//wyong, 20201020 
	err := net.RegisterNodeToPB(node.Host, 30 * time.Second)
	if err != nil {
		log.Fatalf("register node to BP failed: %v", err)
	}

	//initMetrics()

	// stop channel for all daemon routines
	stopCh := make(chan struct{})
	defer close(stopCh)

	//if len(profileServer) > 0 {
	//	go func() {
	//		log.Println(http.ListenAndServe(profileServer, nil))
	//	}()
	//}

	//if len(metricWeb) > 0 {
	//	err = metric.InitMetricWeb(metricWeb)
	//	if err != nil {
	//		log.Errorf("start metric web server on %s failed: %v", metricWeb, err)
	//		os.Exit(-1)
	//	}
	//}

	// start prometheus collector
	//reg := metric.StartMetricCollector()

	// start periodic provide service transaction generator
	go func() {
		tick := time.NewTicker(conf.GConf.Miner.ProvideServiceInterval)
		defer tick.Stop()

		for {
			//wyong, 20201021 
			//sendProvideService(reg)
			sendProvideService(node.Host )

			select {
			case <-stopCh:
				return
			case <-tick.C:
			}
		}
	}()

	// start periodic disk usage metric update
	//go func() {
	//	for {
	//		err := collectDiskUsage()
	//		if err != nil {
	//			log.WithError(err).Error("collect disk usage failed")
	//		}
	//
	//		select {
	//		case <-stopCh:
	//			return
	//		case <-time.After(conf.GConf.Miner.DiskUsageInterval):
	//		}
	//	}
	//}()

	// start rpc server
	//go func() {
	//	server.Serve()
	//}()
	//defer server.Stop()

	// start direct rpc server
	//if direct != nil {
	//	go func() {
	//		direct.Serve()
	//	}()
	//	defer direct.Stop()
	//}

	//int&start network, wyong, 20200916 
	//g, n, err := initNetwork() 
	//if err != nil {
	//	log.WithError(err).Fatal("init network failed")
	//}


	// start frontera 
	//var f *frontera.Frontera 

	//wyong, 20201021 
	//if f, err = startFrontera(server, direct, func() {
	//	sendProvideService(reg)
	//}); err != nil {
	if err = node.Frontera.Start(ctx); err != nil {
		// FIXME(auxten): if restart all miners with the same db,
		// miners will fail to start
		time.Sleep(10 * time.Second)
		//log.WithError(err).Fatal("start dbms failed")
	}

	defer node.Frontera.Shutdown()

	//todo, wyong, 20200723 
        //cancelFunc := startAdapterServer(f, g, n, adapterAddr, "")
        //ExitIfErrors()
        //defer cancelFunc()


	//if metricLog {
	//	go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.StandardLogger())
	//}

	//if metricGraphite != "" {
	//	addr, err := net.ResolveTCPAddr("tcp", metricGraphite)
	//	if err != nil {
	//		log.WithError(err).Error("resolve metric graphite server addr failed")
	//		return
	//	}
	//	minerName := fmt.Sprintf("miner-%s", conf.GConf.ThisNodeID[len(conf.GConf.ThisNodeID)-5:])
	//	go graphite.Graphite(metrics.DefaultRegistry, 5*time.Second, minerName, addr)
	//}

	//if traceFile != "" {
	//	f, err := os.Create(traceFile)
	//	if err != nil {
	//		log.WithError(err).Fatal("failed to create trace output file")
	//	}
	//	defer func() {
	//		if err := f.Close(); err != nil {
	//			log.WithError(err).Fatal("failed to close trace file")
	//		}
	//	}()
	//
	//	if err := trace.Start(f); err != nil {
	//		log.WithError(err).Fatal("failed to start trace")
	//	}
	//	defer trace.Stop()
	//}

	<-utils.WaitForExit()
	utils.StopProfile()

	return nil 
}

/*
func setupServer() (server *mux.Server, direct *rpc.Server,  err error) {
	var masterKey []byte
	if !conf.GConf.UseTestMasterKey {
		// read master key
		fmt.Print("Type in Master key to continue: ")
		masterKey, err = terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			fmt.Printf("Failed to read Master Key: %v", err)
		}
		fmt.Println("")
	}

	// todo, wyong, 20201001 
	//if err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey); err != nil {
	//	log.WithError(err).Error("init local key pair failed")
	//	return
	//}

	log.Info("init routes")

	//todo, wyong, 20201001 
	// init kms routing
	//route.InitKMS(conf.GConf.PubKeyStoreFile)

	err = mux.RegisterNodeToBP(30 * time.Second)
	if err != nil {
		log.Fatalf("register node to BP failed: %v", err)
	}

	// init server
	//utils.RemoveAll(conf.GConf.PubKeyStoreFile + "*")
	//if server, err = createServer(
	//	conf.GConf.PrivateKeyFile, masterKey, conf.GConf.ListenAddr); err != nil {
	//	log.WithError(err).Error("create server failed")
	//	return
	//}
	//if direct, err = createDirectServer(
	//	conf.GConf.PrivateKeyFile, masterKey, conf.GConf.ListenDirectAddr); err != nil {
	//	log.WithError(err).Error("create direct server failed")
	//	return
	//}

	return
}

func createServer(privateKeyPath string, masterKey []byte, listenAddr string) (server *mux.Server, err error) {
	server = mux.NewServer()
	err = server.InitRPCServer(listenAddr, privateKeyPath, masterKey)
	return
}

func createDirectServer(privateKeyPath string, masterKey []byte, listenAddr string) (server *rpc.Server, err error) {
	if listenAddr == "" {
		return nil, nil
	}
	server = rpc.NewServer()
	err = server.InitRPCServer(listenAddr, privateKeyPath, masterKey)
	return
}

func initMetrics() {
	if conf.GConf != nil {
		expvar.NewString(mwMinerAddr).Set(conf.GConf.ListenAddr)
		expvar.NewString(mwMinerExternalAddr).Set(conf.GConf.ExternalListenAddr)
		expvar.NewString(mwMinerNodeID).Set(string(conf.GConf.ThisNodeID))
		expvar.NewString(mwMinerWallet).Set(conf.GConf.WalletAddress)

		if conf.GConf.Miner != nil {
			expvar.NewString(mwMinerDiskRoot).Set(conf.GConf.Miner.RootDir)
		}
	}
}

*/
