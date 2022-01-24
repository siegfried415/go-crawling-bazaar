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
	//"fmt"
	"time"

	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/crypto/hash"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/verifier"
)

//go:generate hsp

type UrlBidding struct {
	Url    string 
	Probability	float64	

	ParentUrl	string	
	ParentProbability float64

	Cancel	bool 		
	ExpectCrawlerCount int	
}


// RequestPayload defines a queries payload.
type UrlBiddingPayload struct {
	Requests []UrlBidding `json:"qs"`
	//Requests UrlBiddingArray `json:"qs"`	
}

// RequestHeader defines a query request header.
type UrlBiddingHeader struct {
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
type SignedUrlBiddingHeader struct {
	UrlBiddingHeader
	verifier.DefaultHashSignVerifierImpl
}

// Verify checks hash and signature in request header.
func (sh *SignedUrlBiddingHeader) Verify() (err error) {
	return sh.DefaultHashSignVerifierImpl.Verify(&sh.UrlBiddingHeader)
}

// Sign the request.
func (sh *SignedUrlBiddingHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	return sh.DefaultHashSignVerifierImpl.Sign(&sh.UrlBiddingHeader, signer)
}


// Request defines a complete query request.
type UrlBiddingMessage struct {
	proto.Envelope
	Header        SignedUrlBiddingHeader `json:"h"`
	Payload       UrlBiddingPayload      `json:"p"`
	_marshalCache []byte              `json:"-"`
}

// Verify checks hash and signature in whole request.
func (r *UrlBiddingMessage) Verify() (err error) {
	// verify payload hash in signed header
	if err = verifyHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}
	// verify header sign
	return r.Header.Verify()
}

// Sign the request.
func (r *UrlBiddingMessage) Sign(signer *asymmetric.PrivateKey) (err error) {
	// set query count
	r.Header.BatchCount = uint64(len(r.Payload.Requests))

	// compute payload hash
	if err = buildHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}

	return r.Header.Sign(signer)
}

func (r *UrlBiddingMessage) Empty() bool {
	return len(r.Payload.Requests) == 0 
}
