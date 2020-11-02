package env

import (
	"context"

	"github.com/ipfs/go-ipfs-cmds"

	host "github.com/libp2p/go-libp2p-core/host"

	//wyong, 20200731 
	"github.com/siegfried415/gdf-rebuild/frontera" 
	
	//wyong, 20200914
	dag "github.com/siegfried415/gdf-rebuild/dag" 
	net "github.com/siegfried415/gdf-rebuild/net" 
)

// Env is the environment for command API handlers.
type Env struct {
	ctx            context.Context

	//wyong, 20201022 
	host	host.Host

	//wyong, 20200723 
	frontera *frontera.Frontera 

	//wyong, 20200908 
	dag *dag.DAG 

	//wyong, 20200912 
	network *net.Network 
}

var _ cmds.Environment = (*Env)(nil)

// NewClientEnv returns a new environment for command API clients.
// This environment lacks direct access to any internal APIs.
func NewClientEnv(ctx context.Context, host host.Host,  frontera *frontera.Frontera, dag *dag.DAG, network *net.Network ) *Env {
	return &Env{
		ctx	: ctx,
		host 	: host, //wyong, 20201022 
		frontera : frontera ,
		dag 	: dag , 
		network : network , 
	}
}

// Context returns the context of the environment.
func (ce *Env) Context() context.Context {
	return ce.ctx
}

//wyong, 20201022 
func (ce *Env) Host() host.Host {
	return ce.host 
}

//wyong, 20200723 
func (ce *Env) Frontera() *frontera.Frontera {
	//ce := env.(*Env)
	return ce.frontera 
}

//wyong, 20200908 
func (ce *Env) DAG() *dag.DAG {
	//ce := env.(*Env)
	return ce.dag 
}

//wyong, 20200912
func (ce *Env) Network() *net.Network {
	//ce := env.(*Env)
	return ce.network
}