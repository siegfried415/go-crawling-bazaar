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

package sqlchain

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"expvar"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	mw "github.com/zserge/metric"

	protocol "github.com/libp2p/go-libp2p-core/protocol" 

	"github.com/siegfried415/go-crawling-bazaar/crypto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
        "github.com/siegfried415/go-crawling-bazaar/crypto/hash"
	"github.com/siegfried415/go-crawling-bazaar/kms"
	net "github.com/siegfried415/go-crawling-bazaar/net" 
	"github.com/siegfried415/go-crawling-bazaar/proto"

	s "github.com/siegfried415/go-crawling-bazaar/state"
	si "github.com/siegfried415/go-crawling-bazaar/state/interfaces"
	ss "github.com/siegfried415/go-crawling-bazaar/state/sqlite"

	"github.com/siegfried415/go-crawling-bazaar/types"
	"github.com/siegfried415/go-crawling-bazaar/utils"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
)

const (
	mwMinerChain               = "service:miner:chain"
	mwMinerChainBlockCount     = "head:count"
	mwMinerChainBlockHeight    = "head:height"
	mwMinerChainBlockHash      = "head:hash"
	mwMinerChainBlockTimestamp = "head:timestamp"
	mwMinerChainRequestsCount  = "requests:count"
)

var (
	metaBlockIndex    = [4]byte{'B', 'L', 'C', 'K'}
	metaResponseIndex = [4]byte{'R', 'E', 'S', 'P'}
	metaAckIndex      = [4]byte{'Q', 'A', 'C', 'K'}

	leveldbConf = opt.Options{
		Compression: opt.SnappyCompression,
	}
	leveldbInit sync.Once
	blkDB       *leveldb.DB
	txDB        *leveldb.DB

	chainVars = expvar.NewMap(mwMinerChain)
)

// heightToKey converts a height in int32 to a key in bytes.
func heightToKey(h int32) (key []byte) {
	key = make([]byte, 4)
	binary.BigEndian.PutUint32(key, uint32(h))
	return
}

// keyWithSymbolToHeight converts a height back from a key(ack/resp/req/block) in bytes.
// ack key:
// ['Q', 'A', 'C', 'K', height, hash]
// resp key:
// ['R', 'E', 'S', 'P', height, hash]
// req key:
// ['R', 'E', 'Q', 'U', height, hash]
// block key:
// ['B', 'L', 'C', 'K', height, hash].
func keyWithSymbolToHeight(k []byte) int32 {
	if len(k) < 8 {
		return -1
	}
	return int32(binary.BigEndian.Uint32(k[4:]))
}

// Chain represents a sql-chain.
type Chain struct {
	bi *blockIndex
	ai *ackIndex
	st *s.State

	host net.RoutedHost 
	rt *runtime

	blocks    chan *types.Block
	heights   chan int32
	responses chan *types.ResponseHeader
	acks      chan *types.AckHeader

	// DBAccount info
	domainID   proto.DomainID
	tokenType    types.TokenType
	gasPrice     uint64
	updatePeriod uint64

	
	// Cached fileds, may need to renew some of this fields later.
	//

	// pk is the private key of the local miner.
	pk *asymmetric.PrivateKey

	// addr is the AccountAddress generate from public key.
	addr *proto.AccountAddress

	// key prefixes
	metaBlockIndex    []byte
	metaResponseIndex []byte
	metaAckIndex      []byte

	// Atomic counters for stats
	cachedBlockCount int32

	// Metric vars to collect
	expVars *expvar.Map
}

// NewChain creates a new sql-chain struct.
func NewChain(c *Config) (chain *Chain, err error) {
	return NewChainWithContext(context.Background(), c)
}

