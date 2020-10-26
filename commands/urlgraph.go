
//wyong, 20190904 
package commands

import (
        "fmt"
        "io"
        //"math/big"


        "github.com/ipfs/go-cid"
        "github.com/ipfs/go-ipfs-cmds"
        "github.com/ipfs/go-ipfs-cmdkit"
        //"github.com/pkg/errors"

	//wyong, 20200803 
	//client "github.com/siegfried415/gdf-rebuild/client" 

	//wyong, 20201022 	
	env "github.com/siegfried415/gdf-rebuild/env" 

)

var urlGraphCmd = &cmds.Command{
        Helptext: cmdkit.HelpText{
                Tagline: "Manage url graph actor",
        },
        Subcommands: map[string]*cmds.Command{
		//"add-page-crawled" : UrlGraphPageCrawledCmd, 
		"get-page-cid-by-url" : UrlGraphGetCidByUrlCmd, 

		//for debug, wyong, 20191118
		//"get-url-nodes"  : UrlGraphGetUrlNodesCmd, 
		//"get-url-node"  : UrlGraphGetUrlNodeCmd, 
        },
}

/*
// MinerSetPriceResult is the return type for miner set-price command
type UrlGraphPageCrawledResult struct {
        GasUsed               types.GasUnits
        UrlGraphAddPageCrawledResponse porcelain.UrlGraphUrlCrawledResponse
}


//wyong, 20190904
var UrlGraphPageCrawledCmd = &cmds.Command {

        Helptext: cmds.HelpText{
                Tagline: "put url 's content to url graph store",
                ShortDescription: `Put url 's content to url graph store, which can be feteched by client in future.
This command waits for the url 's content to be mined.`,
        },

        Arguments: []cmds.Argument{
                cmds.StringArg("url", true, false, "the url to be put to url graph store"),
                cmds.StringArg("cid", true, false, "the cid of the url 's content"),

		//wyong, 20200218 
		cmds.StringArg("hash", true, false, "Hash of page content"),
        },

        Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		url := req.Arguments[0]

		fmt.Printf("commands/urlgraph.go, UrlGraphPageCrawledCmd, input url=%s\n", url ) 

                //out, err := GetPorcelainAPI(env).DAGImportData(req.Context, strings.NewReader(req.Arguments[1]))

		contentCid, err := cid.Decode(req.Arguments[1])
                if err != nil {
                        return err
                }

		//wyong, 20200218 
                pageHash, ok := big.NewInt(0).SetString(req.Arguments[2], 0 )
                if !ok {
                        return fmt.Errorf("pageHash must be a valid integer")
                }


                //cid := out.Cid()
		//fmt.Printf("commands/urlgraph.go, UrlGraphPageCrawledCmd, cid of content(%s) is %s\n", req.Arguments[1], cid.String()) 
                fromAddr, err := fromAddrOrDefault(req, env)
                if err != nil {
                        return err
                }

		//wyong, 20191227 
                ret, err := GetPorcelainAPI(env).ConfigGet("mining.minerAddress")
                if err != nil {
			fmt.Printf("commands/urlgraph.go, UrlGraphPageCrawledCmd, problem getting minner address \n") 
                        return errors.Wrap(err, "problem getting miner address")
                }
                minerAddr, ok := ret.(address.Address)
                if !ok {
			fmt.Printf("commands/urlgraph.go, UrlGraphPageCrawledCmd, problem converting minner address \n") 
                        return errors.New("problem converting miner address")
                }

		//add parameter pageHash, wyong, 20200218 
                res, err := GetPorcelainAPI(env).UrlGraphUrlCrawled(req.Context, fromAddr ,  url, contentCid , pageHash, minerAddr )
                if err != nil {
			fmt.Printf("commands/urlgraph.go, UrlGraphPageCrawledCmd, PorcelainAPI/UrlGraphUrlCrawled completed, got error\n")
                        return err
                }

		fmt.Printf("commands/urlgraph.go, UrlGraphPageCrawledCmd, PorcelainAPI/UrlGraphUrlCrawled completed\n" ) 
                return re.Emit(&UrlGraphPageCrawledResult{
                        GasUsed:               types.NewGasUnits(0),
                        UrlGraphAddPageCrawledResponse: res,
                })

	},

        Type: &UrlGraphPageCrawledResult{},

        Encoders: cmds.EncoderMap{
                cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, res *UrlGraphPageCrawledResult) error {
                        _, err := fmt.Fprintf(w, `put url %s 's content %s to decentralized url store.
        Published url , cid: %s.
        Url confirmed on chain in block: %s.
`,
                                res.UrlGraphAddPageCrawledResponse.Url,
                                res.UrlGraphAddPageCrawledResponse.UrlContentCid.String(),
                                res.UrlGraphAddPageCrawledResponse.UrlCrawledCid.String(),
                                res.UrlGraphAddPageCrawledResponse.BlockCid.String(),
                        )
                        return err
                }),
        },

}
*/


