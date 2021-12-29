package commands 
import (
	"errors"
	"fmt"
	//"io"
	//"time"

 	cid "github.com/ipfs/go-cid"
	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"

	env "github.com/siegfried415/go-crawling-bazaar/env" 
	//log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
	proto "github.com/siegfried415/go-crawling-bazaar/proto" 
)


var bidCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put or get bid.",
		ShortDescription: `
Put Bid the Miner or get Bid from Crawler.
`,
		LongDescription: `
Put Bid the Miner or get Bid from Crawler.

`,
	},

	Subcommands: map[string]*cmds.Command{
		"put": BidPutCmd,
		"get": BidGetCmd,
	},
}



var BidPutCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put bid.",

		ShortDescription: `
Put Bid to Miner who published url bidding before. 
`,
		LongDescription: `
Put Bid to Miner who published url bidding before. 

Examples:
Put bid for a url bidding:

  > gcb bid put http://www.foo.com/index.html #####################################
    (############################ is the cid of content of http://www.foo.com/index.html)

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("url", true, false, "url of the bidding."),
		cmdkit.StringArg("cid", true, false, "Cids to url."),
	},


	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		ce := cmdenv.(*env.Env)
                role := ce.Role()
                if role != proto.Miner {
			return errors.New("this command must issued at Miner mode!")
		}

		url := req.Arguments[0]
		if !IsURL(url){
			return fmt.Errorf("unsupported url syntax: %s", url)
		}

		cid, err := cid.Decode(req.Arguments[1])
		if err != nil {
			return fmt.Errorf("cid syntax error: %s", cid )
		}

		err = ce.Frontera().PutBid(req.Context, url, cid )
		if err != nil {
			return err
		}

		return cmds.EmitOnce(res, 0)
	},

}


type BidResult struct {
	Cid cid.Cid 
}

var BidGetCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Get bid for some url bidding .",
		ShortDescription: `
Get bid for previous published url bidding.
`,
		LongDescription: `
Get bid for previous published url bidding by this Miner.

Examples:
Get all biddings :

  > gcb bid get http://www.foo.com/index.html 

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("url", true, false, "Get bid issued for url."),
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {

		ce := cmdenv.(*env.Env)
                role := ce.Role()
                if role != proto.Miner {
			return errors.New("this command must issued at Miner mode!")
		}

		url := req.Arguments[0]
		if !IsURL(url){
			return fmt.Errorf("unsupported url syntax: %s", url)
		}

		bids, err := ce.Frontera().GetBids(req.Context,  /* , opts... */ url )
		if err != nil {
			return err
		}

		for _, b := range bids{
			if err := res.Emit(&BidResult{Cid: b.GetCid()}); err != nil  {
				return err
			}

		}

		return nil
	},

	Type: BidResult{},
}