// NewChainWithContext creates a new sql-chain struct with context.
func NewChainWithContext(ctx context.Context, c *Config) (chain *Chain, err error) {
	le := log.WithField("domain", c.DomainID)

	leveldbInit.Do(func() {
		// Open LevelDB for block and state
		bdbFile := c.ChainFilePrefix + "-block-state.ldb"
		blkDB, err = leveldb.OpenFile(bdbFile, &leveldbConf)
		if err != nil {
			err = errors.Wrapf(err, "open leveldb %s", bdbFile)
			return
		}
		le.Debugf("opened chain bdb %s", bdbFile)

		// Open LevelDB for ack/request/response
		tdbFile := c.ChainFilePrefix + "-ack-req-resp.ldb"
		txDB, err = leveldb.OpenFile(tdbFile, &leveldbConf)
		if err != nil {
			err = errors.Wrapf(err, "open leveldb %s", tdbFile)
			return
		}
		le.Debugf("opened chain tdb %s", tdbFile)
	})

	if err != nil {
		return
	}

	// Open storage
	var strg si.Storage
	if strg, err = ss.NewSqlite(c.DataFile); err != nil {
		err = errors.Wrapf(err, "open data file %s", c.DataFile)
		return
	}

	// Cache local private key
	var (
		pk   *asymmetric.PrivateKey	
		addr proto.AccountAddress
	)
	if pk, err = kms.GetLocalPrivateKey(); err != nil {
		err = errors.Wrap(err, "failed to cache private key")
		return
	}

	addr, err = crypto.PubKeyHash(pk.PubKey())
	if err != nil {
		err = errors.Wrap(err, "failed to generate address")
		return
	}

	rawID := hash.THashH([]byte(c.DomainID))
        tmpID := proto.DomainID(rawID.String())
        metaKeyPrefix, err := tmpID.AccountAddress()

	if err != nil {
		err = errors.Wrap(err, "failed to generate database meta prefix")
		return
	}

	// Create chain state
	chain = &Chain{
		bi:           newBlockIndex(),
		ai:           newAckIndex(),
		st:           s.NewState(sql.IsolationLevel(c.IsolationLevel), c.Server, strg),
		host :		c.Host,  
		rt:           newRunTime(ctx, c),
		blocks:       make(chan *types.Block),
		heights:      make(chan int32, 1),
		responses:    make(chan *types.ResponseHeader),
		acks:         make(chan *types.AckHeader),
		tokenType:    c.TokenType,
		gasPrice:     c.GasPrice,
		updatePeriod: c.UpdatePeriod,
		domainID:   c.DomainID,

		//todo
		pk:                pk,
		addr:              &addr,

		metaBlockIndex:    utils.ConcatAll(metaKeyPrefix[:], metaBlockIndex[:]),
		metaResponseIndex: utils.ConcatAll(metaKeyPrefix[:], metaResponseIndex[:]),
		metaAckIndex:      utils.ConcatAll(metaKeyPrefix[:], metaAckIndex[:]),

		expVars: new(expvar.Map).Init(),
	}

	chain.expVars.Set(mwMinerChainBlockCount, new(expvar.Int))
	chain.expVars.Set(mwMinerChainBlockHeight, new(expvar.Int))
	chain.expVars.Set(mwMinerChainBlockHash, new(expvar.String))
	chain.expVars.Set(mwMinerChainBlockTimestamp, new(expvar.String))
	chain.expVars.Set(mwMinerChainRequestsCount, mw.NewCounter("5m1m"))

	chainVars.Set(string(c.DomainID), chain.expVars)

	le = le.WithField("peer", chain.rt.getPeerInfoString())

	// Read blocks and rebuild memory index
	var (
		id           uint64
		last, parent *blockNode
		blockIter    = blkDB.NewIterator(util.BytesPrefix(chain.metaBlockIndex), nil)
	)
	defer blockIter.Release()
	for blockIter.Next() {
		var (
			k     = blockIter.Key()
			v     = blockIter.Value()
			block = &types.Block{}
		)

		if err = utils.DecodeMsgPack(v, block); err != nil {
			err = errors.Wrapf(err, "decoding failed at height %d with key %s",
				keyWithSymbolToHeight(k), string(k))
			return
		}
		le.WithField("block", block.BlockHash().String()).Debug("loading block from database")

		if last == nil {
			if err = block.VerifyAsGenesis(); err != nil {
				err = errors.Wrap(err, "genesis verification failed")
				return
			}
			// Set constant fields from genesis block
			chain.rt.setGenesis(block)
		} else if block.ParentHash().IsEqual(&last.hash) {
			if err = block.Verify(); err != nil {
				err = errors.Wrapf(err, "block verification failed at height %d with key %s",
					keyWithSymbolToHeight(k), string(k))
				return
			}
			parent = last
		} else {
			if parent = chain.bi.lookupNode(block.ParentHash()); parent == nil {
				return nil, ErrParentNotFound
			}
		}

		// Update id
		if nid, ok := block.CalcNextID(); ok && nid > id {
			id = nid
		}

		// do not cache block in memory in reloading
		last = newBlockNodeEx(
			chain.rt.getHeightFromTime(block.Timestamp()), block.BlockHash(), nil, parent)
		chain.bi.addBlock(last)
	}

	if err = blockIter.Error(); err != nil {
		err = errors.Wrap(err, "accumulated error of iterator")
		return
	}

	// Initiate chain Genesis if block list is empty
	if last == nil {
		if err = chain.genesis(c.Genesis); err != nil {
			return nil, err
		}
		return
	}

	// Set chain state
	var head = &state{
		node:   last,
		Head:   last.hash,
		Height: last.height,
	}
	chain.rt.setHead(head)
	chain.st.SetSeq(id)

	// update metric
	chain.updateMetrics()

	// Read queries and rebuild memory index
	respIter := txDB.NewIterator(util.BytesPrefix(chain.metaResponseIndex), nil)
	defer respIter.Release()

	for respIter.Next() {
		k := respIter.Key()
		v := respIter.Value()
		h := keyWithSymbolToHeight(k)
		var resp = &types.SignedResponseHeader{}
		if err = utils.DecodeMsgPack(v, resp); err != nil {
			err = errors.Wrapf(err, "load resp, height %d, index %s", h, string(k))
			return
		}
		log.WithFields(log.Fields{
			"height": h,
			"header": resp.Hash().String(),
			"domain":     c.DomainID,
		}).Debug("loaded new resp header")
	}

	if err = respIter.Error(); err != nil {
		err = errors.Wrap(err, "load resp")
		return
	}

	ackIter := txDB.NewIterator(util.BytesPrefix(chain.metaAckIndex), nil)
	defer ackIter.Release()
	for ackIter.Next() {
		k := ackIter.Key()
		v := ackIter.Value()
		h := keyWithSymbolToHeight(k)
		var ack = &types.SignedAckHeader{}
		if err = utils.DecodeMsgPack(v, ack); err != nil {
			err = errors.Wrapf(err, "load ack, height %d, index %s", h, string(k))
			return
		}
		log.WithFields(log.Fields{
			"height": h,
			"header": ack.Hash().String(),
			"domain":     c.DomainID,
		}).Debug("loaded new ack header")
	}

	if err = respIter.Error(); err != nil {
		err = errors.Wrap(err, "load ack")
		return
	}

	return
}

