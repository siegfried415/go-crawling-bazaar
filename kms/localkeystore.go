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

package kms

import (
	"errors"
	"sync"

	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	//"github.com/siegfried415/go-crawling-bazaar/crypto/hash"

	//wyong, 20211206 
	//mine "github.com/siegfried415/go-crawling-bazaar/pow/cpuminer"

	"github.com/siegfried415/go-crawling-bazaar/proto"
)

// LocalKeyStore is the type hold local private & public key.
type LocalKeyStore struct {
	isSet     bool
	private   *asymmetric.PrivateKey
	public    *asymmetric.PublicKey
	nodeID    []byte

	//wyong, 20211206 
	//nodeNonce *mine.Uint256

	sync.RWMutex
}

var (
	// localKey is global accessible local private & public key
	localKey *LocalKeyStore
	once     sync.Once
)

var (
	// ErrNilField indicates field is nil
	ErrNilField = errors.New("local field is nil")
)

func init() {
	initLocalKeyStore()
}

// initLocalKeyStore returns a new LocalKeyStore.
func initLocalKeyStore() {
	once.Do(func() {
		localKey = &LocalKeyStore{
			isSet:     false,
			private:   nil,
			public:    nil,
			nodeID:    nil,
			
			//wyong, 20211206 
			//nodeNonce: nil,
		}
	})
}

// ResetLocalKeyStore FOR UNIT TEST, DO NOT USE IT.
func ResetLocalKeyStore() {
	localKey = &LocalKeyStore{
		isSet:     false,
		private:   nil,
		public:    nil,
		nodeID:    nil,

		//wyong, 20211206 
		//nodeNonce: nil,
	}
}

// SetLocalKeyPair sets private and public key, this is a one time thing.
func SetLocalKeyPair(private *asymmetric.PrivateKey, public *asymmetric.PublicKey) {
	localKey.Lock()
	defer localKey.Unlock()
	if localKey.isSet {
		return
	}
	localKey.isSet = true
	localKey.private = private
	localKey.public = public
}

// SetLocalNodeIDNonce sets private and public key, this is a one time thing.
func SetLocalNodeID /* Nonce */ (rawNodeID []byte /* , nonce *mine.Uint256  wyong, 20201116 */){
	localKey.Lock()
	defer localKey.Unlock()
	localKey.nodeID = make([]byte, len(rawNodeID))
	copy(localKey.nodeID, rawNodeID)

	//wyong, 20201116 
	//if nonce != nil {
	//	localKey.nodeNonce = new(mine.Uint256)
	//	*localKey.nodeNonce = *nonce
	//}
}

// GetLocalNodeID gets current node ID in hash string format.
func GetLocalNodeID() (rawNodeID proto.NodeID, err error) {
	var rawNodeIDBytes []byte
	if rawNodeIDBytes, err = GetLocalNodeIDBytes(); err != nil {
		return
	}

	//todo, wyong, 20201114 
	//var h *hash.Hash
	//if h, err = hash.NewHash(rawNodeIDBytes); err != nil {
	//	return
	//}
	//rawNodeID = proto.NodeID(h.String())
	rawNodeID = proto.NodeID(string(rawNodeIDBytes)) 

	return
}

// GetLocalNodeIDBytes get current node ID copy in []byte.
func GetLocalNodeIDBytes() (rawNodeID []byte, err error) {
	localKey.RLock()
	if localKey.nodeID != nil {
		rawNodeID = make([]byte, len(localKey.nodeID))
		copy(rawNodeID, localKey.nodeID)
	} else {
		err = ErrNilField
	}
	localKey.RUnlock()
	return
}


/* wyong, 20211206 
// GetLocalNonce gets current node nonce copy.
func GetLocalNonce() (nonce *mine.Uint256, err error) {
	localKey.RLock()
	if localKey.nodeNonce != nil {
		nonce = new(mine.Uint256)
		*nonce = *localKey.nodeNonce
	} else {
		err = ErrNilField
	}
	localKey.RUnlock()
	return
}
*/

// GetLocalPublicKey gets local public key, if not set yet returns nil.
func GetLocalPublicKey() (public *asymmetric.PublicKey, err error) {
	localKey.RLock()
	public = localKey.public
	if public == nil {
		err = ErrNilField
	}
	localKey.RUnlock()
	return
}

// GetLocalPrivateKey gets local private key, if not set yet returns nil
//  all call to this func will be logged.
func GetLocalPrivateKey() (private *asymmetric.PrivateKey, err error) {
	localKey.RLock()
	private = localKey.private
	if private == nil {
		err = ErrNilField
	}
	localKey.RUnlock()

	// log the call stack
	//buf := make([]byte, 4096)
	//count := runtime.Stack(buf, false)
	//log.Debugf("###getting private key from###\n%s\n###getting private  key end###\n", buf[:count])
	return
}
