
package commands

import (
	//"errors"
	"fmt"
	//"io"
	//"time"

	//wyong, 20190320
	//cid "github.com/ipfs/go-cid"

	//wyong, 20200729
	//client "github.com/siegfried415/gdf-rebuild/client"
	types "github.com/siegfried415/gdf-rebuild/types" 
	"encoding/json" 

	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	//logging "github.com/ipfs/go-log"

	//wyong, 20201022 	
	env "github.com/siegfried415/gdf-rebuild/env" 
)

//var log = logging.Logger("core/commands/bidding")


var urlRequestCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put and get Bidding.",
		ShortDescription: `
Whenever a node can't access a web page, a url bidding can be issued to his 
connected peers on crawling market,  some of those peers will fetch the url 
for biddee. 
`,
		LongDescription: `
Whenever a node can't access a web page, a url bidding can be issued to his 
connected peers on crawling market,  some of those peers will fetch the url 
for biddee. 

Examples:
create a bidding for a http url:
  > gcm bidding put http://www.foo.com/index.html

get the bidding published by other peers:
  > gcm bidding get 
  http://www.foo.com/index.html

`,
	},

	Subcommands: map[string]*cmds.Command{
		"put": UrlRequestsPutCmd,
		//"del": BiddingDelCmd,	
		//"get": BiddingGetCmd,
		//"get_data":BiddingGetDataCmd, 		    //wyong, 20190320 
	},
}

var UrlRequestsPutCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put url biddings.",

		ShortDescription: `
Whenever you can't access a web page, you can issue url bidding to crawling-
market to let other peers fetch the web page for you.
`,
		LongDescription: `
Whenever you can't access a web page, you can issue url bidding to crawling-
market to let other peers fetch the web page for you.

Examples:
Put bidding for a url:

  > gcm bidding put http://www.foo.com/index.html

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("parent", true, false, "url of the parent to be published.").EnableStdin(),
		cmdkit.StringArg("url", true, false, "url of the bidding to be published.").EnableStdin(),
	},

	//Options: []cmds.Option{
	//	cmds.BoolOption(resolveOptionName, "Resolve given path before publishing.").WithDefault(true),
	//	cmds.StringOption(lifeTimeOptionName, "t",
	//		`Time duration that the record will be valid for. <<default>>
//    This accepts durations such as "300s", "1.5h" or "2h45m". Valid time units are
//    "ns", "us" (or "µs"), "ms", "s", "m", "h".`).WithDefault("24h"),
	//	cmds.BoolOption(allowOfflineOptionName, "When offline, save the IPNS record to the the local datastore without broadcasting to the network instead of simply failing."),
	//	cmds.StringOption(ttlOptionName, "Time duration this record should be cached for (caution: experimental)."),
	//	cmds.StringOption(keyOptionName, "k", "Name of the key to be used or a valid PeerID, as listed by 'ipfs key list -l'. Default: <<default>>.").WithDefault("self"),
	//	cmds.BoolOption(quieterOptionName, "Q", "Write only final hash."),
	//},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		//log.Debugf("BiddingPutCmd, Run() called...") 
			
                var parenturlrequest types.UrlRequest
                var urlrequests []types.UrlRequest

                //wyong, 20191114
		fmt.Printf("commands/urlrequest.go(10), req.Arguments[0]=%s\n", req.Arguments[0]) 
                err := json.Unmarshal([]byte(req.Arguments[0]), &parenturlrequest)
                if err != nil {
                        return err
                }

                fmt.Printf("commands/urlrequest.go(20), PushUrlRequestsCmd, get parent url %s with probability(%f)\n", parenturlrequest.Url, parenturlrequest.Probability)

		fmt.Printf("commands/urlrequest.go(30), req.Arguments[1]=%s\n", req.Arguments[1]) 
                err = json.Unmarshal([]byte(req.Arguments[1]), &urlrequests )
                if err != nil {
                        return err
                }

                for _, urlrequest := range urlrequests {
                        fmt.Printf("commands/urlrequest.go(40), PushUrlRequestsCmd, get url %s with probability(%f)\n", urlrequest.Url, urlrequest.Probability)
                }


		//todo, get dbID from url, wyong, 20200803 
		//var dbID string 

		//todo, wyong, 20200824 
		domain, err := domainForUrl(urlrequests[0].Url) 
		if err != nil {
			return err 
		}

		fmt.Printf("commands/urlrequest.go(50), domain=%s\n", domain) 
		//todo, send bidding request with client to random peers , wyong, 20200723 
		//err := env.GetFrontera().PutBidding(req.Context, url )
		//if err != nil {
		//	//log.Debugf("BiddingPutCmd, Run(30)") 
		//	
		//	//if err == iface.ErrOffline {
		//	//	err = errAllowOffline
		//	//}
		//	return err
		//}


		//todo, wyong, 20200728 
		//var conn *client.Conn

		//todo, get host from env, wyong, 20201007
		e := cmdenv.(*env.Env)
		host := e.Host()

		//todo, define ProtocolUrlRequest, wyong, 20201008 
		conn, err := getConn(host, "FRT.UrlRequest",  domain)
		if err != nil {
			fmt.Printf("commands/urlrequest.go(55), err=%s\n", err.Error()) 
			return nil 
		}
		defer conn.Close()

		fmt.Printf("commands/urlrequest.go(60)\n") 
		//var result sql.Result
		err = conn.PutUrlRequest(req.Context, parenturlrequest, urlrequests ) 

		if err != nil {
			fmt.Printf("commands/urlrequest.go(65), err=%s\n", err.Error()) 
			return nil 
		}

		//if err == nil {
		//	affectedRows, _ = result.RowsAffected()
		//	lastInsertID, _ = result.LastInsertId()
		//}

		//return cmds.EmitOnce(res, &IpnsEntry{
		//	Name:  out.Name(),
		//	Value: out.Value().String(),
		//})

		fmt.Printf("commands/urlrequest.go(70)\n") 
		return cmds.EmitOnce(res, 0)
	},

	//Encoders: cmds.EncoderMap{
	//	cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, ie *IpnsEntry) error {
	//		var err error
	//		quieter, _ := req.Options[quieterOptionName].(bool)
	//		if quieter {
	//			_, err = fmt.Fprintln(w, ie.Name)
	//		} else {
	//			_, err = fmt.Fprintf(w, "Published to %s: %s\n", ie.Name, ie.Value)
	//		}
	//		return err
	//	}),
	//},

	//Type: BiddingEntry{},
}

