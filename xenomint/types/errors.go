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
	"errors"
)

var (
	// ErrMerkleRootNotMatch indicates the merkle root not match error from verifier.
	ErrMerkleRootNotMatch = errors.New("merkle root not match")
	// ErrHashValueNotMatch indicates the hash value not match error from verifier.
	ErrHashValueNotMatch = errors.New("hash value not match")
	// ErrSignatureNotMatch indicates the signature not match error from verifier.
	ErrSignatureNotMatch = errors.New("signature not match")
)
