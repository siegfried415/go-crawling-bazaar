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

	"github.com/siegfried415/gdf-rebuild/proto"
	"github.com/siegfried415/gdf-rebuild/types"

	//wyong, 20201014
	host "github.com/libp2p/go-libp2p-host" 
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

	//wyong, 20201014
	Host	host.Host 

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
