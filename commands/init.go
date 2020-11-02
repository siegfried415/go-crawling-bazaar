package commands

import (
	//"context"
	"fmt"
	//"io"
	"io/ioutil"
	//"net/http"
	//"net/url"
	"os"


        //wyong, 20201022
	"strings" 
	"errors"
        "path/filepath"
	"encoding/hex"
        yaml "gopkg.in/yaml.v2"


	//"github.com/ipfs/go-car"
	//"github.com/ipfs/go-hamt-ipld"
	//"github.com/ipfs/go-ipfs-blockstore"
	"github.com/ipfs/go-ipfs-cmdkit"
	"github.com/ipfs/go-ipfs-cmds"

	//wyong, 20201022
	"github.com/siegfried415/gdf-rebuild/conf/testnet"

	//wyong, 20201005 
	"github.com/siegfried415/gdf-rebuild/crypto"
	"github.com/siegfried415/gdf-rebuild/crypto/asymmetric"

	//wyong, 20201022 
	kms "github.com/siegfried415/gdf-rebuild/kms"
	proto "github.com/siegfried415/gdf-rebuild/proto"
        //mine "github.com/siegfried415/gdf-rebuild/pow/cpuminer"

	//"github.com/siegfried415/gdf-rebuild/address"
	"github.com/siegfried415/gdf-rebuild/conf"
	//"github.com/siegfried415/gdf-rebuild/consensus"
	//"github.com/siegfried415/gdf-rebuild/fixtures"

	//node "github.com/siegfried415/gdf-rebuild/node"
	"github.com/siegfried415/gdf-rebuild/paths"

	//wyong, 20201027 
	//"github.com/siegfried415/gdf-rebuild/repo"

	//"github.com/siegfried415/gdf-rebuild/types"
)

const (
        testnetCN = "cn"
        testnetW  = "w"
)

var initCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Initialize a filecoin repo",
		ShortDescription:     "generate a folder contains config file and private key",
		LongDescription: `
Generates private.key and config.yaml for go-decentralized-frontera.
You can input a passphrase for local encrypt your private key file by set -with-password
e.g.
	gdf init 

or input a passphrase by

	gdf init -with-password
`,
	},
	
	//todo, wyong, 20201002 
	//Options: []cmdkit.Option{
	//	cmdkit.StringOption(GenesisFile, "path of file or HTTP(S) URL containing archive of genesis block DAG data"),
	//	cmdkit.StringOption(PeerKeyFile, "path of file containing key to use for new node's libp2p identity"),
	//	cmdkit.StringOption(WithMiner, "when set, creates a custom genesis block with a pre generated miner account, requires running the daemon using dev mode (--dev)"),
	//	cmdkit.StringOption(OptionSectorDir, "path of directory into which staged and sealed sectors will be written"),
	//	cmdkit.StringOption(DefaultAddress, "when set, sets the daemons's default address to the provided address"),
	//	cmdkit.UintOption(AutoSealIntervalSeconds, "when set to a number > 0, configures the daemon to check for and seal any staged sectors on an interval.").WithDefault(uint(120)),
	//	cmdkit.BoolOption(DevnetStaging, "when set, populates config bootstrap addrs with the dns multiaddrs of the staging devnet and other staging devnet specific bootstrap parameters."),
	//	cmdkit.BoolOption(DevnetNightly, "when set, populates config bootstrap addrs with the dns multiaddrs of the nightly devnet and other nightly devnet specific bootstrap parameters"),
	//	cmdkit.BoolOption(DevnetUser, "when set, populates config bootstrap addrs with the dns multiaddrs of the user devnet and other user devnet specific bootstrap parameters"),
	//},

	Options: []cmdkit.Option{
		cmdkit.StringOption("PrivateKeyParam", "Generate config using an existing private key"), 
		cmdkit.StringOption("Source", "Generate config using the specified config template") ,
		cmdkit.StringOption("MinerListenAddr", "Generate miner config with specified miner address. Conflict with -source param"), 
		cmdkit.StringOption("TestnetRegion", "Generate config using the specified testnet region: cn or w. Default cn. Conflict with -source param"), 
	}, 

	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {


		repoDir, _ := req.Options[OptionRepoDir].(string)	
		repoDir, err := paths.GetRepoPath(repoDir)
		if err != nil {
			return err
		}

		if err := re.Emit(fmt.Sprintf("initializing filecoin node at %s\n", repoDir)); err != nil {
			return err
		}

		// wyong, 20200921 
		//genesisFileSource, _ := req.Options[GenesisFile].(string)
		// Writing to the repo here is messed up; this should create a genesis init function that
		// writes to the repo when invoked.
		//genesisFile, err := loadGenesis(req.Context, rep, genesisFileSource)
		//if err != nil {
		//	return err
		//}

		//peerKeyFile, _ := req.Options[PeerKeyFile].(string)
		//initopts, err := getNodeInitOpts(peerKeyFile)
		//if err != nil {
		//	return err
		//}

		//if err := node.Init(req.Context, rep, // genesisFile,
		//					initopts...); err != nil {
		//	return err
		//}

		//cfg := rep.Config()
		//if err := setConfigFromOptions(cfg, req.Options); err != nil {
		//	return err
		//}
		//if err := rep.ReplaceConfig(cfg); err != nil {
		//	return err
		//}
		//return nil


		//var workingRoot string
		//if len(req.Arguments) == 0 {
		//	workingRoot = utils.HomeDirExpand("~/.cql")
		//} else if req.Arguments[0] == "" {
		//	workingRoot = utils.HomeDirExpand("~/.cql")
		//} else {
		//	workingRoot = utils.HomeDirExpand(req.Arguments[0])
		//}

		//if workingRoot == "" {
		//	ConsoleLog.Error("config directory is required for generate config")
		//	SetExitStatus(1)
		//	return 
		//}

		//if strings.HasSuffix(workingRoot, "config.yaml") {
		//	workingRoot = filepath.Dir(workingRoot)
		//}

		privateKeyFileName := "private.key"
		privateKeyFile := filepath.Join(repoDir, privateKeyFileName)

		var (
			privateKey *asymmetric.PrivateKey
			//err        error

			//wyong, 20201022 
			//privateKeyParam string
			//source          string
			//minerListenAddr string
			//testnetRegion   string

			//password string 
			//withPassword bool 
		)

		//wyong, 20201027 
		password, _ := req.Options["password"].(string)

		//wyong, 20201022 
		withPassword, _ := req.Options["with-password"].(bool) 
		//if withPasswordStr == "yes" { 
		//	withPassword = true 	
		//}

		// detect customized private key
		privateKeyParam, _ := req.Options["PrivateKeyParam"].(string)  
		if privateKeyParam != "" {
			var (
				oldPassword string
			)

			if password == "" {
				fmt.Println("Please enter the passphrase of the existing private key")
				oldPassword = readMasterKey(!withPassword )
			} else {
				oldPassword = password 
			}

			privateKey, err = kms.LoadPrivateKey(privateKeyParam, []byte(oldPassword))

			if err != nil {
				//ConsoleLog.WithError(err).Error("load specified private key failed")
				//SetExitStatus(1)
				return err 
			}
		}

		var port string
		minerListenAddr, _ := req.Options["MinerListenAddr"].(string)
		if minerListenAddr != "" {
			minerListenAddrSplit := strings.Split(minerListenAddr, ":")
			if len(minerListenAddrSplit) != 2 {
				//ConsoleLog.Error("-miner only accepts listen address in ip:port format. e.g. 127.0.0.1:7458")
				//SetExitStatus(1)
				return errors.New("-miner only accepts listen address in ip:port format. e.g. 127.0.0.1:7458")
			}
			port = minerListenAddrSplit[1]
		}

		var rawConfig *conf.Config
		source, _ := req.Options["Source"].(string)
		if source == "" {
			fmt.Printf("Generating testnet %s config\n", req.Options["TestnetRegion"])

			// Load testnet config
			rawConfig = testnet.GetTestNetConfig()
			if minerListenAddr != "" {
				testnet.SetMinerConfig(rawConfig)
				rawConfig.ListenAddr = "0.0.0.0:" + port
			}

			testnetRegion , _ := req.Options["TestnetRegion"].(string) 
			if testnetRegion == testnetW {
				rawConfig.DNSSeed.BPCount = 3
				rawConfig.DNSSeed.Domain = "testnetw.gridb.io"
			}
		} else {
			// Load from template file
			fmt.Printf("Generating config base on %s templete\n", req.Options["Source"])
			sourceConfig, err := ioutil.ReadFile(source)
			if err != nil {
				//ConsoleLog.WithError(err).Error("read config template failed")
				//SetExitStatus(1)
				return err 
			}
			rawConfig = &conf.Config{}
			if err = yaml.Unmarshal(sourceConfig, rawConfig); err != nil {
				//ConsoleLog.WithError(err).Error("load config template failed")
				//SetExitStatus(1)
				return err 
			}
		}

		var fileinfo os.FileInfo
		if fileinfo, err = os.Stat(repoDir); err == nil {
			if fileinfo.IsDir() {
				err = filepath.Walk(repoDir, func(filepath string, f os.FileInfo, err error) error {
					if f == nil {
						return err
					}
					if f.IsDir() {
						return nil
					}
					if strings.Contains(f.Name(), "config.yaml") ||
						strings.Contains(f.Name(), "private.key") ||
						strings.Contains(f.Name(), "public.keystore") ||
						strings.Contains(f.Name(), ".dsn") {
						askDeleteFile(filepath)
					}
					return nil
				})
			} else {
				//wyong, 20201022 
				//askDeleteFile(workingRoot)
				askDeleteFile(repoDir )
			}
		}

		err = os.Mkdir(repoDir, 0755)
		if err != nil && !os.IsExist(err) {
			//ConsoleLog.WithError(err).Error("unexpected error")
			//SetExitStatus(1)
			return err
		}

		fmt.Println("Generating private key...")
		if password == "" {
			fmt.Println("Please enter passphrase for new private key")
			password = readMasterKey(!withPassword)
		}

		if privateKeyParam == "" {
			privateKey, _, err = asymmetric.GenSecp256k1KeyPair()
			if err != nil {
				//ConsoleLog.WithError(err).Error("generate key pair failed")
				//SetExitStatus(1)
				return err
			}
		}

		if err = kms.SavePrivateKey(privateKeyFile, privateKey, []byte(password)); err != nil {
			//ConsoleLog.WithError(err).Error("save generated keypair failed")
			//SetExitStatus(1)
			return err
		}
		fmt.Println("Generated private key.")

		publicKey := privateKey.PubKey()
		keyHash, err := crypto.PubKeyHash(publicKey)
		if err != nil {
			//ConsoleLog.WithError(err).Error("unexpected error")
			//SetExitStatus(1)
			return err
		}

		walletAddr := keyHash.String()

		fmt.Println("Generating nonce...")

		//wyong, 20201027 
		//var nonce *mine.NonceInfo 
		nonce := nonceGen(publicKey)

		cliNodeID := proto.NodeID(nonce.Hash.String())
		fmt.Println("Generated nonce.")

		fmt.Println("Generating config file...")

		// Add client config
		rawConfig.PrivateKeyFile = privateKeyFileName
		rawConfig.WalletAddress = walletAddr
		rawConfig.ThisNodeID = cliNodeID
		if rawConfig.KnownNodes == nil {
			rawConfig.KnownNodes = make([]proto.Node, 0, 1)
		}
		node := proto.Node{
			ID:        cliNodeID,
			Role:      proto.Client,

			//wyong, 20201028 
			Addr:      "/ip4/127.0.0.0/tcp/15151",

			PublicKey: publicKey,
			Nonce:     nonce.Nonce,
		}
		if minerListenAddr != "" {
			node.Role = proto.Miner
			node.Addr = minerListenAddr
		}
		rawConfig.KnownNodes = append(rawConfig.KnownNodes, node)

		// Write config
		configFilePath := filepath.Join(repoDir, "config.yaml")
		err = rawConfig.WriteFile(configFilePath)
		if err != nil {
			//ConsoleLog.WithError(err).Error("unexpected error")
			//SetExitStatus(1)
			return err
		}

		//wyong, 20201027 
		//create repo 
		//if err := repo.InitFSRepo(repoDir, repo.Version, rawConfig ); err != nil {
		//	return err
		//}
		//rep, err := repo.OpenFSRepo(repoDir, repo.Version)
		//if err != nil {
		//	return err
		//}
		//// The only error Close can return is that the repo has already been closed.
		//defer func() { _ = rep.Close() }()


		fmt.Println("Generated config.")
		fmt.Printf("\nConfig file:      %s\n", configFilePath)
		fmt.Printf("Private key file: %s\n", privateKeyFile)
		fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(publicKey.Serialize()))

		fmt.Printf("\nWallet address: %s\n", walletAddr)
		fmt.Printf(`
	Any further command could costs PTC.
	You can get some free PTC from:
		https://testnet.covenantsql.io/wallet/`)
		fmt.Println(walletAddr)

		if password != "" {
			fmt.Println("Your private key had been encrypted by a passphrase, add -with-password in any further command")
		}

		return nil 

	},

	//todo, wyong, 20201022 
	//Encoders: cmds.EncoderMap{
	//	cmds.Text: cmds.MakeEncoder(initTextEncoder),
	//},
}

