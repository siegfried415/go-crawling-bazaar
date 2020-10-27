package commands

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // nolint: golint
	"os"
	"os/signal"
	"syscall"
	"time"

	//wyong, 20201022 
	"path/filepath"


	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdhttp "github.com/ipfs/go-ipfs-cmds/http"
	writer "github.com/ipfs/go-log/writer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/pkg/errors"

	//"github.com/siegfried415/gdf-rebuild/clock"

	//wyong, 20201003 
	"github.com/siegfried415/gdf-rebuild/conf"

	//"github.com/siegfried415/gdf-rebuild/consensus"
	node "github.com/siegfried415/gdf-rebuild/node"
	"github.com/siegfried415/gdf-rebuild/paths"

	//wyong, 20201027 
	//"github.com/siegfried415/gdf-rebuild/repo"

	//wyong, 20201022 
	utils "github.com/siegfried415/gdf-rebuild/utils" 

	env "github.com/siegfried415/gdf-rebuild/env" 
)

var daemonCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Start a long-running daemon process",
	},
	Options: []cmdkit.Option{
		cmdkit.StringOption(SwarmAddress, "multiaddress to listen on for filecoin network connections"),
		cmdkit.StringOption(SwarmPublicRelayAddress, "public multiaddress for routing circuit relay traffic.  Necessary for relay nodes to provide this if they are not publically dialable"),
		cmdkit.BoolOption(OfflineMode, "start the node without networking"),
		cmdkit.BoolOption(ELStdout),
		cmdkit.BoolOption(IsRelay, "advertise and allow filecoin network traffic to be relayed through this node"),
		//cmdkit.StringOption(BlockTime, "time a node waits before trying to mine the next block").WithDefault(consensus.DefaultBlockTime.String()),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		return daemonRun(req, re)
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.Encoders[cmds.Text],
	},
}

func daemonRun(req *cmds.Request, re cmds.ResponseEmitter) error {
	var (
		err error
	)

	repoDir, _ := req.Options[OptionRepoDir].(string)
	repoDir, err = paths.GetRepoPath(repoDir)
	if err != nil {
		return err
	}

	if repoDir == "" {
		repoDir = utils.HomeDirExpand("~/.gdf")
	}

	//wyong, 20201027 
	//repo, err := repo.OpenFSRepo(repoDir, repo.Version)
	//if err != nil {
	//	return err
	//}

	configFile := filepath.Join(repoDir, "config.yaml")
        conf.GConf, err = conf.LoadConfig(configFile)
        if err != nil {
                //log.WithField("config", configFile).WithError(err).Fatal("load config failed")
        }

        if conf.GConf.Miner == nil {
                //log.Fatal("miner config does not exists")
        }
        if conf.GConf.Miner.ProvideServiceInterval.Seconds() <= 0 {
                //log.Fatal("miner metric collect interval is invalid")
        }
        if conf.GConf.Miner.MaxReqTimeGap.Seconds() <= 0 {
                //log.Fatal("miner request time gap is invalid")
        }
        if conf.GConf.Miner.DiskUsageInterval.Seconds() <= 0 {
                // set to default disk usage interval
                //log.Warning("miner disk usage interval not provided, set to default 10 minutes")
                conf.GConf.Miner.DiskUsageInterval = time.Minute * 10
        }

        //log.Debugf("config:\n%#v", conf.GConf)


	// second highest precedence is env vars.
	if envAPI := os.Getenv("FIL_API"); envAPI != "" {
		conf.GConf.API.Address = envAPI
	}

	// highest precedence is cmd line flag.
	if flagAPI, ok := req.Options[OptionAPI].(string); ok && flagAPI != "" {
		conf.GConf.API.Address = flagAPI
	}

	if swarmAddress, ok := req.Options[SwarmAddress].(string); ok && swarmAddress != "" {
		conf.GConf.Swarm.Address = swarmAddress
	}

	if publicRelayAddress, ok := req.Options[SwarmPublicRelayAddress].(string); ok && publicRelayAddress != "" {
		conf.GConf.Swarm.PublicRelayAddress = publicRelayAddress
	}

	//todo, wyong, 20201003 
	//opts, err := node.OptionsFromRepo(rep)
	//if err != nil {
	//	return err
	//}
	var opts []node.BuilderOpt

	if offlineMode, ok := req.Options[OfflineMode].(bool); ok {
		opts = append(opts, node.OfflineMode(offlineMode))
	}

	if isRelay, ok := req.Options[IsRelay].(bool); ok && isRelay {
		opts = append(opts, node.IsRelay())
	}

	// wyong, 20200922 
	//durStr, ok := req.Options[BlockTime].(string)
	//if !ok {
	//	return errors.New("Bad block time passed")
	//}

	//blockTime, err := time.ParseDuration(durStr)
	//if err != nil {
	//	return errors.Wrap(err, "Bad block time passed")
	//}
	//opts = append(opts, node.BlockTime(blockTime))
	//opts = append(opts, node.ClockConfigOption(clock.NewSystemClock()))


	// Instantiate the node.
	n, err := node.New(req.Context, repoDir,  opts...)
	if err != nil {
		return err
	}

	if n.OfflineMode {
		_ = re.Emit("Filecoin node running in offline mode (libp2p is disabled)\n")
	} else {
		_ = re.Emit(fmt.Sprintf("My peer ID is %s\n", n.Host.ID().Pretty()))
		for _, a := range n.Host.Addrs() {
			_ = re.Emit(fmt.Sprintf("Swarm listening on: %s\n", a))
		}
	}

	if _, ok := req.Options[ELStdout].(bool); ok {
		writer.WriterGroup.AddWriter(os.Stdout)
	}

	// Start the node.
	if err := n.Start(req.Context); err != nil {
		return err
	}
	defer n.Stop(req.Context)

	// Run API server around the node.
	ready := make(chan interface{}, 1)
	go func() {
		<-ready

		//wyong, 20201027 
		_ = re.Emit(fmt.Sprintf("API server listening on %s\n", conf.GConf.API.Address))
	}()

	var terminate = make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(terminate)

	// The request is expected to remain open so the daemon uses the request context.
	// Pass a new context here if the flow changes such that the command should exit while leaving
	// a forked deamon running.
	return RunAPIAndWait(req.Context, n, repoDir,  conf.GConf.API, ready, terminate)
}

