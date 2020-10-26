package commands

import (
	//"errors"
	"fmt"
	"io"
	//"time"

	//wyong, 20190320
	//cid "github.com/ipfs/go-cid"

	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	//logging "github.com/ipfs/go-log"

	//wyong, 20200802 
	env "github.com/siegfried415/gdf-rebuild/env" 
)

type BiddingEntry struct {
	Name  string
	Value string
}

//var log = logging.Logger("core/commands/bidding")

var biddingCmd = &cmds.Command{
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
		"put": BiddingPutCmd,
		"del": BiddingDelCmd,	//wyong, 20190131 
		"get": BiddingGetCmd,
		"get_completed": CompletedBiddingGetCmd,
		"get_uncompleted": UncompletedBiddingGetCmd,//wyong, 20190131
		"get_data":BiddingGetDataCmd, 		    //wyong, 20190320 
	},
}

/*
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
	biddingUrlOptionName   = "bidding-url"
)


// IsURL returns true if the string represents a valid URL that the
// urlstore can handle.  More specifically it returns true if a string
// begins with 'http://' or 'https://'.
func IsURL(str string) bool {
	return (len(str) > 7 && str[0] == 'h' && str[1] == 't' && str[2] == 't' && str[3] == 'p') &&
			((len(str) > 8 && str[4] == 's' && str[5] == ':' && str[6] == '/' && str[7] == '/') ||
					(str[4] == ':' && str[5] == '/' && str[6] == '/'))
}

*/

