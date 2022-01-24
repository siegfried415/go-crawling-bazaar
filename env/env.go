/*
 * Copyright 2022 https://github.com/siegfried415
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

package env

import (
	"context"

	"github.com/ipfs/go-ipfs-cmds"

	dag "github.com/siegfried415/go-crawling-bazaar/dag" 
	"github.com/siegfried415/go-crawling-bazaar/frontera" 
	net "github.com/siegfried415/go-crawling-bazaar/net" 
	"github.com/siegfried415/go-crawling-bazaar/proto" 
)

// Env is the environment for command API handlers.
type Env struct {
	ctx            context.Context
        role proto.ServerRole
	host	net.RoutedHost
	frontera *frontera.Frontera 
	dag *dag.DAG 
}

var _ cmds.Environment = (*Env)(nil)

// NewClientEnv returns a new environment for command API clients.
// This environment lacks direct access to any internal APIs.
func NewClientEnv(ctx context.Context, role proto.ServerRole, host net.RoutedHost,  frontera *frontera.Frontera, dag *dag.DAG ) *Env {
	return &Env{
		ctx	: ctx,
		role	: role, 
		host 	: host, 
		frontera : frontera ,
		dag 	: dag , 
	}
}

// Context returns the context of the environment.
func (ce *Env) Context() context.Context {
	return ce.ctx
}

func (ce *Env) Role() proto.ServerRole {
	return ce.role 
}

func (ce *Env) Host() net.RoutedHost {
	return ce.host 
}

func (ce *Env) Frontera() *frontera.Frontera {
	return ce.frontera 
}

func (ce *Env) DAG() *dag.DAG {
	return ce.dag 
}
