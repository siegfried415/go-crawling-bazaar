/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package net

import (
	"fmt"
	"net"
	"sync"

	"github.com/CovenantSQL/beacon/ipv6"
	"github.com/pkg/errors"

	"github.com/siegfried415/go-crawling-bazaar/crypto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

const (
	// ID is node id
	ID = "id."
	// PUBKEY is public key
	PUBKEY = "pub."
	// NONCE is nonce
	NONCE = "n."
	// ADDR is address
	ADDR = "addr."
)

// IPv6SeedClient is IPv6 DNS seed client
type IPv6SeedClient struct{}

// GetPBFromDNSSeed gets PB info from the IPv6 domain
func (isc *IPv6SeedClient) GetPBFromDNSSeed(PBDomain string) (PBNodes map[proto.RawNodeID]proto.Node, err error) {
	// Public key
	var pubKeyBuf []byte
	var pubBuf, addrBuf, nodeIDBuf []byte
	var pubErr, addrErr, nodeIDErr error
	wg := new(sync.WaitGroup)
	wg.Add(4)

	f := func(host string) ([]net.IP, error) {
		return net.LookupIP(host)
	}
	// Public key
	go func() {
		defer wg.Done()
		pubBuf, pubErr = ipv6.FromDomain(PUBKEY+PBDomain, f)
	}()

	// Addr
	go func() {
		defer wg.Done()
		addrBuf, addrErr = ipv6.FromDomain(ADDR+PBDomain, f)
	}()
	// NodeID
	go func() {
		defer wg.Done()
		nodeIDBuf, nodeIDErr = ipv6.FromDomain(ID+PBDomain, f)
	}()

	wg.Wait()

	switch {
	case pubErr != nil:
		err = pubErr
		return

	case addrErr != nil:
		err = addrErr
		return
	case nodeIDErr != nil:
		err = nodeIDErr
		return
	}

	// For bug that trim the public header before or equal cql 0.7.0
	if len(pubBuf) == asymmetric.PublicKeyBytesLen-1 {
		pubKeyBuf = make([]byte, asymmetric.PublicKeyBytesLen)
		pubKeyBuf[0] = asymmetric.PublicKeyFormatHeader
		copy(pubKeyBuf[1:], pubBuf)
	} else if len(pubBuf) == 48 {
		pubKeyBuf, err = crypto.RemovePKCSPadding(pubBuf)
		if err != nil {
			return
		}
	} else {
		return nil, errors.Errorf("error public key bytes len: %d", len(pubBuf))
	}
	var pubKey asymmetric.PublicKey
	err = pubKey.UnmarshalBinary(pubKeyBuf)
	if err != nil {
		return
	}

	addrBytes, err := crypto.RemovePKCSPadding(addrBuf)
	if err != nil {
		return
	}

	var nodeID proto.RawNodeID
	err = nodeID.SetBytes(nodeIDBuf)
	if err != nil {
		return
	}

	PBNodes = make(map[proto.RawNodeID]proto.Node)
	PBNodes[nodeID] = proto.Node{
		ID:        nodeID.ToNodeID(),
		Addr:      string(addrBytes),
		PublicKey: &pubKey,
	}

	return
}

// GenPBIPv6 generates the IPv6 addrs contain PB info
func (isc *IPv6SeedClient) GenPBIPv6(node *proto.Node, domain string) (out string, err error) {
	// NodeID
	nodeIDIps, err := ipv6.ToIPv6(node.ID.ToRawNodeID().AsBytes())
	if err != nil {
		return "", err
	}
	for i, ip := range nodeIDIps {
		out += fmt.Sprintf("%02d.%s%s	1	IN	AAAA	%s\n", i, ID, domain, ip)
	}

	pubKeyIps, err := ipv6.ToIPv6(crypto.AddPKCSPadding(node.PublicKey.Serialize()))
	if err != nil {
		return "", err
	}
	for i, ip := range pubKeyIps {
		out += fmt.Sprintf("%02d.%s%s	1	IN	AAAA	%s\n", i, PUBKEY, domain, ip)
	}

	// Addr
	addrIps, err := ipv6.ToIPv6(crypto.AddPKCSPadding([]byte(node.Addr)))
	if err != nil {
		return "", err
	}
	for i, ip := range addrIps {
		out += fmt.Sprintf("%02d.%s%s	1	IN	AAAA	%s\n", i, ADDR, domain, ip)
	}

	return
}
