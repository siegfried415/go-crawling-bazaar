package commands
import (
	"errors" 
        "fmt"
        "io"

        "github.com/ipfs/go-cid"
        "github.com/ipfs/go-ipfs-cmds"
        "github.com/ipfs/go-ipfs-cmdkit"

	env "github.com/siegfried415/go-crawling-bazaar/env" 
	//log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
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
			return err 
		}
		defer conn.Close()

		result, err := conn.GetCidByUrl(req.Context, req.Arguments[0]) 
		if err != nil {
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

