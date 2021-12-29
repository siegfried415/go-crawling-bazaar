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

	circuit "github.com/libp2p/go-libp2p-circuit"
	lgbl "github.com/libp2p/go-libp2p-loggables"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/siegfried415/go-crawling-bazaar/kms"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)


var (
        // ErrNoChiefPresbyterianAvailable defines failure on find chief presbyterian.
        ErrNoChiefPresbyterianAvailable = errors.New("no chief presbyterian found")
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
	// currentPB represents current chief presbyterian node.
	currentPB proto.NodeID
	// currentPBLock represents the chief presbyterian access lock.
	currentPBLock sync.Mutex
}

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
		
		pi, err = rh.findPeerAddrs(ctx, pi.ID)
		if err != nil {
			err = errors.Wrap(err, "can't find peer address")
			return err
		}

		rh.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.TempAddrTTL)

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

		relayAddrInfo, err := rh.findPeerAddrs(ctx, relayID) 
		if err != nil {
			//log.Debugf("failed to find relay %s: %s", relay, err)
			continue
		}

		relayAddrs := relayAddrInfo.Addrs 
		rh.Peerstore().AddAddrs(relayID, relayAddrs, peerstore.TempAddrTTL)
	}

	// if we're here, we got some addrs. let's use our wrapped host to connect.
	//pi.Addrs = addrs

	return rh.host.Connect(ctx, pi)
}

func (rh RoutedHost) findPeerAddrs(ctx context.Context, id peer.ID) (peer.AddrInfo,  error) {
	pi, err := rh.route.FindPeer(ctx, id)
	if err != nil {
		return peer.AddrInfo{ID:id}, err // couldnt find any :(
	}

	if pi.ID != id {
		err = fmt.Errorf("routing failure: provided addrs for different peer")
		logRoutingErrDifferentPeers(ctx, id, pi.ID, err)
		return peer.AddrInfo{ID:id}, err
	}

	return pi, nil 
}

func logRoutingErrDifferentPeers(ctx context.Context, wanted, got peer.ID, err error) {
	lm := make(lgbl.DeferredMap)
	lm["error"] = err
	lm["wantedPeer"] = func() interface{} { return wanted.Pretty() }
	lm["gotPeer"] = func() interface{} { return got.Pretty() }
	//log.Event(ctx, "routingError", lm)
}

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

// StreamHandler is the type of function used to listen for
// streams opened by the remote side.
type StreamHandler func(Stream)

func (rh RoutedHost) SetStreamHandlerExt(pid protocol.ID, handler StreamHandler) {
	sh := func(s network.Stream) {
		ns := Stream{Stream: s }
		handler(ns)
	}

	rh.host.SetStreamHandler(pid, sh )
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
	ctx := context.Background()
	req := proto.PingReq{
		Node: *node,
	}

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

	resp := new(proto.PingResp)
	err = s.RecvMsg(ctx, &resp) 
	if err!= nil {
		err = errors.Wrap(err, "get DHT.Ping result failed")
		return 
	} 

	return
}

// GetCurrentPB returns nearest hash distance presbyterian as current node chief presbyterian.
func (rh RoutedHost) GetCurrentPB() (pbNodeID proto.NodeID, err error) {
	ctx := context.Background()

	rh.currentPBLock.Lock()
	defer rh.currentPBLock.Unlock()

	if !rh.currentPB.IsEmpty() {
		pbNodeID = rh.currentPB
		return
	}

	var localNodeID proto.NodeID
	if localNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get random presbyterian first
	pbList := rh.route.GetPBs()

	if len(pbList) == 0 {
		err = ErrNoChiefPresbyterianAvailable
		return
	}

	randomPB := pbList[rand.Intn(len(pbList))]

	// call random presbyterian for nearest presbyterian node
	req := &proto.FindNeighborReq{
		ID: localNodeID,
		Roles: []proto.ServerRole{
			proto.Leader,
			proto.Follower,
		},
		Count: 1,
	}

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
		err = errors.Wrap(ErrNoChiefPresbyterianAvailable,
			"get no hash nearest presbyterian nodes")
		return
	}

	if res.Nodes[0].Role != proto.Leader && res.Nodes[0].Role != proto.Follower {
		// not presbyterian  
		err = errors.Wrap(ErrNoChiefPresbyterianAvailable,
			"no suitable nodes with proper presbyterian role")
		return
	}

	rh.currentPB = res.Nodes[0].ID
	pbNodeID = rh.currentPB

	return
}

// SetCurrentPB sets current node chief presbyterian.
func (rh RoutedHost) SetCurrentPB(pbNodeID proto.NodeID) {
	rh.currentPBLock.Lock()
	defer rh.currentPBLock.Unlock()
	rh.currentPB = pbNodeID
}

// RequestPB sends request to main chain.
func (rh RoutedHost)RequestPB(method string, req interface{}, resp interface{}) (err error) {
	ctx := context.Background()

	var pb proto.NodeID
	if pb, err = rh.GetCurrentPB(); err != nil {
		return err
	}

        s, err := rh.NewStreamExt(ctx, pb, protocol.ID(method))
        if err != nil {
                log.Debugf("error opening push stream to %s: %s", pb, err.Error())
                return
        }
        defer s.Close()

	_, err = s.SendMsg(ctx, req )
	if err != nil {
		log.Debugf("send presbyterian request failed") 
		return err 
	}

	err = s.RecvMsg(ctx, resp )
	if err!= nil {
		log.Debugf("get result for presbyterian request failed") 
		return err 	
	} 

	return nil 

}

// RegisterNodeToPB registers the current node to presbyterian network.
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

	log.Debugf("construct local node info\n")

	pingWaitCh := make(chan proto.NodeID)
	pbNodeIDs := rh.route.GetPBs()
	for _, pbNodeID := range pbNodeIDs {
		go func(ch chan proto.NodeID, id proto.NodeID) {
			for {
				log.Debugf("ping PB : localNodeInfo=%s, id=%s\n", localNodeInfo, id )

				err := rh.PingPB(localNodeInfo, id)
				if err == nil {
					log.Debugf("ping PB succeed: %v", localNodeInfo)
					select {
					case ch <- id:
					default:
					}
					return
				}

				log.Debugf("ping PB failed: %v\n", err)
				time.Sleep(3 * time.Second)
			}
		}(pingWaitCh, pbNodeID)
	}

	select {
	case pb := <-pingWaitCh:
		log.Debugf("ping PB(%s) succeed\n", pb)

	case <-time.After(timeout):
		return errors.New("ping PB timeout")
	}

	return
}

var _ (host.Host) = (*RoutedHost)(nil)
