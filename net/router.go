package net

import (
	"context"
	//"errors"
	//"fmt"
	"math/rand"
	"sync"

	//wyong, 20201110 
	"github.com/pkg/errors"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p-core/network"
	//"github.com/libp2p/go-libp2p-kad-dht"
	
	//wyong, 20201110 
        protocol "github.com/libp2p/go-libp2p-core/protocol"


	"github.com/siegfried415/go-crawling-bazaar/conf"
	//"github.com/siegfried415/go-crawling-bazaar/kms"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
)

// This struct wraps the filecoin nodes router.  This router is a
// go-libp2p-core/routing.Routing interface that provides both PeerRouting,
// ContentRouting and a Bootstrap init process. Filecoin nodes in online mode
// use a go-libp2p-kad-dht DHT to satisfy this interface. Nodes run the
// Bootstrap function to join the DHT on start up. The PeerRouting functionality
// enables filecoin nodes to lookup the network addresses of their peers given a
// peerID.  The ContentRouting functionality enables peers to provide and
// discover providers of network services. This is currently used by the
// auto-relay feature in the filecoin network to allow nodes to advertise
// themselves as relay nodes and discover other relay nodes.
//
// The Routing interface and its DHT instantiation also carries ValueStore
// functionality for using the DHT as a key value store.  Filecoin nodes do
// not currently use this functionality.

var (
	// ErrUnknownNodeID indicates we got unknown node id
	ErrUnknownNodeID = errors.New("unknown node id")

	// ErrNilNodeID indicates we got nil node id
	ErrNilNodeID = errors.New("nil node id")
)

// Router exposes the methods on the internal filecoin router that are needed
// by the system plumbing API.
type PBRouter struct {
	//for content routing, wyong, 20201111 
	routing routing.ContentRouting

	//wyong, 20201107 
	//RawNodeID -> NodeID, wyong, 20201114 
        cache    map [proto.NodeID]string 
        bpNodeIDs map[proto.NodeID]string 
        bpNodes   map[proto.NodeID]proto.Node 

	//wyong, 20201107
	host	 host.Host  

        sync.RWMutex
}

// NewRouter builds a new router.
func NewPBRouter(h host.Host, r routing.ContentRouting ) (*PBRouter, error) {
	//return &Router{routing: r}
	pbr := &PBRouter {
		host: h, 
		routing: r, 	//wyong, 20201111 

		//RawNodeID -> NodeID, wyong, 20201114 
		cache:     make(map[proto.NodeID]string),
		bpNodeIDs: make(map[proto.NodeID]string),
        }

	_, err := pbr.initBPNodeIDs()
	return pbr, err 
}

// FindProvidersAsync searches for and returns peers who are able to provide a
// given key.
func (r *PBRouter) FindProvidersAsync(ctx context.Context, key cid.Cid, count int) <-chan peer.AddrInfo {
	return r.routing.FindProvidersAsync(ctx, key, count)
}


// NOTE,NOTE,NOTE, wyong, 20201111
// Provide adds the given cid to the content routing system. If 'true' is
// passed, it also announces it, otherwise it is just kept in the local
// accounting of which objects are being provided.
func (r *PBRouter) Provide(context.Context, cid.Cid, bool) error {
	return nil 	
}

// FindPeer searches the libp2p router for a given peer id
func (r *PBRouter) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {

	//wyong, 20201107 
	//return r.routing.FindPeer(ctx, peerID)

	//wyong, 20201110 
	//nodeID := (proto.NodeID)(peerID)
	//wyong, 20201117
	nodeID := (proto.NodeID)(peerID.Pretty())

	//todo, wyong, 20201114 
	//rawNodeID := nodeID.ToRawNodeID()
	rawNodeID := &nodeID 

	addr, err := r.GetNodeAddrCache(rawNodeID)
	if err != nil {
		//log.WithField("target", id.String()).WithError(err).Debug("get node addr from cache failed")
		if err == ErrUnknownNodeID {
			var node *proto.Node
			
			//wyong, 20201107 
			node, err = r.FindNodeInPB(rawNodeID)
			if err != nil {
				return peer.AddrInfo{}, err 
			}
			_ = r.SetNodeAddrCache(rawNodeID, node.Addr)
			addr = node.Addr 
		}
	}

	//wyong, 20201110 
	result, err := PeerAddrToAddrInfo(addr) 
	if err != nil {
		return peer.AddrInfo{}, err 
	}

	return *result, nil 
}


