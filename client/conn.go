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

package client

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"
	"sync/atomic"
	"time"
	//"os"
	"fmt" 

	"github.com/pkg/errors"

	//wyong, 20201007 
       	//"io/ioutil"
	//peer "github.com/libp2p/go-libp2p-core/peer" 
	protocol "github.com/libp2p/go-libp2p-core/protocol" 

	//go-libp2p-net -> go-libp2p-core/network
	//wyong, 20201029 
        //inet "github.com/libp2p/go-libp2p-core/network"

        //host "github.com/libp2p/go-libp2p-host"


	"github.com/siegfried415/gdf-rebuild/crypto/asymmetric"
	"github.com/siegfried415/gdf-rebuild/kms"
	"github.com/siegfried415/gdf-rebuild/proto"
	//"github.com/siegfried415/gdf-rebuild/route"

	//wyong, 20201007 
	//"github.com/siegfried415/gdf-rebuild/rpc"
	//"github.com/siegfried415/gdf-rebuild/rpc/mux"
	net "github.com/siegfried415/gdf-rebuild/net"

	"github.com/siegfried415/gdf-rebuild/types"
	"github.com/siegfried415/gdf-rebuild/utils/log"
	"github.com/siegfried415/gdf-rebuild/utils/trace"

	//wyong, 20200715 
	"github.com/siegfried415/gdf-rebuild/utils/callinfo"

	//wyong, 20200803 
	"github.com/ipfs/go-cid"

)

// conn implements an interface sql.Conn.
type Conn struct {
	domainID proto.DomainID

	queries     []types.Query
	localNodeID proto.NodeID

	//wyong, 20201007
	host net.RoutedHost 

	privKey     *asymmetric.PrivateKey

	inTransaction bool
	closed        int32

	leader   *pconn
	follower *pconn
}

// pconn represents a connection to a peer.
type pconn struct {
	wg      *sync.WaitGroup
	parent  *Conn
	ackCh   chan *types.Ack

	//wyong, 20201007
	//pCaller rpc.PCaller
	pCaller *net.Stream 
}

const workerCount int = 2

func NewConn(cfg *Config) (c *Conn, err error) {

	//wyong, 20201021 
	ctx := context.Background()

	// get local node id
	var localNodeID proto.NodeID
	if localNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get local private key
	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	c = &Conn{
		domainID:        proto.DomainID(cfg.DomainID),
		localNodeID: localNodeID,

		//wyong, 20201007
		host:		cfg.Host, 

		privKey:     privKey,
		queries:     make([]types.Query, 0),
	}

	// get peers from BP
	var peers *proto.Peers
	if peers, err = cacheGetPeers(c.host, c.domainID, c.privKey); err != nil {
		return nil, errors.WithMessage(err, "cacheGetPeers failed")
	}

	if cfg.Mirror != "" {
		//wyong, 20201007
		caller, err := c.host.NewStreamExt(ctx, proto.NodeID(cfg.Mirror), protocol.ID(cfg.Protocol))
		if err != nil {
			return nil, errors.WithMessage(err, "open stream failed")
		}

		c.leader = &pconn{
			wg:      &sync.WaitGroup{},
			parent:  c,
			pCaller: &caller, //mux.NewRawCaller(cfg.Mirror),
		}

		// no ack workers required, mirror mode does not support ack worker
	} else {
		if cfg.UseLeader {
			//var caller rpc.PCaller
			//if cfg.UseDirectRPC {
			//	caller = rpc.NewPersistentCaller(peers.Leader)
			//} else {
			//	caller = mux.NewPersistentCaller(peers.Leader)
			//}

			//wyong, 20201007
			caller, err := c.host.NewStreamExt(ctx, peers.Leader, protocol.ID(cfg.Protocol))
			if err != nil {
				return nil, errors.WithMessage(err, "open stream failed")
			}

			c.leader = &pconn{
				wg:      &sync.WaitGroup{},
				ackCh:   make(chan *types.Ack, workerCount*4),
				parent:  c,
				pCaller: &caller,
			}
		}

		// choose a random follower node
		if cfg.UseFollower && len(peers.Servers) > 1 {
			for {
				node := peers.Servers[randSource.Intn(len(peers.Servers))]
				if node != peers.Leader {
					//var caller rpc.PCaller
					//if cfg.UseDirectRPC {
					//	caller = rpc.NewPersistentCaller(node)
					//} else {
					//	caller = mux.NewPersistentCaller(node)
					//}

					//wyong, 20201007
					caller, err := c.host.NewStreamExt(ctx, node, protocol.ID(cfg.Protocol))
					if err != nil {
						return nil, errors.WithMessage(err, "open stream failed")
					}

					c.follower = &pconn{
						wg:      &sync.WaitGroup{},
						ackCh:   make(chan *types.Ack, workerCount*4),
						parent:  c,
						pCaller: &caller,
					}
					break
				}
			}
		}

		if c.leader == nil && c.follower == nil {
			return nil, errors.New("no follower peers found")
		}

		if c.leader != nil {
			if err := c.leader.startAckWorkers(); err != nil {
				return nil, errors.WithMessage(err, "leader startAckWorkers failed")
			}
		}
		if c.follower != nil {
			if err := c.follower.startAckWorkers(); err != nil {
				return nil, errors.WithMessage(err, "follower startAckWorkers failed")
			}
		}
	}

	log.WithField("db", c.domainID).Debug("new connection to domain")
	return
}

