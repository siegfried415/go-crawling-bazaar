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

	"github.com/siegfried415/go-crawling-bazaar/crypto/hash"
	"github.com/siegfried415/go-crawling-bazaar/proto"

	//wyong, 20200820 
	//"github.com/ipfs/go-cid"

)

//go:generate hsp

// ResponseRow defines single row of query response.
//type ResponseRow struct {
//	Values []interface{}
//}

// ResponsePayload defines column names and rows of query response.
type UrlCidResponsePayload struct {
	//Columns   []string      `json:"c"`
	//DeclTypes []string      `json:"t"`
	//Rows      []ResponseRow `json:"r"`
	Cids	  []string	`json:"c"`	//wyong, 20200907 
}

// ResponseHeader defines a query response header.
type UrlCidResponseHeader struct {
	Request         UrlCidRequestHeader        `json:"r"`
	RequestHash     hash.Hash            `json:"rh"`
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
func (h *UrlCidResponseHeader) GetRequestHash() hash.Hash {
	return h.RequestHash
}

// GetRequestTimestamp returns the request timestamp.
func (h *UrlCidResponseHeader) GetRequestTimestamp() time.Time {
	return h.Request.Timestamp
}

// SignedResponseHeader defines a signed query response header.
type UrlCidSignedResponseHeader struct {
	UrlCidResponseHeader
	UrlCidResponseHash hash.Hash
}

// Hash returns the response header hash.
func (sh *UrlCidSignedResponseHeader) Hash() hash.Hash {
	return sh.UrlCidResponseHash
}

// VerifyHash verify the hash of the response.
func (sh *UrlCidSignedResponseHeader) VerifyHash() (err error) {
	return errors.Wrap(verifyHash(&sh.UrlCidResponseHeader, &sh.UrlCidResponseHash),
		"verify response header hash failed")
}

// BuildHash computes the hash of the response header.
func (sh *UrlCidSignedResponseHeader) BuildHash() (err error) {
	return errors.Wrap(buildHash(&sh.UrlCidResponseHeader, &sh.UrlCidResponseHash),
		"compute response header hash failed")
}

// Response defines a complete query response.
type UrlCidResponse struct {
	Header  UrlCidSignedResponseHeader `json:"h"`
	Payload UrlCidResponsePayload      `json:"p"`
}

// BuildHash computes the hash of the response.
func (r *UrlCidResponse) BuildHash() (err error) {
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
func (r *UrlCidResponse) VerifyHash() (err error) {
	if err = verifyHash(&r.Payload, &r.Header.PayloadHash); err != nil {
		err = errors.Wrap(err, "verify response payload hash failed")
		return
	}

	return r.Header.VerifyHash()
}

// Hash returns the response header hash.
func (r *UrlCidResponse) Hash() hash.Hash {
	return r.Header.Hash()
}
