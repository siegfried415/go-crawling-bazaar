
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
	"syscall"
	"time"
	"context" 

	//wyong, 20201021
	ecdsa "crypto/ecdsa"

	//wyong, 20200911 
	"github.com/pkg/errors"

	//wyong, 20201021
        "golang.org/x/crypto/ssh/terminal"

	"github.com/ipfs/go-merkledag"
        "github.com/ipfs/go-bitswap"
        bsnet "github.com/ipfs/go-bitswap/network"
        bserv "github.com/ipfs/go-blockservice"
        //"github.com/ipfs/go-datastore"
        //dss "github.com/ipfs/go-datastore/sync"
	bstore "github.com/ipfs/go-ipfs-blockstore"
        //offroute "github.com/ipfs/go-ipfs-routing/offline"

	//wyong, 20201021 
	exchange "github.com/ipfs/go-ipfs-exchange-interface"

        ma "github.com/multiformats/go-multiaddr"

	//wyong, 20201002 
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"

        "github.com/libp2p/go-libp2p"
        "github.com/libp2p/go-libp2p-core/host"
        "github.com/libp2p/go-libp2p-core/routing"
        //rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
        dht "github.com/libp2p/go-libp2p-kad-dht"
        dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
        autonatsvc "github.com/libp2p/go-libp2p-autonat-svc"
        circuit "github.com/libp2p/go-libp2p-circuit"

	//wyong, 20200914
	"github.com/siegfried415/gdf-rebuild/crypto/asymmetric" 
	net "github.com/siegfried415/gdf-rebuild/net" 
	dag "github.com/siegfried415/gdf-rebuild/dag" 
	repo "github.com/siegfried415/gdf-rebuild/repo" 

	//wyong, 20201026 
	//clock "github.com/siegfried415/gdf-rebuild/clock" 

	//wyong, 20200924 
	frontera "github.com/siegfried415/gdf-rebuild/frontera"

	//wyong, 20201014 
        //"github.com/siegfried415/gdf-rebuild/wallet"

	//wyong, 20201021 
	conf "github.com/siegfried415/gdf-rebuild/conf" 
	kms "github.com/siegfried415/gdf-rebuild/kms" 
	route "github.com/siegfried415/gdf-rebuild/route" 
        pb "github.com/siegfried415/gdf-rebuild/presbyterian"

)


const (
        dhtGossipTimeout = time.Second * 20
)


// Builder is a helper to aid in the construction of a filecoin node.
type Builder struct {
        BlockTime   time.Duration
        Libp2pOpts  []libp2p.Option
        OfflineMode bool

        //wyong, 20200412
        //Verifier    verification.Verifier

        //Rewarder    consensus.BlockRewarder
        Repo        repo.Repo
        IsRelay     bool

	//wyong, 20201026 
        //Clock       clock.Clock
}


// BuilderOpt is an option for building a filecoin node.
type BuilderOpt func(*Builder) error

// OfflineMode enables or disables offline mode.
func OfflineMode(offlineMode bool) BuilderOpt {
        return func(c *Builder) error {
                c.OfflineMode = offlineMode
                return nil
        }
}

// IsRelay configures node to act as a libp2p relay.
func IsRelay() BuilderOpt {
        return func(c *Builder) error {
                c.IsRelay = true
                return nil
        }
}


// New creates a new node.
func New(ctx context.Context, opts ...BuilderOpt) (*Node, error) {
        n := &Builder{}
        for _, o := range opts {
                if err := o(n); err != nil {
                        return nil, err
                }
        }

        return n.build(ctx)
}


// BlockTime sets the blockTime.
func BlockTime(blockTime time.Duration) BuilderOpt {
        return func(c *Builder) error {
                c.BlockTime = blockTime
                return nil
        }
}

// Libp2pOptions returns a node config option that sets up the libp2p node
func Libp2pOptions(opts ...libp2p.Option) BuilderOpt {
        return func(nc *Builder) error {
                // Quietly having your options overridden leads to hair loss
                if len(nc.Libp2pOpts) > 0 {
                        panic("Libp2pOptions can only be called once")
                }
                nc.Libp2pOpts = opts
                return nil
        }
}

/*wyong, 20201026 
// ClockConfigOption returns a function that sets the clock to use in the node.
func ClockConfigOption(clk clock.Clock) BuilderOpt {
        return func(c *Builder) error {
                c.Clock = clk
                return nil
        }
}
*/


