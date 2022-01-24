/*
 * Copyright 2018 The CovenantSQL Authors.
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

package types


import (
	"time"

        //"github.com/ipfs/go-cid"

	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/crypto/hash"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/verifier"
)

//go:generate hsp 

type DagCatRequest struct {
	Cid   	string 	
}

// RequestPayload defines a queries payload.
type DagCatRequestPayload struct {
	Requests []DagCatRequest `json:"qs"`

}

// RequestHeader defines a query request header.
type DagCatRequestHeader struct {
	QueryType    QueryType        `json:"qt"`
	NodeID       proto.NodeID     `json:"id"`   // request node id
	DomainID   proto.DomainID `json:"did"` // request domain id
	ConnectionID uint64           `json:"cid"`
	SeqNo        uint64           `json:"seq"`
	Timestamp    time.Time        `json:"t"`  // time in UTC zone
	BatchCount   uint64           `json:"bc"` // query count in this request
	QueriesHash  hash.Hash        `json:"qh"` // hash of query payload
}


// SignedRequestHeader defines a signed query request header.
type SignedDagCatRequestHeader struct {
	DagCatRequestHeader
	verifier.DefaultHashSignVerifierImpl
}

// Verify checks hash and signature in request header.
func (sh *SignedDagCatRequestHeader) Verify() (err error) {
	return sh.DefaultHashSignVerifierImpl.Verify(&sh.DagCatRequestHeader)
}

// Sign the request.
func (sh *SignedDagCatRequestHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	return sh.DefaultHashSignVerifierImpl.Sign(&sh.DagCatRequestHeader, signer)
}


// Request defines a complete query request.
type DagCatRequestMessage struct {
	proto.Envelope
	Header        SignedDagCatRequestHeader `json:"h"`
	Payload       DagCatRequestPayload      `json:"p"`
	_marshalCache []byte              `json:"-"`
}

// Verify checks hash and signature in whole request.
func (r *DagCatRequestMessage) Verify() (err error) {
	// verify payload hash in signed header
	if err = verifyHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}
	// verify header sign
	return r.Header.Verify()
}

// Sign the request.
func (r *DagCatRequestMessage) Sign(signer *asymmetric.PrivateKey) (err error) {
	// set query count
	r.Header.BatchCount = uint64(len(r.Payload.Requests))

	// compute payload hash
	if err = buildHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}

	return r.Header.Sign(signer)
}