func (c *Chain) genesis(b *types.Block) (err error) {
	if b == nil {
		err = errors.New("genesis block not provided")
		return
	}
	if err = b.VerifyAsGenesis(); err != nil {
		err = errors.Wrap(err, "initialize chain state")
		return
	}
	return c.pushBlock(b)
}

// pushBlock pushes the signed block header to extend the current main chain.
func (c *Chain) pushBlock(b *types.Block) (err error) {
	// Prepare and encode
	var (
		h    = c.rt.getHeightFromTime(b.Timestamp())
		node = newBlockNode(h, b, c.rt.getHead().node)
		head = &state{
			node:   node,
			Head:   node.hash,
			Height: node.height,
		}

		blockKey = utils.ConcatAll(c.metaBlockIndex, node.indexKey())
		encBlock *bytes.Buffer
	)
	if encBlock, err = utils.EncodeMsgPack(b); err != nil {
		return
	}

	// Put block
	err = blkDB.Put(blockKey, encBlock.Bytes(), nil)
	if err != nil {
		err = errors.Wrapf(err, "put %s", string(node.indexKey()))
		return
	}
	atomic.AddInt32(&c.cachedBlockCount, 1)
	c.rt.setHead(head)
	c.bi.addBlock(node)

	// update metrics
	c.updateMetrics()

	// Keep track of the queries from the new block
	var (
		ierr error
		le   = log.WithFields(log.Fields{
			"domain":         c.domainID,
			"producer":   b.Producer(),
			"block_hash": b.BlockHash(),
		})
	)
	for i, v := range b.QueryTxs {
		if ierr = c.AddResponse(v.Response); ierr != nil {
			le.WithFields(log.Fields{
				"index": i,
			}).WithError(ierr).Warn("failed to add Response to ackIndex")
		}
	}
	for i, v := range b.Acks {
		if ierr = c.remove(v); ierr != nil {
			le.WithFields(log.Fields{
				"index": i,
			}).WithError(ierr).Warn("failed to remove Ack from ackIndex")
		}
	}

	c.logEntry().WithFields(log.Fields{
		"block":      b.BlockHash().String()[:8],
		"producer":   b.Producer()[:8],
		"queryCount": len(b.QueryTxs),
		"ackCount":   len(b.Acks),
		"blockTime":  b.Timestamp().Format(time.RFC3339Nano),
		"height":     c.rt.getHeightFromTime(b.Timestamp()),
		"head": fmt.Sprintf("%s <- %s",
			func() string {
				if head.node.parent != nil {
					return head.node.parent.hash.String()[:8]
				}
				return "|"
			}(), head.Head.String()[:8]),
		"headHeight": c.rt.getHead().Height,
	}).Info("pushed new block")
	return
}

// pushAckedQuery pushes a acknowledged, signed and verified query into the chain.
func (c *Chain) pushAckedQuery(ack *types.SignedAckHeader) (err error) {
	log.WithField("domain", c.domainID).Debugf("push ack %s", ack.Hash().String())
	h := c.rt.getHeightFromTime(ack.GetResponseTimestamp())
	k := heightToKey(h)
	var enc *bytes.Buffer

	if enc, err = utils.EncodeMsgPack(ack); err != nil {
		return
	}

	tdbKey := utils.ConcatAll(c.metaAckIndex, k, ack.Hash().AsBytes())

	if err = c.register(ack); err != nil {
		err = errors.Wrapf(err, "register ack %v at height %d", ack.Hash(), h)
		return
	}

	if err = txDB.Put(tdbKey, enc.Bytes(), nil); err != nil {
		err = errors.Wrapf(err, "put ack %d %s", h, ack.Hash().String())
		return
	}

	return
}

