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

package presbyterian

import (
	"time"

	"github.com/siegfried415/gdf-rebuild/proto"

	//wyong, 20201015
	host "github.com/libp2p/go-libp2p-core/host"

	//wyong, 20201008 
	//rpc "github.com/siegfried415/gdf-rebuild/rpc/mux"

	"github.com/siegfried415/gdf-rebuild/types"
)

// RunMode defines modes that a bp can run as.
type RunMode int

const (
	// BPMode is the default and normal mode.
	BPMode RunMode = iota

	// APINodeMode makes the bp behaviour like an API gateway. It becomes an API
	// node, who syncs data from the bp network and exposes JSON-RPC API to users.
	APINodeMode
)

// Config is the main chain configuration.
type Config struct {
	Mode    RunMode
	Genesis *types.BPBlock

	DataFile string

	//wyong, 20201008 
	//Server *rpc.Server

	//wyong, 20201015
	Host 	host.Host 

	Peers            *proto.Peers
	NodeID           proto.NodeID
	ConfirmThreshold float64

	Period time.Duration
	Tick   time.Duration

	BlockCacheSize int
}