// GetClosestPeers returns a channel of the K closest peers  to the given key,
// K is the 'K Bucket' parameter of the Kademlia DHT protocol.
//func (r PBRouter) GetClosestPeers(ctx context.Context, key string) (<-chan peer.ID, error) {
//	ipfsDHT, ok := r.routing.(*dht.IpfsDHT)
//	if !ok {
//		return nil, errors.New("underlying routing should be pointer of IpfsDHT")
//	}
//	return ipfsDHT.GetClosestPeers(ctx, key)
//}

// GetNodeAddrCache gets node addr by node id, if cache missed try RPC.
// RawNodeID -> NodeID, wyong, 20201114 
func (r *PBRouter) GetNodeAddrCache(id *proto.NodeID) (addr string, err error) {
	if id == nil {
		return "", ErrNilNodeID
	}
	r.RLock()
	defer r.RUnlock()
	addr, ok := r.cache[*id]
	if !ok {
		return "", ErrUnknownNodeID
	}
	return
}

// setNodeAddrCache sets node id and addr.
// RawNodeID -> NodeID, wyong, 20201114 
func (r *PBRouter) SetNodeAddrCache(id *proto.NodeID, addr string) (err error) {
	if id == nil {
		return ErrNilNodeID
	}
	r.Lock()
	defer r.Unlock()
	r.cache[*id] = addr
	return
}


//wyong, 20201118
func (r *PBRouter) NewStreamExt(ctx context.Context, peerid proto.NodeID, pids ...protocol.ID) (Stream, error) {
        decPeerId, err := peer.IDB58Decode(peerid.String())
        if err != nil {
                err = errors.Wrap(err, "not valid Peer ID")
                return Stream{}, err
        }

        s, err := r.NewStream(ctx, decPeerId, pids...)
        if err!= nil {
                return Stream{}, err
        }

        return Stream{ Stream: s }, nil
}

//wyong, 20201118 
func (r *PBRouter) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
        var result Stream
        if nodial, _ := network.GetNoDial(ctx); !nodial {
		//wyong, 20201118 
                //err := rh.Connect(ctx, peer.AddrInfo{ID: p})
		err := r.host.Connect(ctx, peer.AddrInfo{ID: p})
                if err != nil {
                        return result, err
                }
        }

        s, err := r.host.NewStream(ctx, p, pids...)
        if err != nil {
                return result, err
        }

        return s, nil
}

