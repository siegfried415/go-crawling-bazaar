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

package presbyterian

import (
	"context"

	"github.com/pkg/errors"

	//wyong, 20201018
	//"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/network"

	pi "github.com/siegfried415/gdf-rebuild/presbyterian/interfaces"
	"github.com/siegfried415/gdf-rebuild/types"
	net "github.com/siegfried415/gdf-rebuild/net"
)

// ChainRPCService defines a main chain RPC server.
type ChainRPCService struct {
	chain *Chain
}

// wyong, 20201015,  NewChainRPCService returns a new chain RPC service.
func NewChainRPCService(chain *Chain) (s *ChainRPCService, err error) {
        s = &ChainRPCService{
                chain: chain,
        }

	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainAdviseNewBlock"), s.AdviseNewBlockHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainFetchBlock"), s.FetchBlockHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainFetchLastIrreversibleBlock"), s.FetchLastIrreversibleBlockHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainFetchBlockByCount"), s.FetchBlockByCountHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainFetchTxBilling"), s.FetchTxBillingHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainNextAccountNonce"), s.NextAccountNonceHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainAddTx"), s.AddTxHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainQueryAccountTokenBalance"), s.QueryAccountTokenBalanceHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainQuerySQLChainProfile"), s.QuerySQLChainProfileHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolPBChainQueryTxState"), s.QueryTxStateHandler)
	chain.host.SetStreamHandler(protocol.ID("ProtocolKayakApply"), s.QueryAccountSQLChainProfilesHandler)

        return
}

// AdviseNewBlock is the RPC method to advise a new block to target server.
//func (s *ChainRPCService) AdviseNewBlockHandler(req *types.AdviseNewBlockReq, resp *types.AdviseNewBlockResp) error {
func (cs *ChainRPCService) AdviseNewBlockHandler(s network.Stream) {
	ctx := context.Background()
	var req types.AdviseNewBlockReq 
	if err := net.RecvMsg(ctx, s, &req); err != nil {
		return 
	}

	cs.chain.pendingBlocks <- req.Block
	//return nil
}

//wyong, 20201018 
// FetchBlock is the RPC method to fetch a known block from the target server.
//func (s *ChainRPCService) FetchBlockHandler(req *types.FetchBlockReq, resp *types.FetchBlockResp) error {
func (cs *ChainRPCService) FetchBlockHandler(s network.Stream ) {
	ctx := context.Background()
	var req types.FetchBlockReq  
	err := net.RecvMsg(ctx, s, &req) 
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
	_, err = net.SendMsg(ctx, s, &resp) 

	//return err
}

//wyong, 20201020 
// FetchLastIrreversibleBlock fetches the last block irreversible block from block producer.
func (cs *ChainRPCService) FetchLastIrreversibleBlockHandler(
	//req *types.FetchLastIrreversibleBlockReq, resp *types.FetchLastIrreversibleBlockResp) error {
	s network.Stream ) {

	ctx := context.Background()
	var req types.FetchLastIrreversibleBlockReq 
	err := net.RecvMsg(ctx, s, &req) 
	if err != nil {
		return 
	}

	b, c, h, err := cs.chain.fetchLastIrreversibleBlock()
	if err != nil {
		//return err
		return 
	}

	var resp = types.FetchLastIrreversibleBlockResp {
		Block : b, 
		Count : c, 
		Height : h, 
		SQLChains : cs.chain.loadSQLChainProfiles(req.Address), 
	}	

	net.SendMsg(ctx, s, &resp) 
	//return nil
}

//wyong, 20201020 
// FetchBlockByCount is the RPC method to fetch a known block from the target server.
//func (s *ChainRPCService) FetchBlockByCountHandler(req *types.FetchBlockByCountReq, resp *types.FetchBlockResp) error {
func (cs *ChainRPCService) FetchBlockByCountHandler(s network.Stream ) {
	ctx := context.Background()
	var req types.FetchBlockByCountReq  
	err := net.RecvMsg(ctx, s, &req) 
	if err != nil {
		return 
	}

	block, height, err := cs.chain.fetchBlockByCount(req.Count)
	if err != nil {
		//return err
		return 
	}

	var resp = types.FetchBlockResp {
		Count : req.Count, 
		Block : block, 
		Height: height, 
	}	

	net.SendMsg(ctx, s, &resp) 
	//return err
}

