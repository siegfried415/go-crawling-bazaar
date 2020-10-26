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
	"errors" 
        "fmt"
        "io"

        "github.com/ipfs/go-cid"
        "github.com/ipfs/go-ipfs-cmds"
        "github.com/ipfs/go-ipfs-cmdkit"

	env "github.com/siegfried415/go-crawling-bazaar/env" 
	log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
	proto "github.com/siegfried415/go-crawling-bazaar/proto" 

)

var urlGraphCmd = &cmds.Command{
        Helptext: cmdkit.HelpText{
                Tagline: "manage url graph",
        },
        Subcommands: map[string]*cmds.Command{
		"get-page-cid-by-url" : UrlGraphGetCidByUrlCmd, 
        },
}

var UrlGraphGetCidByUrlCmd = &cmds.Command{
        Helptext: cmdkit.HelpText{
                Tagline: "get cid of url saved in the url graph",
                ShortDescription: `
Get cid of url saved in the url graph. 
`,
		LongDescription: `
Get cid of url saved in the url graph. 

Examples:

  > gcb --role Client urlgraph get-page-cid-by-url http://www.foo.com/index.html 

`,
        },

        Arguments: []cmdkit.Argument{
                cmdkit.StringArg("url", true, false, "the url to be get from url graph"),
        },

        Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		ce := cmdenv.(*env.Env)	
                role := ce.Role()
                if role != proto.Client {
			return errors.New("this command must issued at Client mode!")
		}

		url := req.Arguments[0]
		if !IsURL(url){
			return fmt.Errorf("unsupported url syntax: %s", url)
		}

		domain, err := domainForUrl(url)
		if err != nil {
			return err 
		}
		
		host := ce.Host()
		conn, err := getConn(host, "FRT.UrlCidRequest", domain )
		if err != nil {
			log.Debugf("UrlGraphGetCidByUrlCmd, get Conn failed, err=%s ", err.Error()) 
			return err 
		}
		defer conn.Close()

		result, err := conn.GetCidByUrl(req.Context, req.Arguments[0]) 
		if err != nil {
			log.Debugf("UrlGraphGetCidByUrlCmd, GetCidByUrl failed, err=%s ", err.Error()) 
			return err 
		}

		return re.Emit(result)

        },

        Encoders: cmds.EncoderMap{
                cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, cid cid.Cid) error {
			fmt.Fprintf(w, "%s\n", cid.String()) 
                        return nil
                }),
        },

	Type: cid.Cid{},
}

