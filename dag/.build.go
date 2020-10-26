package dag 

import(
	"context"
	"fmt"
	"github.com/pkg/errors"

	"github.com/ipfs/go-merkledag"
        "github.com/ipfs/go-bitswap"
        bsnet "github.com/ipfs/go-bitswap/network"
        bserv "github.com/ipfs/go-blockservice"
        "github.com/ipfs/go-datastore"
        dss "github.com/ipfs/go-datastore/sync"
        bstore "github.com/ipfs/go-ipfs-blockstore"

        "github.com/libp2p/go-libp2p"
        "github.com/libp2p/go-libp2p-core/host"
        "github.com/libp2p/go-libp2p-core/routing"
        dht "github.com/libp2p/go-libp2p-kad-dht"
        //dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"

)

//wyong, 20200908
func buildHost(ctx context.Context, swarmAddress string, /* privkey , wyong, 20200908 */  makeDHT func(host host.Host) (routing.Routing, error)) (host.Host, error) {

	fmt.Printf("dag/buildHost(10), swarmAddress=%s\n", swarmAddress) 
        // Node must build a host acting as a libp2p relay.  Additionally it
        // runs the autoNAT service which allows other nodes to check for their
        // own dialability by having this node attempt to dial them.
        makeDHTRightType := func(h host.Host) (routing.PeerRouting, error) {
                return makeDHT(h)
        }

	fmt.Printf("dag/buildHost(20)\n") 
	/*todo, wyong, 20200908 
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
                _, err = autonatsvc.NewAutoNATService(ctx, relayHost)
                if err != nil {
                        return nil, err
                }
                return relayHost, nil
        }
	*/

	Libp2pOpts := []libp2p.Option { libp2p.ListenAddrStrings(swarmAddress),
					//todo, wyong, 202009087 
					//libp2p.Identity(privkey), 
				      }
	fmt.Printf("dag/buildHost(30)\n") 
        return libp2p.New(
                ctx,
                libp2p.EnableAutoRelay(),
                libp2p.Routing(makeDHTRightType),
                libp2p.ChainOptions(Libp2pOpts...),
        )
}

type blankValidator struct{}
func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

//wyong, 2020
func BuildDAG(ctx context.Context , swarmAddress string) (d *DAG, err error) {
        var router routing.Routing
        var peerHost host.Host

	fmt.Printf("dag/BuildDAG(10), swarmAddress=%s\n", swarmAddress )
	ds := dss.MutexWrap(datastore.NewMapDatastore())
        bs := bstore.NewBlockstore(ds)

	fmt.Printf("dag/BuildDAG(20)\n") 
	//make a memory repo 
	//rep := repo.NewInMemoryRepo()

	makeDHT := func(h host.Host) (routing.Routing, error) {
		fmt.Printf("dag/makeDGT(10),host=%s\n", h) 
		baseOpts := []dht.Option{
			dht.ProtocolPrefix("/gdf/dht"),
			dht.NamespacedValidator("v", blankValidator{}),
			dht.DisableAutoRefresh(),
			dht.Mode(dht.ModeServer), 
		}

		fmt.Printf("dag/makeDGT(20)\n") 
		r, err := dht.New(
			ctx,
			h,
			//dhtopts.Datastore(ds),
			//dhtopts.NamespacedValidator("v", blankValidator{}),
			//dhtopts.Protocols("/gdf/dht"),
			baseOpts..., 
		)
		if err != nil {
			fmt.Printf("dag/makeDGT(25)\n") 
			return nil, errors.Wrap(err, "failed to setup routing")
		}
		fmt.Printf("dag/makeDGT(30)\n") 
		router = r
		return r, err
	}

	fmt.Printf("dag/BuildDAG(30)\n") 
	//var err error
	peerHost, err = buildHost(ctx, swarmAddress, makeDHT)
	if err != nil {
		fmt.Printf("dag/BuildDAG(35), err=%s\n", err.Error()) 
		return nil, err
	}


	fmt.Printf("dag/BuildDAG(40)\n") 
        // set up bitswap
        nwork := bsnet.NewFromIpfsHost(peerHost, router)
	fmt.Printf("dag/BuildDAG(50)\n") 
        bswap := bitswap.New(ctx, nwork, bs)
	fmt.Printf("dag/BuildDAG(60)\n") 
        bservice := bserv.New(bs, bswap)

	fmt.Printf("dag/BuildDAG(70)\n") 
	s := merkledag.NewDAGService(bservice) 
	fmt.Printf("dag/BuildDAG(80)\n") 
	d = NewDAG(s)

	fmt.Printf("dag/BuildDAG(90)\n") 
	return d, nil 
}
