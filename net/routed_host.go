package net

import (
	"context"
	"fmt"
	"time"
	"sort"
	"sync"
	"math/rand" 

	"github.com/pkg/errors"

	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"

	//logging "github.com/ipfs/go-log"
	circuit "github.com/libp2p/go-libp2p-circuit"
	lgbl "github.com/libp2p/go-libp2p-loggables"

	ma "github.com/multiformats/go-multiaddr"

	//wyong, 20201108 
	"github.com/siegfried415/gdf-rebuild/kms"
	"github.com/siegfried415/gdf-rebuild/proto"
	//"github.com/siegfried415/gdf-rebuild/route"
	"github.com/siegfried415/gdf-rebuild/utils/log"
)

//var log = logging.Logger("routedhost")

var (
        // ErrNoChiefBlockProducerAvailable defines failure on find chief block producer.
        ErrNoChiefBlockProducerAvailable = errors.New("no chief block producer found")
)

// AddressTTL is the expiry time for our addresses.
// We expire them quickly.
const AddressTTL = time.Second * 10

// RoutedHost is a p2p Host that includes a routing system.
// This allows the Host to find the addresses for peers when
// it does not have them.
type RoutedHost struct {
	host  host.Host // embedded other host.
	route *PBRouter  

	//FIXME(auxten): remove currentPB stuff
	// currentPB represents current chief block producer node.
	currentPB proto.NodeID
	// currentPBLock represents the chief block producer access lock.
	currentPBLock sync.Mutex
}

//type Routing interface {
//	FindPeer(context.Context, peer.ID) (peer.AddrInfo, error)
//}

func NewRoutedHost(h host.Host, pbr *PBRouter) RoutedHost {
	return RoutedHost{
			host : h, 
			route: pbr,
		}
}

// Connect ensures there is a connection between this host and the peer with
// given peer.ID. See (host.Host).Connect for more information.
//
// RoutedHost's Connect differs in that if the host has no addresses for a
// given peer, it will use its routing system to try to find some.
func (rh RoutedHost) Connect(ctx context.Context, pi peer.AddrInfo) error {

	// first, check if we're already connected.
	if rh.Network().Connectedness(pi.ID) == network.Connected {
		return nil
	}

	// if we were given some addresses, keep + use them.
	if len(pi.Addrs) > 0 {
		rh.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.TempAddrTTL)
	}

	// Check if we have some addresses in our recent memory.
	addrs := rh.Peerstore().Addrs(pi.ID)
	if len(addrs) < 1 {
		// no addrs? find some with the routing system.
		var err error
		
		//add pi.ID, wyong, 20201117 
		//addrs, err = rh.findPeerAddrs(ctx, pi.ID)
		pi, err = rh.findPeerAddrs(ctx, pi.ID)
		if err != nil {
			err = errors.Wrap(err, "can't find peer address")
			return err
		}

		//wyong, 20201117 
		rh.Peerstore().AddAddrs(/* peer.ID(pi.ID.Pretty()) */ pi.ID, pi.Addrs, peerstore.TempAddrTTL)

		addrs = pi.Addrs 
	}

	// Issue 448: if our address set includes routed specific relay addrs,
	// we need to make sure the relay's addr itself is in the peerstore or else
	// we wont be able to dial it.
	for _, addr := range addrs {
		_, err := addr.ValueForProtocol(circuit.P_CIRCUIT)
		if err != nil {
			// not a relay address
			continue
		}

		if addr.Protocols()[0].Code != ma.P_P2P {
			// not a routed relay specific address
			continue
		}

		relay, _ := addr.ValueForProtocol(ma.P_P2P)

		relayID, err := peer.IDFromString(relay)
		if err != nil {
			log.Debugf("failed to parse relay ID in address %s: %s", relay, err)
			continue
		}

		if len(rh.Peerstore().Addrs(relayID)) > 0 {
			// we already have addrs for this relay
			continue
		}

		//wyong, 20201117 
		//relayAddrs, err := rh.findPeerAddrs(ctx, relayID)
		relayAddrInfo, err := rh.findPeerAddrs(ctx, relayID) 
		if err != nil {
			//log.Debugf("failed to find relay %s: %s", relay, err)
			continue
		}

		//wyong, 20201117
		relayAddrs := relayAddrInfo.Addrs 

		rh.Peerstore().AddAddrs(relayID, relayAddrs, peerstore.TempAddrTTL)
	}

	//wyong, 20201117 
	// if we're here, we got some addrs. let's use our wrapped host to connect.
	//pi.Addrs = addrs

	return rh.host.Connect(ctx, pi)
}

