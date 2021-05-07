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
	pi "github.com/siegfried415/go-crawling-bazaar/presbyterian/interfaces"
	"github.com/siegfried415/go-crawling-bazaar/crypto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/crypto/verifier"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

//go:generate hsp

// CreateDomainHeader defines the database creation transaction header.
type CreateDomainHeader struct {
	Owner          proto.AccountAddress
	ResourceMeta   ResourceMeta
	GasPrice       uint64
	AdvancePayment uint64
	TokenType      TokenType
	Nonce          pi.AccountNonce
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (h *CreateDomainHeader) GetAccountNonce() pi.AccountNonce {
	return h.Nonce
}

// CreateDomain defines the database creation transaction.
type CreateDomain struct {
	CreateDomainHeader
	pi.TransactionTypeMixin
	verifier.DefaultHashSignVerifierImpl
}

// NewCreateDomain returns new instance.
func NewCreateDomain(header *CreateDomainHeader) *CreateDomain {
	return &CreateDomain{
		CreateDomainHeader: *header,
		TransactionTypeMixin: *pi.NewTransactionTypeMixin(pi.TransactionTypeCreateDomain),
	}
}

// Sign implements interfaces/Transaction.Sign.
func (cd *CreateDomain) Sign(signer *asymmetric.PrivateKey) (err error) {
	return cd.DefaultHashSignVerifierImpl.Sign(&cd.CreateDomainHeader, signer)
}

// Verify implements interfaces/Transaction.Verify.
func (cd *CreateDomain) Verify() error {
	return cd.DefaultHashSignVerifierImpl.Verify(&cd.CreateDomainHeader)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (cd *CreateDomain) GetAccountAddress() proto.AccountAddress {
	addr, _ := crypto.PubKeyHash(cd.Signee)
	return addr
}

func init() {
	pi.RegisterTransaction(pi.TransactionTypeCreateDomain, (*CreateDomain)(nil))
}
