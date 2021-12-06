
package commands

import (
	"encoding/json" 

	types "github.com/siegfried415/go-crawling-bazaar/types" 
	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	env "github.com/siegfried415/go-crawling-bazaar/env" 
	log "github.com/siegfried415/go-crawling-bazaar/utils/log" 

)

var urlRequestCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put and get url request.",
		ShortDescription: `
Whenever a node can't access a web page, a url request can be issued to his 
connected peers on crawling market,  some of those peers will fetch the url 
for biddee. 
`,
		LongDescription: `
Whenever a node can't access a web page, a url request can be issued to his 
connected peers on crawling market,  some of those peers will fetch the url 
for biddee. 
`,
	},

	Subcommands: map[string]*cmds.Command{
		"put": UrlRequestsPutCmd,
	},
}

var UrlRequestsPutCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put url request.",

		ShortDescription: `
Whenever you can't access a web page, you can issue url request to crawling-
market to let other peers fetch the web page for you.
`,
		LongDescription: `
Whenever you can't access a web page, you can issue url request to crawling-
market to let other peers fetch the web page for you.

Examples:
Put a request for a url:
  > gcb --role Client urlrequest put '{"Url":"", "Probability":0.0}' '[{"Url":"http://www.foo.com/index.html", "Probability: 1.0 }]' 
`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("parent", true, false, "url of the parent to be published.").EnableStdin(),
		cmdkit.StringArg("url", true, false, "url of the bidding to be published.").EnableStdin(),
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
                var parenturlrequest types.UrlRequest
                var urlrequests []types.UrlRequest

		//log.Debugf("commands/urlrequest.go(10), req.Arguments[0]=%s\n", req.Arguments[0]) 
                err := json.Unmarshal([]byte(req.Arguments[0]), &parenturlrequest)
                if err != nil {
                        return err
                }

                log.Debugf("commands/urlrequest.go(20), PushUrlRequestsCmd, get parent url %s with probability(%f)\n", parenturlrequest.Url, parenturlrequest.Probability)

		//log.Debugf("commands/urlrequest.go(30), req.Arguments[1]=%s\n", req.Arguments[1]) 
                err = json.Unmarshal([]byte(req.Arguments[1]), &urlrequests )
                if err != nil {
                        return err
                }

                for _, urlrequest := range urlrequests {
			//todo, check each requests is child of parenturl? 
                        log.Debugf("commands/urlrequest.go(40), PushUrlRequestsCmd, get url %s with probability(%f)\n", urlrequest.Url, urlrequest.Probability)
                }

		domain, err := domainForUrl(urlrequests[0].Url) 
		if err != nil {
			return err 
		}

		log.Debugf("commands/urlrequest.go(50), domain=%s\n", domain) 
		e := cmdenv.(*env.Env)
		host := e.Host()

		conn, err := getConn(host, "FRT.UrlRequest",  domain)
		if err != nil {
			log.Debugf("commands/urlrequest.go(55), err=%s\n", err.Error()) 
			return nil 
		}
		defer conn.Close()

		log.Debugf("commands/urlrequest.go(60)\n") 
		err = conn.PutUrlRequest(req.Context, parenturlrequest, urlrequests ) 
		if err != nil {
			log.Debugf("commands/urlrequest.go(65), err=%s\n", err.Error()) 
			return nil 
		}

		log.Debugf("commands/urlrequest.go(70)\n") 
		return cmds.EmitOnce(res, 0)
	},

}