func (rh RoutedHost) findPeerAddrs(ctx context.Context, id peer.ID) (/* []ma.Multiaddr, */ peer.AddrInfo,  error) {
	pi, err := rh.route.FindPeer(ctx, id)
	if err != nil {
		return peer.AddrInfo{ID:id}, err // couldnt find any :(
	}

	//bugfix, wyong, 20201117 
	//piID := peer.ID(pi.ID.Pretty()) 
	if pi.ID != id {
		err = fmt.Errorf("routing failure: provided addrs for different peer")
		logRoutingErrDifferentPeers(ctx, id, pi.ID, err)
		return peer.AddrInfo{ID:id}, err
	}


	//return pi.Addrs, nil
	return pi, nil 
}

func logRoutingErrDifferentPeers(ctx context.Context, wanted, got peer.ID, err error) {
	lm := make(lgbl.DeferredMap)
	lm["error"] = err
	lm["wantedPeer"] = func() interface{} { return wanted.Pretty() }
	lm["gotPeer"] = func() interface{} { return got.Pretty() }
	//log.Event(ctx, "routingError", lm)
}

//wyong, 20201112
// Peers lists peers currently available on the network
func (rh RoutedHost) Peers(ctx context.Context, verbose, latency, streams bool) (*SwarmConnInfos, error) {
        if rh.host == nil {
                return nil, errors.New("node must be online")
        }

        conns := rh.host.Network().Conns()

        var out SwarmConnInfos
        for _, c := range conns {
                pid := c.RemotePeer()
                addr := c.RemoteMultiaddr()

                ci := SwarmConnInfo{
                        Addr: addr.String(),
                        Peer: pid.Pretty(),
                }

                if verbose || latency {
                        lat := rh.host.Peerstore().LatencyEWMA(pid)
                        if lat == 0 {
                                ci.Latency = "n/a"
                        } else {
                                ci.Latency = lat.String()
                        }
                }
                if verbose || streams {
                        strs := c.GetStreams()

                        for _, s := range strs {
                                ci.Streams = append(ci.Streams, SwarmStreamInfo{Protocol: string(s.Protocol())})
                        }
                }
                sort.Sort(&ci)
                out.Peers = append(out.Peers, ci)
        }

        sort.Sort(&out)
        return &out, nil
}

//wyong, 20201111 
func (rh RoutedHost) Router() *PBRouter {
	return rh.route
}

func (rh RoutedHost) ID() peer.ID {
	return rh.host.ID()
}

func (rh RoutedHost) Peerstore() peerstore.Peerstore {
	return rh.host.Peerstore()
}

func (rh RoutedHost) Addrs() []ma.Multiaddr {
	return rh.host.Addrs()
}

func (rh RoutedHost) Network() network.Network {
	return rh.host.Network()
}

func (rh RoutedHost) Mux() protocol.Switch {
	return rh.host.Mux()
}

func (rh RoutedHost) EventBus() event.Bus {
	return rh.host.EventBus()
}

func (rh RoutedHost) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {
	rh.host.SetStreamHandler(pid, handler)
}

func (rh RoutedHost) SetStreamHandlerMatch(pid protocol.ID, m func(string) bool, handler network.StreamHandler) {
	rh.host.SetStreamHandlerMatch(pid, m, handler)
}

func (rh RoutedHost) RemoveStreamHandler(pid protocol.ID) {
	rh.host.RemoveStreamHandler(pid)
}