//wyong, 20201020 
// FetchTxBilling is the RPC method to fetch a known billing tx from the target server.
//func (s *ChainRPCService) FetchTxBillingHandler(req *types.FetchTxBillingReq, resp *types.FetchTxBillingResp) error {
func (cs *ChainRPCService) FetchTxBillingHandler(s network.Stream){
	//return nil 
}

//wyong, 20201020 
// NextAccountNonce is the RPC method to query the next nonce of an account.
func (cs *ChainRPCService) NextAccountNonceHandler(
	//req *types.NextAccountNonceReq, resp *types.NextAccountNonceResp) (err error
	s network.Stream,
) {
	ctx := context.Background()
	var req types.NextAccountNonceReq 
	err := net.RecvMsg(ctx, s, &req) 
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

	net.SendMsg(ctx, s, &resp) 
	//return
}

//wyong, 20201020 
// AddTx is the RPC method to add a transaction.
//func (s *ChainRPCService) AddTxHandler(req *types.AddTxReq, _ *types.AddTxResp) (err error) {
func (cs *ChainRPCService) AddTxHandler(s network.Stream) {
	ctx := context.Background()
	var req types.AddTxReq 
	err := net.RecvMsg(ctx, s, &req) 
	if err != nil {
		return 
	}

	cs.chain.addTx(&req)
	//return
}

//wyong, 20201020 
// QueryAccountTokenBalance is the RPC method to query account token balance.
func (cs *ChainRPCService) QueryAccountTokenBalanceHandler(
	//req *types.QueryAccountTokenBalanceReq, resp *types.QueryAccountTokenBalanceResp) (err error,
	s network.Stream,
) {
	ctx := context.Background()
	var req types.QueryAccountTokenBalanceReq 
	err := net.RecvMsg(ctx, s, &req) 
	if err != nil {
		return 
	}

	balance, ok := cs.chain.loadAccountTokenBalance(req.Addr, req.TokenType)
	var resp = types.QueryAccountTokenBalanceResp {
		Addr : req.Addr, 
		Balance : balance,
		OK: ok, 
	}	

	net.SendMsg(ctx, s, &resp) 
	//return
}

//wyong, 20201018 
// QuerySQLChainProfile is the RPC method to query SQLChainProfile.
func (cs *ChainRPCService) QuerySQLChainProfileHandler(
	//req *types.QuerySQLChainProfileReq, resp *types.QuerySQLChainProfileResp) (err error
	s network.Stream ,
) {
	ctx := context.Background()
	var req types.QuerySQLChainProfileReq 
	err := net.RecvMsg(ctx, s, &req) 
	if err != nil {
		return 
	}

	p, ok := cs.chain.loadSQLChainProfile(req.DomainID)
	if !ok {
		err = errors.Wrap(ErrDatabaseNotFound, "rpc query sqlchain profile failed")
		return
	}

	var resp = types.QuerySQLChainProfileResp {
		Profile: *p, 
	}	

	net.SendMsg(ctx, s, &resp) 
	//return

}

//wyong, 20201020 
// QueryTxState is the RPC method to query a transaction state.
func (cs *ChainRPCService) QueryTxStateHandler(
	//req *types.QueryTxStateReq, resp *types.QueryTxStateResp) (err error,
	s network.Stream,
) {
	ctx := context.Background()
	var req types.QueryTxStateReq 
	if err := net.RecvMsg(ctx, s, &req); err != nil {
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

	net.SendMsg(ctx, s, &resp) 
	//return
}

//wyong, 20201020 
// QueryAccountSQLChainProfiles is the RPC method to query account sqlchain profiles.
func (cs *ChainRPCService) QueryAccountSQLChainProfilesHandler(
	//req *types.QueryAccountSQLChainProfilesReq, resp *types.QueryAccountSQLChainProfilesResp) (err error,
	s network.Stream ,
) {
	ctx := context.Background()
	var req types.QueryAccountSQLChainProfilesReq 
	if err := net.RecvMsg(ctx, s, &req); err != nil {
		return 
	}

	var profiles []*types.SQLChainProfile
	profiles, err := cs.chain.queryAccountSQLChainProfiles(req.Addr)
	if err != nil {
		return
	}

	//resp.Addr = req.Addr
	//resp.Profiles = profiles
	var resp = types.QueryAccountSQLChainProfilesResp {
		Addr: req.Addr, 
		Profiles: profiles, 
	}	

	net.SendMsg(ctx, s, &resp) 
	//return
}