func (c *pconn) startAckWorkers() (err error) {
	for i := 0; i < workerCount; i++ {
		c.wg.Add(1)
		go c.ackWorker()
	}
	return
}

func (c *pconn) stopAckWorkers() {
	if c.ackCh != nil {
		close(c.ackCh)
	}
}

func (c *pconn) ackWorker() {
	defer c.wg.Done()

	var (
		//wyong, 20201021 
		//oneTime sync.Once
		//pc      rpc.PCaller

		err     error
	)

	ctx := context.Background()

ackWorkerLoop:
	for {
		ack, got := <-c.ackCh
		if !got { // closed and empty
			break ackWorkerLoop
		}

		//wyong, 20201021 
		//oneTime.Do(func() {
		//	pc = c.pCaller.New()
		//})

		if err = ack.Sign(c.parent.privKey); err != nil {
			//wyong, 20201021 
			//log.WithField("target", pc.Target()).WithError(err).Error("failed to sign ack")
			log.WithField("target", string(c.pCaller.Conn().RemotePeer())).WithError(err).Error("failed to sign ack")
			continue
		}


		//wyong, 20201008 
		// send ack back
		//if err = pc.Call(route.DBSAck.String(), ack, &ackRes); err != nil {
		if _, err = c.pCaller.SendMsg(ctx, ack ); err != nil {
			log.WithError(err).Debug("send ack failed")
			continue
		}
		
		//wyong, 20201021 
		var ackRes types.AckResponse
		err = c.pCaller.RecvMsg(ctx, ackRes) 
		if err != nil { 
			log.WithError(err).Debug("receice ack response failed")
			continue
		}
		
	}

	//todo, wyong, 20201021 
	//if pc != nil {
	//	pc.Close()
	//}

	log.Debug("ack worker quiting")
}

func (c *pconn) close() error {
	c.stopAckWorkers()
	c.wg.Wait()
	if c.pCaller != nil {
		c.pCaller.Close()
	}
	return nil
}


