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

package asymmetric

import (
	"crypto/ecdsa"

	//wyong, 20201002 
	"crypto/x509"

	//wyong, 20201028
	//"crypto/elliptic" 
	//"errors"

	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
	ec "github.com/btcsuite/btcd/btcec"

	mine "github.com/siegfried415/go-crawling-bazaar/pow/cpuminer"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"

	//wyong, 20201002 
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"

)

// PrivateKeyBytesLen defines the length in bytes of a serialized private key.
const PrivateKeyBytesLen = ec.PrivKeyBytesLen

// PublicKeyBytesLen defines the length in bytes of a serialized public key.
const PublicKeyBytesLen = ec.PubKeyBytesLenCompressed

// PublicKeyFormatHeader is the default header of PublicKey.Serialize()
//  see: github.com/btcsuite/btcd/btcec/pubkey.go#L63
const PublicKeyFormatHeader byte = 0x2

var parsedPublicKeyCache sync.Map

// PrivateKey wraps an ec.PrivateKey as a convenience mainly for signing things with the the
// private key without having to directly import the ecdsa package.
type PrivateKey ec.PrivateKey

// PublicKey wraps an ec.PublicKey as a convenience mainly verifying signatures with the the
// public key without having to directly import the ecdsa package.
type PublicKey ec.PublicKey

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message.
func (k PublicKey) Msgsize() (s int) {
	s = hsp.BytesPrefixSize + ec.PubKeyBytesLenCompressed
	return
}

// MarshalHash marshals for hash.
func (k *PublicKey) MarshalHash() (keyBytes []byte, err error) {
	return k.MarshalBinary()
}

// MarshalBinary does the serialization.
func (k *PublicKey) MarshalBinary() (keyBytes []byte, err error) {
	return k.Serialize(), nil
}

// UnmarshalBinary does the deserialization.
func (k *PublicKey) UnmarshalBinary(keyBytes []byte) (err error) {
	pubKeyI, ok := parsedPublicKeyCache.Load(string(keyBytes))
	if ok {
		*k = *pubKeyI.(*PublicKey)
	} else {
		pubNew, err := ParsePubKey(keyBytes)
		if err == nil {
			*k = *pubNew
			parsedPublicKeyCache.Store(string(keyBytes), pubNew)
		}
	}
	return
}

// MarshalYAML implements the yaml.Marshaler interface.
func (k PublicKey) MarshalYAML() (interface{}, error) {
	return fmt.Sprintf("%x", k.Serialize()), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (k *PublicKey) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	// load public key string
	pubKeyBytes, err := hex.DecodeString(str)
	if err != nil {
		return err
	}

	err = k.UnmarshalBinary(pubKeyBytes)

	return err
}

// IsEqual return true if two keys are equal.
func (k *PublicKey) IsEqual(public *PublicKey) bool {
	return (*ec.PublicKey)(k).IsEqual((*ec.PublicKey)(public))
}

// Serialize is a function that converts a public key
// to uncompressed byte array
//
// TODO(leventeliu): use SerializeUncompressed, which is about 40 times faster than
// SerializeCompressed.
//
// BenchmarkParsePublicKey-12                 50000             39819 ns/op            2401 B/op         35 allocs/op
// BenchmarkParsePublicKey-12               1000000              1039 ns/op             384 B/op          9 allocs/op.
func (k *PublicKey) Serialize() []byte {
	return (*ec.PublicKey)(k).SerializeCompressed()
}

// ParsePubKey recovers the public key from pubKeyStr.
func ParsePubKey(pubKeyStr []byte) (*PublicKey, error) {
	key, err := ec.ParsePubKey(pubKeyStr, ec.S256())
	return (*PublicKey)(key), err
	
	//todo, wyong, 20201028 
        //pubIfc, err := x509.ParsePKIXPublicKey(pubKeyStr)
        //if err != nil {
        //        return nil, err
        //}
        //pub, ok := pubIfc.(*ecdsa.PublicKey)
        //if !ok {
        //        return nil, errors.New("not an ecdsa public key") 
        //}
	//return (*PublicKey)(pub), nil  

}

