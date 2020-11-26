package commands

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"syscall"

	"github.com/ipfs/go-ipfs-cmdkit"
	"github.com/ipfs/go-ipfs-cmds"
	"github.com/ipfs/go-ipfs-cmds/cli"
	cmdhttp "github.com/ipfs/go-ipfs-cmds/http"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multiaddr-net"
	"github.com/pkg/errors"

	"github.com/siegfried415/gdf-rebuild/paths"
	"github.com/siegfried415/gdf-rebuild/conf"
	//"github.com/siegfried415/gdf-rebuild/types"

	env "github.com/siegfried415/gdf-rebuild/env" 
	gdfnet "github.com/siegfried415/gdf-rebuild/net" 

	//wyong, 20201126 
	"github.com/siegfried415/gdf-rebuild/proto" 
)

const (
	// OptionAPI is the name of the option for specifying the api port.
	OptionAPI = "apiaddr"

	// OptionRepoDir is the name of the option for specifying the directory of the repo.
	OptionRepoDir = "repodir"

	//wyong, 20201028
	OptionRole = "role"

	// OptionSectorDir is the name of the option for specifying the directory into which staged and sealed sectors will be written.
	OptionSectorDir = "sectordir"

	// APIPrefix is the prefix for the http version of the api.
	APIPrefix = "/api"

	// OfflineMode tells us if we should try to connect this Filecoin node to the network
	OfflineMode = "offline"

	// ELStdout tells the daemon to write event logs to stdout.
	ELStdout = "elstdout"

	// AutoSealIntervalSeconds configures the daemon to check for and seal any staged sectors on an interval.
	AutoSealIntervalSeconds = "auto-seal-interval-seconds"

	// SwarmAddress is the multiaddr for this Filecoin node
	SwarmAddress = "swarmlisten"

	// SwarmPublicRelayAddress is a public address that the filecoin node
	// will listen on if it is operating as a relay.  We use this to specify
	// the public ip:port of a relay node that is sitting behind a static
	// NAT mapping.
	SwarmPublicRelayAddress = "swarmrelaypublic"

	// BlockTime is the duration string of the block time the daemon will
	// run with.  TODO: this should eventually be more explicitly grouped
	// with testing as we won't be able to set blocktime in production.
	BlockTime = "block-time"

	//wyong, 20201125 
	//wyong, 20201030 
	//AdapterAddress = "adapter"

	// PeerKeyFile is the path of file containing key to use for new nodes libp2p identity
	PeerKeyFile = "peerkeyfile"

	// WithMiner when set, creates a custom genesis block with a pre generated miner account, requires to run the daemon using dev mode (--dev)
	WithMiner = "with-miner"

	// DefaultAddress when set, sets the daemons's default address to the provided address
	DefaultAddress = "default-address"

	// GenesisFile is the path of file containing archive of genesis block DAG data
	GenesisFile = "genesisfile"

	// DevnetStaging populates config bootstrap addrs with the dns multiaddrs of the staging devnet and other staging devnet specific bootstrap parameters
	DevnetStaging = "devnet-staging"

	// DevnetNightly populates config bootstrap addrs with the dns multiaddrs of the nightly devnet and other nightly devnet specific bootstrap parameters
	DevnetNightly = "devnet-nightly"

	// DevnetUser populates config bootstrap addrs with the dns multiaddrs of the user devnet and other user devnet specific bootstrap parameters
	DevnetUser = "devnet-user"

	// IsRelay when set causes the the daemon to provide libp2p relay
	// services allowing other filecoin nodes behind NATs to talk directly.
	IsRelay = "is-relay"
)

