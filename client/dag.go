
package client

import (
	"context"
	"sync/atomic"

        //"github.com/ipfs/go-cid"

	"github.com/siegfried415/go-crawling-bazaar/types"
)

func (c *Conn) sendDagCatRequest(ctx context.Context, request types.DagCatRequest ) (result []byte, err error) {
	var uc *pconn // peer connection used to execute the queries
	uc = c.leader

	// todo, select target peer accrodingto hash of url 
	// use follower pconn only when the query is readonly
	//if queryType == types.ReadQuery && c.follower != nil {
	//	uc = c.follower
	//}
	//if uc == nil {
	//	uc = c.follower
	//}

	// allocate sequence
	connID, seqNo := allocateConnAndSeq()
	defer putBackConn(connID)

	// build request
	reqMsg := &types.DagCatRequestMessage {
		Header: types.SignedDagCatRequestHeader{
			DagCatRequestHeader: types.DagCatRequestHeader{
				QueryType:    types.ReadQuery, //queryType,
				NodeID:       c.localNodeID,
				//DomainID:   c.domainID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    getLocalTime(),
			},
		},
		Payload: types.DagCatRequestPayload{
			Requests : []types.DagCatRequest{request}, 
		},
	}

	if err = reqMsg.Sign(c.privKey); err != nil {
		return
	}

	// set receipt if key exists in context
	if val := ctx.Value(&ctxReceiptKey); val != nil {
		val.(*atomic.Value).Store(&Receipt{
			RequestHash: reqMsg.Header.Hash(),
		})
	}

	
	if _, err = uc.pCaller.SendMsg(ctx, reqMsg ); err != nil {
		return
	}

	var response types.UrlCidResponse
	uc.pCaller.RecvMsg(ctx, &response) 
	if err != nil {
		return
	}

	//todo
	//cids := response.Payload.Cids 
	//if len(cids) > 0 {
	//	result, _  = cid.Decode(cids[0])
	//}

	return
}

func (c *Conn) DagCat(ctx context.Context, cid string ) (result []byte,  err error) {
	request := types.DagCatRequest {
		Cid : cid,
	}

	if result,  err = c.sendDagCatRequest(ctx, request); err != nil {
		return
	}

	return
}
