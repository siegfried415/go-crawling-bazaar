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

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	graphite "github.com/cyberdelia/go-metrics-graphite"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/siegfried415/gdf-rebuild/conf"

	/*
	"github.com/siegfried415/gdf-rebuild/crypto"
	"github.com/siegfried415/gdf-rebuild/metric"
	"github.com/siegfried415/gdf-rebuild/rpc"
	"github.com/siegfried415/gdf-rebuild/rpc/mux"

	"github.com/siegfried415/gdf-rebuild/utils"
	"github.com/siegfried415/gdf-rebuild/utils/log"
	_ "github.com/siegfried415/gdf-rebuild/utils/log/debug"
	"github.com/siegfried415/gdf-rebuild/utils/trace"
	"github.com/siegfried415/gdf-rebuild/frontera"
	*/

	//wyong, 20200903 
	"syscall" 
	"net/url" 
	"context" 
        cmds "github.com/ipfs/go-ipfs-cmds"
        cmdhttp "github.com/ipfs/go-ipfs-cmds/http"
	cli "github.com/ipfs/go-ipfs-cmds/cli"
	//adapter "github.com/siegfried415/gdf-rebuild/cql-minerd/adapter" 

	//wyong, 20200914 
	//dag "github.com/siegfried415/gdf-rebuild/cql-minerd/dag" 

)

const logo = `
   ______                                  __  _____ ____    __ 
  / ____/___ _   _____  ____  ____ _____  / /_/ ___// __ \  / / 
 / /   / __ \ | / / _ \/ __ \/ __  / __ \/ __/\__ \/ / / / / /
/ /___/ /_/ / |/ /  __/ / / / /_/ / / / / /_ ___/ / /_/ / / /___
\____/\____/|___/\___/_/ /_/\__,_/_/ /_/\__//____/\___\_\/_____/

   _____ ______   ___  ________   _______   ________     
  |\   _ \  _   \|\  \|\   ___  \|\  ___ \ |\   __  \    
  \ \  \\\__\ \  \ \  \ \  \\ \  \ \   __/|\ \  \|\  \   
   \ \  \\|__| \  \ \  \ \  \\ \  \ \  \_|/_\ \   _  _\  
    \ \  \    \ \  \ \  \ \  \\ \  \ \  \_|\ \ \  \\  \| 
     \ \__\    \ \__\ \__\ \__\\ \__\ \_______\ \__\\ _\ 
      \|__|     \|__|\|__|\|__| \|__|\|_______|\|__|\|__|
                                                       
`

var (
	version = "unknown"
)

var (
	// config
	configFile string
	genKeyPair bool
	metricLog  bool
	metricWeb  string

	// profile
	cpuProfile     string
	memProfile     string
	profileServer  string
	metricGraphite string
	traceFile      string

	// other
	noLogo      bool
	showVersion bool
	logLevel    string

	//wyong, 20200816
	adapterAddr string 
)

const name = `cql-minerd`
const desc = `CovenantSQL is a Distributed Database running on BlockChain`

func init() {
	//wyong, 20200816 
	flag.StringVar(&adapterAddr, "adapter", "", "adapter address ")

	flag.BoolVar(&noLogo, "nologo", false, "Do not print logo")
	flag.BoolVar(&metricLog, "metric-log", false, "Print metrics in log")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&genKeyPair, "gen-keypair", false, "Gen new key pair when no private key found")
	flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")

	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file path")

	flag.StringVar(&profileServer, "profile-server", "", "Profile server address, default not started")
	flag.StringVar(&cpuProfile, "cpu-profile", "", "Path to file for CPU profiling information")
	flag.StringVar(&memProfile, "mem-profile", "", "Path to file for memory profiling information")
	flag.StringVar(&metricGraphite, "metric-graphite-server", "", "Metric graphite server to push metrics")
	flag.StringVar(&metricWeb, "metric-web", "", "Address and port to get internal metrics")

	flag.StringVar(&traceFile, "trace-file", "", "Trace profile")
	flag.StringVar(&logLevel, "log-level", "", "Service log level")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "\n%s\n\n", desc)
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [arguments]\n", name)
		flag.PrintDefaults()
	}
}

func initLogs() {
	log.Infof("%#v starting, version %#v", name, version)
	log.Infof("%#v, target architecture is %#v, operating system target is %#v", runtime.Version(), runtime.GOARCH, runtime.GOOS)
	log.Infof("role: %#v", conf.RoleTag)
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


func main() {
	flag.Parse()
	// set random
	rand.Seed(time.Now().UnixNano())
	log.SetStringLevel(logLevel, log.InfoLevel)

	if showVersion {
		fmt.Printf("%v %v %v %v %v\n",
			name, version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		os.Exit(0)
	}

	configFile = utils.HomeDirExpand(configFile)

	flag.Visit(func(f *flag.Flag) {
		log.Infof("args %#v : %s", f.Name, f.Value)
	})


	var err error
	conf.GConf, err = conf.LoadConfig(configFile)
	if err != nil {
		log.WithField("config", configFile).WithError(err).Fatal("load config failed")
	}

	if conf.GConf.Miner == nil {
		log.Fatal("miner config does not exists")
	}
	if conf.GConf.Miner.ProvideServiceInterval.Seconds() <= 0 {
		log.Fatal("miner metric collect interval is invalid")
	}
	if conf.GConf.Miner.MaxReqTimeGap.Seconds() <= 0 {
		log.Fatal("miner request time gap is invalid")
	}
	if conf.GConf.Miner.DiskUsageInterval.Seconds() <= 0 {
		// set to default disk usage interval
		log.Warning("miner disk usage interval not provided, set to default 10 minutes")
		conf.GConf.Miner.DiskUsageInterval = time.Minute * 10
	}

	log.Debugf("config:\n%#v", conf.GConf)

	// init log
	initLogs()


	//wyong, 20200903 
	if adapterAddr == "" {

		log.Debugf("Main(100), adapterAddr==null \n")
		//parse the command path, arguments and options from the command line
		req, err := cli.Parse(context.TODO(), flag.Args(), os.Stdin, adapter.GetRoot())
		if err != nil {
			panic(err)
		}

		log.Debugf("Main(110)\n")
		req.Options["encoding"] = cmds.Text

		// create http rpc client
		client := cmdhttp.NewClient(conf.GConf.AdapterAddr)


		// create an emitter
		re, err := cli.NewResponseEmitter(os.Stdout, os.Stderr, req)
		if err != nil {
			panic(err)
		}

		log.Debugf("Main(120)\n")
		// send request to server
		err = client.Execute(req, re, nil)
		if err != nil {
			panic(err)
		}

		log.Debugf("Main(130)\n")
		os.Exit(0)
	}


	//wyong, 20200911 
	ctx := context.Background()

	if !noLogo {
		fmt.Print(logo)
	}

	log.Info("miner stopped")
}