//wyong, 20201117 
func (rh RoutedHost) NewStreamExt(ctx context.Context, peerid proto.NodeID, pids ...protocol.ID) (Stream, error) {
	decPeerId, err := peer.IDB58Decode(peerid.String())
	if err != nil {
		err = errors.Wrap(err, "not valid Peer ID")
		return Stream{}, err 
	}

	s, err := rh.NewStream(ctx, decPeerId, pids...) 
	if err!= nil {
		return Stream{}, err 
	}

	return Stream{ Stream: s }, nil 
}

func (rh RoutedHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	// Ensure we have a connection, with peer addresses resolved by the routing system (#207)
	// It is not sufficient to let the underlying host connect, it will most likely not have
	// any addresses for the peer without any prior connections.
	// If the caller wants to prevent the host from dialing, it should use the NoDial option.

	var result Stream
	if nodial, _ := network.GetNoDial(ctx); !nodial {
		err := rh.Connect(ctx, peer.AddrInfo{ID: p})
		if err != nil {
			err = errors.Wrap(err, "can't connect to remote peer")
			return result, err
		}
	}

	s, err := rh.host.NewStream(ctx, p, pids...)
	if err != nil {
		err = errors.Wrap(err, "can't create stream")
		return result, err
	}
	
	return s, nil 
}

func (rh RoutedHost) Close() error {
	// no need to close IpfsRouting. we dont own it.
	return rh.host.Close()
}
func (rh RoutedHost) ConnManager() connmgr.ConnManager {
	return rh.host.ConnManager()
}

// PingPB Send DHT.Ping Request with Anonymous ETLS session.
func (rh RoutedHost)PingPB(node *proto.Node, PBNodeID proto.NodeID) (err error) {
	//wyong, 20201020 
	ctx := context.Background()

	//client := NewCaller()
	req := proto.PingReq{
		Node: *node,
	}

	//wyong, 20201008 
	//err = client.CallNode(PBNodeID, "DHT.Ping", req, resp)

	//wyong, 20201118 
        s, err := rh.NewStreamExt(ctx, PBNodeID, protocol.ID("DHT.Ping"))
	if err != nil {
		err = errors.Wrap(err, "call DHT.Ping failed")
		return
	}
        defer s.Close()

	_, err = s.SendMsg(ctx, &req )
	if err != nil {
		err = errors.Wrap(err, "send DHT.Ping failed")
		return 
	}

	//resp, err = ioutil.ReadAll(s)
	resp := new(proto.PingResp)
	err = s.RecvMsg(ctx, &resp) 
	if err!= nil {
		err = errors.Wrap(err, "get DHT.Ping result failed")
		return 
	} 

	return
}

// GetCurrentPB returns nearest hash distance block producer as current node chief block producer.
func (rh RoutedHost) GetCurrentPB() (bpNodeID proto.NodeID, err error) {
	//wyong, 20201020 
	ctx := context.Background()

	rh.currentPBLock.Lock()
	defer rh.currentPBLock.Unlock()

	if !rh.currentPB.IsEmpty() {
		bpNodeID = rh.currentPB
		return
	}

	var localNodeID proto.NodeID
	if localNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get random block producer first
	bpList := rh.route.GetPBs()

	if len(bpList) == 0 {
		err = ErrNoChiefBlockProducerAvailable
		return
	}

	randomPB := bpList[rand.Intn(len(bpList))]

	// call random block producer for nearest block producer node
	req := &proto.FindNeighborReq{
		ID: localNodeID,
		Roles: []proto.ServerRole{
			proto.Leader,
			proto.Follower,
		},
		Count: 1,
	}
	//wyong, 20201008 
	//if err = NewCaller().CallNode(randomPB, "DHT.FindNeighbor", req, res); err != nil {
        s, err := rh.NewStreamExt(ctx, randomPB, protocol.ID("DHT.FindNeighbor"))
	if err != nil { 
		return
	}
        defer s.Close()

	_, err = s.SendMsg(ctx, &req )
	if err != nil {
		err = errors.Wrap(err, "send DHT.FindNeighbor failed")
		return 
	}

	//res, err = ioutil.ReadAll(s)
	res := new(proto.FindNeighborResp)
	err = s.RecvMsg(ctx, &res) 
	if err!= nil {
		err = errors.Wrap(err, "get DHT.FindNeighbor result failed")
		return 
	} 


	if len(res.Nodes) <= 0 {
		// node not found
		err = errors.Wrap(ErrNoChiefBlockProducerAvailable,
			"get no hash nearest block producer nodes")
		return
	}

	if res.Nodes[0].Role != proto.Leader && res.Nodes[0].Role != proto.Follower {
		// not block producer
		err = errors.Wrap(ErrNoChiefBlockProducerAvailable,
			"no suitable nodes with proper block producer role")
		return
	}

	rh.currentPB = res.Nodes[0].ID
	bpNodeID = rh.currentPB

	return
}