// produceBlock prepares, signs and advises the pending block to the other peers.
func (c *Chain) produceBlock(now time.Time) (err error) {
	var (
		frs []*types.Request
		qts []*s.QueryTracker
	)
	if frs, qts, err = c.st.CommitEx(); err != nil {
		err = errors.Wrap(err, "failed to fetch query list from db state")
		return
	}
	if len(frs) == 0 && len(qts) == 0 {
		c.logEntryWithHeadState().Debug("no query found in current period, skip block producing")
		return
	}
	var block = &types.Block{
		SignedHeader: types.SignedHeader{
			Header: types.Header{
				Version:     0x01000000,
				Producer:    c.rt.getServer(),
				GenesisHash: c.rt.genesisHash,
				ParentHash:  c.rt.getHead().Head,
				// MerkleRoot: will be set by PBBlock.PackAndSignBlock(PrivateKey)
				Timestamp: now,
			},
		},
		FailedReqs: frs,
		QueryTxs:   make([]*types.QueryAsTx, len(qts)),
		Acks:       c.ai.acks(c.rt.getHeightFromTime(now)),
	}
	for i, v := range qts {
		// TODO(leventeliu): maybe block waiting at a ready channel instead?
		for !v.Ready() {
			time.Sleep(c.rt.period / 10)
			if c.rt.ctx.Err() != nil {
				err = c.rt.ctx.Err()
				return
			}
		}
		block.QueryTxs[i] = &types.QueryAsTx{
			// TODO(leventeliu): add acks for billing.
			Request:  v.Req,
			Response: &v.Resp.Header,
		}
	}

	/* todo
	// Sign block
	if err = block.PackAndSignBlock(c.pk); err != nil {
		return
	}
	*/

	// Send to pending list
	le := c.logEntryWithHeadState().WithFields(log.Fields{
		"using_timestamp": now.Format(time.RFC3339Nano),
		"block_hash":      block.BlockHash().String(),
	})
	select {
	case c.blocks <- block:
	case <-c.rt.ctx.Done():
		err = c.rt.ctx.Err()
		le.WithError(err).Info("abort block producing")
		return
	}
	le.Debug("produced new block")
	// Advise new block to the other peers
	peers := c.rt.getPeers()
	for _, s := range peers.Servers {
		if s != c.rt.getServer() {
			func(remote proto.NodeID) { // bind remote node id to closure
				c.rt.goFuncWithTimeout(func(ctx context.Context) {
					req := MuxAdviseNewBlockReq{
						DomainID: c.domainID,
						AdviseNewBlockReq: AdviseNewBlockReq{
							Block: block,
							Count: func() int32 {
								if nd := c.bi.lookupNode(block.BlockHash()); nd != nil {
									return nd.count
								}
								if pn := c.bi.lookupNode(block.ParentHash()); pn != nil {
									return pn.count + 1
								}
								return -1
							}(),
						},
					}
					resp := MuxAdviseNewBlockResp{}
					s, err := c.host.NewStreamExt(ctx, remote, protocol.ID("URLC.AdviseNewBlock"))
					if err != nil {
						le.WithError(err).Error("error opening push stream")
						return 
					}

	                		if _, err := s.SendMsg(ctx, &req ) ; err != nil { 
						le.WithError(err).Error("failed to advise new block")
					}

	                		if err := s.RecvMsg(ctx, &resp ) ; err != nil { 
						le.WithError(err).Error("failed to receive new block advise response")
					}


				}, c.rt.tick)
			}(s)
		}
	}

	return
}

func (c *Chain) syncHead() (err error) {
	// Try to fetch if the block of the current turn is not advised yet
	h := c.rt.getNextTurn() - 1
	if c.rt.getHead().Height >= h {
		return
	}

	var (
		peers = c.rt.getPeers()

		//todo 
		//l     = len(peers.Servers)

		//todo
		//le    = c.logEntryWithHeadState()

		child, cancel = context.WithTimeout(c.rt.ctx, c.rt.tick)
		wg            = &sync.WaitGroup{}

		totalCount, succCount, initiatingCount uint32
	)
	defer func() {
		wg.Wait()
		cancel()

		if totalCount > 0 && succCount == 0 {
			if initiatingCount == totalCount {
				err = ErrInitiating
			} else {
				// Set error if all RPC calls are failed
				err = errors.New("all remote peers are unreachable")
			}
		}
	}()

	for i, s := range peers.Servers {
		// Skip local server
		if s == c.rt.getServer() {
			continue
		}

		//todo, 20220103 
		log.Debugf("syncHead, begin sync with %s", s.String()) 

		wg.Add(1)
		go func(i int, node proto.NodeID) {
			defer wg.Done()
			var (
				//todo 
				//ile = le.WithFields(log.Fields{"remote": fmt.Sprintf("[%d/%d] %s", i, l, node)})
				req = MuxFetchBlockReq{
					DomainID: c.domainID,
					FetchBlockReq: FetchBlockReq{
						Height: h,
					},
				}
				resp = MuxFetchBlockResp{}
			)

			atomic.AddUint32(&totalCount, 1)

			//todo, 20220103 
			log.Debugf("syncHead, before open stream to %s", node.String()) 
			s, err := c.host.NewStreamExt(child, node, protocol.ID("URLC.FetchBlock"))
			if err != nil {
				//todo, ile->log, 20220103 
				log.WithError(err).Error("error opening push stream")

				log.Debugf("syncHead, failed to open stream to %s, err = %s", node.String(), err.Error()) 

				return 
			}

			//todo, 20220103 
			log.Debugf("syncHead, after open stream to %s", node.String()) 

	                if _, err := s.SendMsg(child, &req ) ; err != nil { 
				//todo, ile -> log 
				log.WithError(err).Error("failed to fetch block from peer")
				return 
			}


			err = s.RecvMsg(child, &resp) 
			if err != nil {
				//todo, ile -> log 
				log.WithError(err).Error("failed to get response")
				return 
			}

			if resp.Block == nil {
				log.Debug("fetch block request reply: no such block")
				// If block is nil, resp.Height returns the current head height of the remote peer
				if resp.Height <= req.Height {
					atomic.AddUint32(&initiatingCount, 1)
				} else {
					atomic.AddUint32(&succCount, 1)
				}
				return
			}

			//todo, ile 
			log.WithFields(log.Fields{
				"parent": resp.Block.ParentHash().Short(4),
				"hash":   resp.Block.BlockHash().Short(4),
			}).Debug("fetch block request reply: found block")
			select {
			case c.blocks <- resp.Block:
				atomic.AddUint32(&succCount, 1)
			case <-child.Done():
				//todo, le
				log.WithError(child.Err()).Info("abort head block synchronizing")
				return
			}
		}(i, s)
	}

	return
}

