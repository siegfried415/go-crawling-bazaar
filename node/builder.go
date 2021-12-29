
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


	"github.com/pkg/errors"
        "golang.org/x/crypto/ssh/terminal"

	"github.com/ipfs/go-merkledag"
        "github.com/ipfs/go-bitswap"
        bsnet "github.com/ipfs/go-bitswap/network"
        bserv "github.com/ipfs/go-blockservice"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"

        ma "github.com/multiformats/go-multiaddr"

	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
        "github.com/libp2p/go-libp2p"
        "github.com/libp2p/go-libp2p-core/host"
        "github.com/libp2p/go-libp2p-core/routing"
        dht "github.com/libp2p/go-libp2p-kad-dht"
        dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
        autonatsvc "github.com/libp2p/go-libp2p-autonat-svc"
        circuit "github.com/libp2p/go-libp2p-circuit"

	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric" 
	conf "github.com/siegfried415/go-crawling-bazaar/conf" 
	//crypto "github.com/siegfried415/go-crawling-bazaar/crypto" 
	dag "github.com/siegfried415/go-crawling-bazaar/dag" 
	frontera "github.com/siegfried415/go-crawling-bazaar/frontera"
        "github.com/siegfried415/go-crawling-bazaar/utils/log"
	kms "github.com/siegfried415/go-crawling-bazaar/kms" 
	net "github.com/siegfried415/go-crawling-bazaar/net" 
        pb "github.com/siegfried415/go-crawling-bazaar/presbyterian"
	proto "github.com/siegfried415/go-crawling-bazaar/proto" 

)


const (
        dhtGossipTimeout = time.Second * 20
)


// Builder is a helper to aid in the construction of a filecoin node.
type Builder struct {
        BlockTime   time.Duration
        Libp2pOpts  []libp2p.Option
        OfflineMode bool
        IsRelay     bool
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
func New(ctx context.Context, repoPath string, role proto.ServerRole, opts ...BuilderOpt) (*Node, error) {
        n := &Builder{}
        for _, o := range opts {
                if err := o(n); err != nil {
                        return nil, err
                }
        }

        return n.build(ctx, repoPath, role )
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
        return func(nb *Builder) error {
                // Quietly having your options overridden leads to hair loss
                if len(nb.Libp2pOpts) > 0 {
                        panic("Libp2pOptions can only be called once")
                }
                nb.Libp2pOpts = opts
                return nil
        }
}


type blankValidator struct{}
func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func (nb *Builder) build(ctx context.Context, repoPath string, role proto.ServerRole) (*Node, error ) {
	var (
		masterKey []byte

		bswap exchange.Interface 	
		g *dag.DAG 
		f *frontera.Frontera 
		chain *pb.Chain
		err error 
	)

	if !conf.GConf.UseTestMasterKey {
		// read master key
		fmt.Print("Type in Master key to continue: ")
		masterKey, err = terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			log.Debugf("Failed to read Master Key: %v", err)
		}
		fmt.Println("")
	}

	if err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey); err != nil {
		log.WithError(err).Error("init local key pair failed")
		return nil, err 
	}


        //log.WithField("node", nodeID).Info("init peers")
        peers, err := GetPeersFromConf(conf.GConf.PubKeyStoreFile)
        if err != nil {
                log.WithError(err).Error("init nodes and peers failed")
                return nil, err 
        }

        // Get accountAddress
        //var pubKey *asymmetric.PublicKey
        //if pubKey, err = kms.GetLocalPublicKey(); err != nil {
        //        return nil, err 
        //}

        //accountAddr, err := crypto.PubKeyHash(pubKey)
	//if err != nil {
        //        return nil, err 
        //}


	ds, _ := conf.NewDatastore(repoPath) 
        bs := bstore.NewBlockstore(ds)
        validator := blankValidator{}

        var peerHost net.RoutedHost
	network := "gcb"

	makeDHT := func(h host.Host) (routing.ContentRouting, error) {
		r, err := dht.New(
			ctx,
			h,
			dhtopts.Datastore(ds),
			dhtopts.NamespacedValidator("v", validator),
			dhtopts.Protocols(net.FilecoinDHT(network)),
		)

		if err != nil {
			return nil, errors.Wrap(err, "failed to setup routing")
		}

		return r, err
	}

	//var err error
	privKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		log.Debugf("dag/BuildDAG(35), err=%s\n", err.Error()) 
		return nil, err
	}


	peerHost, err = nb.buildHost(ctx, conf.GConf.ListenAddr, privKey, makeDHT)
	if err != nil {
		log.Debugf("dag/BuildDAG(45), err=%s\n", err.Error()) 
		return nil, err
	}


	// init kms routing
	net.InitKMS(peerHost, conf.GConf.PubKeyStoreFile)

	switch role  {
	case proto.Client:
		//todo

	case proto.Leader, 
	     proto.Follower: 
		genesis, err := loadGenesis()
		if err != nil {
			return nil, err 
		}

               	// init storage
                log.Info("init storage")
                var st *LocalStorage
                if st, err = initStorage(conf.GConf.DHTFileName); err != nil {
                        log.WithError(err).Error("init storage failed")
                        return nil, err
                }

                // init dht node server
                //log.Info("init consistent runtime")
                kvServer := NewKVServer(peerHost, conf.GConf.ThisNodeID, peers, st, dhtGossipTimeout)
                dht, err := net.NewDHTService(peerHost, conf.GConf.DHTFileName, kvServer, true)
                if err != nil {
                        log.WithError(err).Error("init consistent hash failed")
                        return nil, err
                }
                defer kvServer.Stop()

                // set consistent handler to local storage
                kvServer.storage.consistent = dht.Consistent

                // create gossip service 
                _, err = NewGossipService(kvServer)
                if err != nil {
                        log.WithError(err).Error("register dht gossip service failed")
                        return nil, err
                }


		// init main chain service
		//log.Info("register main chain service rpc")
		chainConfig := &pb.Config{
			Mode:           pb.PBMode,
			Genesis:        genesis,
			DataFile:       conf.GConf.PB.ChainFileName,
			Host: peerHost, 
			Peers:          peers,
			NodeID:         conf.GConf.ThisNodeID,	//nodeID,
			Period:         conf.GConf.PBPeriod,
			Tick:           conf.GConf.PBTick,
			BlockCacheSize: 1000,
		}

		chain, err = pb.NewChain(chainConfig)
		if err != nil {
			//log.WithError(err).Error("init chain failed")
			return nil, err
		}
		


	case proto.Miner: 
		// set up bitswap
		nwork := bsnet.NewFromIpfsHost(peerHost, peerHost.Router())
		bswap = bitswap.New(ctx, nwork, bs)
		bservice := bserv.New(bs, bswap)

		s := merkledag.NewDAGService(bservice) 
		g = dag.NewDAG(s)

                _, err = NewDagService("DAG", peerHost, g)
                if err != nil {
                        log.WithError(err).Error("register dag service failed")
                        return nil, err
                }

		//create frontera
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

		//if err = f.Init(); err != nil {
		//	err = errors.Wrap(err, "init Frontera failed")
		//	return nil, err 
		//}


	case proto.Unknown:
                return nil, err
	}	

        nd := &Node{
		Host : peerHost, 
		Role : role, 
                OfflineMode: nb.OfflineMode,
		Chain : chain, 
		DAG : g, 
		Frontera : f , 
        }

        //log.WithField("node", nodeID).Info("init peers")
        _,  _, err = nd.InitNodePeers(conf.GConf.PubKeyStoreFile)
        if err != nil {
                log.WithError(err).Error("init nodes and peers failed")
                return nil, err 
        }

	return nd, nil 
}


