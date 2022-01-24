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

package presbyterian

import (
	"context"

	"github.com/pkg/errors"
	"github.com/libp2p/go-libp2p-core/protocol"

	pi "github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/types"

)

// ChainRPCService defines a main chain RPC server.
type ChainRPCService struct {
	chain *Chain
}

// NewChainRPCService returns a new chain RPC service.
func NewChainRPCService(chain *Chain) (s *ChainRPCService, err error) {
        s = &ChainRPCService{
                chain: chain,
        }

	chain.host.SetStreamHandlerExt(protocol.ID("MCC.AdviseNewBlock"), s.AdviseNewBlockHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.FetchBlock"), s.FetchBlockHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.FetchLastIrreversibleBlock"), s.FetchLastIrreversibleBlockHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.FetchBlockByCount"), s.FetchBlockByCountHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.FetchTxBilling"), s.FetchTxBillingHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.NextAccountNonce"), s.NextAccountNonceHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.AddTx"), s.AddTxHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.QueryAccountTokenBalance"), s.QueryAccountTokenBalanceHandler)

	chain.host.SetStreamHandlerExt(protocol.ID("MCC.QueryDomainAccountTokenBalanceAndTotal"), s.QueryDomainAccountTokenBalanceAndTotalHandler)

	chain.host.SetStreamHandlerExt(protocol.ID("MCC.QuerySQLChainProfile"), s.QuerySQLChainProfileHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.QueryTxState"), s.QueryTxStateHandler)
	chain.host.SetStreamHandlerExt(protocol.ID("MCC.QueryAccountSQLChainProfiles"), s.QueryAccountSQLChainProfilesHandler)

        return
}

// AdviseNewBlock is the RPC method to advise a new block to target server.
//func (s *ChainRPCService) AdviseNewBlockHandler(req *types.AdviseNewBlockReq, resp *types.AdviseNewBlockResp) error {
func (cs *ChainRPCService) AdviseNewBlockHandler(s net.Stream) {
	ctx := context.Background()
	var req types.AdviseNewBlockReq 

	if err := s.RecvMsg(ctx, &req); err != nil {
		return 
	}

	cs.chain.pendingBlocks <- req.Block
}

// FetchBlock is the RPC method to fetch a known block from the target server.
func (cs *ChainRPCService) FetchBlockHandler(s net.Stream ) {
	ctx := context.Background()
	var req types.FetchBlockReq  

	err := s.RecvMsg(ctx, &req) 
	if err != nil {
		return 
	}

	block, count, err := cs.chain.fetchBlockByHeight(req.Height)
	if err != nil {
		return 
	}

	var resp = types.FetchBlockResp {
		Height : req.Height, 
		Block : block, 
		Count : count, 
	}	
	_, err = s.SendMsg(ctx, &resp) 

}

// FetchLastIrreversibleBlock fetches the last block irreversible block from presbyterian.
func (cs *ChainRPCService) FetchLastIrreversibleBlockHandler( s net.Stream ) {

	ctx := context.Background()
	var req types.FetchLastIrreversibleBlockReq 

	err := s.RecvMsg(ctx, &req) 
	if err != nil {
		return 
	}

	b, c, h, err := cs.chain.fetchLastIrreversibleBlock()
	if err != nil {
		return 
	}

	var resp = types.FetchLastIrreversibleBlockResp {
		Block : b, 
		Count : c, 
		Height : h, 
		SQLChains : cs.chain.loadSQLChainProfiles(req.Address), 
	}	

	s.SendMsg(ctx, &resp) 
}

// FetchBlockByCount is the RPC method to fetch a known block from the target server.
func (cs *ChainRPCService) FetchBlockByCountHandler(s net.Stream ) {
	ctx := context.Background()
	var req types.FetchBlockByCountReq  

	err := s.RecvMsg(ctx, &req) 
	if err != nil {
		return 
	}

	block, height, err := cs.chain.fetchBlockByCount(req.Count)
	if err != nil {
		return 
	}

	var resp = types.FetchBlockResp {
		Count : req.Count, 
		Block : block, 
		Height: height, 
	}	

	s.SendMsg(ctx, &resp) 
}

// FetchTxBilling is the RPC method to fetch a known billing tx from the target server.
func (cs *ChainRPCService) FetchTxBillingHandler(s net.Stream){

}

// NextAccountNonce is the RPC method to query the next nonce of an account.
func (cs *ChainRPCService) NextAccountNonceHandler( s net.Stream) {
	ctx := context.Background()
	var req types.NextAccountNonceReq 

	err := s.RecvMsg(ctx, &req) 
	if err != nil {
		return 
	}

	nonce, err := cs.chain.nextNonce(req.Addr)
	if  err != nil {
		return
	}

	var resp = types.NextAccountNonceResp {
		Addr : req.Addr, 
		Nonce : nonce, 
	}	

	s.SendMsg(ctx, &resp) 
}