// runCurrentTurn does the check and runs block producing if its my turn.
func (c *Chain) runCurrentTurn(now time.Time, d time.Duration) {
	elapsed := -d
	h := c.rt.getNextTurn()
	le := c.logEntryWithHeadState().WithFields(log.Fields{
		"using_timestamp": now.Format(time.RFC3339Nano),
		"elapsed_seconds": elapsed.Seconds(),
	})

	defer func() {
		c.stat()
		c.pruneBlockCache()
		c.rt.IncNextTurn()
		c.ai.advance(c.rt.getMinValidHeight())
		// Info the block processing goroutine that the chain height has grown, so please return
		// any stashed blocks for further check.
		select {
		case c.heights <- h:
		case <-c.rt.ctx.Done():
			le.Debug("abort publishing height")
		}
	}()

	le.Debug("run current turn")
	if c.rt.getHead().Height < c.rt.getNextTurn()-1 {
		le.Debug("a block will be skipped")
	}
	if !c.rt.isMyTurn() {
		return
	}
	if elapsed+c.rt.tick > c.rt.period {
		le.Warn("too much time elapsed in the new period, skip this block")
		return
	}
	if err := c.produceBlock(now); err != nil {
		le.WithError(err).Error("failed to produce block")
	}
}

// mainCycle runs main cycle of the sql-chain.
func (c *Chain) mainCycle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logEntry().WithError(ctx.Err()).Info("abort main cycle")
			return
		default:
			if err := c.syncHead(); err != nil {
				if err != ErrInitiating {
					c.logEntry().WithError(err).Error("failed to sync head")
					continue
				}
			}
			if t, d := c.rt.nextTick(); d > 0 {
				time.Sleep(d)
			} else {
				c.runCurrentTurn(t, d)
			}
		}
	}
}

// sync synchronizes blocks and queries from the other peers.
func (c *Chain) sync() (err error) {
	le := c.logEntry()
	le.Debug("synchronizing chain state")
	defer func() {
		c.stat()
		c.pruneBlockCache()
		c.ai.advance(c.rt.getMinValidHeight())
	}()
	for {
		now := c.rt.now()
		height := c.rt.getHeightFromTime(now)
		if now.Before(c.rt.chainInitTime) {
			le.Debug("now time is before genesis time, waiting for genesis")
			return
		}
		if c.rt.getNextTurn() > height {
			break
		}
		for c.rt.getNextTurn() <= height {
			if err = c.syncHead(); err != nil {
				if err != ErrInitiating {
					le.WithError(err).Errorf("failed to sync block at height %d", height)
					return
				}
				// Skip sync and reset error
				c.rt.SetNextTurn(height + 1)
				err = nil
			} else {
				c.rt.IncNextTurn()
			}
		}
	}
	return
}