type blankValidator struct{}
func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func (nc *Builder) build(ctx context.Context) (*Node, error ) {
        //if nc.Repo == nil {
        //        nc.Repo = repo.NewInMemoryRepo()
        //}
        //if nc.Clock == nil {
        //        nc.Clock = clock.NewSystemClock()
        //}

	var (
		masterKey []byte

		//wyong, 20201021 
		bswap exchange.Interface 	
		g *dag.DAG 
		n *net.Network
		f *frontera.Frontera 
		
		//wyong, 20201021
		err error 
	)

	if !conf.GConf.UseTestMasterKey {
		// read master key
		fmt.Print("Type in Master key to continue: ")
		masterKey, err = terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			fmt.Printf("Failed to read Master Key: %v", err)
		}
		fmt.Println("")
	}

	if err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey); err != nil {
		//log.WithError(err).Error("init local key pair failed")
		return nil, err 
	}

        //wyong, 20201015  init nodes
        //log.WithField("node", nodeID).Info("init peers")
        _, peers, /* thisNode */ _, err := initNodePeers(conf.GConf.ThisNodeID, conf.GConf.PubKeyStoreFile)
        if err != nil {
                //log.WithError(err).Error("init nodes and peers failed")
                return nil, err 
        }

	//log.Info("init routes")

	// init kms routing
	route.InitKMS(conf.GConf.PubKeyStoreFile)


        bs := bstore.NewBlockstore(nc.Repo.Datastore())
        validator := blankValidator{}

	//wyong, 20200911 
        var router routing.Routing
        var peerHost host.Host

	//fmt.Printf("dag/BuildDAG(10), swarmAddress=%s\n", swarmAddress )
	//ds := dss.MutexWrap(datastore.NewMapDatastore())
        //bs := bstore.NewBlockstore(ds)

	//wyong, 20200921 
	network := "gdf"

	//fmt.Printf("dag/BuildDAG(20)\n") 
        //if !nc.OfflineMode {
	makeDHT := func(h host.Host) (routing.Routing, error) {
		//fmt.Printf("dag/makeDGT(10),host=%s\n", h) 

		/* wyong, 20200921 
		baseOpts := []dht.Option{
			dht.ProtocolPrefix("/gdf/dht"),
			dht.NamespacedValidator("v", blankValidator{}),
			dht.DisableAutoRefresh(),
			dht.Mode(dht.ModeServer), 
		}

		//fmt.Printf("dag/makeDGT(20)\n") 
		r, err := dht.New(ctx, h, baseOpts..., )
		*/

		r, err := dht.New(
			ctx,
			h,
			dhtopts.Datastore(nc.Repo.Datastore()),
			dhtopts.NamespacedValidator("v", validator),
			dhtopts.Protocols(net.FilecoinDHT(network)),
		)

		if err != nil {
			fmt.Printf("dag/makeDGT(25)\n") 
			return nil, errors.Wrap(err, "failed to setup routing")
		}
		//fmt.Printf("dag/makeDGT(30)\n") 
		router = r
		return r, err
	}

	fmt.Printf("dag/BuildDAG(30)\n") 
	//var err error
	privKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		fmt.Printf("dag/BuildDAG(35), err=%s\n", err.Error()) 
		return nil, err
	}

	//fmt.Printf("dag/BuildDAG(40)\n") 
	peerHost, err = nc.buildHost(ctx, conf.GConf.Swarm.Address, privKey, makeDHT)
	if err != nil {
		fmt.Printf("dag/BuildDAG(45), err=%s\n", err.Error()) 
		return nil, err
	}

	//} else {
        //      router = offroute.NewOfflineRouter(nc.Repo.Datastore(), validator)
        //        peerHost = rhost.Wrap(noopLibP2PHost{}, router)
	//}

	//todo, wyong, 20201015 
        // set up peer tracking
        //peerTracker := net.NewPeerTracker(peerHost.ID())


	//todo, wyong, 20201007 
	switch conf.RoleTag[0] {
	
	case conf.ClientBuildTag[0]:

	case conf.BlockProducerBuildTag[0]:

		//wyong, 20201021 
		genesis, err := loadGenesis()
		if err != nil {
			return nil, err 
		}

               	// init storage
                //log.Info("init storage")
                var st *LocalStorage
                if st, err = initStorage(conf.GConf.DHTFileName); err != nil {
                        //log.WithError(err).Error("init storage failed")
                        return nil, err
                }

                // init dht node server
                //log.Info("init consistent runtime")
                kvServer := NewKVServer(peerHost, conf.GConf.ThisNodeID, peers, st, dhtGossipTimeout)
                dht, err := route.NewDHTService(conf.GConf.DHTFileName, kvServer, true)
                if err != nil {
                        //log.WithError(err).Error("init consistent hash failed")
                        return nil, err
                }
                defer kvServer.Stop()

                // register dht service rpc
                //log.Info("register dht service rpc")
                //err = server.RegisterService(route.DHTRPCName, dht)
                //if err != nil {
                //        log.WithError(err).Error("register dht service failed")
                //        return err
                //}

                // set consistent handler to local storage
                kvServer.storage.consistent = dht.Consistent

                // register gossip service rpc
                /* gossipService := */  NewGossipService(kvServer)
                //log.Info("register dht gossip service rpc")
                //err = server.RegisterService(route.DHTGossipRPCName, gossipService)
                //if err != nil {
                //        log.WithError(err).Error("register dht gossip service failed")
                //        return err
                //}


		// init main chain service
		//log.Info("register main chain service rpc")
		chainConfig := &pb.Config{
			Mode:           pb.BPMode,
			Genesis:        genesis,
			DataFile:       conf.GConf.BP.ChainFileName,

			//wyong, 20201015 
			//Server:         server,
			Host: peerHost, 

			Peers:          peers,
			NodeID:         conf.GConf.ThisNodeID,	//nodeID,
			Period:         conf.GConf.BPPeriod,
			Tick:           conf.GConf.BPTick,
			BlockCacheSize: 1000,
		}
		chain, err := pb.NewChain(chainConfig)
		if err != nil {
			//log.WithError(err).Error("init chain failed")
			return nil, err
		}
		chain.Start()
		defer chain.Stop()


	case conf.MinerBuildTag[0]: 
		//fmt.Printf("dag/BuildDAG(50)\n") 
		// set up bitswap
		nwork := bsnet.NewFromIpfsHost(peerHost, router)
		//fmt.Printf("dag/BuildDAG(60)\n") 
		bswap = bitswap.New(ctx, nwork, bs)
		//fmt.Printf("dag/BuildDAG(70)\n") 
		bservice := bserv.New(bs, bswap)

		//fmt.Printf("dag/BuildDAG(80)\n") 
		s := merkledag.NewDAGService(bservice) 
		//fmt.Printf("dag/BuildDAG(90)\n") 
		g = dag.NewDAG(s)


		fmt.Printf("dag/BuildDAG(100)\n") 
		n = net.New(peerHost, net.NewRouter(router))

		//wyong, 20200929 
		//RootDir = nc.Repo.Path()
		//if nc.Repo.Config().Mining != nil && !path.IsAbs(nc.Repo.Config().Mining.RootDir) {
		//	RootDir = path.Join(nc.Repo.Path(), nc.Repo.Config().Mining.RootDir)
		//}

		//create frontera, wyong, 20200924
		fcfg := &frontera.FronteraConfig{
			RootDir:          conf.GConf.Miner.RootDir,
			//Server:           server,
			//DirectServer:     direct,
			MaxReqTimeGap:    conf.GConf.Miner.MaxReqTimeGap,
			//OnCreateDatabase: onCreateDB,
		}

		f, err = frontera.NewFrontera(fcfg, peerHost )
		if err != nil {
			err = errors.Wrap(err, "create new Frontera failed")
			return nil, err 
		}

		if err = f.Init(); err != nil {
			err = errors.Wrap(err, "init Frontera failed")
			return nil, err 
		}

	case conf.UnknownBuildTag[0]:
	//default :
                return nil, err
	}	

	//wyong, 20201014 
	//create wallet 
        //backend, err := wallet.NewDSBackend(nc.Repo.WalletDatastore())
        //if err != nil {
        //        return nil, errors.Wrap(err, "failed to set up wallet backend")
        //}
        //fcWallet := wallet.New(backend)

        nd := &Node{
		//wyong, 20200924 
		Host : peerHost, 
                PeerHost: peerHost,

                //Clock:       nc.Clock,
                OfflineMode: nc.OfflineMode,
                Repo:        nc.Repo,
                Network: NetworkSubmodule{
                        //host:        peerHost,
                        //PeerHost:    peerHost,
                        NetworkName: network,

			//todo, wyong, 20201015 
                        //PeerTracker: peerTracker,

                        Router:      router,
                },

		//wyong, 20201014 
                //Wallet: WalletSubmodule{
                //        Wallet: fcWallet,
                //},


                //Messaging: MessagingSubmodule{
                //        Inbox:  inbox,
                //        Outbox: outbox,
                //},
	
                //Blockstore: BlockstoreSubmodule{
                //        blockservice: bservice,
                //        Blockstore:   bs,
                //        cborStore:    &ipldCborStore,
                //},

                StorageNetworking: StorageNetworkingSubmodule{
                        Exchange: bswap,
                },
		
                //Chain: ChainSubmodule{
                //        Fetcher:      fetcher,
                //        Consensus:    nodeConsensus,
                //        ChainReader:  chainStore,
                //        ChainSynced:  moresync.NewLatch(1),
                //        MessageStore: messageStore,
                //        Syncer:       chainSyncer,
                //},
		

		//wyong, 20200921 
		DAG : g, 
		Net : n, 
		Frontera : f , 
        }

	//todo, wyong, 20200921 
        //nd.PorcelainAPI = porcelain.New(plumbing.New(&plumbing.APIDeps{
        //        Bitswap:       bswap,
		
        //        //Chain:         chainState,
        //        //Sync:          cst.NewChainSyncProvider(chainSyncer),
        //        Config:        cfg.NewConfig(nc.Repo),
        //        DAG:           dag.NewDAG(merkledag.NewDAGService(bservice)),

        //        //wyong, 20200410
        //        //Deals:         strgdls.New(nc.Repo.DealsDatastore()),

        //        //Expected:      nodeConsensus,
        //        //MsgPool:       msgPool,
        //        //MsgPreviewer:  msg.NewPreviewer(chainStore, &ipldCborStore, bs),
        //        //ActState:      actorState,
        //        //MsgWaiter:     msg.NewWaiter(chainStore, messageStore, bs, &ipldCborStore),
        //        Network:       net.New(peerHost, pubsub.NewPublisher(fsub), pubsub.NewSubscriber(fsub), net.NewRouter(router), bandwidthTracker, net.NewPinger(peerHost, pingService)),
        //        //Outbox:        outbox,

        //        //wyong, 20200410
        //        //SectorBuilder: nd.SectorBuilder,

        //        Wallet:        fcWallet,
        //}))


	/*todo, wyong, 20201021 
	//fmt.Printf("dag/BuildDAG(110)\n") 
        // Bootstrapping network peers.
        periodStr := nd.Repo.Config().Bootstrap.Period
        period, err := time.ParseDuration(periodStr)
        if err != nil {
                return nil, errors.Wrapf(err, "couldn't parse bootstrap period %s", periodStr)
        }

        // Bootstrapper maintains connections to some subset of addresses
        ba := nd.Repo.Config().Bootstrap.Addresses
        bpi, err := net.PeerAddrsToAddrInfo(ba)
        if err != nil {
                return nil, errors.Wrapf(err, "couldn't parse bootstrap addresses [%s]", ba)
        }
        minPeerThreshold := nd.Repo.Config().Bootstrap.MinPeerThreshold
        nd.Network.Bootstrapper = net.NewBootstrapper(bpi, nd.Host(), nd.Host().Network(), nd.Network.Router, minPeerThreshold, period)
	*/

	return nd, nil 
}

