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
        "encoding/json"
	"errors" 
	"fmt" 
        "io"
	"io/ioutil" 

	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	cmds "github.com/ipfs/go-ipfs-cmds"
        "github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs-files"
	"github.com/mfonda/simhash"

	env "github.com/siegfried415/go-crawling-bazaar/env" 
	//log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
	proto "github.com/siegfried415/go-crawling-bazaar/proto" 
)

var dagCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with IPLD DAG objects.",
	},
	Subcommands: map[string]*cmds.Command{
		"get": 		DagGetCmd,
		"cat":          DagCatCmd,
		"import":       DagImportDataCmd,
	},
}

var DagGetCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Get a DAG node by its CID",
		ShortDescription: `
Get a DAG node by ite CID. 
`,
		LongDescription: `
Get a DAG node by ite CID. 

Examples:

  > gcb dag get #####################################
    (############################ is the cid of content of http://www.foo.com/index.html)

`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("ref", true, false, "CID of object to get"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		c, err := cid.Decode(req.Arguments[0])
		if err != nil {
			return fmt.Errorf("cid syntax error: %s", c )
		}

		e := cmdenv.(*env.Env)
		out, err := e.DAG().GetNode(req.Context, c.String()) 
		if err != nil {
			return err
		}

		return re.Emit(out)
	},
}

type Page struct {
	Url string
	Body string 
}

var DagCatCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Read out data stored on the DAG",
		ShortDescription: `
Prints data stored on DAG by ite CID. 
`,
		LongDescription: `
Prints data from the DAG network specified with a given CID to stdout. The
only argument should be the CID to return. 

Examples:

  > gcb dag cat parent_url url  #####################################
    (############################ is the cid of content of http://www.foo.com/index.html)

`,
	},
	Arguments: []cmdkit.Argument{
                cmdkit.StringArg("parent", true, false, "the parent url to be get from crawling bazaar"),
                cmdkit.StringArg("url", true, false, "the url to be get from crawling bazaar"),

		cmdkit.StringArg("cid", true, false, "CID of url content"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {

		e := cmdenv.(*env.Env)
		role := e.Role() 
		
                //parenturl := req.Arguments[0]
		var parenturl string 
		if err := json.Unmarshal([]byte(req.Arguments[0]), &parenturl); err != nil {
			return err
		}

                //url := req.Arguments[1]
		var url string 
		if err := json.Unmarshal([]byte(req.Arguments[1]), &url); err != nil {
			return err
		}

		//get domain of this url request 
		var domain string 
		if parenturl != "" { 
			domain, _ = domainForUrl(parenturl) 
		}

		if domain == "" {
			if url != "" { 
				var err error 
				domain, err = domainForUrl(url) 
				if err != nil {
					return err 
				}
			}
		}

		if domain == "" {
			return errors.New("can't get domain of this url request!")
		}

		switch role {
		case proto.Client : 
			host := e.Host()

			conn, err := getConn(host, "DAG.Cat", domain )
			if err != nil {
				return err 
			}
			defer conn.Close()

			result, err := conn.DagCat(req.Context, req.Arguments[2]) 
			if err != nil {
				return err  
			}

			return re.Emit(string(result)) 

		case proto.Leader:
			fallthrough 
		case proto.Follower: 
			return nil 

		case proto.Miner: 
			c, err := cid.Decode(req.Arguments[2])
			if err != nil {
				return err
			}

			dr, err := e.DAG().Cat(req.Context, c) 
			if err != nil {
				return err
			}

			return re.Emit(dr)

		case proto.Unknown: 
			return nil 
		}

		return nil 
	},
}

type DagImportDataResult struct {
        Cid cid.Cid  
        Hash uint64 
}

var DagImportDataCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Import data into the DAG",
		ShortDescription: `
Imports data previously exported with the client cat command into the DAG.
`,
		LongDescription: `
Prints data from the DAG network specified with a given CID to stdout. The
only argument should be the CID to return. 

Examples:

  > gcb dag import index.html 

`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.FileArg("file", true, false, "Path of file to import").EnableStdin(),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {

		e := cmdenv.(*env.Env)
		role := e.Role() 
		if role != proto.Miner {
			return errors.New("this command must issued at Miner mode!")
		}

		iter := req.Files.Entries()
		if !iter.Next() {
			return fmt.Errorf("no file given: %s", iter.Err())
		}

		fi, ok := iter.Node().(files.File)
		if !ok {
			return fmt.Errorf("given file was not a files.File")
		}

		out, err := e.DAG().ImportData(req.Context, fi) 
		if err != nil {
			return err
		}

		b, err := ioutil.ReadAll(fi)
		if err != nil {
			return err 	
		}

		hash := simhash.Simhash(simhash.NewWordFeatureSet(b))
                return re.Emit(&DagImportDataResult{
                        Cid:	out.Cid(),
                        Hash:	hash ,
                })
	},

	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, res *DagImportDataResult) error {
			fmt.Fprintf(w, "{Cid:\"%s\", Hash:%d},", res.Cid, res.Hash)
                        return nil
		}),
	},

	Type:&DagImportDataResult{},
}