//wyong, 20190905 
var UrlGraphGetCidByUrlCmd = &cmds.Command{
        Helptext: cmdkit.HelpText{
                Tagline: "get cid of url saved in the url store",
                ShortDescription: `
Get cid of url in the url store. This command takes no arguments. Results
will be returned as a space separated table with url. 
`,
        },

        Arguments: []cmdkit.Argument{
                cmdkit.StringArg("url", true, false, "the url to be get from url store"),
        },

        Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		//wyong, 20200729 
                //result, err := GetPorcelainAPI(env).UrlGraphGetUrlNode(req.Context, url )
                //if err != nil {
                //        return err
                //}

		domain, err := domainForUrl(req.Arguments[0])
		if err != nil {
			return err 
		}
		
		fmt.Printf("UrlGraphGetCidOfUrl(10), domain=%s\n", domain ) 

		//todo, get host from env, wyong, 20201008 
		e := cmdenv.(*env.Env)	
		host := e.Host()

		//todo, check cid verfied by two crawlers. wyong, 20191031
		//var conn *client.Conn
		conn, err := getConn(host, "ProtocolGetUrlCidRequest", domain )
		if err != nil {
			return err 
		}
		defer conn.Close()

		//var result cid.Cid  
		fmt.Printf("UrlGraphGetCidOfUrl(20), url =%s\n", req.Arguments[0] ) 
		result, err := conn.GetCidByUrl(req.Context, req.Arguments[0]) 

		fmt.Printf("UrlGraphGetCidOfUrl(30), result=%s\n", result.String()) 
		return re.Emit(result)

        },

	Type: cid.Cid{},
        Encoders: cmds.EncoderMap{
                cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, cid cid.Cid) error {
			fmt.Fprintf(w, "%s\n", cid.String()) 
                        return nil
                }),
        },

}

/* wyong, 20200729 
//wyong, 20190905 
type UrlGraphGetUrlNodeResult struct {
        GasUsed        types.GasUnits
        UrlNode		porcelain.UrlNode
}

var UrlGraphGetUrlNodeCmd = &cmds.Command{
        Helptext: cmds.HelpText{
                Tagline: "get url node saved in the url graph store",
                ShortDescription: `
Get cid of url in the url store. This command takes no arguments. Results
will be returned as a space separated table with url. 
`,
        },

        Arguments: []cmds.Argument{
                cmds.StringArg("url", true, false, "the url to be get from url store"),
        },

        Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		url := req.Arguments[0]

		fmt.Printf("commands/urlgraph.go, UrlGraphGetUrlNodeCmd(10)\n" ) 
                res, err := GetPorcelainAPI(env).UrlGraphGetUrlNode(req.Context, url )
                if err != nil {
                        return err
                }

		fmt.Printf("commands/urlgraph.go, UrlGraphGetUrlNodeCmd(20)\n" ) 
                return re.Emit(&UrlGraphGetUrlNodeResult{
                        GasUsed:       types.NewGasUnits(0),
                        UrlNode: 	res,
                })

        },

        Type: &UrlGraphGetUrlNodeResult{},
        Encoders: cmds.EncoderMap{
                cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, res *UrlGraphGetUrlNodeResult ) error {

			//todo, output cid,  wyong, 20191029 
			fmt.Fprintf(w, "UrlNode(%s): CrawledCount =%d, RequestedCount=%d, RetrivedCount=%d\n", res.UrlNode.Url, 
				//res.UrlGraphGetUrlNodeResponse.Cid,
				res.UrlNode.CrawledCount, 
				res.UrlNode.RequestedCount ,
				
				//wyong, 20191115 
				res.UrlNode.RetrivedCount ) 

                        return nil
                }),
        },

}

//wyong, 20191118 
type UrlGraphGetUrlNodesResult struct {
        GasUsed        types.GasUnits
        UrlNodes	[]porcelain.UrlNode
}

var UrlGraphGetUrlNodesCmd = &cmds.Command{
        Helptext: cmds.HelpText{
                Tagline: "get all url node saved in the url graph store",
                ShortDescription: `
Get cid of url in the url store. This command takes no arguments. Results
will be returned as a space separated table with url. 
`,
        },

        //Arguments: []cmds.Argument{
        //        cmds.StringArg("url", true, false, "the url to be get from url store"),
        //},

        Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		//url := req.Arguments[0]

		fmt.Printf("commands/urlgraph.go, UrlGraphGetUrlNodesCmd(10)\n" ) 
                res, err := GetPorcelainAPI(env).UrlGraphGetUrlNodes(req.Context )
                if err != nil {
                        return err
                }

		fmt.Printf("commands/urlgraph.go, UrlGraphGetUrlNodesCmd(20)\n" ) 
                return re.Emit(&UrlGraphGetUrlNodesResult{
                        GasUsed:       types.NewGasUnits(0),
                        UrlNodes: 	res,
                })

        },

        Type: &UrlGraphGetUrlNodesResult{},
        Encoders: cmds.EncoderMap{
                cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, res *UrlGraphGetUrlNodesResult ) error {

			for _, urlnode := range res.UrlNodes {
				//todo, output cid,  wyong, 20191029 
				fmt.Fprintf(w, "UrlNode(%s): CrawledCount =%d, RequestedCount=%d, RetrivedCount=%d\n", urlnode.Url, 
					//urlnode.Cid,
					urlnode.CrawledCount, 
					urlnode.RequestedCount ,
					
					//wyong, 20191115 
					urlnode.RetrivedCount ) 
			}

                        return nil

                }),

        },

}
*/