// PrivKeyFromBytes returns a private and public key for `curve' based on the private key passed
// as an argument as a byte slice.
func PrivKeyFromBytes(pk []byte) (*PrivateKey, *PublicKey) {
	//todo, wyong, 20201028 
	x, y := ec.S256().ScalarBaseMult(pk)
	//x, y := elliptic.P256().ScalarBaseMult(pk)

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			//todo, wyong, 20201028, 
			Curve:  ec.S256(),
			//Curve:  elliptic.P256(), 	

			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}

	return (*PrivateKey)(priv), (*PublicKey)(&priv.PublicKey)
}

//wyong, 20200916 
func (private *PrivateKey) Bytes() ([]byte, error) {
	return private.Serialize(), nil 
}

//wyong, 20200916 
// Equals compares two private keys
func (private *PrivateKey) Equals(target PrivateKey) bool {
	//wyong, 20201013 
        //targetPriv, ok := ecdsa.PrivateKey(target)
        //if !ok {
        //        return false
        //}
        //return private.(*ecdsa.PrivateKey).D.Cmp(targetPriv.D) == 0

        ecdsaTargetPriv := (ecdsa.PrivateKey)(target)
	ecdsaPrivateKey := (*ecdsa.PrivateKey) (private )
        return ecdsaPrivateKey.D.Cmp(ecdsaTargetPriv.D) == 0
}

// Serialize returns the private key number d as a big-endian binary-encoded number, padded to a
// length of 32 bytes.
func (private *PrivateKey) Serialize() []byte {
	b := make([]byte, 0, PrivateKeyBytesLen)
	return paddedAppend(PrivateKeyBytesLen, b, private.D.Bytes())
}

//wyong, 20201002 
// Raw returns x509 bytes from a private key
func (private *PrivateKey) Raw() ([]byte, error) {
	//wyong, 20201013 
        //return x509.MarshalECPrivateKey(private)
        return x509.MarshalECPrivateKey((*ecdsa.PrivateKey)(private))
}

//wyong, 20201002 
// Type returns the key type
func (private *PrivateKey) Type() pb.KeyType {
        return pb.KeyType_ECDSA
}

//Alias of PubKey(), wyong, 20201002 
func (private *PrivateKey) GetPublic() *PublicKey {
	return private.PubKey() 
}

// PubKey return the public key.
func (private *PrivateKey) PubKey() *PublicKey {
	return (*PublicKey)((*ec.PrivateKey)(private).PubKey())
}

// paddedAppend appends the src byte slice to dst, returning the new slice. If the length of the
// source is smaller than the passed size, leading zero bytes are appended to the dst slice before
// appending src.
func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}

// GenSecp256k1KeyPair generate Secp256k1(used by Bitcoin) key pair.
func GenSecp256k1KeyPair() (
	privateKey *PrivateKey,
	publicKey *PublicKey,
	err error) {

	privateKeyEc, err := ec.NewPrivateKey( /* todo, wyong, 20201028*/  ec.S256() /* elliptic.P256() */  )
	if err != nil {
		log.WithError(err).Error("private key generation failed")
		return nil, nil, err
	}
	publicKey = (*PublicKey)(privateKeyEc.PubKey())
	privateKey = (*PrivateKey)(privateKeyEc)
	return
}

// GetPubKeyNonce will make his best effort to find a difficult enough
// nonce.
func GetPubKeyNonce(
	publicKey *PublicKey,
	difficulty int,
	timeThreshold time.Duration,
	quit chan struct{}) (nonce mine.NonceInfo) {

	miner := mine.NewCPUMiner(quit)
	nonceCh := make(chan mine.NonceInfo)
	// if miner finished his work before timeThreshold
	// make sure writing to the Stop chan non-blocking.
	stop := make(chan struct{}, 1)
	block := mine.MiningBlock{
		Data:      publicKey.Serialize(),
		NonceChan: nonceCh,
		Stop:      stop,
	}

	go miner.ComputeBlockNonce(block, mine.Uint256{}, difficulty)

	time.Sleep(timeThreshold)
	// stop miner
	block.Stop <- struct{}{}

	return <-block.NonceChan
}
