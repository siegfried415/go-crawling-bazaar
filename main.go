package main

import (
	"context"
	"os"
	//"strconv"
	//"fmt"

	//logging "github.com/ipfs/go-log"
	//oldlogging "github.com/whyrusleeping/go-logging"

	"github.com/siegfried415/gdf-rebuild/commands"
	//"github.com/siegfried415/gdf-rebuild/metrics"

	//wyong, 20201215
	log "github.com/siegfried415/gdf-rebuild/utils/log"

)

func main() {
	log.Debugf("gdf-minerd/main(10)\n")

	//wyong, 20200922 
	// TODO: make configurable - this should be done via a command like go-ipfs
	// something like:
	//		`go-filecoin log level "system" "level"`
	// TODO: find a better home for this
	// TODO fix this in go-log 4 == INFO

	//n, err := strconv.Atoi(os.Getenv("GO_FILECOIN_LOG_LEVEL"))
	//if err != nil {
	//	n = 4
	//}

	//wyong, 20200922 
	//if os.Getenv("GO_FILECOIN_LOG_JSON") == "1" {
	//	oldlogging.SetFormatter(&metrics.JSONFormatter{})
	//}

	//logging.SetAllLoggers(oldlogging.Level(n))

	// wyong, 20200918 
	//logging.SetLogLevel("dht", "error")          // nolint: errcheck
	//logging.SetLogLevel("bitswap", "error")      // nolint: errcheck
	//logging.SetLogLevel("graphsync", "info")     // nolint: errcheck
	//logging.SetLogLevel("heartbeat", "error")    // nolint: errcheck
	//logging.SetLogLevel("blockservice", "error") // nolint: errcheck
	//logging.SetLogLevel("peerqueue", "error")    // nolint: errcheck
	//logging.SetLogLevel("swarm", "error")        // nolint: errcheck
	//logging.SetLogLevel("swarm2", "error")       // nolint: errcheck
	//logging.SetLogLevel("basichost", "error")    // nolint: errcheck
	//logging.SetLogLevel("dht_net", "error")      // nolint: errcheck
	//logging.SetLogLevel("pubsub", "error")       // nolint: errcheck
	//logging.SetLogLevel("relay", "error")        // nolint: errcheck

	// TODO implement help text like so:
	// https://github.com/ipfs/go-ipfs/blob/master/core/commands/root.go#L91
	// TODO don't panic if run without a command.
	code, _ := commands.Run(context.Background(), os.Args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}
