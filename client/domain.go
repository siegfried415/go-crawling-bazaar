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
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/crypto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/crypto/hash"

	"github.com/siegfried415/go-crawling-bazaar/kms"
	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"

	pb "github.com/siegfried415/go-crawling-bazaar/presbyterian"
	"github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"

	"github.com/siegfried415/go-crawling-bazaar/types"
	"github.com/siegfried415/go-crawling-bazaar/utils"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
)

const (
	// DBScheme defines the dsn scheme.
	DBScheme = "covenantsql"
	// DBSchemeAlias defines the alias dsn scheme.
	DBSchemeAlias = "cql"
	// DefaultGasPrice defines the default gas price for new created database.
	DefaultGasPrice = 1
	// DefaultAdvancePayment defines the default advance payment for new created database.
	DefaultAdvancePayment = 20000000
)

var (
	// PeersUpdateInterval defines peers list refresh interval for client.
	PeersUpdateInterval = time.Second * 5

	driverInitialized   uint32

	peersUpdaterRunning uint32
	peerList            sync.Map // map[proto.DatabaseID]*proto.Peers
	connIDLock          sync.Mutex
	connIDAvail         []uint64
	globalSeqNo         uint64
	randSource          = rand.New(rand.NewSource(time.Now().UnixNano()))

	// DefaultConfigFile is the default path of config file
	DefaultConfigFile = "~/.cql/config.yaml"
)

func init() {
	/* 
	d := new(covenantSQLDriver)
	sql.Register(DBScheme, d)
	sql.Register(DBSchemeAlias, d)
	log.Debug("CovenantSQL driver registered.")
	*/
}

/* 
// covenantSQLDriver implements sql.Driver interface.
type covenantSQLDriver struct {
}

// Open returns new db connection.
func (d *covenantSQLDriver) Open(dsn string) (conn driver.Conn, err error) {
	var cfg *Config
	if cfg, err = ParseDSN(dsn); err != nil {
		return
	}

	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = defaultInit()
		if err != nil && err != ErrAlreadyInitialized {
			return
		}
	}

	return newConn(cfg)

}
*/


// ResourceMeta defines new database resources requirement descriptions.
type ResourceMeta struct {
	// copied fields from types.ResourceMeta
	TargetMiners           []proto.AccountAddress `json:"target-miners,omitempty"`        // designated miners
	Node                   uint16                 `json:"node,omitempty"`                 // reserved node count

	Domain			string			`json:"domain-name, omitempty"`

	Space                  uint64                 `json:"space,omitempty"`                // reserved storage space in bytes
	Memory                 uint64                 `json:"memory,omitempty"`               // reserved memory in bytes
	LoadAvgPerCPU          float64                `json:"load-avg-per-cpu,omitempty"`     // max loadAvg15 per CPU
	EncryptionKey          string                 `json:"encrypt-key,omitempty"`          // encryption key for database instance
	UseEventualConsistency bool                   `json:"eventual-consistency,omitempty"` // use eventual consistency replication if enabled
	ConsistencyLevel       float64                `json:"consistency-level,omitempty"`    // customized strong consistency level
	IsolationLevel         int                    `json:"isolation-level,omitempty"`      // customized isolation level

	GasPrice       uint64 `json:"gas-price"`       // customized gas price
	AdvancePayment uint64 `json:"advance-payment"` // customized advance payment
}

func defaultInit(host net.RoutedHost) (err error) {
	configFile := utils.HomeDirExpand(DefaultConfigFile)
	if configFile == DefaultConfigFile {
		//System not support ~ dir, need Init manually.
		log.Debugf("Could not find CovenantSQL default config location: %v", configFile)
		return ErrNotInitialized
	}

	log.Debugf("Using CovenantSQL default config location: %v", configFile)
	return Init(host, configFile, []byte(""))
}

