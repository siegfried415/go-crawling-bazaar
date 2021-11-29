
// package decision implements the decision engine for the bitswap service.
package frontera 

import (
	//"context"
	//"sync"
	//"time"
	//"fmt" 
	//"math/big" 
	"errors" 

	//wyong, 20210202
	//"crypto/ecdsa"

	"github.com/siegfried415/go-crawling-bazaar/proto"
        //"github.com/siegfried415/go-crawling-bazaar/kms"

	//wyong, 20200827 
        "github.com/siegfried415/go-crawling-bazaar/types"

	//cid "github.com/ipfs/go-cid"

	//wyong, 20201215
        log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	
	//wyong, 20210118 
	//ecvrf "github.com/vechain/go-ecvrf"

	//wyong, 20210630
        sortition "github.com/siegfried415/go-crawling-bazaar/frontera/sortition"

	//wyong, 20210706 
	net "github.com/siegfried415/go-crawling-bazaar/net"
	//crypto "github.com/siegfried415/go-crawling-bazaar/crypto"

	//wyong, 20210719
	pi "github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
)

//wyong, 20210719 
func GetNextAccountNonce(host net.RoutedHost, addr proto.AccountAddress )(nonce pi.AccountNonce, err error ) {
	// allocate nonce
	nonceReq := &types.NextAccountNonceReq{}
	nonceResp := &types.NextAccountNonceResp{}
	nonceReq.Addr = addr

	if err = host.RequestPB("MCC.NextAccountNonce", nonceReq, nonceResp); err != nil {
		// allocate nonce failed
		//le.WithError(err).Warning("allocate nonce for transaction failed")
		return pi.AccountNonce(0), errors.New("allocate nonce for transaction failed")

	}

	return nonceResp.Nonce, nil 
}

func GetDomainTokenBalanceAndTotal(host net.RoutedHost,  domainID proto.DomainID,  addr proto.AccountAddress, tt types.TokenType) (balance uint64, totalBalance uint64, err error) {

	balance = 0 
	totalBalance = 0 

        req := new(types.QueryDomainAccountTokenBalanceAndTotalReq)
        resp := new(types.QueryDomainAccountTokenBalanceAndTotalResp)

        //var pubKey *asymmetric.PublicKey
        //if pubKey, err = kms.GetLocalPublicKey(); err != nil {
        //        return
        //}
	//
        //if req.Addr, err = crypto.PubKeyHash(pubKey); err != nil {
        //        return
        //}

        req.DomainID = domainID
	req.Addr = addr 
        req.TokenType = tt

        if err = host.RequestPB("MCC.QueryDomainAccountTokenBalanceAndTotal", &req, &resp); err == nil {
                if !resp.OK {
                        //err = ErrNoSuchTokenBalance
			log.Debugf("BiddingServer/getDomainTokenBalanceAndTotal,  can't get balance and totalBalance\n")
                	err = errors.New("can't get account 's balance or total balance")
                        return
                }
                balance = resp.Balance
		totalBalance = resp.TotalBalance 
        }

	log.Debugf("BiddingServer/getDomainTokenBalanceAndTotal, got balance=%d, totalBalance=%d\n", balance, totalBalance)

        return
}

/*wyong, 20210630 
//todo, wyong, 20210118 
// IsWinner returns true if the input challengeTicket wins the election
func (bs *BiddingServer) IsWinner(challengeTicket []byte, expectCrawlerCount int64 , totalPeersCount int64 ) bool {
	//wyong, 20210127 
        // (ChallengeTicket / MaxChallengeTicket) < (ExpectedCrawlerCount / totalPeersCount )
        // ->
        // ChallengeTicket * totalPeersCount < ExpectedCrawlerCount * MaxChallengeTicket
	
	//wyong, 20210202 
        lhs := &big.Int{}
        lhs.SetBytes(challengeTicket[:])
	log.Debugf("BiddingServer/IsWinner(10), lhs=%s\n", lhs.String())
	lhs.Mul(lhs, big.NewInt(totalPeersCount)) 
	log.Debugf("BiddingServer/IsWinner(20), lhs=%s\n", lhs.String())

	rhs := big.NewInt(expectCrawlerCount)
	log.Debugf("BiddingServer/IsWinner(30), rhs=%s\n", rhs.String())

	//wyong, 20210219 
	//MaxChallengeTicket := bs.getMaxChallengeTicket(challengeTicket )
	//log.Debugf("BiddingServer/IsWinner(40), MaxChallengeTicket=%s\n", MaxChallengeTicket.String())
	rhs.Mul(rhs, MaxChallengeTicket ) 
	log.Debugf("BiddingServer/IsWinner(50), rhs=%s\n", rhs.String())

	return lhs.Cmp(rhs) < 0 

}
*/

// IsWinner returns true if the input challengeTicket wins the election, wyong, 20210630 
func IsWinner(challengeTicket []byte, expectedCrawlerCount int64 , money uint64, totalMoney uint64 ) bool {
	return sortition.Select(money, totalMoney, float64(expectedCrawlerCount), challengeTicket ) > 0 
}

