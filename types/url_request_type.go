package types


import (
	//"fmt"
	"time"

	"github.com/siegfried415/gdf-rebuild/crypto/asymmetric"
	"github.com/siegfried415/gdf-rebuild/crypto/hash"
	"github.com/siegfried415/gdf-rebuild/crypto/verifier"
	"github.com/siegfried415/gdf-rebuild/proto"
)

//wyong, 20191113 
type UrlRequest struct {
	Url    string 
	Probability	float64	

	//wyong, 20210205 
	ParentUrl 	string
	ParentProbability float64 
}

// RequestPayload defines a queries payload.
type UrlRequestPayload struct {
	Requests []UrlRequest `json:"qs"`
}

// RequestHeader defines a query request header.
type UrlRequestHeader struct {
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
type SignedUrlRequestHeader struct {
	UrlRequestHeader
	verifier.DefaultHashSignVerifierImpl
}

// Verify checks hash and signature in request header.
func (sh *SignedUrlRequestHeader) Verify() (err error) {
	return sh.DefaultHashSignVerifierImpl.Verify(&sh.UrlRequestHeader)
}

// Sign the request.
func (sh *SignedUrlRequestHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	return sh.DefaultHashSignVerifierImpl.Sign(&sh.UrlRequestHeader, signer)
}


// Request defines a complete query request.
type UrlRequestMessage struct {
	proto.Envelope
	Header        SignedUrlRequestHeader `json:"h"`
	Payload       UrlRequestPayload      `json:"p"`
	_marshalCache []byte              `json:"-"`
}

// Verify checks hash and signature in whole request.
func (r *UrlRequestMessage) Verify() (err error) {
	// verify payload hash in signed header
	if err = verifyHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}
	// verify header sign
	return r.Header.Verify()
}

// Sign the request.
func (r *UrlRequestMessage) Sign(signer *asymmetric.PrivateKey) (err error) {
	// set query count
	r.Header.BatchCount = uint64(len(r.Payload.Requests))

	// compute payload hash
	if err = buildHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}

	return r.Header.Sign(signer)
}

