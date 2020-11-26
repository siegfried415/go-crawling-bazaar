// Package commands implements the command to print the blockchain.
package commands

import (
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	cmds "github.com/ipfs/go-ipfs-cmds"

	"fmt" 
        "io"

        "github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs-files"

	//wyong, 20191115
	"io/ioutil" 

	//wyong, 20200218 
	"github.com/mfonda/simhash"

	env "github.com/siegfried415/gdf-rebuild/env" 

	//wyong, 20201126 
	"github.com/siegfried415/gdf-rebuild/proto" 
)

var dagCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with IPLD DAG objects.",
	},
	Subcommands: map[string]*cmds.Command{
		"get": dagGetCmd,
		"cat":                  DagCatCmd,
		"import":               DagImportDataCmd,
	},
}

var dagGetCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Get a DAG node by its CID",
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("ref", true, false, "CID of object to get"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		fmt.Printf("commands/dag.go, DagGetCmd(10) \n") 
		//out, err := GetPorcelainAPI(env).DAGGetNode(req.Context, req.Arguments[0])
		e := cmdenv.(*env.Env)
		out, err := e.DAG().GetNode(req.Context, req.Arguments[0]) 
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
		Tagline: "Read out data stored on the network",
		ShortDescription: `
Prints data from the storage market specified with a given CID to stdout. The
only argument should be the CID to return. The data will be returned in whatever
format was provided with the data initially.
`,
	},
	Arguments: []cmdkit.Argument{
		//todo, wyong, 20201126 
                //cmdkit.StringArg("parent", true, false, "the parent url to be get from url store"),
                //cmdkit.StringArg("url", true, false, "the url to be get from url store"),

		cmdkit.StringArg("cid", true, false, "CID of page content"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		fmt.Printf("commands/dag.go, DagCatCmd(10) \n") 

		/* todo, wyong, 20201126
		parenturl := req.Arguments[0]
		fmt.Printf("commands/dag.go, DagCatCmd(10), parenturl=%s\n", parenturl)

		url := req.Arguments[1]
		fmt.Printf("commands/dag.go, DagCatCmd(20), url=%s\n", url ) 

		//todo, wyong, 20200824 
		domain, err := domainForUrl(url) 
		if err != nil {
			return err 
		}
		*/

		c, err := cid.Decode(req.Arguments[0])
		if err != nil {
			return err
		}


		fmt.Printf("commands/dag.go, DagCatCmd(20), cid=%s \n", c.String()) 

		//bugfix, wyong, 20201126 
		//wyong, 20201007 
		//role := req.Options["role"] 
		//role, _ := req.Options[OptionRole].(string)
		e := cmdenv.(*env.Env)
		role := e.Role() 
		fmt.Printf("commands/dag.go, DagCatCmd(30), role =%s \n", role.String()) 

		switch role {
		//if role == "Client" { 	
		case proto.Client : 
			fmt.Printf("commands/dag.go, DagCatCmd(40) \n") 
			//get host from env, wyong, 20201008
			host := e.Host()

			//todo, wyong, 20201126 
			conn, err := getConn(host, "DAG.CatRequest", /* domain */ "" )
			if err != nil {
				fmt.Printf("commands/urlrequest.go(45), err=%s\n", err.Error()) 
				return nil 
			}
			defer conn.Close()

			//todo, wyong, 20201022 
			//var result sql.Result
			//err = conn.DagCat(req.Context, c) 
			//if err != nil {
			//	fmt.Printf("commands/urlrequest.go(65), err=%s\n", err.Error()) 
			//	return err  
			//}

		//}else if role == "Miner" {	
		case proto.Leader:
			fallthrough 
		case proto.Follower: 
			fmt.Printf("commands/dag.go, DagCatCmd(50)\n") 
			return nil 

		case proto.Miner: 
			fmt.Printf("commands/dag.go, DagCatCmd(60)\n") 
			//dr, err := GetPorcelainAPI(env).DAGCat(req.Context, c)
			//e := cmdenv.(*env.Env)
			dr, err := e.DAG().Cat(req.Context, c) 
			if err != nil {
				fmt.Printf("commands/dag.go, DagCatCmd(65) \n") 
				return err
			}
			fmt.Printf("commands/dag.go, DagCatCmd(70)\n") 
			return re.Emit(dr)

		//} else { 	
		case proto.Unknown: 
			fmt.Printf("commands/dag.go, DagCatCmd(80)\n") 
			return nil 
		}

		fmt.Printf("commands/dag.go, DagCatCmd(90) \n") 
		//get url from dr , wyong, 20191115 
		//var p Page 
		//err := json.Unmarshal(dr, &p)
		//if err != nil {
		//	return err 
		//}

		// todo, wyong, 20200922 
		//wyong, 20191025
                //fromAddr, err := fromAddrOrDefault(req, cmdenv)
                //if err != nil {
		//	fmt.Printf("commands/dag.go, DagCatCmd(45) \n") 
                //      return err
                //}

		//fmt.Printf("commands/dag.go, DagCatCmd(50) \n") 
                //_, err = GetPorcelainAPI(env).UrlGraphPageRetrived(req.Context, fromAddr, parenturl, url )
                //if err != nil {
		//	fmt.Printf("commands/dag.go, DagCatCmd(55) \n") 
                //      return err
                //}

		//fmt.Printf("commands/dag.go, DagCatCmd(90) \n") 

		//return re.Emit(dr)
		return nil 
	},
}

//wyong, 20200218 
type DagImportDataResult struct {
        Cid cid.Cid  
        Hash uint64 
}

var DagImportDataCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Import data into the local node",
		ShortDescription: `
Imports data previously exported with the client cat command into the storage
market. This command takes only one argument, the path of the file to import.
See the go-filecoin client cat command for more details.
`,
	},
	Arguments: []cmdkit.Argument{
                //cmdkit.StringArg("url", true, false, "the url to be get from url store"),
		cmdkit.FileArg("file", true, false, "Path to file to import").EnableStdin(),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		fmt.Printf("commands/dag.go, DagImportCmd(10) \n") 

		//wyong, 20191115 
		//url := req.Arguments[0]

		iter := req.Files.Entries()
		if !iter.Next() {
			return fmt.Errorf("no file given: %s", iter.Err())
		}

		fi, ok := iter.Node().(files.File)
		if !ok {
			return fmt.Errorf("given file was not a files.File")
		}

		/*
		//embed url with body into dag, wyong, 20191115 
		b, err := ioutil.ReadAll(fi)
		if err != nil  {
			return fmt.Errorf("can't read body from File")
		}

		r, _ := json.Marshal( {'Url':url, 'Body':b})
		*/

		//out, err := GetPorcelainAPI(env).DAGImportData(req.Context, fi )
		e := cmdenv.(*env.Env)
		out, err := e.DAG().ImportData(req.Context, fi) 
		if err != nil {
			return err
		}

		//wyong, 20200218 
		//return re.Emit(out.Cid())

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

	//wyong, 20200218 
	//Type: cid.Cid{},
	Type:&DagImportDataResult{},

	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, res *DagImportDataResult  /* c cid.Cid , wyong, 20200218  */ ) error {
			//return PrintString(w, c)
			fmt.Fprintf(w, "{Cid:\"%s\", Hash:%d},", res.Cid, res.Hash)
                        return nil
		}),
	},
}
