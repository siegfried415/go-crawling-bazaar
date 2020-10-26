package commands 
import (
	"errors"
	"fmt"
	"io"
	//"time"

	//wyong, 20190320 
	//files "github.com/ipfs/go-ipfs-files"

	//wyong, 20190123
 	cid "github.com/ipfs/go-cid"

	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	logging "github.com/ipfs/go-log"

	//wyong, 20200802 
	env "github.com/siegfried415/gdf-rebuild/env" 
)


var log = logging.Logger("core/commands/bidding")

//type BidEntry struct {
//	Name  string
//	Value string
//}

var bidCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put bid.",
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
		"put": BidPutCmd,
		"get": BidGetCmd,
		"put_data":BidPutDataCmd, 		    //wyong, 20190320 
	},
}




var (
	errAllowOffline = errors.New("can't put bidding while offline: pass `--allow-offline` to override")
)

const (
	//ipfsPathOptionName     = "ipfs-path"
	//resolveOptionName      = "resolve"
	//allowOfflineOptionName = "allow-offline"
	//lifeTimeOptionName     = "lifetime"
	//ttlOptionName          = "ttl"
	//keyOptionName          = "key"
	//quieterOptionName      = "quieter"

	biddingUrlOptionName   = "url"
)

// IsURL returns true if the string represents a valid URL that the
// urlstore can handle.  More specifically it returns true if a string
// begins with 'http://' or 'https://'.
func IsURL(str string) bool {
	return (len(str) > 7 && str[0] == 'h' && str[1] == 't' && str[2] == 't' && str[3] == 'p') &&
			((len(str) > 8 && str[4] == 's' && str[5] == ':' && str[6] == '/' && str[7] == '/') ||
					(str[4] == ':' && str[5] == '/' && str[6] == '/'))
}


var BidPutCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Put bid.",

		ShortDescription: `
Whenever you can't access a web page, you can issue url bidding to crawling-
market to let other peers fetch the web page for you.
`,
		LongDescription: `
Whenever you can't access a web page, you can issue url bidding to crawling-
market to let other peers fetch the web page for you.

Examples:
Put bid for a url bidding:

  > gcm bid put http://www.foo.com/index.html #############################

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("url", true, false, "url of the bidding."),
		cmdkit.StringArg("cid", true, false, "Cids to url."),
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
		//api, err := cmdenv.GetApi(env)
		//if err != nil {
		//	return err
		//}

		//allowOffline, _ := req.Options[allowOfflineOptionName].(bool)
		//kname, _ := req.Options[keyOptionName].(string)

		//validTimeOpt, _ := req.Options[lifeTimeOptionName].(string)
		//validTime, err := time.ParseDuration(validTimeOpt)
		//if err != nil {
		//	return fmt.Errorf("error parsing lifetime option: %s", err)
		//}

		//opts := []options.NamePublishOption{
		//	options.Name.AllowOffline(allowOffline),
		//	options.Name.Key(kname),
		//	options.Name.ValidTime(validTime),
		//}

		//if ttl, found := req.Options[ttlOptionName].(string); found {
		//	d, err := time.ParseDuration(ttl)
		//	if err != nil {
		//		return err
		//	}

		//	opts = append(opts, options.Name.TTL(d))
		//}

		//p, err := iface.ParsePath(req.Arguments[0])
		//if err != nil {
		//	return err
		//}

		//if verifyExists, _ := req.Options[resolveOptionName].(bool); verifyExists {
		//	_, err := api.ResolveNode(req.Context, p)
		//	if err != nil {
		//		return err
		//	}
		//}

		fmt.Printf("BidPutCmd/Run(10)\n") 
		url := req.Arguments[0]
		if !IsURL(url){
			fmt.Printf("BidPutCmd/Run(15)\n") 
			return fmt.Errorf("unsupported url syntax: %s", url)
		}

		fmt.Printf("BidPutCmd/Run(20), url=%s\n", url ) 
		//wyong, 20190123
		cid, err := cid.Decode(req.Arguments[1])
		if err != nil {
			return fmt.Errorf("cid syntax error: %s", cid )
		}

		fmt.Printf("BidPutCmd/Run(30), cid=%s\n", cid.String()) 
		ce := cmdenv.(*env.Env)
		err = ce.Frontera().PutBid(req.Context, url, cid )
		if err != nil {
			fmt.Printf("BidPutCmd/Run(35), err=%s\n", err.Error()) 
			//if err == ErrNodeOffline {
			//	err = errAllowOffline
			//}
			return err
		}

		//return cmds.EmitOnce(res, &IpnsEntry{
		//	Name:  out.Name(),
		//	Value: out.Value().String(),
		//})

		fmt.Printf("BidPutCmd/Run(40)\n") 
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


//var log = logging.Logger("core/commands/bidding")
type BidResult struct {
	Cid cid.Cid 
}

const (
	recursiveOptionName      = "recursive"
	nocacheOptionName        = "nocache"
	dhtRecordCountOptionName = "dht-record-count"
	dhtTimeoutOptionName     = "dht-timeout"
	streamOptionName         = "stream"
)

var BidGetCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Get url bidding requests.",
		ShortDescription: `
Get all url biddings published by your peers.
`,
		LongDescription: `
Get all url biddings published by your peers.

