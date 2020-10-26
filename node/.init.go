package node

import (
	"context"

	//bserv "github.com/ipfs/go-blockservice"
	//"github.com/ipfs/go-hamt-ipld"
	//bstore "github.com/ipfs/go-ipfs-blockstore"
	//offline "github.com/ipfs/go-ipfs-exchange-offline"
	keystore "github.com/ipfs/go-ipfs-keystore"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"

	//"github.com/siegfried415/gdf-rebuild/chain"
	//"github.com/siegfried415/gdf-rebuild/consensus"
	"github.com/siegfried415/gdf-rebuild/repo"
	"github.com/siegfried415/gdf-rebuild/types"
	"github.com/siegfried415/gdf-rebuild/wallet"

)

/* wyong, 20201002 
const defaultPeerKeyBits = 2048

// initCfg contains configuration for initializing a node's repo.
type initCfg struct {
	peerKey    crypto.PrivKey
	defaultKey *types.KeyInfo
}

// InitOpt is an option for initialization of a node's repo.
type InitOpt func(*initCfg)

// PeerKeyOpt sets the private key for a node's 'self' libp2p identity.
// If unspecified, initialization will create a new one.
func PeerKeyOpt(k crypto.PrivKey) InitOpt {
	return func(opts *initCfg) {
		opts.peerKey = k
	}
}

// DefaultKeyOpt sets the private key for the wallet's default account.
// If unspecified, initialization will create a new one.
func DefaultKeyOpt(ki *types.KeyInfo) InitOpt {
	return func(opts *initCfg) {
		opts.defaultKey = ki
	}
}


// copy code from /gdf/cql/internal/generate.go, todo, wyong, 20201002 
// Init initializes a Filecoin repo with genesis state and keys.
// This will always set the configuration for wallet default address (to the specified default
// key or a newly generated one), but otherwise leave the repo's config object intact.
// Make further configuration changes after initialization.
func Init(ctx context.Context, r repo.Repo, /* gen consensus.GenesisInitFunc,*/  opts ...InitOpt) error {
	cfg := new(initCfg)
	for _, o := range opts {
		o(cfg)
	}

	bs := bstore.NewBlockstore(r.Datastore())
	cst := &hamt.CborIpldStore{Blocks: bserv.New(bs, offline.Exchange(bs))}
	if _, err := chain.Init(ctx, r, bs, cst, gen); err != nil {
		return errors.Wrap(err, "Could not Init Node")
	}

	if err := initPeerKey(r.Keystore(), cfg.peerKey); err != nil {
		return err
	}

	defaultKey, err := initDefaultKey(r.WalletDatastore(), cfg.defaultKey)
	if err != nil {
		return err
	}

	defaultAddress, err := defaultKey.Address()
	if err != nil {
		return errors.Wrap(err, "failed to extract address from default key")
	}
	r.Config().Wallet.DefaultAddress = defaultAddress
	if err = r.ReplaceConfig(r.Config()); err != nil {
		return errors.Wrap(err, "failed to write config")
	}

	return nil
}

func initPeerKey(store keystore.Keystore, key crypto.PrivKey) error {
	var err error

	if key == nil {
		key, _, err = crypto.GenerateKeyPair(crypto.RSA, defaultPeerKeyBits)
		if err != nil {
			return errors.Wrap(err, "failed to create peer key")
		}
	}

	if err := store.Put("self", key); err != nil {
		return errors.Wrap(err, "failed to store private key")
	}

	return nil
}

func initDefaultKey(store repo.Datastore, key *types.KeyInfo) (*types.KeyInfo, error) {
	backend, err := wallet.NewDSBackend(store)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open wallet datastore")
	}
	w := wallet.New(backend)
	if key == nil {
		key, err = w.NewKeyInfo()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create default key")
		}
	} else {
		if _, err := w.Import(key); err != nil {
			return nil, errors.Wrap(err, "failed to import default key")
		}
	}
	return key, nil
}
*/

