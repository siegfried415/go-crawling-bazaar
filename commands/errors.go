/*
 * Copyright (c) 2018 Filecoin Project
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

package commands

import (
	"fmt"
	"github.com/pkg/errors"
)

var (
	// ErrInvalidSize indicates that the provided size was invalid.
	ErrInvalidSize = fmt.Errorf("invalid size")

	// ErrInvalidPrice indicates that the provided price was invalid.
	ErrInvalidPrice = fmt.Errorf("invalid price")

	// ErrInvalidAmount indicates that the provided amount was invalid.
	ErrInvalidAmount = fmt.Errorf("invalid amount")

	// ErrInvalidCollateral indicates that provided collateral was invalid.
	ErrInvalidCollateral = fmt.Errorf("invalid collateral")

	// ErrInvalidPledge indicates that provided pledge was invalid.
	ErrInvalidPledge = fmt.Errorf("invalid pledge")

	// ErrInvalidBlockHeight indicates that the provided block height was invalid.
	ErrInvalidBlockHeight = fmt.Errorf("invalid block height")

	// ErrMissingDaemon is the error returned when trying to execute a command that requires the daemon to be started.
	ErrMissingDaemon = errors.New("daemon must be started before using this command")

	// ErrNoWalletAddresses indicates that there are no addresses in wallet to mine to.
	ErrNoWalletAddresses = fmt.Errorf("no addresses in wallet to mine to")
)
