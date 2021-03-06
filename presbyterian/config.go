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

	net "github.com/siegfried415/go-crawling-bazaar/net"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	"github.com/siegfried415/go-crawling-bazaar/types"
)

// RunMode defines modes that a presbyterian can run as.
type RunMode int

const (
	// PBMode is the default and normal mode.
	PBMode RunMode = iota

	// APINodeMode makes the presbyterian behaviour like an API gateway. It becomes an API
	// node, who syncs data from the presbyterian network and exposes JSON-RPC API to users.
	APINodeMode
)

// Config is the main chain configuration.
type Config struct {
	Mode    RunMode
	Genesis *types.PBBlock

	DataFile string

	Host 	net.RoutedHost 

	Peers            *proto.Peers
	NodeID           proto.NodeID
	ConfirmThreshold float64

	Period time.Duration
	Tick   time.Duration

	BlockCacheSize int
}
