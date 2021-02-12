package types


import (
	//"fmt"
	"time"

	"github.com/siegfried415/gdf-rebuild/crypto/asymmetric"
	"github.com/siegfried415/gdf-rebuild/crypto/hash"
	"github.com/siegfried415/gdf-rebuild/crypto/verifier"
	"github.com/siegfried415/gdf-rebuild/proto"
)

//go:generate hsp

//wyong, 20191113 
type UrlBidding struct {
	Url    string 
	Probability	float64	

	ParentUrl	string	//wyong, 20210125 
	ParentProbability float64	//wyong, 20210205 

	Cancel	bool 		//wyong, 20200827 
	ExpectCrawlerCount int	//wyong, 20210203 
}


//define UrlBiddingArray for sort operation, wyong, 20210125  
//type UrlBiddingArray []UrlBidding
// 
//func (s UrlBiddingArray) Len() int { 
//	return len(s) 
//}
// 
//func (s UrlBiddingArray) Swap(i, j int) { 
//	s[i], s[j] = s[j], s[i] 
//}
// 
//func (s UrlBiddingArray) Less(i, j int) bool { 
//	return s[i].Probability < s[j].Probability 
//}
 
// RequestPayload defines a queries payload.
type UrlBiddingPayload struct {
	Requests []UrlBidding `json:"qs"`
	//Requests UrlBiddingArray `json:"qs"`	//wyong, 20210125 
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

//wyong, 20200827 
func (r *UrlBiddingMessage) Empty() bool {
	return len(r.Payload.Requests) == 0 
}