//wyong, 20200729
 func (c *Conn) sendUrlRequest(ctx context.Context, requests []types.UrlRequest ) (err error) {
	fmt.Printf("Client/Conn/sendUrlRequest(10)\n") 

	var uc *pconn // peer connection used to execute the queries
	uc = c.leader

	/* todo, select target peer accrodingto hash of url , wyong, 20200729  
	// use follower pconn only when the query is readonly
	if queryType == types.ReadQuery && c.follower != nil {
		uc = c.follower
	}
	if uc == nil {
		uc = c.follower
	}
	*/

	// allocate sequence
	connID, seqNo := allocateConnAndSeq()
	defer putBackConn(connID)


	/*
	defer func() {
		log.WithFields(log.Fields{
			"count":  len(queries),
			"type":   queryType.String(),
			"connID": connID,
			"seqNo":  seqNo,
			"target": uc.pCaller.Target(),
			"source": c.localNodeID,
		}).WithError(err).Debug("send query")
	}()
	*/

	// build request
	reqMsg := &types.UrlRequestMessage {
		Header: types.SignedUrlRequestHeader{
			UrlRequestHeader: types.UrlRequestHeader{
				QueryType:    types.WriteQuery, //queryType,
				NodeID:       c.localNodeID,
				DomainID:   c.domainID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    getLocalTime(),
			},
		},
		Payload: types.UrlRequestPayload{
			//Queries: queries,
			Requests : requests, 
		},
	}

	fmt.Printf("Client/Conn/sendUrlRequest(20)\n") 
	if err = reqMsg.Sign(c.privKey); err != nil {
		fmt.Printf("Client/Conn/sendUrlRequest(25), err=%s\n", err.Error()) 
		return
	}

	fmt.Printf("Client/Conn/sendUrlRequest(30)\n") 
	// set receipt if key exists in context
	if val := ctx.Value(&ctxReceiptKey); val != nil {
		val.(*atomic.Value).Store(&Receipt{
			RequestHash: reqMsg.Header.Hash(),
		})
	}

	fmt.Printf("Client/Conn/sendUrlRequest(40)\n") 

	
	//wyong, 20201007 
	//if err = uc.pCaller.Call(route.FronteraURLRequest.String(), reqMsg, &response); err != nil {
	if _, err = uc.pCaller.SendMsg(ctx, reqMsg ); err != nil {
		fmt.Printf("Client/Conn/sendUrlRequest(45), err=%s\n", err.Error()) 
		return
	}

	//wyong, 20201021 
	var response types.Response
	err = uc.pCaller.RecvMsg(ctx, response) 
	if err != nil {
		fmt.Printf("Client/Conn/sendUrlRequest(47), err=%s\n", err.Error()) 
		return
	}

	/*
	rows = newRows(&response)
	if queryType == types.WriteQuery {
		affectedRows = response.Header.AffectedRows
		lastInsertID = response.Header.LastInsertID
	}
	*/

	// build ack
	func() {
		defer trace.StartRegion(ctx, "ackEnqueue").End()
		if uc.ackCh != nil {
			uc.ackCh <- &types.Ack{
				Header: types.SignedAckHeader{
					AckHeader: types.AckHeader{
						Response:     response.Header.ResponseHeader,
						ResponseHash: response.Header.Hash(),
						NodeID:       c.localNodeID,
						Timestamp:    getLocalTime(),
					},
				},
			}
		}
	}()

	fmt.Printf("Client/Conn/sendUrlRequest(50)\n") 
	return
}

// ExecContext implements the driver.ExecerContext.ExecContext method.
func (c *Conn) PutUrlRequest(ctx context.Context, parent types.UrlRequest, requests []types.UrlRequest )( err error) {

	defer trace.StartRegion(ctx, "dbExec").End()

	//wyong, 20200519 
	//log.SetOutput(os.Stdout) 
	//log.WithField("query", query).Debug("ExecContext called...")
	fmt.Printf("Client/Conn/PutUrlRequest(10)\n") 

	/*
	if atomic.LoadInt32(&c.closed) != 0 {
		err = driver.ErrBadConn
		return
	}
	*/

	// TODO(xq262144): make use of the ctx argument
	//sq := convertQuery(query, args)


	//var affectedRows, lastInsertID int64
	if err = c.sendUrlRequest(ctx, requests ); err != nil {
		fmt.Printf("Client/Conn/PutUrlRequest(15), err=%s\n", err.Error()) 
		return
	}

	/*
	result = &execResult{
		affectedRows: affectedRows,
		lastInsertID: lastInsertID,
	}
	*/

	fmt.Printf("Client/Conn/PutUrlRequest(20)\n") 
	return
}