func (c *Chain) processBlocks(ctx context.Context) {
	var (
		cld, ccl = context.WithCancel(ctx)
		wg       = &sync.WaitGroup{}
	)

	returnStash := func(stash []*types.Block) {
		defer wg.Done()
		for i, block := range stash {
			select {
			case c.blocks <- block:
			case <-cld.Done():
				c.logEntry().WithFields(log.Fields{
					"remaining": len(stash) - i,
				}).WithError(cld.Err()).Debug("abort stash returning")
				return
			}
		}
	}

	defer func() {
		ccl()
		wg.Wait()
	}()

	var stash []*types.Block
	for {
		le := c.logEntryWithHeadState()
		select {
		case h := <-c.heights:
			// Trigger billing
			index, total := c.rt.getIndexTotal()
			period := int32(c.updatePeriod)
			isBillingPeriod := (h%period == 0)
			isMyTurnBilling := (h/period%total == index)
			if isBillingPeriod && isMyTurnBilling {
				ub, err := c.billing(h, c.rt.getHead().node)
				if err != nil {
					le.WithError(err).Error("billing failed")
				}
				// allocate nonce
				nonceReq := &types.NextAccountNonceReq{}
				nonceResp := &types.NextAccountNonceResp{}

				if err = c.host.RequestPB("MCC.NextAccountNonce", nonceReq, nonceResp); err != nil {
					// allocate nonce failed
					le.WithError(err).Warning("allocate nonce for transaction failed")
				}
				ub.Nonce = nonceResp.Nonce

				if err = ub.Sign(c.pk); err != nil {
					le.WithError(err).Warning("sign tx failed")
				}

				addTxReq := &types.AddTxReq{TTL: 1}
				addTxResp := &types.AddTxResp{}
				addTxReq.Tx = ub
				
				if err = c.host.RequestPB("MCC.AddTx", addTxReq, addTxResp); err != nil {
					le.WithError(err).Warning("send tx failed")
				}
			}
			// Return all stashed blocks to pending channel
			c.logEntryWithHeadState().WithFields(log.Fields{
				"height": h,
				"stashs": len(stash),
			}).Debug("read new height from channel")
			if stash != nil {
				wg.Add(1)
				go returnStash(stash)
				stash = nil
			}
		case block := <-c.blocks:
			height := c.rt.getHeightFromTime(block.Timestamp())
			le.WithFields(log.Fields{
				"block_height": height,
				"block_hash":   block.BlockHash().String(),
			}).Debug("processing new block")

			if height > c.rt.getNextTurn()-1 {
				// Stash newer blocks for later check
				stash = append(stash, block)
			} else {
				// Process block
				if height < c.rt.getNextTurn()-1 {
					// TODO(leventeliu): check and add to fork list.
				} else {
					if err := c.CheckAndPushNewBlock(block); err != nil {
						le.WithError(err).Error("failed to check and push new block")
					}
				}
			}
		case <-ctx.Done():
			c.logEntryWithHeadState().WithError(ctx.Err()).Debug("abort block processing")
			return
		}
	}
}

// Start starts the main process of the sql-chain.
func (c *Chain) Start() (err error) {
	c.rt.goFunc(c.processBlocks)
	if err = c.sync(); err != nil {
		c.logEntryWithHeadState().WithError(err).Error("failed to start, chain process terminated")
		_ = c.Stop()
		return
	}
	c.rt.goFunc(c.mainCycle)
	c.rt.startService(c)
	c.logEntryWithHeadState().Info("started successfully")
	return
}

// Stop stops the main process of the sql-chain.
func (c *Chain) Stop() (err error) {
	// Stop main process
	le := c.logEntry()
	le.Debug("stopping chain")
	c.rt.stop(c.domainID)
	le.Debug("chain service and workers stopped")
	// Close state
	var ierr error
	if ierr = c.st.Close(false); ierr != nil && err == nil {
		err = ierr
	}
	le.WithError(ierr).Debug("chain state storage closed")
	return
}

// FetchBlock fetches the block at specified height from local cache.
func (c *Chain) FetchBlock(height int32) (b *types.Block, err error) {
	if n := c.rt.getHead().node.ancestor(height); n != nil {
		return c.fetchBlockByIndexKey(n.indexKey())
	}
	return
}

// FetchBlockByCount fetches the block at specified count from local cache.
func (c *Chain) FetchBlockByCount(count int32) (b *types.Block, realCount int32, height int32, err error) {
	var n *blockNode

	if count < 0 {
		n = c.rt.getHead().node
	} else {
		n = c.rt.getHead().node.ancestorByCount(count)
	}

	if n != nil {
		b, err = c.fetchBlockByIndexKey(n.indexKey())
		if err != nil {
			return
		}

		height = n.height
		realCount = n.count
	}

	return
}

func (c *Chain) fetchBlockByIndexKey(indexKey []byte) (b *types.Block, err error) {
	k := utils.ConcatAll(c.metaBlockIndex, indexKey)
	var v []byte
	v, err = blkDB.Get(k, nil)
	if err != nil {
		err = errors.Wrapf(err, "fetch block %s", string(k))
		return
	}

	b = &types.Block{}
	err = utils.DecodeMsgPack(v, b)
	if err != nil {
		err = errors.Wrapf(err, "fetch block %s", string(k))
		return
	}

	return
}