func (nb *Builder) buildHost(ctx context.Context, 
					swarmAddress string, 
				privkey *asymmetric.PrivateKey, 
	makeDHT func(host host.Host) (routing.ContentRouting, error),
) (net.RoutedHost, error) {
        // Node must build a host acting as a libp2p relay.  Additionally it
        // runs the autoNAT service which allows other nodes to check for their
        // own dialability by having this node attempt to dial them.

        makeDHTRightType := func(h host.Host) (*net.PBRouter, error) {
		dht, err := makeDHT(h) 
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup dht routing")
		}		

		r, err := net.NewPBRouter(h, dht )
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup routing")
		}

		return r, nil 
        }


        if nb.IsRelay {
                //cfg := nb.Repo.Config()
                publicAddr, err := ma.NewMultiaddr(conf.GConf.PublicRelayAddress)
                if err != nil {
                        return net.RoutedHost{}, err
                }
                publicAddrFactory := func(lc *libp2p.Config) error {
                        lc.AddrsFactory = func(addrs []ma.Multiaddr) []ma.Multiaddr {
                                if conf.GConf.PublicRelayAddress == "" {
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
                        publicAddrFactory,
                        libp2p.ChainOptions(nb.Libp2pOpts...),
                )
                if err != nil {
                        return net.RoutedHost{}, err
                }
                // Set up autoNATService as a streamhandler on the host.
		// add parameter forceEnabled
		// todo, Deprecated - use autonat.EnableService
                _, err = autonatsvc.NewAutoNATService(ctx, relayHost, true )
                if err != nil {
                        return net.RoutedHost{}, err
                }

		router, err := makeDHTRightType(relayHost.(host.Host))
		if err != nil {
			return net.RoutedHost{}, err 
		}

		rh := net.NewRoutedHost(relayHost.(host.Host), router)
		return rh, nil  
        }

	libp2pPrivKey := (*libp2pcrypto.Secp256k1PrivateKey) (privkey) 
	Libp2pOpts := []libp2p.Option { libp2p.ListenAddrStrings(swarmAddress),
					libp2p.Identity(libp2pPrivKey), 
				      }
        host, err := libp2p.New(ctx, libp2p.ChainOptions(Libp2pOpts...),
        )
	if err != nil {
		return net.RoutedHost{}, err 
	}

	router, err := makeDHTRightType(host)
	if err != nil {
		return net.RoutedHost{}, err 
	}

	rh := net.NewRoutedHost(host, router)
	return rh, nil 
}

