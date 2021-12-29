package commands

import (
	"fmt"
	"encoding/json" 
	"errors"
	"io"
	"strings" 

	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"

	env "github.com/siegfried415/go-crawling-bazaar/env" 
	log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
	proto "github.com/siegfried415/go-crawling-bazaar/proto" 
	types "github.com/siegfried415/go-crawling-bazaar/types" 
)


var biddingCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put and get Bidding.",
		ShortDescription: `
Put biddings to other crawlers or get biddings from Miners.
`,
		LongDescription: `
Whenever a miner can't access a web page, a url bidding can be issued to other 
connected crawlers on crawling market,  some of those crawlers will fetch the url 
for the miner. 
`,
	},

	Subcommands: map[string]*cmds.Command{
		"put": BiddingPutCmd,
		"get": BiddingGetCmd,
		"del": BiddingDelCmd,	
		"get_completed": CompletedBiddingGetCmd,
		"get_uncompleted": UncompletedBiddingGetCmd,
	},
}

var BiddingPutCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put url biddings.",

		ShortDescription: `
Put url bidding to crawler miner.
`,
		LongDescription: `
Whenever a miner can't access a web page,  miner can issue url bidding to crawler
to let other crawlers fetch the web page for him.

Examples:
Put bidding for a url:

  > gcb --role Miner bidding put '{"Url":"", "Probability":0.0}' '[{"Url":"http://www.foo.com/index.html", "Probability: 1.0 }]' 

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("parent", true, false, "url of the parent to be published.").EnableStdin(),
		cmdkit.StringArg("url", true, false, "url of the bidding to be published.").EnableStdin(),
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		ce := cmdenv.(*env.Env)
                role := ce.Role()
                if role != proto.Miner {
			return errors.New("this command must issued at Miner mode!")
		}

                var parenturlrequest types.UrlRequest
                err := json.Unmarshal([]byte(req.Arguments[0]), &parenturlrequest)
                if err != nil {
                        return err
                }

                var urlrequests []types.UrlRequest
                err = json.Unmarshal([]byte(req.Arguments[1]), &urlrequests )
                if err != nil {
                        return err
                }

		//get domain of this url request 
		var domain string 
		if parenturlrequest.Url != "" { 
			domain, _ = domainForUrl(parenturlrequest.Url) 
		}

		if domain == "" {
			if urlrequests[0].Url != "" { 
				domain, err = domainForUrl(urlrequests[0].Url) 
				if err != nil {
					return err 
				}
			}
		}

		if domain == "" {
			return errors.New("can't get domain of this url request!")
		}

                for _, urlrequest := range urlrequests {
			if !strings.HasPrefix(urlrequest.Url, domain) {
				return errors.New("requested url must belongs to the same domain of parent")
			}
                }

		log.Debugf("BiddingPutCmd, Run(20)") 
                //send bidding request with client to random peers 
                err = ce.Frontera().PutBidding(req.Context, domain, urlrequests, parenturlrequest)
                if err != nil {
                      log.Debugf("BiddingPutCmd, Run(30)")
                      return err
                }

		//log.Debugf("BiddingPutCmd, Run(40)") 
		return cmds.EmitOnce(res, 0)
	},

}

type BiddingUrl struct {
	Url string  
}

var BiddingGetCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Get url bidding.",
		ShortDescription: `
Get all url biddings published by other miners.
`,
		LongDescription: `
Get all url biddings published by other miners.

Examples:
Get biddings :

  > gcb --role Crawler  bidding get 

`,
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {

		ce := cmdenv.(*env.Env) 
                role := ce.Role()
                if role != proto.Miner {
			return errors.New("this command must issued at Miner mode!")
		}

		output, err := ce.Frontera().GetBidding(req.Context /* , opts... */ )
		if err != nil {
			return err
		}

		if len(output) ==  0 {
			if err := res.Emit(&BiddingUrl{Url:""}); err != nil  {
				return err
			}
		}else {
			for _, v := range output {
				if err := res.Emit(&BiddingUrl{v.GetUrl()}); err != nil  {
					return err
				}

			}
		}

		return nil
	},

	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{}) error{
			u, ok := v.(*BiddingUrl)
			if !ok {
				return fmt.Errorf("cast error, got type %T", v)
			}

			fmt.Fprintln(w, "url:%s}",  u.Url )
			return nil 
		}),
	},

	Type: &BiddingUrl{},
}

var BiddingDelCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Delete url bidding.",

		ShortDescription: `
Delete url bidding when the bidding has been completed.
`,
		LongDescription: `
When the url bidding has been completed, you should delete the url bidding 
from local queue.

Examples:
Delete a url bidding :

  > gcb --role Miner bidding del http://www.foo.com/index.html

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("url", true, false, "url of the bidding to be published.").EnableStdin(),
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

		err := ce.Frontera().DelBidding(req.Context, url )
		if err != nil {
			return err
		}

		return cmds.EmitOnce(res, 0)
	},

}

type CompletedBidding struct {
	Url string  
	//Cid []cid.Cid 
}


var CompletedBiddingGetCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Get completed bidding requests.",
		ShortDescription: `
Get all completed biddings published by this miner.
`,
		LongDescription: `
Get all completed biddings published by this miner.

Examples:
Get all completed biddings :

  > gcb --role Miner bidding get_completed 

`,
	},

	//TODO
	//Arguments: []cmds.Argument{
	//	cmds.StringArg("peer", false, false, "Get biddings issued from peer. return all biddings received if no peer given."),
	//},


	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		ce := cmdenv.(*env.Env) 
                role := ce.Role()
                if role != proto.Miner {
			return errors.New("this command must issued at Miner mode!")
		}

		output, err := ce.Frontera().GetCompletedBiddings(req.Context /* , opts... */ )
		if err != nil {
			return err
		}

		for _, v := range output {
			if err := res.Emit(&CompletedBidding{Url:v.GetUrl()}); err != nil  {
				return err
			}
		}

		return nil
	},

	Type: CompletedBidding{},
}

type UncompletedBidding struct {
	Url string  
	//Cid []cid.Cid 
}


var UncompletedBiddingGetCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Get uncompleted bidding requests.",
		ShortDescription: `
Get all uncompleted biddings published by this miner.
`,
		LongDescription: `
Get all uncompleted biddings published by this miner.

Examples:
Get all uncompleted biddings :

  > gcm --role Miner bidding get_uncompleted 

`,
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		ce := cmdenv.(*env.Env)
                role := ce.Role()
                if role != proto.Miner {
			return errors.New("this command must issued at Miner mode!")
		}

		output, err := ce.Frontera().GetUncompletedBiddings(req.Context /* , opts... */ )
		if err != nil {
			return err
		}

		for _, v := range output {
			if err := res.Emit(&UncompletedBidding{Url:v.GetUrl() }); err != nil  {
				return err
			}

		}

		return nil
	},

	Type: UncompletedBidding{},
}


