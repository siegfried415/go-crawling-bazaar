/*
 * Copyright (c) 2018 Filecoin Project
 * Copyright (c) 2022 https://github.com/siegfried415
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


package net

import (
	"context"
	"fmt"
	"sort"
	"sync"

	logging "github.com/ipfs/go-log"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

var lognetwork = logging.Logger("net.network")

// SwarmConnInfo represents details about a single swarm connection.
type SwarmConnInfo struct {
	Addr    string
	Peer    string
	Latency string
	Muxer   string
	Streams []SwarmStreamInfo
}

// SwarmStreamInfo represents details about a single swarm stream.
type SwarmStreamInfo struct {
	Protocol string
}

func (ci *SwarmConnInfo) Less(i, j int) bool {
	return ci.Streams[i].Protocol < ci.Streams[j].Protocol
}

func (ci *SwarmConnInfo) Len() int {
	return len(ci.Streams)
}

func (ci *SwarmConnInfo) Swap(i, j int) {
	ci.Streams[i], ci.Streams[j] = ci.Streams[j], ci.Streams[i]
}

// SwarmConnInfos represent details about a list of swarm connections.
type SwarmConnInfos struct {
	Peers []SwarmConnInfo
}

func (ci SwarmConnInfos) Less(i, j int) bool {
	return ci.Peers[i].Addr < ci.Peers[j].Addr
}

func (ci SwarmConnInfos) Len() int {
	return len(ci.Peers)
}

func (ci SwarmConnInfos) Swap(i, j int) {
	ci.Peers[i], ci.Peers[j] = ci.Peers[j], ci.Peers[i]
}

// Network is a unified interface for dealing with libp2p
type Network struct {
	host host.Host
}

// New returns a new Network
func New( host host.Host) *Network {
	return &Network{
		host:       host,
	}
}

// GetPeerAddresses gets the current addresses of the node
func (network *Network) GetPeerAddresses() []ma.Multiaddr {
	return network.host.Addrs()
}

// GetPeerID gets the current peer id from libp2p-host
func (network *Network) GetPeerID() peer.ID {
	return network.host.ID()
}

// ConnectionResult represents the result of an attempted connection from the
// Connect method.
type ConnectionResult struct {
	PeerID peer.ID
	Err    error
}

// Connect connects to peers at the given addresses. Does not retry.
func (network *Network) Connect(ctx context.Context, addrs []string) (<-chan ConnectionResult, error) {
        lognetwork.Event(ctx, "swarmConnectCmdTo", func() logging.Loggable {
                return logging.Metadata{"to": addrs[0]}
        }())

	outCh := make(chan ConnectionResult)
	swrm, ok := network.host.Network().(*swarm.Swarm)
	if !ok {
		return nil, fmt.Errorf("peerhost network was not a swarm")
	}

	pis, err := PeerAddrsToAddrInfo(addrs)
	if err != nil {
		return nil, err
	}

	go func() {
		var wg sync.WaitGroup
		wg.Add(len(pis))

		for _, pi := range pis {
			go func(pi peer.AddrInfo) {
				swrm.Backoff().Clear(pi.ID)
				err := network.host.Connect(ctx, pi)
				outCh <- ConnectionResult{
					PeerID: pi.ID,
					Err:    err,
				}
				wg.Done()
			}(pi)
		}

		wg.Wait()
		close(outCh)
	}()

	return outCh, nil
}

// Peers lists peers currently available on the network
func (network *Network) Peers(ctx context.Context, verbose, latency, streams bool) (*SwarmConnInfos, error) {
	if network.host == nil {
		return nil, errors.New("node must be online")
	}

	conns := network.host.Network().Conns()

	var out SwarmConnInfos
	for _, c := range conns {
		pid := c.RemotePeer()
		addr := c.RemoteMultiaddr()

		ci := SwarmConnInfo{
			Addr: addr.String(),
			Peer: pid.Pretty(),
		}

		if verbose || latency {
			lat := network.host.Peerstore().LatencyEWMA(pid)
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
