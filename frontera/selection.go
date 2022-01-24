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

// package decision implements the decision engine for the bitswap service.
package frontera 

import (
	"errors" 

	pi "github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
        log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
        sortition "github.com/siegfried415/go-crawling-bazaar/frontera/sortition"
        "github.com/siegfried415/go-crawling-bazaar/types"
)

func GetNextAccountNonce(host net.RoutedHost, addr proto.AccountAddress )(nonce pi.AccountNonce, err error ) {
	// allocate nonce
	nonceReq := &types.NextAccountNonceReq{}
	nonceResp := &types.NextAccountNonceResp{}
	nonceReq.Addr = addr

	if err = host.RequestPB("MCC.NextAccountNonce", nonceReq, nonceResp); err != nil {
		// allocate nonce failed
		log.WithError(err).Warning("allocate nonce for transaction failed")
		return pi.AccountNonce(0), errors.New("allocate nonce for transaction failed")

	}

	return nonceResp.Nonce, nil 
}

func GetDomainTokenBalanceAndTotal(host net.RoutedHost,  domainID proto.DomainID,  addr proto.AccountAddress, tt types.TokenType) (balance uint64, totalBalance uint64, err error) {

	balance = 0 
	totalBalance = 0 

        req := new(types.QueryDomainAccountTokenBalanceAndTotalReq)
        resp := new(types.QueryDomainAccountTokenBalanceAndTotalResp)

        req.DomainID = domainID
	req.Addr = addr 
        req.TokenType = tt

        if err = host.RequestPB("MCC.QueryDomainAccountTokenBalanceAndTotal", &req, &resp); err == nil {
                if !resp.OK {
                        //err = ErrNoSuchTokenBalance
			log.Debugf("can't get balance and totalBalance\n")
                	err = errors.New("can't get account 's balance or total balance")
                        return
                }
                balance = resp.Balance
		totalBalance = resp.TotalBalance 
        }

        return
}

// IsWinner returns true if the input challengeTicket wins the election
func IsWinner(challengeTicket []byte, expectedCrawlerCount int64 , money uint64, totalMoney uint64 ) bool {
	return sortition.Select(money, totalMoney, float64(expectedCrawlerCount), challengeTicket ) > 0 
}