// command object for the local cli
var rootCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "A decentralized storage network",
		Subcommands: `
START RUNNING FILECOIN
  go-filecoin init                   - Initialize a filecoin repo
  go-filecoin config <key> [<value>] - Get and set filecoin config values
  go-filecoin daemon                 - Start a long-running daemon process
  go-filecoin wallet                 - Manage your filecoin wallets
  go-filecoin address                - Interact with addresses

STORE AND RETRIEVE DATA
  go-filecoin client                 - Make deals, store data, retrieve data
  go-filecoin retrieval-client       - Manage retrieval client operations

MINE
  go-filecoin miner                  - Manage a single miner actor
  go-filecoin mining                 - Manage all mining operations for a node

VIEW DATA STRUCTURES
  go-filecoin chain                  - Inspect the filecoin blockchain
  go-filecoin dag                    - Interact with IPLD DAG objects
  go-filecoin deals                  - Manage deals made by or with this node
  go-filecoin show                   - Get human-readable representations of filecoin objects

NETWORK COMMANDS
  go-filecoin bitswap                - Explore libp2p bitswap
  go-filecoin bootstrap              - Interact with bootstrap addresses
  go-filecoin dht                    - Interact with the dht
  go-filecoin id                     - Show info about the network peers
  go-filecoin ping <peer ID>...      - Send echo request packets to p2p network members
  go-filecoin swarm                  - Interact with the swarm
  go-filecoin stats                  - Monitor statistics on your network usage

ACTOR COMMANDS
  go-filecoin actor                  - Interact with actors. Actors are built-in smart contracts
  go-filecoin paych                  - Payment channel operations

MESSAGE COMMANDS
  go-filecoin message                - Manage messages
  go-filecoin mpool                  - Manage the message pool
  go-filecoin outbox                 - Manage the outbound message queue

TOOL COMMANDS
  go-filecoin inspect                - Show info about the go-filecoin node
  go-filecoin leb128                 - Leb128 cli encode/decode
  go-filecoin log                    - Interact with the daemon event log output
  go-filecoin protocol               - Show protocol parameter details
  go-filecoin version                - Show go-filecoin version information
`,
	},
	Options: []cmdkit.Option{
		cmdkit.StringOption(OptionAPI, "set the api port to use"),
		cmdkit.StringOption(OptionRepoDir, "set the repo directory, defaults to ~/.filecoin/repo"),
		cmds.OptionEncodingType,
		cmdkit.BoolOption("help", "Show the full command help text."),
		cmdkit.BoolOption("h", "Show a short version of the command help text."),

		//wyong, 20201002 
        	cmdkit.BoolOption("with-password", "Enter the passphrase for private.key") , 
        	cmdkit.StringOption("password", "Passphrase for encrypting private.key (NOT SAFE, for debug or script only)"),
		cmdkit.StringOption("log-level", "Console log level: trace debug info warning error fatal panic"), 

		//wyong, 20201007
		cmdkit.StringOption(OptionRole, "The role of this node."), 

	},
	Subcommands: make(map[string]*cmds.Command),
}

// command object for the daemon
var rootCmdDaemon = &cmds.Command{
	Subcommands: make(map[string]*cmds.Command),
}

// all top level commands, not available to daemon
var rootSubcmdsLocal = map[string]*cmds.Command{
	"daemon":  daemonCmd,
	"init":    initCmd,
	"version": versionCmd,
	//"leb128":  leb128Cmd,
}

// all top level commands, available on daemon. set during init() to avoid configuration loops.
var rootSubcmdsDaemon = map[string]*cmds.Command{
	"domain":            domainCmd,
	"urlrequest":            urlRequestCmd,

	//wyong, 20200819 
	"urlgraph":            urlGraphCmd,

	//wyong, 20200910
	"dag" :		     dagCmd, 

	//"actor":            actorCmd,
	//"address":          addrsCmd,
	//"bitswap":          bitswapCmd,
	//"bootstrap":        bootstrapCmd,
	//"chain":            chainCmd,
	//"config":           configCmd,

	//wyong, 20200410 
	//"client":           clientCmd,


	//wyong, 20201007
	"bidding":	biddingCmd,
	"bid":		bidCmd, 

	
	//wyong, 20200410 
	//"deals":            dealsCmd,

	//"dht":              dhtCmd,
	"id":               idCmd,
	//"inspect":          inspectCmd,
	//"leb128":           leb128Cmd,
	//"log":              logCmd,
	//"message":          msgCmd,
	//"miner":            minerCmd,

	//wyong, 20200416 
	//wyong, 20190813
	//"frontera":	    fronteraCmd, 
	//"crawlingmarket":	    crawlingmarketCmd, 
	//"urlgraph":	    urlgraphCmd, 

	//wyong, 20191008
	//"urlstore":	urlstoreCmd, 

	//"mining":           miningCmd,
	//"mpool":            mpoolCmd,
	//"outbox":           outboxCmd,

	//wyong, 20200412 
	//"paych":            paymentChannelCmd,

	//"ping":             pingCmd,
	//"protocol":         protocolCmd,

	//wyong, 20200410 
	//"retrieval-client": retrievalClientCmd,

	//"show":             showCmd,
	//"stats":            statsCmd,
	"swarm":            swarmCmd,
	//"wallet":           walletCmd,
	"version":          versionCmd,
}