//todo
// Init defines init process for client.
func Init(host net.RoutedHost, configFile string, masterKey []byte) (err error) {
	if !atomic.CompareAndSwapUint32(&driverInitialized, 0, 1) {
		err = ErrAlreadyInitialized
		return
	}

	// load config
	if conf.GConf, err = conf.LoadConfig(configFile); err != nil {
		return
	}

	//todo
	//route.InitKMS(conf.GConf.PubKeyStoreFile)
	if err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey); err != nil {
		return
	}

	// ping presbyterian to register node
	if err = registerNode(host); err != nil {
		return
	}

	// run peers updater
	if err = runPeerListUpdater(host); err != nil {
		return
	}

	return
}


//todo
// CreateDomain sends create domain operation to presbyterian.
func CreateDomain(host net.RoutedHost, meta ResourceMeta) (txHash hash.Hash, /* dsn string */  err error) {
	//if atomic.LoadUint32(&driverInitialized) == 0 {
	//	err = ErrNotInitialized
	//	return
	//}

	var (
		nonceReq   = new(types.NextAccountNonceReq)
		nonceResp  = new(types.NextAccountNonceResp)
		req        = new(types.AddTxReq)
		resp       = new(types.AddTxResp)
		privateKey *asymmetric.PrivateKey
		clientAddr proto.AccountAddress
	)

	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		err = errors.Wrap(err, "get local private key failed")
		return
	}
	if clientAddr, err = crypto.PubKeyHash(privateKey.PubKey()); err != nil {
		err = errors.Wrap(err, "get local account address failed")
		return
	}
	// allocate nonce
	nonceReq.Addr = clientAddr

	if err = host.RequestPB("MCC.NextAccountNonce", &nonceReq, &nonceResp); err != nil {
		err = errors.Wrap(err, "allocate create database transaction nonce failed")
		return
	}

	if meta.GasPrice == 0 {
		meta.GasPrice = DefaultGasPrice
	}
	if meta.AdvancePayment == 0 {
		meta.AdvancePayment = DefaultAdvancePayment
	}

	req.TTL = 1
	req.Tx = types.NewCreateDomain(&types.CreateDomainHeader{
		Owner: clientAddr,
		ResourceMeta: types.ResourceMeta{
			TargetMiners:           meta.TargetMiners,
			Node:                   meta.Node,

			Domain:			meta.Domain, 	

			//Space:                  meta.Space,
			//Memory:                 meta.Memory,
			//LoadAvgPerCPU:          meta.LoadAvgPerCPU,

			EncryptionKey:          meta.EncryptionKey,
			UseEventualConsistency: meta.UseEventualConsistency,
			ConsistencyLevel:       meta.ConsistencyLevel,
			IsolationLevel:         meta.IsolationLevel,
		},
		GasPrice:       meta.GasPrice,
		AdvancePayment: meta.AdvancePayment,
		TokenType:      types.Particle,
		Nonce:          nonceResp.Nonce,
	})

	if err = req.Tx.Sign(privateKey); err != nil {
		err = errors.Wrap(err, "sign request failed")
		return
	}

	if err = host.RequestPB("MCC.AddTx", &req, &resp); err != nil {
		err = errors.Wrap(err, "call create database transaction failed")
		return
	}

	txHash = req.Tx.Hash()

	//cfg := NewConfig()
	//cfg.DomainID = string(proto.FromAccountAndNonce(clientAddr, uint32(nonceResp.Nonce)))
	//dsn = cfg.FormatDSN()

	return
}

// WaitDomainCreation waits for domain creation complete.
func WaitDomainCreation(ctx context.Context, host net.RoutedHost, dsn string) (err error) {
	//dsnCfg, err := ParseDSN(dsn)
	//if err != nil {
	//	return
	//}

	//todo
	//db, err := sql.Open("covenantsql", dsn)
	//defer db.Close()
	//if err != nil {
	//	return
	//}

	// wait for creation
	err = WaitPBDomainCreation(ctx, host, proto.DomainID(/* dsnCfg.DomainID */ dsn ), /* db, */ 3*time.Second)
	return
}

