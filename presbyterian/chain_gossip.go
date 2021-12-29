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
	"sync"
	"sync/atomic"

	pi "github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/libp2p/go-libp2p-core/protocol" 
	"github.com/siegfried415/go-crawling-bazaar/types"

)

func (c *Chain) nonblockingBroadcastBlock(block *types.PBBlock) {
	for _, info := range c.getRemotePBInfos() {
		func(remote *PresbyterianInfo) {
			c.goFuncWithTimeout(func(ctx context.Context) {
				var (
					req = types.AdviseNewBlockReq{
						Envelope: proto.Envelope{
							// TODO(lambda): Add fields.
						},
						Block: block,
					}
				)

				s, err := c.host.NewStreamExt(ctx, remote.nodeID, protocol.ID("MCC.AdviseNewBlock"))
				if err != nil {
					log.WithError(err).Error("error opening block advise stream")
					return
				}

				if _, err := s.SendMsg(ctx, &req ) ; err != nil {
					log.WithError(err).Error("failed to advise new block")
				}
				
				log.WithFields(log.Fields{
					"local":       c.getLocalPBInfo(),
					"remote":      remote,
					"block_time":  block.Timestamp(),
					"block_hash":  block.BlockHash().Short(4),
					"parent_hash": block.ParentHash().Short(4),
				}).WithError(err).Debug("broadcast new block to other peers")
			}, c.period)
		}(info)
	}
}

func (c *Chain) nonblockingBroadcastTx(ttl uint32, tx pi.Transaction) {
	for _, info := range c.getRemotePBInfos() {
		func(remote *PresbyterianInfo) {
			c.goFuncWithTimeout(func(ctx context.Context) {
				var (
					req = types.AddTxReq{
						Envelope: proto.Envelope{
							// TODO(lambda): Add fields.
						},
						TTL: ttl,
						Tx:  tx,
					}
				)
				s, err := c.host.NewStreamExt(ctx, remote.nodeID, protocol.ID("MCC.AddTx"))
				if err != nil {
					log.WithError(err).Error("error opening addtx stream")
					return
				}

				if _, err := s.SendMsg(ctx, &req ) ; err != nil {
					log.WithError(err).Error("failed to advise new block")
				}

				log.WithFields(log.Fields{
					"local":   c.getLocalPBInfo(),
					"remote":  remote,
					"hash":    tx.Hash().Short(4),
					"address": tx.GetAccountAddress(),
					"type":    tx.GetTransactionType(),
				}).WithError(err).Debug("broadcast transaction to other peers")
			}, c.tick)
		}(info)
	}
}

func (c *Chain) blockingFetchBlock(ctx context.Context, h uint32) (unreachable uint32) {
	var (
		cld, ccl = context.WithTimeout(ctx, c.tick)
		wg       = &sync.WaitGroup{}
	)
	defer func() {
		wg.Wait()
		ccl()
	}()
	for _, info := range c.getRemotePBInfos() {
		wg.Add(1)
		go func(remote *PresbyterianInfo) {
			defer wg.Done()
			var (
				err error
				req = types.FetchBlockReq{
					Envelope: proto.Envelope{
						// TODO(lambda): Add fields.
					},
					Height: h,
				}
				resp = types.FetchBlockResp{}
			)
			var le = log.WithFields(log.Fields{
				"local":  c.getLocalPBInfo(),
				"remote": remote,
				"height": h,
			})

			s, err := c.host.NewStreamExt(cld, remote.nodeID, protocol.ID("MCC.FetchBlock"))
			if err != nil {
				le.WithError(err).Error("error opening block-fetching stream")
				atomic.AddUint32(&unreachable, 1)
				return
			}

			if _, err := s.SendMsg(cld, &req ) ; err != nil {
				le.WithError(err).Error("failed to fetch block")
				atomic.AddUint32(&unreachable, 1)
				return
			}

			err = s.RecvMsg(cld, &resp) 
                        if err != nil {
                                le.WithError(err).Error("failed to get response")
				atomic.AddUint32(&unreachable, 1)
                                return
                        }

			if resp.Block == nil {
				le.Debug("fetch block request reply: no such block")
				return
			}
			// Push new block from other peers
			le.WithFields(log.Fields{
				"parent": resp.Block.ParentHash().Short(4),
				"hash":   resp.Block.BlockHash().Short(4),
			}).Debug("fetch block request reply: found block")
			select {
			case c.pendingBlocks <- resp.Block:
			case <-cld.Done():
				log.WithError(cld.Err()).Warn("add pending block aborted")
			}
		}(info)
	}
	return
}