// FindNodeInPB find node in block producer dht service.
// RawNodeID -> NodeID, wyong, 20201114 
func (r *PBRouter) FindNodeInPB(id *proto.NodeID) (*proto.Node, error) {
	bps := r.GetPBs()
	if len(bps) == 0 {
		err := errors.New("no available PB")
		return nil, err 
	}

	//wyong, 20201107
	ctx := context.Background()

	//wyong, 20201008 
	//client := NewCaller()

	req := &proto.FindNodeReq{
		ID: proto.NodeID(id.String()),
	}
	resp := new(proto.FindNodeResp)
	bpCount := len(bps)
	offset := rand.Intn(bpCount)
	method := protocol.ID("DHT.FindNode")	//DHTFindNode.String()

	for i := 0; i != bpCount; i++ {
		bp := bps[(offset+i)%bpCount]

		//wyong, 20201008 
		//err = client.CallNode(bp, method, req, resp)

		//wyong, 20201118 
		s, err := r.NewStreamExt(ctx, bp, method )
		if err != nil {
			log.Debugf("error opening push stream to %s: %s", bp, err.Error())
			continue 	
		}

		//s = s.(Stream)
		_, err = s.SendMsg(ctx, &req )
		if err != nil {
			log.WithFields(log.Fields{
				"method": method,
				"bp":     bp,
			}).WithError(err).Warning("call dht rpc failed")
			continue 	
		}

		err = s.RecvMsg(ctx, &resp)
		if err!= nil {
			log.WithFields(log.Fields{
				"method": method,
				"bp":     bp,
			}).WithError(err).Warning("call dht rpc failed")
			continue 	
		}

		s.Close()
		//if err == nil {
		node := resp.Node
		return node, nil 
		//}


	}

	err := errors.New("could not find node in all block producers")
	return nil, err 
}
// initBPNodeIDs initializes BlockProducer route and map from config file and DNS Seed.
// RawNodeID -> NodeID, wyong, 20201114 
func (r *PBRouter) initBPNodeIDs() (bpNodeIDs map[proto.NodeID]string , err error ) {
	if conf.GConf == nil {
		log.Fatal("call conf.LoadConfig to init conf first")
	}

	// clear address map before init
	//wyong, 20201114 
	//r.bpNodeIDs = make(map[proto.RawNodeID]string)
	r.bpNodeIDs = make(map[proto.NodeID]string)
	bpNodeIDs = r.bpNodeIDs

	//var err error

	//todo, wyong, 20201114 
	//if conf.GConf.DNSSeed.Domain != "" {
	//	var bpIndex int
	//	dc := IPv6SeedClient{}
	//	bpIndex = rand.Intn(conf.GConf.DNSSeed.BPCount)
	//	bpDomain := fmt.Sprintf("bp%02d.%s", bpIndex, conf.GConf.DNSSeed.Domain)
	//	log.Infof("Geting bp address from dns: %v", bpDomain)
	//	r.bpNodes, err = dc.GetBPFromDNSSeed(bpDomain)
	//	if err != nil {
	//		log.WithField("seed", bpDomain).WithError(err).Error(
	//			"getting BP info from DNS failed")
	//		return nil, err 
	//	}
	//}

	if r.bpNodes == nil {
		//wyong, 20201114 
		//r.bpNodes = make(map[proto.RawNodeID]proto.Node)
		r.bpNodes = make(map[proto.NodeID]proto.Node)
	}

	if conf.GConf.KnownNodes != nil {
		for _, n := range conf.GConf.KnownNodes {
			//todo, wyong, 20201114 
			//rawID := n.ID.ToRawNodeID()
			rawID := &n.ID 
			if rawID != nil {
				if n.Role == proto.Leader || n.Role == proto.Follower {
					r.bpNodes[*rawID] = n
				}
				r.SetNodeAddrCache(rawID, n.Addr)
			}
		}
	}

	conf.GConf.SeedBPNodes = make([]proto.Node, 0, len(r.bpNodes))
	for _, n := range r.bpNodes {
		//todo, wyong, 20201114 
		//rawID := n.ID.ToRawNodeID()
		rawID := &n.ID 
		if rawID != nil {
			conf.GConf.SeedBPNodes = append(conf.GConf.SeedBPNodes, n)
			r.SetNodeAddrCache(rawID, n.Addr)
			r.bpNodeIDs[*rawID] = n.Addr
		}
	}

	return r.bpNodeIDs, nil 
}

// GetPBs returns the known BP node id list.
func (r *PBRouter) GetPBs() (bpAddrs []proto.NodeID) {
	bpAddrs = make([]proto.NodeID, 0, len(r.bpNodeIDs))
	for id := range r.bpNodeIDs {
		bpAddrs = append(bpAddrs, proto.NodeID(id.String()))
	}
	return
}