// CheckAndPushNewBlock implements ChainRPCServer.CheckAndPushNewBlock.
func (c *Chain) CheckAndPushNewBlock(block *types.Block) (err error) {
	height := c.rt.getHeightFromTime(block.Timestamp())
	head := c.rt.getHead()
	peers := c.rt.getPeers()
	total := int32(len(peers.Servers))
	next := func() int32 {
		if total > 0 {
			return (c.rt.getNextTurn() - 1) % total
		}
		return -1
	}()
	le := c.logEntryWithHeadState().WithFields(log.Fields{
		"block":       block.BlockHash().String(),
		"producer":    block.Producer(),
		"blocktime":   block.Timestamp().Format(time.RFC3339Nano),
		"blockheight": height,
		"blockparent": block.ParentHash().String(),
	})
	le.Debug("checking new block from other peer")

	if head.Height == height && head.Head.IsEqual(block.BlockHash()) {
		// Maybe already set by FetchBlock
		return nil
	} else if !block.ParentHash().IsEqual(&head.Head) {
		err = ErrInvalidBlock
		le.WithError(err).Error("invalid new block for the current chain")
		return ErrInvalidBlock
	}

	// Verify block signatures
	if err = block.Verify(); err != nil {
		le.WithError(err).Error("failed to verify block")
		return
	}

	// Short circuit the checking process if it's a self-produced block
	if block.Producer() == c.rt.server {
		return c.pushBlock(block)
	}
	// Check block producer
	index, found := peers.Find(block.Producer())
	if !found {
		err = ErrUnknownProducer
		le.WithError(err).Error("unknown producer of new block")
		return ErrUnknownProducer
	}

	if index != next {
		err = ErrInvalidProducer
		le.WithFields(log.Fields{
			"expected": next,
			"actual":   index,
		}).WithError(err).Error("invalid producer of new block")
		return
	}

	// TODO(leventeliu): check if too many periods are skipped or store block for future use.
	// if height-c.rt.getHead().Height > X {
	// 	...
	// }

	// Replicate local state from the new block
	if err = c.st.ReplayBlockWithContext(c.rt.ctx, block); err != nil {
		le.WithError(err).Error("failed to replay new block")
		return
	}

	return c.pushBlock(block)
}

// VerifyAndPushAckedQuery verifies a acknowledged and signed query, and pushed it if valid.
func (c *Chain) VerifyAndPushAckedQuery(ack *types.SignedAckHeader) (err error) {
	// TODO(leventeliu): check ack.
	if c.rt.queryTimeIsExpired(ack.GetResponseTimestamp()) {
		err = errors.Wrapf(ErrQueryExpired,
			"Verify ack query, min valid height %d, ack height %d",
			c.rt.getMinValidHeight(), c.rt.getHeightFromTime(ack.Timestamp))
		return
	}

	if err = ack.Verify(); err != nil {
		return
	}

	return c.pushAckedQuery(ack)
}

// UpdatePeers updates peer list of the sql-chain.
func (c *Chain) UpdatePeers(peers *proto.Peers) error {
	return c.rt.updatePeers(peers)
}

// Query queries req from local chain state and returns the query results in resp.
func (c *Chain) Query(
	req *types.Request, isLeader bool) (tracker *s.QueryTracker, resp *types.Response, err error,
) {
	// TODO(leventeliu): we're using an external context passed by request. Make sure that
	// cancelling will be propagated to this context before chain instance stops.
	// update metrics
	c.expVars.Get(mwMinerChainRequestsCount).(mw.Metric).Add(1)

	return c.st.QueryWithContext(req.GetContext(), req, isLeader)
}

// AddResponse addes a response to the ackIndex, awaiting for acknowledgement.
func (c *Chain) AddResponse(resp *types.SignedResponseHeader) (err error) {
	return c.ai.addResponse(c.rt.getHeightFromTime(resp.GetRequestTimestamp()), resp)
}

func (c *Chain) register(ack *types.SignedAckHeader) (err error) {
	return c.ai.register(c.rt.getHeightFromTime(ack.GetRequestTimestamp()), ack)
}

func (c *Chain) remove(ack *types.SignedAckHeader) (err error) {
	return c.ai.remove(c.rt.getHeightFromTime(ack.GetRequestTimestamp()), ack)
}

func (c *Chain) pruneBlockCache() {
	var (
		head     = c.rt.getHead().node
		nextTurn = c.rt.getNextTurn()
		lastCnt  int32
	)
	if head == nil {
		return
	}
	lastCnt = nextTurn - c.rt.blockCacheTTL
	if h := c.rt.getLastBillingHeight(); h < lastCnt {
		lastCnt = h // also keep cache for billing if possible
	}
	// Move to last count position
	for ; head != nil && head.count > lastCnt; head = head.parent {
	}
	// Prune block references
	for ; head != nil && head.clear(); head = head.parent {
		atomic.AddInt32(&c.cachedBlockCount, -1)
	}
}

func (c *Chain) stat() {
	var (
		ic = atomic.LoadInt32(&c.ai.multiIndexCount)
		rc = atomic.LoadInt32(&c.ai.responseCount)
		tc = atomic.LoadInt32(&c.ai.ackCount)
		bc = atomic.LoadInt32(&c.cachedBlockCount)
	)
	// Print chain stats
	c.logEntry().WithFields(log.Fields{
		"multiIndex_count":      ic,
		"response_header_count": rc,
		"query_tracker_count":   tc,
		"cached_block_count":    bc,
	}).Info("chain mem stats")
	// Print xeno stats
	c.st.Stat(c.domainID)
}