//wyong, 20200908
func (nc *Builder) buildHost(ctx context.Context, 
					swarmAddress string, 
				privkey *asymmetric.PrivateKey, 
	makeDHT func(host host.Host) (routing.Routing, error)) (host.Host, error) {

	//fmt.Printf("dag/buildHost(10), swarmAddress=%s\n", swarmAddress) 
        // Node must build a host acting as a libp2p relay.  Additionally it
        // runs the autoNAT service which allows other nodes to check for their
        // own dialability by having this node attempt to dial them.
        makeDHTRightType := func(h host.Host) (routing.PeerRouting, error) {
                return makeDHT(h)
        }

	fmt.Printf("dag/buildHost(20)\n") 
        if nc.IsRelay {
                cfg := nc.Repo.Config()
                publicAddr, err := ma.NewMultiaddr(cfg.Swarm.PublicRelayAddress)
                if err != nil {
                        return nil, err
                }
                publicAddrFactory := func(lc *libp2p.Config) error {
                        lc.AddrsFactory = func(addrs []ma.Multiaddr) []ma.Multiaddr {
                                if cfg.Swarm.PublicRelayAddress == "" {
                                        return addrs
                                }
                                return append(addrs, publicAddr)
                        }
                        return nil
                }
                relayHost, err := libp2p.New(
                        ctx,
                        libp2p.EnableRelay(circuit.OptHop),
                        libp2p.EnableAutoRelay(),
                        libp2p.Routing(makeDHTRightType),
                        publicAddrFactory,
                        libp2p.ChainOptions(nc.Libp2pOpts...),
                )
                if err != nil {
                        return nil, err
                }
                // Set up autoNATService as a streamhandler on the host.
		// add parameter forceEnabled, wyong, 20201021 
		// todo, Deprecated - use autonat.EnableService
                _, err = autonatsvc.NewAutoNATService(ctx, relayHost, true )
                if err != nil {
                        return nil, err
                }
                return relayHost, nil
        }

	//wyong, 20201021 
	libp2pPrivKey, _, _ := libp2pcrypto.ECDSAKeyPairFromKey((*ecdsa.PrivateKey)(privkey))

	Libp2pOpts := []libp2p.Option { libp2p.ListenAddrStrings(swarmAddress),

					//wyong, 20201021 
					//libp2p.Identity(privkey.(*libp2pcrypto.PrivKey)), 
					libp2p.Identity(libp2pPrivKey), 
				      }
	fmt.Printf("dag/buildHost(30)\n") 
        return libp2p.New(
                ctx,
                libp2p.EnableAutoRelay(),
                libp2p.Routing(makeDHTRightType),
                libp2p.ChainOptions(Libp2pOpts...),
        )
}
