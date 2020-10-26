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

package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/ipfs/go-ipfs-cmdkit"
	"github.com/ipfs/go-ipfs-cmds"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/siegfried415/go-crawling-bazaar/net"
	env "github.com/siegfried415/go-crawling-bazaar/env" 
)

// swarmCmd contains swarm commands.
var swarmCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with other go-crawling-bazaar node",
		ShortDescription: `
'gcb swarm' is a tool to manipulate the libp2p swarm. The swarm is the
component that opens, listens for, and maintains connections to other
libp2p peers on the internet.
`,
	},
	Subcommands: map[string]*cmds.Command{
		"connect": swarmConnectCmd,
		"peers":   swarmPeersCmd,
	},
}

var swarmPeersCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "List peers with open connections.",
		ShortDescription: `
'go-filecoin swarm peers' lists the set of peers this node is connected to.
`,
	},
	Options: []cmdkit.Option{
		cmdkit.BoolOption("verbose", "v", "Display all extra information"),
		cmdkit.BoolOption("streams", "Also list information about open streams for each peer"),
		cmdkit.BoolOption("latency", "Also list information about latency to each peer"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		verbose, _ := req.Options["verbose"].(bool)
		latency, _ := req.Options["latency"].(bool)
		streams, _ := req.Options["streams"].(bool)

                e := cmdenv.(*env.Env)
		host := e.Host()
                out, err := host.Peers(req.Context, verbose, latency, streams)

		if err != nil {
			return err
		}

		return re.Emit(&out)
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, ci *net.SwarmConnInfos) error {
			pipfs := ma.ProtocolWithCode(ma.P_IPFS).Name
			for _, info := range ci.Peers {
				ids := fmt.Sprintf("/%s/%s", pipfs, info.Peer)
				if strings.HasSuffix(info.Addr, ids) {
					fmt.Fprintf(w, "%s", info.Addr) // nolint: errcheck
				} else {
					fmt.Fprintf(w, "%s%s", info.Addr, ids) // nolint: errcheck
				}
				if info.Latency != "" {
					fmt.Fprintf(w, " %s", info.Latency) // nolint: errcheck
				}
				fmt.Fprintln(w) // nolint: errcheck

				for _, s := range info.Streams {
					if s.Protocol == "" {
						s.Protocol = "<no protocol name>"
					}

					fmt.Fprintf(w, "  %s\n", s.Protocol) // nolint: errcheck
				}
			}

			return nil
		}),
	},
	Type: net.SwarmConnInfos{},
}

var swarmConnectCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Open connection to a given address.",
		ShortDescription: `
'gcb swarm connect' opens a new direct connection to a peer address.

The address format is a multiaddr:

gcb swarm connect /ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", true, true, "Address of peer to connect to.").EnableStdin(),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		e := cmdenv.(*env.Env)
		host := e.Host()

		addrInfo, err := net.PeerAddrToAddrInfo(req.Arguments[0])
		if err != nil {
			return err
		}

                err = host.Connect(req.Context, *addrInfo)
		if err != nil {
			return err
		}

		peerID := peer.ID(addrInfo.ID.Pretty())
		if err := re.Emit(peerID); err != nil {
			return err
		}

		return nil
	},
	Type: peer.ID(""),
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, result peer.ID) error {
			fmt.Fprintf(w, "connect %s success\n", result.Pretty()) // nolint: errcheck
			return nil
		}),
	},
}