//wyong, 20200817 
 func (c *Conn) sendUrlCidRequest(ctx context.Context, request types.UrlCidRequest ) (result cid.Cid, err error) {
	fmt.Printf("conn/sendUrlCidRequest(10)\n") 
	var uc *pconn // peer connection used to execute the queries
	uc = c.leader

	// todo, select target peer accrodingto hash of url , wyong, 20200729  
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


	//defer func() {
	//	log.WithFields(log.Fields{
	//		"count":  len(queries),
	//		"type":   queryType.String(),
	//		"connID": connID,
	//		"seqNo":  seqNo,
	//		"target": uc.pCaller.Target(),
	//		"source": c.localNodeID,
	//	}).WithError(err).Debug("send query")
	//}()


	fmt.Printf("conn/sendUrlCidRequest(20)\n") 
	// build request
	reqMsg := &types.UrlCidRequestMessage {
		Header: types.SignedUrlCidRequestHeader{
			UrlCidRequestHeader: types.UrlCidRequestHeader{
				QueryType:    types.ReadQuery, //queryType,
				NodeID:       c.localNodeID,
				DomainID:   c.domainID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    getLocalTime(),
			},
		},
		Payload: types.UrlCidRequestPayload{
			Requests : []types.UrlCidRequest{request}, 
		},
	}

	if err = reqMsg.Sign(c.privKey); err != nil {
		return
	}

	fmt.Printf("conn/sendUrlCidRequest(30)\n") 
	// set receipt if key exists in context
	if val := ctx.Value(&ctxReceiptKey); val != nil {
		val.(*atomic.Value).Store(&Receipt{
			RequestHash: reqMsg.Header.Hash(),
		})
	}

	
	//wyong, 20201008 
	//if err = uc.pCaller.Call(route.FronteraURLCidRequest.String(), reqMsg, &response); err != nil {
	if _, err = uc.pCaller.SendMsg(ctx, reqMsg ); err != nil {
		fmt.Printf("conn/sendUrlCidRequest(35), err=%s\n", err.Error()) 
		return
	}

	//wyong, 20201021 
	var response types.UrlCidResponse
	uc.pCaller.RecvMsg(ctx, response) 
	if err != nil {
		fmt.Printf("Client/Conn/sendUrlRequest(47), err=%s\n", err.Error()) 
		return
	}

	cids := response.Payload.Cids 
	if len(cids) > 0 {
		//wyong, 20200907 
		result, _  = cid.Decode(cids[0])
	}

	fmt.Printf("conn/sendUrlCidRequest(40), result =%s\n", result.String() ) 
	//rows = newRows(&response)
	//if queryType == types.WriteQuery {
	//	affectedRows = response.Header.AffectedRows
	//	lastInsertID = response.Header.LastInsertID
	//}

	// build ack
	//func() {
	//	defer trace.StartRegion(ctx, "ackEnqueue").End()
	//	if uc.ackCh != nil {
	//		fmt.Printf("conn/sendUrlCidRequest(50)\n") 
	//		uc.ackCh <- &types.Ack{
	//			Header: types.SignedAckHeader{
	//				AckHeader: types.AckHeader{
	//					Response:     response.Header.ResponseHeader,
	//					ResponseHash: response.Header.Hash(),
	//					NodeID:       c.localNodeID,
	//					Timestamp:    getLocalTime(),
	//				},
	//			},
	//		}
	//	}
	//}()

	fmt.Printf("conn/sendUrlCidRequest(60)\n") 
	return
}

//wyong, 20200729 
func (c *Conn) GetCidByUrl(ctx context.Context, url string ) (/* result driver.Result, */ result cid.Cid,  err error) {

	fmt.Printf("conn/GetCidByUrl(10), url=%s\n", url ) 
	defer trace.StartRegion(ctx, "dbExec").End()

	//wyong, 20200519 
	//log.SetOutput(os.Stdout) 
	//log.WithField("query", query).Debug("ExecContext called...")
	//fmt.Printf("ExecContext called, query=%s\n, stack=%s", query, callinfo.Stacks()) 

	if atomic.LoadInt32(&c.closed) != 0 {
		err = driver.ErrBadConn
		return
	}

	fmt.Printf("conn/GetCidByUrl(20)\n") 
	// TODO(xq262144): make use of the ctx argument
	//query := fmt.Sprintf("select cid from UrlGraph where url=%s", url)
	//sq := convertQuery(query, []driver.NamedValue{})

	//todo, create requests accroding to url, wyong, 20200817 
	request := types.UrlCidRequest {
		Url : url ,
	}


	//var affectedRows, lastInsertID int64
	//if /* affectedRows */ _, /* lastInsertID */ _, _, err = c.sendQuery(ctx, types.ReadQuery, []types.Query{*sq}); err != nil {
	//	return
	//}

	if result,  err = c.sendUrlCidRequest(ctx, request); err != nil {
		fmt.Printf("conn/GetCidByUrl(25), err=%s\n", err.Error()) 
		return
	}

	fmt.Printf("conn/GetCidByUrl(30), result=%s\n", result.String() ) 

	// todo, get cid from result,  wyong, 20200803 
	//result = &execResult{
	//	affectedRows: affectedRows,
	//	lastInsertID: lastInsertID,
	//}

	return
}