Examples:
Get all biddings :

  > gcm bidding get 
    http://www.foo.com/index.html 

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("url", true, false, "Get bid issued for url."),
	},

	//Options: []cmds.Option{
	//	cmds.BoolOption(recursiveOptionName, "r", "Resolve until the result is not an IPNS name."),
	//	cmds.BoolOption(nocacheOptionName, "n", "Do not use cached entries."),
	//	cmds.UintOption(dhtRecordCountOptionName, "dhtrc", "Number of records to request for DHT resolution."),
	//	cmds.StringOption(dhtTimeoutOptionName, "dhtt", "Max time to collect values during DHT resolution eg \"30s\". Pass 0 for no timeout."),
	//	cmds.BoolOption(streamOptionName, "s", "Stream entries as they are found."),
	//},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		log.Debugf("CompletedBiddingGetCmd, Run() called...") 
		//api, err := cmdenv.GetApi(env)
		//if err != nil {
		//	return err
		//}

		//nocache, _ := req.Options["nocache"].(bool)
		//local, _ := req.Options["local"].(bool)

		//var name string
		//if len(req.Arguments) == 0 {
		//	self, err := api.Key().Self(req.Context)
		//	if err != nil {
		//		return err
		//	}
		//	name = self.ID().Pretty()
		//} else {
		//	name = req.Arguments[0]
		//}

		//recursive, _ := req.Options[recursiveOptionName].(bool)
		//rc, rcok := req.Options[dhtRecordCountOptionName].(int)
		//dhtt, dhttok := req.Options[dhtTimeoutOptionName].(string)
		//stream, _ := req.Options[streamOptionName].(bool)

		//opts := []options.NameResolveOption{
		//	options.Name.Local(local),
		//	options.Name.Cache(!nocache),
		//}

		//if !recursive {
		//	opts = append(opts, options.Name.ResolveOption(nsopts.Depth(1)))
		//}
		//if rcok {
		//	opts = append(opts, options.Name.ResolveOption(nsopts.DhtRecordCount(uint(rc))))
		//}
		//if dhttok {
		//	d, err := time.ParseDuration(dhtt)
		//	if err != nil {
		//		return err
		//	}
		//	if d < 0 {
		//		return errors.New("DHT timeout value must be >= 0")
		//	}
		//	opts = append(opts, options.Name.ResolveOption(nsopts.DhtTimeout(d)))
		//}

		////"/ipns" -> "/ipus", wyong, 20181212 
		//if !strings.HasPrefix(name, "/ipus/") {
		//	name = "/ipus/" + name
		//}

		//if !stream {
		//	output, err := api.Name().Resolve(req.Context, name, opts...)
		//	if err != nil {
		//		return err
		//	}

		//	return cmds.EmitOnce(res, &ResolvedPath{path.FromString(output.String())})
		//}

		log.Debugf("BidGetCmd, Run(10)") 
		url := req.Arguments[0]
		//if !IsURL(url){
		//	return fmt.Errorf("unsupported url syntax: %s", url)
		//}

		//TODO, get url from Arguments[0], wyong, 20190120 
		//peer, err := peer.IDFromString(req.Arguments[0])
		//if err != nil {
		//	return err
		//}

		log.Debugf("BidGetCmd, Run(20)") 
		ce := cmdenv.(*env.Env)
		bids, err := ce.Frontera().GetBids(req.Context,  /* , opts... */ url )
		if err != nil {
			return err
		}

		log.Debugf("BidGetCmd, Run(20)") 
		//output := make([]cid.Cid, len(bids))
		for _, b := range bids{
			//if v.Err != nil {
			//	return err
			//}
			//if err := res.Emit(&ResolvedPath{path.FromString(v.Path.String())}); err != nil {
			log.Debugf("BidGetCmd, Run(30), %s", b.GetCid()) 

			if err := res.Emit(&BidResult{Cid: b.GetCid()}); err != nil  {
				return err
			}

		}

		log.Debugf("BidGetCmd, Run(40)") 
		return nil
	},

	//Encoders: cmds.EncoderMap{
	//	cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, rp *ResolvedPath) error {
	//		_, err := fmt.Fprintln(w, rp.Path)
	//		return err
	//	}),
	//},

	Type: BidResult{},
}


//wyong, 20190322 
var BidPutDataCmd = &cmds.Command{
        Helptext: cmdkit.HelpText{
                Tagline: "Import data into the local node",
                ShortDescription: `
Imports data previously exported with the client cat command into the storage
market. This command takes only one argument, the path of the file to import.
See the go-filecoin client cat command for more details.
`,
        },
        Arguments: []cmdkit.Argument{
                cmdkit.FileArg("file", true, false, "Path to file to import").EnableStdin(),
        },
        Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		//todo, wyong, 20200802 
                //iter := req.Files.Entries()
                //if !iter.Next() {
                //        return fmt.Errorf("no file given: %s", iter.Err())
                //}

                //fi, ok := iter.Node().(files.File)
                //if !ok {
                //        return fmt.Errorf("given file was not a files.File")
                //}

		//ce := cmdenv.(env.Env)
                //out, err := ce.Frontera().DAGImportData(req.Context, fi)
                //if err != nil {
                //        return err
                //}

                //return re.Emit(out.Cid())
		
		return nil  

        },
        Type: cid.Cid{},
        Encoders: cmds.EncoderMap{
                cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, c cid.Cid) error {
                        return PrintString(w, c)
                }),
        },
}

