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
	"time"

	"github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/types"
)

// Config represents a sql-chain config.
type Config struct {
	DomainID      proto.DomainID
	ChainFilePrefix string
	DataFile        string

	Genesis *types.Block
	Period  time.Duration
	Tick    time.Duration

	MuxService *MuxService
	Peers      *proto.Peers
	Server     proto.NodeID

	Host	net.RoutedHost 

	// QueryTTL sets the unacknowledged query TTL in block periods.
	QueryTTL      int32
	BlockCacheTTL int32

	// DBAccount info
	TokenType         types.TokenType
	GasPrice          uint64
	UpdatePeriod      uint64
	LastBillingHeight int32
	IsolationLevel    int
}