func init() {
	for k, v := range rootSubcmdsLocal {
		rootCmd.Subcommands[k] = v
	}

	for k, v := range rootSubcmdsDaemon {
		rootCmd.Subcommands[k] = v
		rootCmdDaemon.Subcommands[k] = v
	}
}

// Run processes the arguments and stdin
func Run(ctx context.Context, args []string, stdin, stdout, stderr *os.File) (int, error) {
	fmt.Printf("Run(10)\n") 
	err := cli.Run(ctx, rootCmd, args, stdin, stdout, stderr, buildEnv, makeExecutor)
	if err == nil {
		fmt.Printf("Run(15)\n") 
		return 0, nil
	}
	fmt.Printf("Run(20)\n") 
	if exerr, ok := err.(cli.ExitError); ok {
		fmt.Printf("Run(25)\n") 
		return int(exerr), nil
	}
	fmt.Printf("Run(30)\n") 
	return 1, err
}

func buildEnv(ctx context.Context, _ *cmds.Request) (cmds.Environment, error) {
	fmt.Printf("buildEnv(10)\n") 
	return env.NewClientEnv(ctx, proto.Unknown, gdfnet.RoutedHost{}, nil, nil ), nil
}

type executor struct {
	api  string
	exec cmds.Executor
}

func (e *executor) Execute(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
	fmt.Printf("executor/Execute(10)\n") 
	if e.api == "" {
		fmt.Printf("executor/Execute(15)\n") 
		return e.exec.Execute(req, re, env)
	}

	fmt.Printf("executor/Execute(20)\n") 
	client := cmdhttp.NewClient(e.api, cmdhttp.ClientWithAPIPrefix(APIPrefix))

	fmt.Printf("executor/Execute(30)\n") 
	res, err := client.Send(req)
	if err != nil {
		fmt.Printf("executor/Execute(35)\n") 
		if isConnectionRefused(err) {
			fmt.Printf("executor/Execute(37)\n") 
			return cmdkit.Errorf(cmdkit.ErrFatal, "Connection Refused. Is the daemon running?")
		}
		fmt.Printf("executor/Execute(39)\n") 
		return cmdkit.Errorf(cmdkit.ErrFatal, err.Error())
	}

	fmt.Printf("executor/Execute(40)\n") 
	// copy received result into cli emitter
	err = cmds.Copy(re, res)
	if err != nil {
		fmt.Printf("executor/Execute(45)\n") 
		return cmdkit.Errorf(cmdkit.ErrFatal|cmdkit.ErrNormal, err.Error())
	}
	fmt.Printf("executor/Execute(50)\n") 
	return nil
}

func makeExecutor(req *cmds.Request, env interface{}) (cmds.Executor, error) {
	fmt.Printf("makeExecutor(10)\n") 
	var api string
	isDaemonRequired := requiresDaemon(req)
	if isDaemonRequired {
		fmt.Printf("makeExecutor(20)\n") 
		var err error
		maddr, err := getAPIAddress(req)
		if err != nil {
			fmt.Printf("makeExecutor(25)\n") 
			return nil, err
		}

		_, api, err = manet.DialArgs(maddr)
		if err != nil {
			fmt.Printf("getAPIAddress(75)\n") 
			return nil, errors.Wrap(err, fmt.Sprintf("unable to dial API endpoint address %s", maddr))
		}

		fmt.Printf("makeExecutor(30)\n") 
		if api == "" {
			fmt.Printf("makeExecutor(35)\n") 
			return nil, ErrMissingDaemon
		}
	}

	fmt.Printf("makeExecutor(40)\n") 
	return &executor{
		api:  api,
		exec: cmds.NewExecutor(rootCmd),
	}, nil

}