/* wyong, 20201002 
func setConfigFromOptions(cfg *config.Config, options cmdkit.OptMap) error {
	var err error
	if dir, ok := options[OptionSectorDir].(string); ok {
		cfg.SectorBase.RootDir = dir
	}

	if m, ok := options[WithMiner].(string); ok {
		if cfg.Mining.MinerAddress, err = address.NewFromString(m); err != nil {
			return err
		}
	}

	if autoSealIntervalSeconds, ok := options[AutoSealIntervalSeconds]; ok {
		cfg.Mining.AutoSealIntervalSeconds = autoSealIntervalSeconds.(uint)
	}

	if m, ok := options[DefaultAddress].(string); ok {
		if cfg.Wallet.DefaultAddress, err = address.NewFromString(m); err != nil {
			return err
		}
	}

	devnetTest, _ := options[DevnetStaging].(bool)
	devnetNightly, _ := options[DevnetNightly].(bool)
	devnetUser, _ := options[DevnetUser].(bool)
	if (devnetTest && devnetNightly) || (devnetTest && devnetUser) || (devnetNightly && devnetUser) {
		return fmt.Errorf(`cannot specify more than one "devnet-" option`)
	}

	// Setup devnet specific config options.
	if devnetTest || devnetNightly || devnetUser {
		cfg.Bootstrap.MinPeerThreshold = 1
		cfg.Bootstrap.Period = "10s"
	}

	//todo, wyong, 20200922 
	// Setup devnet staging specific config options.
	//if devnetTest {
	//	cfg.Bootstrap.Addresses = fixtures.DevnetStagingBootstrapAddrs
	//}

	//// Setup devnet nightly specific config options.
	//if devnetNightly {
	//	cfg.Bootstrap.Addresses = fixtures.DevnetNightlyBootstrapAddrs
	//}

	//// Setup devnet user specific config options.
	//if devnetUser {
	//	cfg.Bootstrap.Addresses = fixtures.DevnetUserBootstrapAddrs
	//}

	return nil
}

func initTextEncoder(_ *cmds.Request, w io.Writer, val interface{}) error {
	_, err := fmt.Fprintf(w, val.(string))
	return err
}


func loadGenesis(ctx context.Context, rep repo.Repo, sourceName string) (consensus.GenesisInitFunc, error) {
	if sourceName == "" {
		return consensus.MakeGenesisFunc(consensus.ProofsMode(types.LiveProofsMode)), nil
	}

	sourceURL, err := url.Parse(sourceName)
	if err != nil {
		return nil, fmt.Errorf("invalid filepath or URL for genesis file: %s", sourceURL)
	}

	var source io.ReadCloser
	if sourceURL.Scheme == "http" || sourceURL.Scheme == "https" {
		// NOTE: This code is temporary. It allows downloading a genesis block via HTTP(S) to be able to join a
		// recently deployed staging devnet.
		response, err := http.Get(sourceName)
		if err != nil {
			return nil, err
		}
		source = response.Body
	} else if sourceURL.Scheme != "" {
		return nil, fmt.Errorf("unsupported protocol for genesis file: %s", sourceURL.Scheme)
	} else {
		file, err := os.Open(sourceName)
		if err != nil {
			return nil, err
		}
		source = file
	}
	defer func() { _ = source.Close() }()

	bs := blockstore.NewBlockstore(rep.Datastore())
	ch, err := car.LoadCar(bs, source)
	if err != nil {
		return nil, err
	}

	if len(ch.Roots) != 1 {
		return nil, fmt.Errorf("expected car with only a single root")
	}

	gif := func(cst *hamt.CborIpldStore, bs blockstore.Blockstore) (*types.Block, error) {
		var blk types.Block

		if err := cst.Get(ctx, ch.Roots[0], &blk); err != nil {
			return nil, err
		}

		return &blk, nil
	}

	return gif, nil
}

func getNodeInitOpts(peerKeyFile string) ([]node.InitOpt, error) {
	var initOpts []node.InitOpt
	if peerKeyFile != "" {
		data, err := ioutil.ReadFile(peerKeyFile)
		if err != nil {
			return nil, err
		}
		peerKey, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return nil, err
		}
		initOpts = append(initOpts, node.PeerKeyOpt(peerKey))
	}

	return initOpts, nil
}
*/