// WaitPBDomainCreation waits for database creation complete.
func WaitPBDomainCreation(
	ctx context.Context,
	host net.RoutedHost, 
	domainID proto.DomainID,
	//db *sql.DB, 
	period time.Duration,
) (err error) {
	var (
		ticker = time.NewTicker(period)
		req    = &types.QuerySQLChainProfileReq{
			DomainID: domainID,
		}
		
		resp   = new(types.QuerySQLChainProfileResp)
		count = 0
	)
	defer ticker.Stop()
	defer log.Debugf("\n")

	for {
		select {
		case <-ticker.C:
			count++

			if err = host.RequestPB("MCC.QuerySQLChainProfile", &req, &resp,); err != nil {
				if !strings.Contains(err.Error(), pb.ErrDatabaseNotFound.Error()) {
					// err != nil && err != ErrDatabaseNotFound (unexpected error)
					return
				}
			} else {
				/* 
				// err == nil (creation done on Presbyterian ): try to use database connection
				if db == nil {
					return
				}
				if _, err = db.ExecContext(ctx, "SHOW TABLES"); err == nil {
					// err == nil (connect to Miner OK)
					return
				}
				*/ 
				return 
			}

			log.Debugf("\rQuerying SQLChain Profile %vs", count*int(period.Seconds()))

		case <-ctx.Done():
			if err != nil {
				return errors.Wrapf(ctx.Err(), "last error: %s", err.Error())
			}
			return ctx.Err()
		}
	}
}

// Drop sends drop database operation to presbyterian.
func Drop(host net.RoutedHost, dsn string) (txHash hash.Hash, err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	var cfg *Config
	if cfg, err = ParseDSN(dsn); err != nil {
		return
	}

	peerList.Delete(cfg.DomainID)

	//TODO(laodouya) currently not supported
	//err = errors.New("drop db current not support")

	return
}

/* 
// GetTokenBalance get the token balance of current account.
func GetTokenBalance(tt types.TokenType) (balance uint64, err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	req := new(types.QueryAccountTokenBalanceReq)
	resp := new(types.QueryAccountTokenBalanceResp)

	var pubKey *asymmetric.PublicKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	if req.Addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}
	req.TokenType = tt

	if err = net.RequestPB("MCC.QueryAccountTokenBalance", &req, &resp); err == nil {
		if !resp.OK {
			err = ErrNoSuchTokenBalance
			return
		}
		balance = resp.Balance
	}

	return
}

// UpdatePermission sends UpdatePermission transaction to chain.
func UpdatePermission(targetUser proto.AccountAddress,
	targetChain proto.AccountAddress, perm *types.UserPermission) (txHash hash.Hash, err error) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	var (
		pubKey  *asymmetric.PublicKey
		privKey *asymmetric.PrivateKey
		addr    proto.AccountAddress
		nonce   interfaces.AccountNonce
	)
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}

	nonce, err = getNonce(addr)
	if err != nil {
		return
	}

	up := types.NewUpdatePermission(&types.UpdatePermissionHeader{
		TargetSQLChain: targetChain,
		TargetUser:     targetUser,
		Permission:     perm,
		Nonce:          nonce,
	})
	err = up.Sign(privKey)
	if err != nil {
		log.WithError(err).Warning("sign failed")
		return
	}
	addTxReq := new(types.AddTxReq)
	addTxResp := new(types.AddTxResp)
	addTxReq.Tx = up
	err = net.RequestPB("MCC.AddTx", &addTxReq, &addTxResp)
	if err != nil {
		log.WithError(err).Warning("send tx failed")
		return
	}

	txHash = up.Hash()
	return
}

// TransferToken send Transfer transaction to chain.
func TransferToken(targetUser proto.AccountAddress, amount uint64, tokenType types.TokenType) (
	txHash hash.Hash, err error,
) {
	if atomic.LoadUint32(&driverInitialized) == 0 {
		err = ErrNotInitialized
		return
	}

	var (
		pubKey  *asymmetric.PublicKey
		privKey *asymmetric.PrivateKey
		addr    proto.AccountAddress
		nonce   interfaces.AccountNonce
	)
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}

	nonce, err = getNonce(addr)
	if err != nil {
		return
	}

	tran := types.NewTransfer(&types.TransferHeader{
		Sender:    addr,
		Receiver:  targetUser,
		Amount:    amount,
		TokenType: tokenType,
		Nonce:     nonce,
	})
	err = tran.Sign(privKey)
	if err != nil {
		log.WithError(err).Warning("sign failed")
		return
	}
	addTxReq := new(types.AddTxReq)
	addTxResp := new(types.AddTxResp)
	addTxReq.Tx = tran
	err = net.RequestPB("MCC.AddTx", &addTxReq, &addTxResp)
	if err != nil {
		log.WithError(err).Warning("send tx failed")
		return
	}

	txHash = tran.Hash()
	return
}
*/


