
package commands

import (
	//"errors"
	"fmt"
	//"io"
	//"time"
	"strconv" 
	"context" 

	//wyong, 20190320
	//cid "github.com/ipfs/go-cid"

	cmds "github.com/ipfs/go-ipfs-cmds"
	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	//logging "github.com/ipfs/go-log"
	
	client "github.com/siegfried415/go-crawling-bazaar/client" 
	//conf "github.com/siegfried415/go-crawling-bazaar/conf"  

	//wyong, 20201022 
	env "github.com/siegfried415/go-crawling-bazaar/env"  

	//wyong, 20201215 
	log "github.com/siegfried415/go-crawling-bazaar/utils/log"  

)

const(
	waitTxConfirmationOptionName = "wait" 
)

/*
var (
	waitTxConfirmationMaxDuration = 20 * conf.GConf.BPPeriod
)
*/

//var log = logging.Logger("core/commands/bidding")

var domainCmd = &cmds.Command{
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
		"create": DomainCreateCmd,
		"drop": DomainDropCmd,	
		//"get": BiddingGetCmd,
	},
}

var DomainCreateCmd = &cmds.Command{
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
		cmdkit.StringArg("name", true, false, "url of the domain."),
		cmdkit.StringArg("count", true, false, "how many miners need to create the domain."),
	},

	Options: []cmdkit.Option{
		cmdkit.BoolOption(waitTxConfirmationOptionName, "Resolve given path before publishing.").WithDefault(true),

		//cmds.StringOption(lifeTimeOptionName, "t",
		//	`Time duration that the record will be valid for. <<default>>
//    This accepts durations such as "300s", "1.5h" or "2h45m". Valid time units are
//    "ns", "us" (or "µs"), "ms", "s", "m", "h".`).WithDefault("24h"),
		//cmds.BoolOption(allowOfflineOptionName, "When offline, save the IPNS record to the the local datastore without broadcasting to the network instead of simply failing."),
		//cmds.StringOption(ttlOptionName, "Time duration this record should be cached for (caution: experimental)."),
		//cmds.StringOption(keyOptionName, "k", "Name of the key to be used or a valid PeerID, as listed by 'ipfs key list -l'. Default: <<default>>.").WithDefault("self"),
		//cmds.BoolOption(quieterOptionName, "Q", "Write only final hash."),

	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, cmdenv cmds.Environment) error {
		log.Debugf("DomainCreateCmd, called...\n") 

		//log.Debugf("BiddingPutCmd, Run(10)") 
		domain_name := req.Arguments[0]
		log.Debugf("DomainCreateCmd, domain_name =%s\n", domain_name ) 

		nodeCnt, err := strconv.ParseInt(req.Arguments[1], 10, 32) 
		if err != nil {
			return fmt.Errorf("unsupported nodeCnt syntax: %s", req.Arguments[1])
		}


		log.Debugf("DomainCreateCmd, nodeCnt=%d\n", nodeCnt ) 

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


		//wyong, 20200728
		var meta = client.ResourceMeta{}
		meta.Node = uint16(nodeCnt)

		//wyong, 20200816 
		meta.Domain = domain_name 

		//var dsn string
		//todo, get host from env, wyong, 20201008 
		e := cmdenv.(*env.Env)
		host := e.Host()
		
		txHash, err := client.CreateDomain(host, meta)
		if err != nil {
			return err 
		}

		waitTxConfirmation, _ := req.Options[waitTxConfirmationOptionName].(bool)
		if waitTxConfirmation {
			err = wait(host, txHash)
			if err != nil {
				//ConsoleLog.WithError(err).Error("create database failed durating bp creation")
				//SetExitStatus(1)
				return err 
			}
			log.Debugf("\nThe domain is accecpted by presbyterian\n")

			var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
			defer cancel()
			err = client.WaitDomainCreation(ctx, host, domain_name )
			if err != nil {
				//ConsoleLog.WithError(err).Error("create database failed durating miner creation")
				//SetExitStatus(1)
				return err 
			}
		}

		//todo, wyong, 20200813 
		//var cfg *client.Config
		// cfg 
		//_, err = client.ParseDSN(dsn)
		//if err != nil {
		//	return err 
		//}
		//dbID = cfg.DatabaseID

		//return cmds.EmitOnce(res, &IpnsEntry{
		//	Name:  out.Name(),
		//	Value: out.Value().String(),
		//})

		//log.Debugf("BiddingPutCmd, Run(40)") 
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

var DomainDropCmd = &cmds.Command{
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
		cmdkit.StringArg("name", true, false, "url of the bidding to be published.").EnableStdin(),
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
		//log.Debugf("BiddingDelCmd, Run() called...") 

		//log.Debugf("BiddingDelCmd, Run(10)") 
		dsn := req.Arguments[0]
		//if !IsURL(url){
		//	return fmt.Errorf("unsupported url syntax: %s", url)
		//}

		//log.Debugf("BiddingDelCmd, Run(20)") 
		//err := GetBiddingsysAPI(env).DelBidding(req.Context, url )
		//if err != nil {
		//	//log.Debugf("BiddingDelCmd, Run(30)") 
		//	//if err == iface.ErrOffline {
		//	//	err = errAllowOffline
		//	//}
		//	return err
		//}

		//todo, wyong, 20200728 
		//dsn := args[0]

		// drop database
		if _, err := client.ParseDSN(dsn); err != nil {
			// not a dsn/dbid
			//ConsoleLog.WithField("db", dsn).WithError(err).Error("not a valid dsn")
			//SetExitStatus(1)
			return err 
		}

		//txHash, err := client.Drop(dsn)
		_, err := client.Drop(dsn)
		if err != nil {
			// drop database failed
			//ConsoleLog.WithField("db", dsn).WithError(err).Error("drop database failed")
			//SetExitStatus(1)
			return err 
		}

		//todo, wyong, 20200802 
		//if waitTxConfirmation {
		//	err = client.wait(txHash)
		//	if err != nil {
		//		ConsoleLog.WithField("db", dsn).WithError(err).Error("drop database failed")
		//		SetExitStatus(1)
		//		return
		//	}
		//}

		//return cmds.EmitOnce(res, &IpnsEntry{
		//	Name:  out.Name(),
		//	Value: out.Value().String(),
		//})

		//log.Debugf("BiddingDelCmd, Run(40)") 
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