// SetCurrentPB sets current node chief block producer.
func (rh RoutedHost) SetCurrentPB(bpNodeID proto.NodeID) {
	rh.currentPBLock.Lock()
	defer rh.currentPBLock.Unlock()
	rh.currentPB = bpNodeID
}

// RequestPB sends request to main chain.
func (rh RoutedHost)RequestPB(method string, req interface{}, resp interface{}) (err error) {
	//wyong, 20201020
	ctx := context.Background()

	var bp proto.NodeID
	if bp, err = rh.GetCurrentPB(); err != nil {
		return err
	}

	fmt.Printf("RoutedHost/RequestPB, current pb is %s\n", bp.String()) 

	//todo, method->protocol, wyong, 20201008 
	//return NewCaller().CallNode(bp, method, req, resp)
        s, err := rh.NewStreamExt(ctx, bp, protocol.ID(method))
        if err != nil {
                //log.Debugf("error opening push stream to %s: %s", bp, err.Error())
                return
        }
        defer s.Close()

	_, err = s.SendMsg(ctx, &req )
	if err == nil {
		//log.WithFields(log.Fields{
		//	"index":    i.index,
		//	"instance": i.r.instanceID,
		//}).WithError(err).Debug("send fetch request failed")
		//log.Debugf("send presbyterian request failed") 
		return err 
	}

	err = s.RecvMsg(ctx, &resp )
	if err!= nil {
		//log.WithFields(log.Fields{
		//	"index":    i.index,
		//	"instance": i.r.instanceID,
		//}).WithError(err).Debug("get fetch result failed")
		//log.Debugf("get result for presbyterian request failed") 
		return err 	
	} 

	return nil 

}

// RegisterNodeToPB registers the current node to bp network.
func (rh RoutedHost) RegisterNodeToPB(timeout time.Duration) (err error) {
	// get local node id
	localNodeID, err := kms.GetLocalNodeID()
	if err != nil {
		err = errors.Wrap(err, "register node to PB")
		return
	}

	// get local node info
	localNodeInfo, err := kms.GetNodeInfo(localNodeID)
	if err != nil {
		err = errors.Wrap(err, "register node to PB")
		return
	}

	//log.WithField("node", localNodeInfo).Debug("construct local node info")
	fmt.Printf("construct local node info\n")

	pingWaitCh := make(chan proto.NodeID)
	bpNodeIDs := rh.route.GetPBs()
	for _, bpNodeID := range bpNodeIDs {
		go func(ch chan proto.NodeID, id proto.NodeID) {
			for {
				//wyong, 20201020 
				//log.Infof("ping PB %s", id )
				fmt.Printf("ping PB : localNodeInfo=%s, id=%s\n", localNodeInfo, id )

				err := rh.PingPB(localNodeInfo, id)
				if err == nil {
					//log.Infof("ping PB succeed: %v", localNodeInfo)
					fmt.Printf("ping PB succeed: %v", localNodeInfo)
					select {
					case ch <- id:
					default:
					}
					return
				}

				//log.Warnf("ping PB failed: %v", err)
				fmt.Printf("ping PB failed: %v\n", err)
				time.Sleep(3 * time.Second)
			}
		}(pingWaitCh, bpNodeID)
	}

	select {
	case bp := <-pingWaitCh:
		//log.WithField("PB", bp).Infof("ping PB succeed")
		fmt.Printf("ping PB(%s) succeed\n", bp)

	case <-time.After(timeout):
		return errors.New("ping PB timeout")
	}

	return
}

//todo, wyong, 20201110 
var _ (host.Host) = (*RoutedHost)(nil)
