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

package types

import (
	"time"

	pi "github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/crypto/hash"
	"github.com/siegfried415/go-crawling-bazaar/crypto/verifier"
	"github.com/siegfried415/go-crawling-bazaar/merkle"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

//go:generate hsp

// PBHeader defines the main chain block header.
type PBHeader struct {
	Version    int32
	Producer   proto.AccountAddress
	MerkleRoot hash.Hash
	ParentHash hash.Hash
	Timestamp  time.Time
}

// PBSignedHeader defines the main chain header with the signature.
type PBSignedHeader struct {
	PBHeader
	verifier.DefaultHashSignVerifierImpl
}

func (s *PBSignedHeader) verifyHash() error {
	return s.DefaultHashSignVerifierImpl.VerifyHash(&s.PBHeader)
}

func (s *PBSignedHeader) verify() error {
	return s.DefaultHashSignVerifierImpl.Verify(&s.PBHeader)
}

func (s *PBSignedHeader) setHash() error {
	return s.DefaultHashSignVerifierImpl.SetHash(&s.PBHeader)
}

func (s *PBSignedHeader) sign(signer *asymmetric.PrivateKey) error {
	return s.DefaultHashSignVerifierImpl.Sign(&s.PBHeader, signer)
}

// PBBlock defines the main chain block.
type PBBlock struct {
	SignedHeader PBSignedHeader
	Transactions []pi.Transaction
}

// GetTxHashes returns all hashes of tx in block.{Billings, ...}.
func (b *PBBlock) GetTxHashes() []*hash.Hash {
	// TODO(lambda): when you add new tx type, you need to put new tx's hash in the slice
	// get hashes in block.Transactions
	hs := make([]*hash.Hash, len(b.Transactions))

	for i, v := range b.Transactions {
		h := v.Hash()
		hs[i] = &h
	}
	return hs
}

func (b *PBBlock) setMerkleRoot() {
	var merkleRoot = merkle.NewMerkle(b.GetTxHashes()).GetRoot()
	b.SignedHeader.MerkleRoot = *merkleRoot
}

func (b *PBBlock) verifyMerkleRoot() error {
	var merkleRoot = *merkle.NewMerkle(b.GetTxHashes()).GetRoot()
	if !merkleRoot.IsEqual(&b.SignedHeader.MerkleRoot) {
		return ErrMerkleRootVerification
	}
	return nil
}

// SetHash sets the block header hash, including the merkle root of the packed transactions.
func (b *PBBlock) SetHash() error {
	b.setMerkleRoot()
	return b.SignedHeader.setHash()
}

// VerifyHash verifies the block header hash, including the merkle root of the packed transactions.
func (b *PBBlock) VerifyHash() error {
	if err := b.verifyMerkleRoot(); err != nil {
		return err
	}
	return b.SignedHeader.verifyHash()
}

// PackAndSignBlock computes block's hash and sign it.
func (b *PBBlock) PackAndSignBlock(signer *asymmetric.PrivateKey) error {
	b.setMerkleRoot()
	return b.SignedHeader.sign(signer)
}

// Verify verifies whether the block is valid.
func (b *PBBlock) Verify() error {
	if err := b.verifyMerkleRoot(); err != nil {
		return err
	}
	return b.SignedHeader.verify()
}

// Timestamp returns timestamp of block.
func (b *PBBlock) Timestamp() time.Time {
	return b.SignedHeader.Timestamp
}

// Producer returns the producer of block.
func (b *PBBlock) Producer() proto.AccountAddress {
	return b.SignedHeader.Producer
}

// ParentHash returns the parent hash field of the block header.
func (b *PBBlock) ParentHash() *hash.Hash {
	return &b.SignedHeader.ParentHash
}

// BlockHash returns the parent hash field of the block header.
func (b *PBBlock) BlockHash() *hash.Hash {
	return &b.SignedHeader.DataHash
}