var BiddingPutCmd = &cmds.Command{
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
		cmdkit.StringArg(biddingUrlOptionName, true, false, "url of the bidding to be published.").EnableStdin(),
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

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		log.Debugf("BiddingPutCmd, Run() called...") 

		log.Debugf("BiddingPutCmd, Run(10)") 
		url := req.Arguments[0]
		if !IsURL(url){
			return fmt.Errorf("unsupported url syntax: %s", url)
		}


		log.Debugf("BiddingPutCmd, Run(20)") 

		//todo, send bidding request with client to random peers , wyong, 20200723 
		//err := env.GetFrontera().PutBidding(req.Context, url )
		//if err != nil {
		//	log.Debugf("BiddingPutCmd, Run(30)") 
		//	
		//	//if err == iface.ErrOffline {
		//	//	err = errAllowOffline
		//	//}
		//	return err
		//}


		//return cmds.EmitOnce(res, &IpnsEntry{
		//	Name:  out.Name(),
		//	Value: out.Value().String(),
		//})

		log.Debugf("BiddingPutCmd, Run(40)") 
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

type BiddingUrl struct {
	Url string  
}

var BiddingGetCmd = &cmds.Command{
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

	//TODO, wyong, 20181227
	//Arguments: []cmds.Argument{
	//	cmds.StringArg("peer", false, false, "Get biddings issued from peer. return all biddings received if no peer given."),
	//},

	//Options: []cmds.Option{
	//	cmds.BoolOption(recursiveOptionName, "r", "Resolve until the result is not an IPNS name."),
	//	cmds.BoolOption(nocacheOptionName, "n", "Do not use cached entries."),
	//	cmds.UintOption(dhtRecordCountOptionName, "dhtrc", "Number of records to request for DHT resolution."),
	//	cmds.StringOption(dhtTimeoutOptionName, "dhtt", "Max time to collect values during DHT resolution eg \"30s\". Pass 0 for no timeout."),
	//	cmds.BoolOption(streamOptionName, "s", "Stream entries as they are found."),
	//},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		fmt.Printf("BiddingGetCmd/Run(10)\n") 

		//TODO,wyong, 20181227
		//peer, err := peer.IDFromString(req.Arguments[0])
		//if err != nil {
		//	return err
		//}

		fmt.Printf("BiddingGetCmd/Run(20)\n") 
		ce := cmdenv.(*env.Env) 

		output, err := ce.Frontera().GetBidding(req.Context /* , opts... */ )
		if err != nil {
			fmt.Printf("BiddingGetCmd/Run(25), err=%s\n", err.Error()) 
			return err
		}

		fmt.Printf("BiddingGetCmd/Run(30)\n") 
		if len(output) ==  0 {
			fmt.Printf("BiddingGetCmd/Run(40)\n") 
			if err := res.Emit(&BiddingUrl{Url:""}); err != nil  {
				return err
			}
		}else {
			for _, v := range output {
				//if v.Err != nil {
				//	return err
				//}
				//if err := res.Emit(&ResolvedPath{path.FromString(v.Path.String())}); err != nil {
				fmt.Printf("BiddingGetCmd/Run(50), %s\n", v.GetUrl()) 

				if err := res.Emit(&BiddingUrl{v.GetUrl()}); err != nil  {
					return err
				}

			}
		}

		fmt.Printf("BiddingGetCmd/Run(60)\n") 
		return nil
	},

	//wyong, 20200903 
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeEncoder(func(req *cmds.Request, w io.Writer, v interface{} ) error {
			
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

//var log = logging.Logger("core/commands/bidding")
type CompletedBidding struct {
	Url string  
	//Cid []cid.Cid 
}


var CompletedBiddingGetCmd = &cmds.Command{
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

	//TODO, wyong, 20181227
	//Arguments: []cmds.Argument{
	//	cmds.StringArg("peer", false, false, "Get biddings issued from peer. return all biddings received if no peer given."),
	//},

	//Options: []cmds.Option{
	//	cmds.BoolOption(recursiveOptionName, "r", "Resolve until the result is not an IPNS name."),
	//	cmds.BoolOption(nocacheOptionName, "n", "Do not use cached entries."),
	//	cmds.UintOption(dhtRecordCountOptionName, "dhtrc", "Number of records to request for DHT resolution."),
	//	cmds.StringOption(dhtTimeoutOptionName, "dhtt", "Max time to collect values during DHT resolution eg \"30s\". Pass 0 for no timeout."),
	//	cmds.BoolOption(streamOptionName, "s", "Stream entries as they are found."),
	//},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		log.Debugf("CompletedBiddingGetCmd, Run() called...") 

		//TODO,wyong, 20181227
		//peer, err := peer.IDFromString(req.Arguments[0])
		//if err != nil {
		//	return err
		//}

		log.Debugf("CompletedBiddingGetCmd, Run(10)") 
		ce := cmdenv.(*env.Env) 
		output, err := ce.Frontera().GetCompletedBiddings(req.Context /* , opts... */ )
		if err != nil {
			return err
		}

		log.Debugf("CompletedBiddingGetCmd, Run(20)") 
		for _, v := range output {
			//if v.Err != nil {
			//	return err
			//}
			//if err := res.Emit(&ResolvedPath{path.FromString(v.Path.String())}); err != nil {
			log.Debugf("CompletedBiddingGetCmd, Run(30), %s", v.GetUrl()) 

			//cids := make([]cid.Cid, len(v.Bids()) 
			//for _, v := range v.Bids() {
			//	cids = append(cids, v.Cid()) 
			//}

			if err := res.Emit(&CompletedBidding{Url:v.GetUrl() /* , cid: cids */ }); err != nil  {
				return err
			}

		}

		log.Debugf("CompletedBiddingGetCmd, Run(40)") 
		return nil
	},

	//Encoders: cmds.EncoderMap{
	//	cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, rp *ResolvedPath) error {
	//		_, err := fmt.Fprintln(w, rp.Path)
	//		return err
	//	}),
	//},

	Type: CompletedBidding{},
}

type UncompletedBidding struct {
	Url string  
	//Cid []cid.Cid 
}


var UncompletedBiddingGetCmd = &cmds.Command{
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

	//TODO, wyong, 20181227
	//Arguments: []cmds.Argument{
	//	cmds.StringArg("peer", false, false, "Get biddings issued from peer. return all biddings received if no peer given."),
	//},

	//Options: []cmds.Option{
	//	cmds.BoolOption(recursiveOptionName, "r", "Resolve until the result is not an IPNS name."),
	//	cmds.BoolOption(nocacheOptionName, "n", "Do not use cached entries."),
	//	cmds.UintOption(dhtRecordCountOptionName, "dhtrc", "Number of records to request for DHT resolution."),
	//	cmds.StringOption(dhtTimeoutOptionName, "dhtt", "Max time to collect values during DHT resolution eg \"30s\". Pass 0 for no timeout."),
	//	cmds.BoolOption(streamOptionName, "s", "Stream entries as they are found."),
	//},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		log.Debugf("CompletedBiddingGetCmd, Run() called...") 

		//TODO,wyong, 20181227
		//peer, err := peer.IDFromString(req.Arguments[0])
		//if err != nil {
		//	return err
		//}

		log.Debugf("UncompletedBiddingGetCmd, Run(10)") 
		ce := cmdenv.(*env.Env)
		output, err := ce.Frontera().GetUncompletedBiddings(req.Context /* , opts... */ )
		if err != nil {
			return err
		}

		log.Debugf("UncompletedBiddingGetCmd, Run(20)") 
		for _, v := range output {
			//if v.Err != nil {
			//	return err
			//}
			//if err := res.Emit(&ResolvedPath{path.FromString(v.Path.String())}); err != nil {
			log.Debugf("UncompletedBiddingGetCmd, Run(30), %s", v.GetUrl()) 

			//cids := make([]cid.Cid, len(v.Bids()) 
			//for _, v := range v.Bids() {
			//	cids = append(cids, v.Cid()) 
			//}

			if err := res.Emit(&UncompletedBidding{Url:v.GetUrl() /* , cid: cids */ }); err != nil  {
				return err
			}

		}

		log.Debugf("UncompletedBiddingGetCmd, Run(40)") 
		return nil
	},

	//Encoders: cmds.EncoderMap{
	//	cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, rp *ResolvedPath) error {
	//		_, err := fmt.Fprintln(w, rp.Path)
	//		return err
	//	}),
	//},

	Type: UncompletedBidding{},
}


var BiddingDelCmd = &cmds.Command{
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
		cmdkit.StringArg(biddingUrlOptionName, true, false, "url of the bidding to be published.").EnableStdin(),
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
		log.Debugf("BiddingDelCmd, Run() called...") 

		log.Debugf("BiddingDelCmd, Run(10)") 
		url := req.Arguments[0]
		if !IsURL(url){
			return fmt.Errorf("unsupported url syntax: %s", url)
		}


		log.Debugf("BiddingDelCmd, Run(20)") 
		ce := cmdenv.(*env.Env)
		err := ce.Frontera().DelBidding(req.Context, url )
		if err != nil {
			log.Debugf("BiddingDelCmd, Run(30)") 
			//if err == iface.ErrOffline {
			//	err = errAllowOffline
			//}
			return err
		}

		//return cmds.EmitOnce(res, &IpnsEntry{
		//	Name:  out.Name(),
		//	Value: out.Value().String(),
		//})

		log.Debugf("BiddingDelCmd, Run(40)") 
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

/*
var BiddingGetDataCmd = &cmds.Command{
        Helptext: cmdkit.HelpText{
                Tagline: "Read out data stored on the network",
                ShortDescription: `
Prints data from the storage market specified with a given CID to stdout. The
only argument should be the CID to return. The data will be returned in whatever
format was provided with the data initially.
`,
        },
        Arguments: []cmdkit.Argument{
                cmdkit.StringArg("cid", true, false, "CID of data to read"),
        },
        Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
                c, err := cid.Decode(req.Arguments[0])
                if err != nil {
                        return err
                }

                dr, err := GetAPI(env).Biddingsys().GetData(req.Context, c)
                if err != nil {
                        return err
                }

                return re.Emit(dr)
        },
}
*/

var BiddingGetDataCmd = &cmds.Command{
        Helptext: cmdkit.HelpText{
                Tagline: "Read out data stored on the network",
                ShortDescription: `
Prints data from the storage market specified with a given CID to stdout. The
only argument should be the CID to return. The data will be returned in whatever
format was provided with the data initially.
`,
        },
        Arguments: []cmdkit.Argument{
                cmdkit.StringArg("cid", true, false, "CID of data to read"),
        },
        Run: func(req *cmds.Request, re cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		// todo, wyong, 20200802 
                //, err := cid.Decode(req.Arguments[0])
                //f err != nil {
                //       return err
                //}

		//ce := cmdenv.(env.Env)
                //dr, err := ce.Frontera().DAGCat(req.Context, c)
                //if err != nil {
                //        return err
                //}

                //return re.Emit(dr)

		return nil 
        },
}
