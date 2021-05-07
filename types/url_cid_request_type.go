package types


import (
	//"fmt"
	"time"

	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/crypto/hash"
	"github.com/siegfried415/go-crawling-bazaar/crypto/verifier"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

//go:generate hsp 

//wyong, 20191113 
type UrlCidRequest struct {
	Url    string 
	ParentUrl string 	//wyong, 20210205 
}

// RequestPayload defines a queries payload.
type UrlCidRequestPayload struct {
	Requests []UrlCidRequest `json:"qs"`
}

// RequestHeader defines a query request header.
type UrlCidRequestHeader struct {
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
type SignedUrlCidRequestHeader struct {
	UrlCidRequestHeader
	verifier.DefaultHashSignVerifierImpl
}

// Verify checks hash and signature in request header.
func (sh *SignedUrlCidRequestHeader) Verify() (err error) {
	return sh.DefaultHashSignVerifierImpl.Verify(&sh.UrlCidRequestHeader)
}

// Sign the request.
func (sh *SignedUrlCidRequestHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	return sh.DefaultHashSignVerifierImpl.Sign(&sh.UrlCidRequestHeader, signer)
}


// Request defines a complete query request.
type UrlCidRequestMessage struct {
	proto.Envelope
	Header        SignedUrlCidRequestHeader `json:"h"`
	Payload       UrlCidRequestPayload      `json:"p"`
	_marshalCache []byte              `json:"-"`
}

// Verify checks hash and signature in whole request.
func (r *UrlCidRequestMessage) Verify() (err error) {
	// verify payload hash in signed header
	if err = verifyHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}
	// verify header sign
	return r.Header.Verify()
}

// Sign the request.
func (r *UrlCidRequestMessage) Sign(signer *asymmetric.PrivateKey) (err error) {
	// set query count
	r.Header.BatchCount = uint64(len(r.Payload.Requests))

	// compute payload hash
	if err = buildHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}

	return r.Header.Sign(signer)
}

