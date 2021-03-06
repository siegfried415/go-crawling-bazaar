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
	"strings"

	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	cmds "github.com/ipfs/go-ipfs-cmds"

	env "github.com/siegfried415/go-crawling-bazaar/env" 
	//log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
	proto "github.com/siegfried415/go-crawling-bazaar/proto" 
	types "github.com/siegfried415/go-crawling-bazaar/types" 

)

var urlRequestCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put url request.",
		ShortDescription: `
Put a url request to crawling bazaar.
`,
		LongDescription: `
Whenever a node can't access a web page, a url request can be issued to his 
connected peers on crawling market,  some of those peers will fetch the url 
for the client. 
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
Put url request to crawling-bazaar.
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
                ce := cmdenv.(*env.Env)
		role := ce.Role() 
                if role != proto.Client {
			return errors.New("this command must issued at Client mode!")
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

		host := ce.Host()
		conn, err := getConn(host, "FRT.UrlRequest",  domain)
		if err != nil {
			return err  
		}
		defer conn.Close()

		err = conn.PutUrlRequest(req.Context, parenturlrequest, urlrequests ) 
		if err != nil {
			return err 
		}

		return cmds.EmitOnce(res, 0)
	},

}

