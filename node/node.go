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

package node

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/siegfried415/go-crawling-bazaar/conf"
	dag "github.com/siegfried415/go-crawling-bazaar/dag" 
	frontera "github.com/siegfried415/go-crawling-bazaar/frontera" 
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	pb "github.com/siegfried415/go-crawling-bazaar/presbyterian" 
)

const (
	mwMinerAddr         = "service:miner:addr"
	mwMinerExternalAddr = "service:miner:addr:external"
	mwMinerNodeID       = "service:miner:node"
	mwMinerWallet       = "service:miner:wallet"
	mwMinerDiskRoot     = "service:miner:disk:root"
)

var (
	// ErrNoMinerAddress is returned when the node is not configured to have any miner addresses.
	ErrNoMinerAddress = errors.New("no miner addresses configured")
)

// Node represents a full Filecoin node.
type Node struct {
        Host     net.RoutedHost
	Role proto.ServerRole 

	// OfflineMode, when true, disables libp2p.
	OfflineMode bool

	Chain *pb.Chain 
	DAG *dag.DAG 
	Frontera *frontera.Frontera 
}

// Stop initiates the shutdown of the node.
func (node *Node) Stop(ctx context.Context) {
	//todo
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
                log.Debugf("error closing host: %s\n", err)
        }

        //node.Network.Bootstrapper.Stop()
        fmt.Println("stopping filecoin :(")
}


func (node *Node) Start(ctx context.Context) error {
	//todo
	// set generate key pair config
	//conf.GConf.GenerateKeyPair = genKeyPair

	if !node.OfflineMode {
		// Start bootstrapper.
		//context.Background()->ctx
		//node.Network.Bootstrapper.Start(ctx)

		//todo
		// Register peer tracker disconnect function with network.
		//net.TrackerRegisterDisconnect(node.Network.host.Network(), node.Network.PeerTracker)
	}

	switch node.Role {
	case proto.Client: 
		return nil

	case proto.Leader, 
	     proto.Follower:
		node.Chain.Start()

	case proto.Miner: 
		go func() {
			err := node.Host.RegisterNodeToPB(30 * time.Second)
			if err != nil {
				log.Fatalf("register node to Presbyterian failed: %v", err)
			}
		}()


		// start periodic provide service transaction generator
		go func() {
			tick := time.NewTicker(conf.GConf.Miner.ProvideServiceInterval)
			defer tick.Stop()

			for {
				//sendProvideService(reg)
				node.Host.SendProvideService()

				select {
				//case <-stopCh:
				//	return
				case <-tick.C:
				}
			}
		}()

		if err := node.Frontera.Init(); err != nil {
			err = errors.Wrap(err, "init Frontera failed")
			return err 
		}

		if err := node.Frontera.Start(ctx); err != nil {
			// FIXME(auxten): if restart all miners with the same db,
			// miners will fail to start
			time.Sleep(10 * time.Second)
			log.WithError(err).Fatal("start dbms failed")
		}


	case proto.Unknown:
                return nil

	}

	//todo
	//<-utils.WaitForExit()
	//utils.StopProfile()

	return nil 
}

