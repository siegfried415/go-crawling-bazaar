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

package commands

import (
	//"errors"
	"fmt"
	//"io"
	//"time"
	"strconv" 
	"context" 


	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	
	client "github.com/siegfried415/go-crawling-bazaar/client" 
	env "github.com/siegfried415/go-crawling-bazaar/env"  
	log "github.com/siegfried415/go-crawling-bazaar/utils/log"  

)

const(
	waitTxConfirmationOptionName = "wait" 
)


var domainCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Create or drop domain group.",
	},

	Subcommands: map[string]*cmds.Command{
		"create": DomainCreateCmd,
		"drop": DomainDropCmd,	
	},
}

var DomainCreateCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Create a domain group.",

		ShortDescription: `
Create a domain group.
`,
		LongDescription: `
Create command creates a go-crawling-bazaar domain group by meta params. The meta info must include
node count.
Examples:

    gcb domain create "http://www.foo.com" 2

Since go-crawling-bazaar is built on top of blockchains, you may want to wait for the transaction
confirmation before the creation takes effect.

    gcb domain create -wait-tx-confirm "http://www.foo.com" 2

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("name", true, false, "url of the domain."),
		cmdkit.StringArg("count", true, false, "how many miners need to create the domain."),
	},

	Options: []cmdkit.Option{
		cmdkit.BoolOption("wait-tx-confirm", "If wait for the creation completed.").WithDefault(true),

	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		domain_name := req.Arguments[0]

		nodeCnt, err := strconv.ParseInt(req.Arguments[1], 10, 32) 
		if err != nil {
			return fmt.Errorf("unsupported nodeCnt syntax: %s", req.Arguments[1])
		}


		e := cmdenv.(*env.Env)
		host := e.Host()

		var meta = client.ResourceMeta{}
		meta.Node = uint16(nodeCnt)
		meta.Domain = domain_name 

		txHash, err := client.CreateDomain(host, meta)
		if err != nil {
			log.Error("create database failed, err= %s", err )
			return err 
		}

		waitTxConfirmation, _ := req.Options["wait-tx-confirm"].(bool)
		if waitTxConfirmation {
			log.Debugf("DomainCreateCmd, wait presbyterian ... \n") 
			err = wait(host, txHash)
			if err != nil {
				log.WithError(err).Error("create database failed durating presbyterian creation")
				return err 
			}

			var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
			defer cancel()
			err = client.WaitDomainCreation(ctx, host, domain_name )
			if err != nil {
				log.WithError(err).Error("create database failed durating miner creation")
				return err 
			}
		}

		return cmds.EmitOnce(res, 0)
	},

}

var DomainDropCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Drop a domain group.",

		ShortDescription: `
Drop a domain group.
`,
		LongDescription: `
Drop command drop a go-crawling-bazaar domain group. 
Examples:

    gcb domain drop -name "http://www.foo.com" 

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("name", true, false, "url of the domain group to be droped.").EnableStdin(),
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		dsn := req.Arguments[0]

		e := cmdenv.(*env.Env)
		host := e.Host()

		// drop database
		if _, err := client.ParseDSN(dsn); err != nil {
			// not a dsn/dbid
			log.WithField("db", dsn).WithError(err).Error("not a valid dsn")
			return err 
		}

		txHash, err := client.Drop(host, dsn)
		if err != nil {
			// drop database failed
			log.WithField("db", dsn).WithError(err).Error("drop database failed")
			return err 
		}

		//err = client.wait(txHash)
		//if err != nil {
		//	log.WithField("db", dsn).WithError(err).Error("drop database failed")
		//	return
		//}

		waitTxConfirmation, _ := req.Options[waitTxConfirmationOptionName].(bool)
		if waitTxConfirmation {
			err = wait(host, txHash)
			if err != nil {
				log.WithError(err).Error("drop database failed ")
				return err 
			}
			log.Debugf("\nThe domain is droped by presbyterian\n")
		}

		return cmds.EmitOnce(res, 0)
	},

}
