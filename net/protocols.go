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


package net

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/protocol"
)

// FilecoinDHT is creates a protocol for the filecoin DHT.
func FilecoinDHT(network string) protocol.ID {
	return protocol.ID(fmt.Sprintf("/fil/kad/%s", network))
}
