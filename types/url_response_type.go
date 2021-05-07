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

/* wyong, 20200824 
// ResponsePayload defines column names and rows of query response.
type UrlResponsePayload struct {
	//Columns   []string      `json:"c"`
	//DeclTypes []string      `json:"t"`
	//Rows      []ResponseRow `json:"r"`
}
*/

// ResponseHeader defines a query response header.
type UrlResponseHeader struct {
	Request         UrlRequestHeader        `json:"r"`
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
func (h *UrlResponseHeader) GetRequestHash() hash.Hash {
	return h.RequestHash
}

// GetRequestTimestamp returns the request timestamp.
func (h *UrlResponseHeader) GetRequestTimestamp() time.Time {
	return h.Request.Timestamp
}

// SignedResponseHeader defines a signed query response header.
type UrlSignedResponseHeader struct {
	UrlResponseHeader
	UrlResponseHash hash.Hash
}

// Hash returns the response header hash.
func (sh *UrlSignedResponseHeader) Hash() hash.Hash {
	return sh.UrlResponseHash
}

// VerifyHash verify the hash of the response.
func (sh *UrlSignedResponseHeader) VerifyHash() (err error) {
	return errors.Wrap(verifyHash(&sh.UrlResponseHeader, &sh.UrlResponseHash),
		"verify response header hash failed")
}

// BuildHash computes the hash of the response header.
func (sh *UrlSignedResponseHeader) BuildHash() (err error) {
	return errors.Wrap(buildHash(&sh.UrlResponseHeader, &sh.UrlResponseHash),
		"compute response header hash failed")
}

// Response defines a complete query response.
type UrlResponse struct {
	Header  UrlSignedResponseHeader `json:"h"`
	//Payload UrlResponsePayload      `json:"p"`
}

// BuildHash computes the hash of the response.
func (r *UrlResponse) BuildHash() (err error) {
	// set rows count
	//r.Header.RowCount = uint64(len(r.Payload.Rows))

	/*wyong, 20200824
	// build hash in header
	if err = buildHash(&r.Payload, &r.Header.PayloadHash); err != nil {
		err = errors.Wrap(err, "compute response payload hash failed")
		return
	}
	*/

	// compute header hash
	return r.Header.BuildHash()
}

// VerifyHash verify the hash of the response.
func (r *UrlResponse) VerifyHash() (err error) {
	/* wyong, 20200824 
	if err = verifyHash(&r.Payload, &r.Header.PayloadHash); err != nil {
		err = errors.Wrap(err, "verify response payload hash failed")
		return
	}
	*/

	return r.Header.VerifyHash()
}

// Hash returns the response header hash.
func (r *UrlResponse) Hash() hash.Hash {
	return r.Header.Hash()
}
