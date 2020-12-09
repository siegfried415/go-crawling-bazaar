/*
 *  Copyright 2018 The CovenantSQL Authors.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package frontera

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	//wyong, 20201008
	//host "github.com/libp2p/go-libp2p-core/host"

	"github.com/siegfried415/gdf-rebuild/presbyterian/interfaces"
	"github.com/siegfried415/gdf-rebuild/frontera/chainbus"
	"github.com/siegfried415/gdf-rebuild/proto"
	//"github.com/siegfried415/gdf-rebuild/route"

	//wyong, 20201008 
	//rpc "github.com/siegfried415/gdf-rebuild/rpc/mux"
	net "github.com/siegfried415/gdf-rebuild/net"

	"github.com/siegfried415/gdf-rebuild/types"
	"github.com/siegfried415/gdf-rebuild/utils/log"
)

// BusService defines the man chain bus service type.
type BusService struct {
	chainbus.Bus

	host net.RoutedHost 	
	//caller *rpc.Caller

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	checkInterval time.Duration
	localAddress  proto.AccountAddress

	lock             sync.RWMutex // a lock for the map
	blockCount       uint32
	sqlChainProfiles map[proto.DomainID]*types.SQLChainProfile
	sqlChainState    map[proto.DomainID]map[proto.AccountAddress]*types.PermStat
}

// NewBusService creates a new chain bus instance.
func NewBusService(
	ctx context.Context, host net.RoutedHost, addr proto.AccountAddress, checkInterval time.Duration) (_ *BusService,
) {
	ctd, ccl := context.WithCancel(ctx)
	bs := &BusService{
		Bus:           chainbus.New(),
		wg:            sync.WaitGroup{},

		//wyong, 20201008 
		//caller:        rpc.NewCaller(),
		host:		host, 

		ctx:           ctd,
		cancel:        ccl,
		checkInterval: checkInterval,
		localAddress:  addr,
	}
	// State initialization: fetch last block and update fields `blockCount` and `sqlChainProfiles`
	var _, profiles, count = bs.requestLastBlock()
	bs.updateState(count, profiles)
	return bs
}

// GetCurrentDBMapping returns current cached db mapping.
func (bs *BusService) GetCurrentDomainMapping() (domainMap map[proto.DomainID]*types.SQLChainProfile) {
	domainMap = make(map[proto.DomainID]*types.SQLChainProfile)
	bs.lock.RLock()
	defer bs.lock.RUnlock()
	for k, v := range bs.sqlChainProfiles {
		domainMap[k] = v
	}
	return
}

func (bs *BusService) updateState(count uint32, profiles []*types.SQLChainProfile) {
	bs.lock.Lock()
	defer bs.lock.Unlock()
	var (
		rebuilt       = make(map[proto.DomainID]*types.SQLChainProfile)
		sqlchainState = make(map[proto.DomainID]map[proto.AccountAddress]*types.PermStat)
	)
	for _, v := range profiles {
		rebuilt[v.ID] = v
		sqlchainState[v.ID] = make(map[proto.AccountAddress]*types.PermStat)
		for _, user := range v.Users {
			sqlchainState[v.ID][user.Address] = &types.PermStat{
				Permission: user.Permission,
				Status:     user.Status,
			}
		}
	}
	atomic.StoreUint32(&bs.blockCount, count)
	bs.sqlChainProfiles = rebuilt
	bs.sqlChainState = sqlchainState
}

func (bs *BusService) subscribeBlock(ctx context.Context) {
	defer bs.wg.Done()

	log.Info("start to subscribe blocks")
	for {
		select {
		case <-ctx.Done():
			log.Info("exit subscription service")
			return
		case <-time.After(bs.checkInterval):
			// fetch block from remote block producer
			c := atomic.LoadUint32(&bs.blockCount)
			log.Debugf("fetch block in count: %d", c)
			b, profiles, newCount := bs.requestLastBlock()
			if b == nil {
				continue
			}
			if newCount <= c {
				continue
			}

			log.WithFields(log.Fields{
				"last_count": c,
				"new_count":  newCount,
				"block_hash": b.BlockHash().Short(4),
				"tx_num":     len(b.Transactions),
			}).Debug("success fetch block")

			// Write sqlchain profile state first (bound to the last irreversible block)
			bs.updateState(newCount, profiles)

			// Fetch any intermediate irreversible blocks and extract txs
			for i := c + 1; i < newCount; i++ {
				var (
					block *types.BPBlock
					err   error
				)
				if block, err = bs.fetchBlockByCount(i); err != nil {
					log.WithError(err).WithFields(log.Fields{
						"count": i,
					}).Warn("failed to fetch block")
					continue
				}
				bs.extractTxs(block, i)
			}

			// Extract txs in last irreversible block
			bs.extractTxs(b, c)
		}
	}
}

func (bs *BusService) fetchBlockByCount(count uint32) (block *types.BPBlock, err error) {
	var (
		//wyong, 20201204 
		req = types.FetchBlockByCountReq{
			Count: count,
		}
		resp = types.FetchBlockResp{}
	)
	if err = bs.host.RequestPB("MCC.FetchBlockByCount", &req, &resp); err != nil {
		return
	}
	block = resp.Block
	return
}

func (bs *BusService) requestLastBlock() (
	block *types.BPBlock, profiles []*types.SQLChainProfile, count uint32,
) {
	req := types.FetchLastIrreversibleBlockReq{
		Address: bs.localAddress,
	}
	resp := types.FetchLastIrreversibleBlockResp{}

	if err := bs.host.RequestPB("MCC.FetchLastIrreversibleBlock", &req, &resp); err != nil {
		log.WithError(err).Warning("fetch last block failed")
		return
	}

	block = resp.Block
	profiles = resp.SQLChains
	count = resp.Count
	
	//just for debug, wyong, 20201204
	for _ , prof := range profiles {  
		log.WithFields(log.Fields{
			"profile_domain_id":  prof.ID,
		}).Debugf("BusService/requestLastBlock")
	}

	return
}

// RequestSQLProfile get specified database profile.
func (bs *BusService) RequestSQLProfile(domainID proto.DomainID) (p *types.SQLChainProfile, ok bool) {
	bs.lock.RLock()
	defer bs.lock.RUnlock()
	p, ok = bs.sqlChainProfiles[domainID]
	return
}

// RequestPermStat fetches permission state from bus service.
func (bs *BusService) RequestPermStat(
	domainID proto.DomainID, user proto.AccountAddress) (permStat *types.PermStat, ok bool,
) {
	bs.lock.RLock()
	defer bs.lock.RUnlock()
	userState, ok := bs.sqlChainState[domainID]
	if ok {
		permStat, ok = userState[user]
	}
	return
}

//wyong, 20201008
//func (bs *BusService) requestBP(method string, request interface{}, response interface{}) (err error) {
//	var bpNodeID proto.NodeID
//	if bpNodeID, err = rpc.GetCurrentBP(); err != nil {
//		return
//	}
//
//	//todo, wyong, 20200929 
//	return bs.caller.CallNode(bpNodeID, method, request, response)
//}

func (bs *BusService) extractTxs(blocks *types.BPBlock, count uint32) {
	for _, tx := range blocks.Transactions {
		t := bs.unwrapTx(tx)
		eventName := fmt.Sprintf("/%s/", t.GetTransactionType().String())
		bs.Publish(eventName, t, count)
	}
}

func (bs *BusService) unwrapTx(tx interfaces.Transaction) interfaces.Transaction {
	switch t := tx.(type) {
	case *interfaces.TransactionWrapper:
		return bs.unwrapTx(t.Unwrap())
	default:
		return tx
	}
}

// Start starts a chain bus service.
func (bs *BusService) Start() {
	bs.wg.Add(1)
	go bs.subscribeBlock(bs.ctx)
}

// Stop stops the chain bus service.
func (bs *BusService) Stop() {
	bs.cancel()
	bs.wg.Wait()
}