func getAPIAddress(req *cmds.Request) (ma.Multiaddr, error) {
	fmt.Printf("getAPIAddress(10)\n") 
	var rawAddr string
	var err error
	// second highest precedence is env vars.
	if envapi := os.Getenv("FIL_API"); envapi != "" {
		fmt.Printf("getAPIAddress(15)\n") 
		rawAddr = envapi
	}

	fmt.Printf("getAPIAddress(20)\n") 
	// first highest precedence is cmd flag.
	if apiAddress, ok := req.Options[OptionAPI].(string); ok && apiAddress != "" {
		fmt.Printf("getAPIAddress(25)\n") 
		rawAddr = apiAddress
	}

	fmt.Printf("getAPIAddress(30)\n") 

	// we will read the api file if no other option is given.
	if len(rawAddr) == 0 {
		fmt.Printf("getAPIAddress(40)\n") 
		repoDir, _ := req.Options[OptionRepoDir].(string)
		repoDir, err = paths.GetRepoPath(repoDir)
		if err != nil {
			fmt.Printf("getAPIAddress(45)\n") 
			return nil, err
		}

		fmt.Printf("getAPIAddress(50), repoDir=%s\n", repoDir ) 
		rawAddr, err = conf.APIAddrFromRepoPath(repoDir)
		if err != nil {
			fmt.Printf("getAPIAddress(55)\n") 
			return nil, errors.Wrap(err, "can't find API endpoint address in environment, command-line, or local repo (is the daemon running?)")
		}
	}

	fmt.Printf("getAPIAddress(60), rawAddr=%s\n", rawAddr ) 
	maddr, err := ma.NewMultiaddr(rawAddr)
	if err != nil {
		fmt.Printf("getAPIAddress(65)\n") 
		return nil, errors.Wrap(err, fmt.Sprintf("unable to convert API endpoint address %s to a multiaddr", rawAddr))
	}

	//fmt.Printf("getAPIAddress(70)\n") 
	//_, host, err := manet.DialArgs(maddr)
	//if err != nil {
	//	fmt.Printf("getAPIAddress(75)\n") 
	//	return "", errors.Wrap(err, fmt.Sprintf("unable to dial API endpoint address %s", maddr))
	//}

	fmt.Printf("getAPIAddress(80)\n") 
	return maddr, nil
}

func requiresDaemon(req *cmds.Request) bool {
	for cmd := range rootSubcmdsLocal {
		if len(req.Path) > 0 && req.Path[0] == cmd {
			return false
		}
	}
	return true
}

func isConnectionRefused(err error) bool {
	urlErr, ok := err.(*url.Error)
	if !ok {
		return false
	}

	opErr, ok := urlErr.Err.(*net.OpError)
	if !ok {
		return false
	}

	syscallErr, ok := opErr.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	return syscallErr.Err == syscall.ECONNREFUSED
}

var priceOption = cmdkit.StringOption("gas-price", "Price (FIL e.g. 0.00013) to pay for each GasUnit consumed mining this message")
var limitOption = cmdkit.Uint64Option("gas-limit", "Maximum GasUnits this message is allowed to consume")
var previewOption = cmdkit.BoolOption("preview", "Preview the Gas cost of this command without actually executing it")

/*
func parseGasOptions(req *cmds.Request) (types.AttoFIL, types.GasUnits, bool, error) {
	priceOption := req.Options["gas-price"]
	if priceOption == nil {
		return types.ZeroAttoFIL, types.NewGasUnits(0), false, errors.New("gas-price option is required")
	}

	price, ok := types.NewAttoFILFromFILString(priceOption.(string))
	if !ok {
		return types.ZeroAttoFIL, types.NewGasUnits(0), false, errors.New("invalid gas price (specify FIL as a decimal number)")
	}

	limitOption := req.Options["gas-limit"]
	if limitOption == nil {
		return types.ZeroAttoFIL, types.NewGasUnits(0), false, errors.New("gas-limit option is required")
	}

	gasLimitInt, ok := limitOption.(uint64)
	if !ok {
		msg := fmt.Sprintf("invalid gas limit: %s", limitOption)
		return types.ZeroAttoFIL, types.NewGasUnits(0), false, errors.New(msg)
	}

	preview, _ := req.Options["preview"].(bool)

	return price, types.NewGasUnits(gasLimitInt), preview, nil
}
*/