func runGenerate(cmd *Command, args []string) {
	commonFlagsInit(cmd)

	var workingRoot string
	if len(args) == 0 {
		workingRoot = utils.HomeDirExpand("~/.cql")
	} else if args[0] == "" {
		workingRoot = utils.HomeDirExpand("~/.cql")
	} else {
		workingRoot = utils.HomeDirExpand(args[0])
	}

	if workingRoot == "" {
		ConsoleLog.Error("config directory is required for generate config")
		SetExitStatus(1)
		return
	}

	if strings.HasSuffix(workingRoot, "config.yaml") {
		workingRoot = filepath.Dir(workingRoot)
	}

	privateKeyFileName := "private.key"
	privateKeyFile := path.Join(workingRoot, privateKeyFileName)

	var (
		privateKey *asymmetric.PrivateKey
		err        error
	)

	// detect customized private key
	if privateKeyParam != "" {
		var oldPassword string

		if password == "" {
			fmt.Println("Please enter the passphrase of the existing private key")
			oldPassword = readMasterKey(!withPassword)
		} else {
			oldPassword = password
		}

		privateKey, err = kms.LoadPrivateKey(privateKeyParam, []byte(oldPassword))

		if err != nil {
			ConsoleLog.WithError(err).Error("load specified private key failed")
			SetExitStatus(1)
			return
		}
	}

	var port string
	if minerListenAddr != "" {
		minerListenAddrSplit := strings.Split(minerListenAddr, ":")
		if len(minerListenAddrSplit) != 2 {
			ConsoleLog.Error("-miner only accepts listen address in ip:port format. e.g. 127.0.0.1:7458")
			SetExitStatus(1)
			return
		}
		port = minerListenAddrSplit[1]
	}

	var rawConfig *conf.Config

	if source == "" {
		fmt.Printf("Generating testnet %s config\n", testnetRegion)

		// Load testnet config
		rawConfig = testnet.GetTestNetConfig()
		if minerListenAddr != "" {
			testnet.SetMinerConfig(rawConfig)
			rawConfig.ListenAddr = "0.0.0.0:" + port
		}

		if testnetRegion == testnetW {
			rawConfig.DNSSeed.BPCount = 3
			rawConfig.DNSSeed.Domain = "testnetw.gridb.io"
		}
	} else {
		// Load from template file
		fmt.Printf("Generating config base on %s templete\n", source)
		sourceConfig, err := ioutil.ReadFile(source)
		if err != nil {
			ConsoleLog.WithError(err).Error("read config template failed")
			SetExitStatus(1)
			return
		}
		rawConfig = &conf.Config{}
		if err = yaml.Unmarshal(sourceConfig, rawConfig); err != nil {
			ConsoleLog.WithError(err).Error("load config template failed")
			SetExitStatus(1)
			return
		}
	}

	var fileinfo os.FileInfo
	if fileinfo, err = os.Stat(workingRoot); err == nil {
		if fileinfo.IsDir() {
			err = filepath.Walk(workingRoot, func(filepath string, f os.FileInfo, err error) error {
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
			askDeleteFile(workingRoot)
		}
	}

	err = os.Mkdir(workingRoot, 0755)
	if err != nil && !os.IsExist(err) {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	fmt.Println("Generating private key...")
	if password == "" {
		fmt.Println("Please enter passphrase for new private key")
		password = readMasterKey(!withPassword)
	}

	if privateKeyParam == "" {
		privateKey, _, err = asymmetric.GenSecp256k1KeyPair()
		if err != nil {
			ConsoleLog.WithError(err).Error("generate key pair failed")
			SetExitStatus(1)
			return
		}
	}

	if err = kms.SavePrivateKey(privateKeyFile, privateKey, []byte(password)); err != nil {
		ConsoleLog.WithError(err).Error("save generated keypair failed")
		SetExitStatus(1)
		return
	}
	fmt.Println("Generated private key.")

	publicKey := privateKey.PubKey()
	keyHash, err := crypto.PubKeyHash(publicKey)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	walletAddr := keyHash.String()

	fmt.Println("Generating nonce...")
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
		Addr:      "0.0.0.0:15151",
		PublicKey: publicKey,
		Nonce:     nonce.Nonce,
	}
	if minerListenAddr != "" {
		node.Role = proto.Miner
		node.Addr = minerListenAddr
	}
	rawConfig.KnownNodes = append(rawConfig.KnownNodes, node)

	// Write config
	out, err := yaml.Marshal(rawConfig)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}
	configFilePath := path.Join(workingRoot, "config.yaml")
	err = ioutil.WriteFile(configFilePath, out, 0644)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}
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

}