// WaitTxConfirmation waits for the transaction with target hash txHash to be confirmed. It also
// returns if any error occurs or a final state is returned from Presbyterian.
func WaitTxConfirmation(
	ctx context.Context, host net.RoutedHost, txHash hash.Hash) (state interfaces.TransactionState, err error,
) {
	var (
		ticker = time.NewTicker(1 * time.Second)
		//method = route.MCCQueryTxState
		req    = &types.QueryTxStateReq{Hash: txHash}
		resp   = &types.QueryTxStateResp{}
		count  = 0
	)
	defer ticker.Stop()
	defer log.Debugf("\n")
	for {
		if err = host.RequestPB("MCC.QueryTxState", req, resp); err != nil {
			err = errors.Wrapf(err, "failed to call %s", "MCC.QueryTxState" )
			return
		}

		state = resp.State

		count++
		log.Debugf("\rWaiting presbyterians confirmation %vs, state: %v\033[0K", count, state)
		log.WithFields(log.Fields{
			"tx_hash":  txHash,
			"tx_state": state,
		}).Debug("waiting for tx confirmation")

		switch state {
		case interfaces.TransactionStatePending:
		case interfaces.TransactionStatePacked:
		case interfaces.TransactionStateConfirmed,
			interfaces.TransactionStateExpired,
			interfaces.TransactionStateNotFound:
			return
		default:
			err = errors.Errorf("unknown transaction state %d", state)
			return
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

/*
func getNonce(addr proto.AccountAddress) (nonce interfaces.AccountNonce, err error) {
	nonceReq := new(types.NextAccountNonceReq)
	nonceResp := new(types.NextAccountNonceResp)
	nonceReq.Addr = addr
	err = net.RequestPB(route.MCCNextAccountNonce, &nonceReq, &nonceResp)
	if err != nil {
		log.WithError(err).Warning("get nonce failed")
		return
	}
	nonce = nonceResp.Nonce
	return
}
*/


//func requestPB(method route.RemoteFunc, request interface{}, response interface{}) (err error) {
//	var pbNodeID proto.NodeID
//	if pbNodeID, err = rpc.GetCurrentPB(); err != nil {
//		return
//	}
//
//	return rpc.NewCaller().CallNode(pbNodeID, method.String(), request, response)
//}

func registerNode(host net.RoutedHost ) (err error) {
	var nodeID proto.NodeID

	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	var nodeInfo *proto.Node
	if nodeInfo, err = kms.GetNodeInfo(nodeID); err != nil {
		return
	}

	if nodeInfo.Role != proto.Leader && nodeInfo.Role != proto.Follower {
		log.Infof("Self register to presbyterian: %v", conf.GConf.PB.NodeID)
		err = host.PingPB(nodeInfo, conf.GConf.PB.NodeID)
	}

	return
}

func runPeerListUpdater(host net.RoutedHost ) (err error) {
	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	if !atomic.CompareAndSwapUint32(&peersUpdaterRunning, 0, 1) {
		return
	}

	go func() {
		for {
			if atomic.LoadUint32(&peersUpdaterRunning) == 0 {
				return
			}

			var wg sync.WaitGroup

			peerList.Range(func(rawDomainID, _ interface{}) bool {
				domainID := rawDomainID.(proto.DomainID)

				wg.Add(1)
				go func(domainID proto.DomainID) {
					defer wg.Done()
					var err error

					if _, err = getPeers(host, domainID, privKey); err != nil {
						log.WithField("domain", domainID).
							WithError(err).
							Debug("update peers failed")

						// TODO(xq262144), better rpc remote error judgement
						if strings.Contains(err.Error(), pb.ErrNoSuchDatabase.Error()) {
							log.WithField("domain", domainID).
								Warning("domain no longer exists, stopping peers update")
							peerList.Delete(domainID)
						}
					}
				}(domainID)

				return true
			})

			wg.Wait()

			time.Sleep(PeersUpdateInterval)
		}
	}()

	return
}

func stopPeersUpdater() {
	atomic.StoreUint32(&peersUpdaterRunning, 0)
}

func cacheGetPeers(host net.RoutedHost, domainID proto.DomainID, privKey *asymmetric.PrivateKey) (peers *proto.Peers, err error) {
	var ok bool
	var rawPeers interface{}
	var cacheHit bool

	defer func() {
		log.WithFields(log.Fields{
			"domain":  domainID,
			"hit": cacheHit,
		}).WithError(err).Debug("cache get peers for database")
	}()

	if rawPeers, ok = peerList.Load(domainID); ok {
		if peers, ok = rawPeers.(*proto.Peers); ok {
			cacheHit = true
			return
		}
	}

	// get peers using non-cache method
	return getPeers(host, domainID, privKey)
}

func getPeers(host net.RoutedHost, domainID proto.DomainID, privKey *asymmetric.PrivateKey) (peers *proto.Peers, err error) {
	defer func() {
		log.WithFields(log.Fields{
			"domain":    domainID,
			"peers": peers,
		}).WithError(err).Debug("get peers for database")
	}()

	profileReq := types.QuerySQLChainProfileReq{}
	profileResp := types.QuerySQLChainProfileResp{}
	profileReq.DomainID = domainID

	err = host.RequestPB("MCC.QuerySQLChainProfile", &profileReq, &profileResp)
	if err != nil {
		err = errors.Wrap(err, "get sqlchain profile failed in getPeers")
		return
	}

	nodeIDs := make([]proto.NodeID, len(profileResp.Profile.Miners))
	if len(profileResp.Profile.Miners) <= 0 {
		err = errors.Wrap(ErrInvalidProfile, "unexpected error in getPeers")
		return
	}
	for i, mi := range profileResp.Profile.Miners {
		nodeIDs[i] = mi.NodeID
	}
	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Leader:  nodeIDs[0],
			Servers: nodeIDs[:],
		},
	}
	err = peers.Sign(privKey)
	if err != nil {
		err = errors.Wrap(err, "sign peers failed in getPeers")
		return
	}

	// set peers in the updater cache
	peerList.Store(domainID, peers)

	return
}

func allocateConnAndSeq() (connID uint64, seqNo uint64) {
	connIDLock.Lock()
	defer connIDLock.Unlock()

	if len(connIDAvail) == 0 {
		// generate one
		connID = randSource.Uint64()
		seqNo = atomic.AddUint64(&globalSeqNo, 1)
		return
	}

	// pop one conn
	connID = connIDAvail[0]
	connIDAvail = connIDAvail[1:]
	seqNo = atomic.AddUint64(&globalSeqNo, 1)

	return
}

func putBackConn(connID uint64) {
	connIDLock.Lock()
	defer connIDLock.Unlock()

	connIDAvail = append(connIDAvail, connID)
}
