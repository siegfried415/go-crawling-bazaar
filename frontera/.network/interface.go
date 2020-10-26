package network

import (
	"context"

	bsmsg "github.com/siegfried415/go-crawling-market/protocol/biddingsys/message"

	ifconnmgr "gx/ipfs/QmcCk4LZRJPAKuwY9dusFea7LckELZgo5HagErTbm39o38/go-libp2p-interface-connmgr"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peer "gx/ipfs/QmTu65MVbemtUxJEWgsTtzv9Zv9P8rvmqNA4eG9TrTRGYc/go-libp2p-peer"
)

var (
	//wyong, 20190322
	// These two are equivalent, legacy
	//ProtocolBitswapOne    protocol.ID = "/ipfs/bitswap/1.0.0"
	//ProtocolBitswapNoVers protocol.ID = "/ipfs/bitswap"
	//
	//ProtocolBitswap protocol.ID = "/ipfs/bitswap/1.1.0"
	
	ProtocolBiddingsys protocol.ID = "/ipfs/biddingsys"
)

// BitSwapNetwork provides network connectivity for BitSwap sessions
type BiddingNetwork interface {

	// SendMessage sends a BitSwap message to a peer.
	SendMessage(
		context.Context,
		peer.ID,
		bsmsg.BiddingMessage) error

	// SetDelegate registers the Reciver to handle messages received from the
	// network.
	SetDelegate(Receiver)

	ConnectTo(context.Context, peer.ID) error

	NewMessageSender(context.Context, peer.ID) (MessageSender, error)

	ConnectionManager() ifconnmgr.ConnManager

	Stats() NetworkStats

	NodeId() peer.ID	//wyong, 20190118	

	Routing
}

type MessageSender interface {
	SendMsg(context.Context, bsmsg.BiddingMessage) error
	Close() error
	Reset() error
}

// Implement Receiver to receive messages from the BitSwapNetwork
type Receiver interface {
	ReceiveMessage(
		ctx context.Context,
		sender peer.ID,
		incoming bsmsg.BiddingMessage)

	ReceiveError(error)

	// Connected/Disconnected warns bitswap about peer connections
	PeerConnected(peer.ID)
	PeerDisconnected(peer.ID)
}

type Routing interface {
	// FindProvidersAsync returns a channel of providers for the given key
	FindProvidersAsync(context.Context, cid.Cid, int) <-chan peer.ID

	// Provide provides the key to the network
	Provide(context.Context, cid.Cid) error
}

// NetworkStats is a container for statistics about the bitswap network
// the numbers inside are specific to bitswap, and not any other protocols
// using the same underlying network.
type NetworkStats struct {
	MessagesSent  uint64
	MessagesRecvd uint64
}