/*
func getRepo(req *cmds.Request) (repo.Repo, error) {
	repoDir, _ := req.Options[OptionRepoDir].(string)
	repoDir, err := paths.GetRepoPath(repoDir)
	if err != nil {
		return nil, err
	}
	return repo.OpenFSRepo(repoDir, repo.Version)
}
*/


// RunAPIAndWait starts an API server and waits for it to finish.
// The `ready` channel is closed when the server is running and its API address has been
// saved to the node's repo.
// A message sent to or closure of the `terminate` channel signals the server to stop.
func RunAPIAndWait(ctx context.Context, nd *node.Node, repoDir string, config *conf.APIConfig, ready chan interface{}, terminate chan os.Signal) error {

	// wyong, 20200921 
	//servenv := &Env{
	//	blockMiningAPI: nd.BlockMining.BlockMiningAPI,
	//	ctx:            ctx,
	//	inspectorAPI:   NewInspectorAPI(nd.Repo),
	//	porcelainAPI:   nd.PorcelainAPI,
	//
	//	//wyong, 20200410 
	//	//retrievalAPI:   nd.RetrievalProtocol.RetrievalAPI,
	//	//storageAPI:     nd.StorageProtocol.StorageAPI,
	//}

	//wyong, 20201022 
	servenv := env.NewClientEnv(ctx, nd.Host,  nd.Frontera, nd.DAG, nd.Net)

	cfg := cmdhttp.NewServerConfig()
	cfg.APIPath = APIPrefix
	cfg.SetAllowedOrigins(config.AccessControlAllowOrigin...)
	cfg.SetAllowedMethods(config.AccessControlAllowMethods...)
	cfg.SetAllowCredentials(config.AccessControlAllowCredentials)

	maddr, err := ma.NewMultiaddr(config.Address)
	if err != nil {
		return err
	}

	// Listen on the configured address in order to bind the port number in case it has
	// been configured as zero (i.e. OS-provided)
	apiListener, err := manet.Listen(maddr)
	if err != nil {
		return err
	}

	handler := http.NewServeMux()
	handler.Handle("/debug/pprof/", http.DefaultServeMux)
	handler.Handle(APIPrefix+"/", cmdhttp.NewHandler(servenv, rootCmdDaemon, cfg))

	apiserv := http.Server{
		Handler: handler,
	}

	go func() {
		err := apiserv.Serve(manet.NetListener(apiListener))
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Write the resolved API address to the repo
	config.Address = apiListener.Multiaddr().String()
	if err := conf.SetAPIAddr(repoDir, config.Address); err != nil {
		return errors.Wrap(err, "Could not save API address to repo")
	}
	// Signal that the sever has started and then wait for a signal to stop.
	close(ready)
	received := <-terminate
	if received != nil {
		fmt.Println("Received signal", received)
	}
	fmt.Println("Shutting down...")

	// Allow a grace period for clean shutdown.
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if err := apiserv.Shutdown(ctx); err != nil {
		fmt.Println("Error shutting down API server:", err)
	}

	return nil
}
