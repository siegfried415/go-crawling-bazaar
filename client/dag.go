/*
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

package client

import (
	"context"
	"sync/atomic"

        //"github.com/ipfs/go-cid"

	
	//log "github.com/siegfried415/go-crawling-bazaar/utils/log"
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

	var response types.DagCatResponse
	err = uc.pCaller.RecvMsg(ctx, &response) 
	if err != nil {
		return
	}

	//todo
	//cids := response.Payload.Cids 
	//if len(cids) > 0 {
	//	result, _  = cid.Decode(cids[0])
	//}

	//20220116
	result = response.Payload.Data 

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