func (c *Chain) billing(h int32, node *blockNode) (ub *types.UpdateBilling, err error) {
	le := c.logEntryWithHeadState()
	le.WithFields(log.Fields{"given_height": h}).Info("begin to billing")
	var (
		i, j      uint64
		iter      *blockNode
		minerAddr proto.AccountAddress
		userAddr  proto.AccountAddress
		minHeight = c.rt.getLastBillingHeight()
		usersMap  = make(map[proto.AccountAddress]uint64)
		minersMap = make(map[proto.AccountAddress]map[proto.AccountAddress]uint64)
	)

	for iter = node; iter != nil && iter.height > h; iter = iter.parent {
	}
	for iter != nil && iter.height > minHeight {
		var block = iter.load()
		// Not cached, recover from storage
		if block == nil {
			if block, err = c.FetchBlock(iter.height); err != nil {
				return
			}
		}
		for _, tx := range block.QueryTxs {
			minerAddr = tx.Response.ResponseAccount
			if userAddr, err = crypto.PubKeyHash(tx.Request.Header.Signee); err != nil {
				//todo 
				//le.WithError(err).Warning("billing fail: miner addr")
				return
			}

			if _, ok := minersMap[userAddr]; !ok {
				minersMap[userAddr] = make(map[proto.AccountAddress]uint64)
			}
			if tx.Request.Header.QueryType == types.ReadQuery {
				minersMap[userAddr][minerAddr] += tx.Response.RowCount
				usersMap[userAddr] += tx.Response.RowCount
			} else {
				minersMap[userAddr][minerAddr] += uint64(tx.Response.AffectedRows)
				usersMap[userAddr] += uint64(tx.Response.AffectedRows)
			}
		}

		for _, req := range block.FailedReqs {
			if minerAddr, err = crypto.PubKeyHash(block.Signee()); err != nil {
				//todo
				//le.WithError(err).Warning("billing fail: miner addr")
				return
			}
			if userAddr, err = crypto.PubKeyHash(req.Header.Signee); err != nil {
				//todo
				//le.WithError(err).Warning("billing fail: user addr")
				return
			}
			if _, ok := minersMap[userAddr][minerAddr]; !ok {
				minersMap[userAddr] = make(map[proto.AccountAddress]uint64)
			}

			minersMap[userAddr][minerAddr] += uint64(len(req.Payload.Queries))
			usersMap[userAddr] += uint64(len(req.Payload.Queries))
		}
		iter = iter.parent
	}

	ub = types.NewUpdateBilling(&types.UpdateBillingHeader{
		Users: make([]*types.UserCost, len(usersMap)),
	})

	//todo, check if the following line is necessary
	//ub.Version = int32(ub.HSPDefaultVersion())

	i = 0
	j = 0
	for userAddr, cost := range usersMap {
		le.Debugf("user %s, cost %d", userAddr.String(), cost)
		ub.Users[i] = &types.UserCost{
			User: userAddr,
			Cost: cost,
		}
		miners := minersMap[userAddr]
		ub.Users[i].Miners = make([]*types.MinerIncome, len(miners))

		for k1, v1 := range miners {
			ub.Users[i].Miners[j] = &types.MinerIncome{
				Miner:  k1,
				Income: v1,
			}
			j++
		}
		j = 0
		i++
	}
	ub.Receiver, err = c.domainID.AccountAddress()
	ub.Range.From = uint32(minHeight)
	ub.Range.To = uint32(h)
	return
}

// SetLastBillingHeight sets the last billing height of this chain instance.
func (c *Chain) SetLastBillingHeight(h int32) {
	c.logEntryWithHeadState().WithFields(
		log.Fields{"new_height": h}).Debug("set last billing height")
	c.rt.setLastBillingHeight(h)
}

func (c *Chain) logEntry() *log.Entry {
	return log.WithFields(log.Fields{
		"domain":     c.domainID,
		"peer":   c.rt.getPeerInfoString(),
		"offset": c.rt.getChainTimeString(),
	})
}

func (c *Chain) logEntryWithHeadState() *log.Entry {
	return log.WithFields(log.Fields{
		"domain":          c.domainID,
		"peer":        c.rt.getPeerInfoString(),
		"offset":      c.rt.getChainTimeString(),
		"curr_turn":   c.rt.getNextTurn(),
		"head_height": c.rt.getHead().Height,
		"head_block":  c.rt.getHead().Head.String(),
	})
}

func (c *Chain) updateMetrics() {
	head := c.rt.getHead()
	c.expVars.Get(mwMinerChainBlockCount).(*expvar.Int).Set(int64(head.node.count))
	c.expVars.Get(mwMinerChainBlockHeight).(*expvar.Int).Set(int64(head.Height))
	c.expVars.Get(mwMinerChainBlockHash).(*expvar.String).Set(head.Head.String())

	b := head.node.load()
	if b == nil {
		// load manually
		var err error
		b, err = c.FetchBlock(head.Height)
		if err != nil {
			return
		}
	}

	c.expVars.Get(mwMinerChainBlockTimestamp).(*expvar.String).Set(b.Timestamp().String())
}

//export this fuction
func (c *Chain) GetCurrentHeight() int32 {
	now := c.rt.now()
	return c.rt.getHeightFromTime(now)
}

func (c *Chain) GetCurrentHeightFromTime(t time.Time) int32 {
	return c.rt.getHeightFromTime(t)
}