// AddTx is the RPC method to add a transaction.
func (cs *ChainRPCService) AddTxHandler(s net.Stream) {
	ctx := context.Background()
	var req types.AddTxReq 

	err := s.RecvMsg(ctx, &req) 
	if err != nil {
		return 
	}

	log.WithFields(log.Fields{
		"tx_type":  req.Tx.GetTransactionType().String(),
		"tx_account_addr": req.Tx.GetAccountAddress().String(),
	}).Debugf("ChainRPCService/AddTxHandler")

	cs.chain.addTx(&req)
	var resp = types.AddTxResp {
	}	

	s.SendMsg(ctx, &resp) 
}

// QueryAccountTokenBalance is the RPC method to query account token balance.
func (cs *ChainRPCService) QueryAccountTokenBalanceHandler( s net.Stream) {
	ctx := context.Background()
	var req types.QueryAccountTokenBalanceReq 

	err := s.RecvMsg(ctx, &req) 
	if err != nil {
		return 
	}

	balance, ok := cs.chain.loadAccountTokenBalance(req.Addr, req.TokenType)
	var resp = types.QueryAccountTokenBalanceResp {
		Addr : req.Addr, 
		Balance : balance,
		OK: ok, 
	}	

	s.SendMsg(ctx, &resp) 
}

// QueryAccountTokenBalance is the RPC method to query account token balance.
func (cs *ChainRPCService) QueryDomainAccountTokenBalanceAndTotalHandler(s net.Stream ) {
	ctx := context.Background()
	var req types.QueryDomainAccountTokenBalanceAndTotalReq 

	err := s.RecvMsg(ctx, &req) 
	if err != nil {
		return 
	}

	balance, totalBalance, ok := cs.chain.loadDomainAccountTokenBalanceAndTotal(req.DomainID, req.Addr, req.TokenType)
	var resp = types.QueryDomainAccountTokenBalanceAndTotalResp {
		DomainID : req.DomainID, 
		Addr : req.Addr, 
		Balance : balance,
		TotalBalance : totalBalance, 
		OK: ok, 
	}	

	log.WithFields(log.Fields{
		"addr": req.Addr,
		"balance" : balance, 
		"total" : totalBalance, 
	}).Debugf("ChainRPCService/QueryDomainAccountTokenBalanceAndTotalHandler")

	s.SendMsg(ctx, &resp) 
}

// QuerySQLChainProfile is the RPC method to query SQLChainProfile.
func (cs *ChainRPCService) QuerySQLChainProfileHandler( s net.Stream ) {
	ctx := context.Background()
	var req types.QuerySQLChainProfileReq 

	err := s.RecvMsg(ctx, &req) 
	if err != nil {
		return 
	}

	p, ok := cs.chain.loadSQLChainProfile(req.DomainID)
	if !ok {
		err = errors.Wrap(ErrDatabaseNotFound, "rpc query sqlchain profile failed")
		return
	}

	log.WithFields(log.Fields{
		"ID": p.ID,
		"Address" : p.Address.String(), 
	}).Debugf("ChainRPCService/QuerySQLChainProfileHandler")

	var resp = types.QuerySQLChainProfileResp {
		Profile: *p, 
	}	

	s.SendMsg(ctx, &resp) 

}

// QueryTxState is the RPC method to query a transaction state.
func (cs *ChainRPCService) QueryTxStateHandler( s net.Stream) {
	ctx := context.Background()
	var req types.QueryTxStateReq 

	if err := s.RecvMsg(ctx, &req); err != nil {
		return 
	}

	var state pi.TransactionState
	state, err := cs.chain.queryTxState(req.Hash)
	if err != nil {
		return
	}

	var resp = types.QueryTxStateResp {
		Hash: req.Hash, 
		State: state , 
	}	

	s.SendMsg(ctx, &resp) 
}

// QueryAccountSQLChainProfiles is the RPC method to query account sqlchain profiles.
func (cs *ChainRPCService) QueryAccountSQLChainProfilesHandler( s net.Stream ) {
	ctx := context.Background()
	var req types.QueryAccountSQLChainProfilesReq 

	if err := s.RecvMsg(ctx, &req); err != nil {
		return 
	}

	var profiles []*types.SQLChainProfile
	profiles, err := cs.chain.queryAccountSQLChainProfiles(req.Addr)
	if err != nil {
		return
	}

	var resp = types.QueryAccountSQLChainProfilesResp {
		Addr: req.Addr, 
		Profiles: profiles, 
	}	

	s.SendMsg(ctx, &resp) 
}
