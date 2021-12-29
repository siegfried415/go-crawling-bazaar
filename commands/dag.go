// Package commands implements the command to print the blockchain.
package commands

import (
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

  > gcb dag cat #####################################
    (############################ is the cid of content of http://www.foo.com/index.html)

`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("cid", true, false, "CID of page content"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {

		e := cmdenv.(*env.Env)
		role := e.Role() 

		switch role {
		case proto.Client : 
			host := e.Host()

			//todo
			conn, err := getConn(host, "DAG.CatRequest", /* domain */ "" )
			if err != nil {
				return err 
			}
			defer conn.Close()

			result, err := conn.DagCat(req.Context, req.Arguments[0]) 
			if err != nil {
				return err  
			}

			return re.Emit(result)

		case proto.Leader:
			fallthrough 
		case proto.Follower: 
			return nil 

		case proto.Miner: 
			c, err := cid.Decode(req.Arguments[0])
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
		cmdkit.FileArg("file", true, false, "Path to file to import").EnableStdin(),
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
