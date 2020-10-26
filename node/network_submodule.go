package node

import (
	"github.com/siegfried415/gdf-rebuild/net"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
)

// NetworkSubmodule enhances the `Node` with networking capabilities.
type NetworkSubmodule struct {
	NetworkName string

	host host.Host

	// TODO: do we need a second host? (see: https://github.com/siegfried415/go-decentralized-frontera/issues/3477)
	PeerHost host.Host

	Bootstrapper *net.Bootstrapper

	//todo, wyong, 20201015 
	// PeerTracker maintains a list of peers good for fetching.
	//PeerTracker *net.PeerTracker

	// Router is a router from IPFS
	Router routing.Routing
}
