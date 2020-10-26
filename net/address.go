/*
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


package net

import (
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func PeerAddrToAddrInfo(addr string) (*peer.AddrInfo, error) {
	a, err := ma.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}

	pinfo, err := peer.AddrInfoFromP2pAddr(a)
	if err != nil {
		return nil, err
	}
	
	return pinfo, nil 

}

// PeerAddrsToAddrInfo converts a slice of string peer addresses
// (multiaddr + ipfs peerid) to PeerInfos.
func PeerAddrsToAddrInfo(addrs []string) ([]peer.AddrInfo, error) {
	var pis []peer.AddrInfo
	for _, addr := range addrs {
		pinfo, err := PeerAddrToAddrInfo(addr)
		if err!= nil {
			return nil, err 
		}
		pis = append(pis, *pinfo)
	}
	return pis, nil
}

// AddrInfoToPeerIDs converts a slice of AddrInfo to a slice of peerID's.
func AddrInfoToPeerIDs(ai []peer.AddrInfo) []peer.ID {
	var pis []peer.ID
	for _, a := range ai {
		pis = append(pis, a.ID)
	}
	return pis
}
