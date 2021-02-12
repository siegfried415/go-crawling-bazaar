/*
 * Copyright 2018 The CovenantSQL Authors.
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

	"github.com/pkg/errors"

	"github.com/siegfried415/gdf-rebuild/crypto/hash"
	"github.com/siegfried415/gdf-rebuild/proto"

	//wyong, 20200820 
	//"github.com/ipfs/go-cid"

)

//go:generate hsp

// ResponseRow defines single row of query response.
//type ResponseRow struct {
//	Values []interface{}
//}

//wyong, 20210203 
type UrlBid struct {
	Url    	string 
	Cid 	string	
	
	Hash	[]byte	//wyong, 20210203
	Proof 	[]byte 	//wyong, 20210203
}

// ResponsePayload defines column names and rows of query response.
type UrlBidPayload struct {
	//Columns   []string      `json:"c"`
	//DeclTypes []string      `json:"t"`
	//Rows      []ResponseRow `json:"r"`

	//Cids	    map[string]string	`json:"c"`	//wyong, 20200907 
	Bids	    []UrlBid		`json:"b"`	//wyong, 20210203 
}

// ResponseHeader defines a query response header.
type UrlBidHeader struct {
	RequestHash     hash.Hash            `json:"rh"`
	Target		proto.NodeID	     `json:"tg"` // wyong, 20200827 
	NodeID          proto.NodeID         `json:"id"` // response node id
	Timestamp       time.Time            `json:"t"`  // time in UTC zone
	RowCount        uint64               `json:"c"`  // response row count of payload
	LogOffset       uint64               `json:"o"`  // request log offset
	LastInsertID    int64                `json:"l"`  // insert insert id
	AffectedRows    int64                `json:"a"`  // affected rows
	PayloadHash     hash.Hash            `json:"dh"` // hash of query response payload
	ResponseAccount proto.AccountAddress `json:"aa"` // response account
}

// GetRequestHash returns the request hash.
func (h *UrlBidHeader) GetRequestHash() hash.Hash {
	return h.RequestHash
}

// GetRequestTimestamp returns the request timestamp.
func (h *UrlBidHeader) GetRequestTimestamp() time.Time {
	return h.Timestamp
}

// SignedResponseHeader defines a signed query response header.
type UrlBidSignedHeader struct {
	UrlBidHeader
	UrlBidHash hash.Hash
}

// Hash returns the response header hash.
func (sh *UrlBidSignedHeader) Hash() hash.Hash {
	return sh.UrlBidHash
}

// VerifyHash verify the hash of the response.
func (sh *UrlBidSignedHeader) VerifyHash() (err error) {
	return errors.Wrap(verifyHash(&sh.UrlBidHeader, &sh.UrlBidHash),
		"verify response header hash failed")
}

// BuildHash computes the hash of the response header.
func (sh *UrlBidSignedHeader) BuildHash() (err error) {
	return errors.Wrap(buildHash(&sh.UrlBidHeader, &sh.UrlBidHash),
		"compute response header hash failed")
}

// Response defines a complete query response.
type UrlBidMessage struct {
	Header  UrlBidSignedHeader `json:"h"`
	Payload UrlBidPayload      `json:"p"`
}

// BuildHash computes the hash of the response.
func (r *UrlBidMessage) BuildHash() (err error) {
	// set rows count
	//r.Header.RowCount = uint64(len(r.Payload.Rows))

	// build hash in header
	if err = buildHash(&r.Payload, &r.Header.PayloadHash); err != nil {
		err = errors.Wrap(err, "compute response payload hash failed")
		return
	}

	// compute header hash
	return r.Header.BuildHash()
}

// VerifyHash verify the hash of the response.
func (r *UrlBidMessage) VerifyHash() (err error) {
	if err = verifyHash(&r.Payload, &r.Header.PayloadHash); err != nil {
		err = errors.Wrap(err, "verify response payload hash failed")
		return
	}

	return r.Header.VerifyHash()
}

// Hash returns the response header hash.
func (r *UrlBidMessage) Hash() hash.Hash {
	return r.Header.Hash()
}
