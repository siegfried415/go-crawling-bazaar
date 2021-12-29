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