/* wyong, 20200802 
// Prepare implements the driver.Conn.Prepare method.
func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}
*/

// Close implements the driver.Conn.Close method.
func (c *Conn) Close() error {
	// close the meta connection
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		log.WithField("domain", c.domainID).Debug("closed connection")
	}
	if c.leader != nil {
		c.leader.close()
	}
	if c.follower != nil {
		c.follower.close()
	}
	return nil
}

// Begin implements the driver.Conn.Begin method.
func (c *Conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

// BeginTx implements the driver.ConnBeginTx.BeginTx method.
func (c *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if atomic.LoadInt32(&c.closed) != 0 {
		return nil, driver.ErrBadConn
	}

	// start transaction
	log.WithField("inTx", c.inTransaction).Debug("begin transaction")

	if c.inTransaction {
		return nil, sql.ErrTxDone
	}

	// TODO(xq262144): make use of the ctx argument
	c.inTransaction = true
	c.queries = c.queries[:0]

	return c, nil
}


/* wyong, 20200802 
// PrepareContext implements the driver.ConnPrepareContext.ConnPrepareContext method.
func (c *Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if atomic.LoadInt32(&c.closed) != 0 {
		return nil, driver.ErrBadConn
	}

	log.WithField("query", query).Debug("prepared statement")

	// prepare the statement
	return newStmt(c, query), nil
}
*/

// ExecContext implements the driver.ExecerContext.ExecContext method.
func (c *Conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (result driver.Result, err error) {
	defer trace.StartRegion(ctx, "dbExec").End()

	//wyong, 20200519 
	//log.SetOutput(os.Stdout) 
	//log.WithField("query", query).Debug("ExecContext called...")
	fmt.Printf("ExecContext called, query=%s\n, stack=%s", query, callinfo.Stacks()) 

	if atomic.LoadInt32(&c.closed) != 0 {
		err = driver.ErrBadConn
		return
	}

	// TODO(xq262144): make use of the ctx argument
	sq := convertQuery(query, args)

	var affectedRows, lastInsertID int64
	if affectedRows, lastInsertID, _, err = c.addQuery(ctx, types.WriteQuery, sq); err != nil {
		return
	}

	result = &execResult{
		affectedRows: affectedRows,
		lastInsertID: lastInsertID,
	}
	return
}

// QueryContext implements the driver.QueryerContext.QueryContext method.
func (c *Conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	defer trace.StartRegion(ctx, "dbQuery").End()

	//wyong, 20200519 
	//log.SetOutput(os.Stdout) 
	//log.WithField("query", query).Debug("QueryContext called...")
	fmt.Printf("QueryContext called, query=%s\n, stack=%s", query, callinfo.Stacks()) 

	if atomic.LoadInt32(&c.closed) != 0 {
		err = driver.ErrBadConn
		return
	}

	// TODO(xq262144): make use of the ctx argument
	sq := convertQuery(query, args)
	_, _, rows, err = c.addQuery(ctx, types.ReadQuery, sq)

	return
}

// Commit implements the driver.Tx.Commit method.
func (c *Conn) Commit() (err error) {
	if atomic.LoadInt32(&c.closed) != 0 {
		return driver.ErrBadConn
	}

	if !c.inTransaction {
		return sql.ErrTxDone
	}

	defer func() {
		c.queries = c.queries[:0]
		c.inTransaction = false
	}()

	if len(c.queries) > 0 {
		// send query
		if _, _, _, err = c.sendQuery(context.Background(), types.WriteQuery, c.queries); err != nil {
			return
		}
	}

	return
}

// Rollback implements the driver.Tx.Rollback method.
func (c *Conn) Rollback() error {
	if atomic.LoadInt32(&c.closed) != 0 {
		return driver.ErrBadConn
	}

	if !c.inTransaction {
		return sql.ErrTxDone
	}

	defer func() {
		c.queries = c.queries[:0]
		c.inTransaction = false
	}()

	if len(c.queries) == 0 {
		return sql.ErrTxDone
	}

	return nil
}

func (c *Conn) addQuery(ctx context.Context, queryType types.QueryType, query *types.Query) (affectedRows int64, lastInsertID int64, rows driver.Rows, err error) {
	if c.inTransaction {
		// check query type, enqueue query
		if queryType == types.ReadQuery {
			// read query is not supported in transaction
			err = ErrQueryInTransaction
			return
		}

		// append queries
		c.queries = append(c.queries, *query)

		log.WithFields(log.Fields{
			"pattern": query.Pattern,
			"args":    query.Args,
		}).Debug("add query to tx")

		return
	}

	log.WithFields(log.Fields{
		"pattern": query.Pattern,
		"args":    query.Args,
	}).Debug("execute query")

	return c.sendQuery(ctx, queryType, []types.Query{*query})
}

func (c *Conn) sendQuery(ctx context.Context, queryType types.QueryType, queries []types.Query) (affectedRows int64, lastInsertID int64, rows driver.Rows, err error) {
	var uc *pconn // peer connection used to execute the queries

	uc = c.leader
	// use follower pconn only when the query is readonly
	if queryType == types.ReadQuery && c.follower != nil {
		uc = c.follower
	}
	if uc == nil {
		uc = c.follower
	}

	// allocate sequence
	connID, seqNo := allocateConnAndSeq()
	defer putBackConn(connID)

	defer func() {
		log.WithFields(log.Fields{
			"count":  len(queries),
			"type":   queryType.String(),
			"connID": connID,
			"seqNo":  seqNo,

			//wyong, 20201021 
			//"target": uc.pCaller.Target(),
			"target": string(uc.pCaller.Conn().RemotePeer()),

			"source": c.localNodeID,
		}).WithError(err).Debug("send query")
	}()

	// build request
	req := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    queryType,
				NodeID:       c.localNodeID,
				DomainID:   c.domainID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    getLocalTime(),
			},
		},
		Payload: types.RequestPayload{
			Queries: queries,
		},
	}

	if err = req.Sign(c.privKey); err != nil {
		return
	}

	// set receipt if key exists in context
	if val := ctx.Value(&ctxReceiptKey); val != nil {
		val.(*atomic.Value).Store(&Receipt{
			RequestHash: req.Header.Hash(),
		})
	}


	//wyong, 20201008 
	//if err = uc.pCaller.Call(route.DBSQuery.String(), req, &response); err != nil {
	if _, err = uc.pCaller.SendMsg(ctx, req); err != nil {
		return
	}

	//wyong, 20201021 
	var response types.Response
	err = uc.pCaller.RecvMsg(ctx, response) 
	if err != nil {
		fmt.Printf("Client/Conn/sendUrlRequest(47), err=%s\n", err.Error()) 
		return
	}


	rows = newRows(&response)
	if queryType == types.WriteQuery {
		affectedRows = response.Header.AffectedRows
		lastInsertID = response.Header.LastInsertID
	}

	// build ack
	func() {
		defer trace.StartRegion(ctx, "ackEnqueue").End()
		if uc.ackCh != nil {
			uc.ackCh <- &types.Ack{
				Header: types.SignedAckHeader{
					AckHeader: types.AckHeader{
						Response:     response.Header.ResponseHeader,
						ResponseHash: response.Header.Hash(),
						NodeID:       c.localNodeID,
						Timestamp:    getLocalTime(),
					},
				},
			}
		}
	}()

	return
}


func getLocalTime() time.Time {
	return time.Now().UTC()
}

func convertQuery(query string, args []driver.NamedValue) (sq *types.Query) {
	// rebuild args to named args
	sq = &types.Query{
		Pattern: query,
	}

	sq.Args = make([]types.NamedArg, len(args))

	for i, v := range args {
		sq.Args[i].Name = v.Name
		sq.Args[i].Value = v.Value
	}

	return
}
