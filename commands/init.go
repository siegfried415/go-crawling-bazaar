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
package commands

import (
	"strings" 
	"errors"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
        "path/filepath"

        yaml "gopkg.in/yaml.v2"

	"github.com/ipfs/go-ipfs-cmdkit"
	"github.com/ipfs/go-ipfs-cmds"
	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
        ma "github.com/multiformats/go-multiaddr"
	peer "github.com/libp2p/go-libp2p-core/peer" 

	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/crypto"
	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	log "github.com/siegfried415/go-crawling-bazaar/utils/log"
	kms "github.com/siegfried415/go-crawling-bazaar/kms"
	"github.com/siegfried415/go-crawling-bazaar/paths"
	proto "github.com/siegfried415/go-crawling-bazaar/proto"
)

var initCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Initialize a go-crawling-bazaar repo",
		ShortDescription:     "generate a folder contains config file and private key",
		LongDescription: `
Generates private.key and config.yaml for go-crawling-bazaar.
You can input a passphrase for local encrypt your private key file by set -with-password
e.g.
	gcb init 

or input a passphrase by

	gcb init -with-password
`,
	},
	
	Options: []cmdkit.Option{
		cmdkit.StringOption("PrivateKeyParam", "Generate config using an existing private key"), 
		cmdkit.StringOption("Source", "Generate config using the specified config template") ,
		cmdkit.StringOption("MinerListenAddr", "Generate miner config with specified miner address. Conflict with -source param"), 
	}, 

	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		repoDir, _ := req.Options[OptionRepoDir].(string)	
		repoDir, err := paths.GetRepoPath(repoDir)
		if err != nil {
			return err
		}

		if err := re.Emit(fmt.Sprintf("initializing crawling bazaar node at %s\n", repoDir)); err != nil {
			return err
		}

		privateKeyFileName := "private.key"
		privateKeyFile := filepath.Join(repoDir, privateKeyFileName)

		var (
			privateKey *asymmetric.PrivateKey
		)

		password, _ := req.Options["password"].(string)
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
				log.WithError(err).Error("load specified private key failed")
				//SetExitStatus(1)
				return err 
			}
		}

		host := "127.0.0.1" 
		port := "1551" 
		
		minerListenAddr, _ := req.Options["MinerListenAddr"].(string)
		if minerListenAddr != "" {
			minerListenAddrSplit := strings.Split(minerListenAddr, ":")
			if len(minerListenAddrSplit) != 2 {
				//ConsoleLog.Error("-miner only accepts listen address in ip:port format. e.g. 127.0.0.1:7458")
				//SetExitStatus(1)
				return errors.New("-miner only accepts listen address in ip:port format. e.g. 127.0.0.1:7458")
			}
			host = minerListenAddrSplit[0]
			port = minerListenAddrSplit[1]
		}

		var rawConfig *conf.Config
		source, _ := req.Options["Source"].(string)

		if source != "" {
			// Load from template file
			log.Debugf("Generating config base on %s templete\n", req.Options["Source"])
			sourceConfig, err := ioutil.ReadFile(source)
			if err != nil {
				log.WithError(err).Error("read config template failed")
				return err 
			}
			rawConfig = &conf.Config{}
			if err = yaml.Unmarshal(sourceConfig, rawConfig); err != nil {
				log.WithError(err).Error("load config template failed")
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
				askDeleteFile(repoDir )
			}
		}

		err = os.Mkdir(repoDir, 0755)
		if err != nil && !os.IsExist(err) {
			log.WithError(err).Error("unexpected error")
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
				log.WithError(err).Error("generate key pair failed")
				return err
			}
		}

		if err = kms.SavePrivateKey(privateKeyFile, privateKey, []byte(password)); err != nil {
			log.WithError(err).Error("save generated keypair failed")
			return err
		}
		fmt.Println("Generated private key.")

		//get publickey 
		publicKey := privateKey.PubKey()

		// generate wallet address 
		keyHash, err := crypto.PubKeyHash(publicKey)
		if err != nil {
			log.WithError(err).Error("unexpected error")
			return err
		}
		walletAddr := keyHash.String()

		// Obtain Peer ID from public key
		libp2pPubKey := (*libp2pcrypto.Secp256k1PublicKey) (publicKey)
		nodeID, err := peer.IDFromPublicKey(libp2pPubKey)
		if err != nil {
			return err
		}

		fmt.Println("Generating config file...")

		// Add config
		rawConfig.PrivateKeyFile = privateKeyFileName
		rawConfig.WalletAddress = walletAddr
		rawConfig.ThisNodeID = proto.NodeID(nodeID.Pretty())
		if rawConfig.KnownNodes == nil {
			rawConfig.KnownNodes = make([]proto.Node, 0, 1)
		}

		pipfs := ma.ProtocolWithCode(ma.P_P2P).Name
		multiaddr := fmt.Sprintf("/ip4/%s/tcp/%s/%s/%s", host, port, pipfs, nodeID.Pretty())

		node := proto.Node{
			ID:        proto.NodeID(nodeID.Pretty()),

			//todo, what about Leader or Folloer? 
			Role:      proto.Client,

			Addr: multiaddr, 
			PublicKey: publicKey,
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
			log.WithError(err).Error("unexpected error")
			return err
		}

		fmt.Println("Generated config.")
		log.Debugf("\nConfig file:      %s\n", configFilePath)
		log.Debugf("Private key file: %s\n", privateKeyFile)
		log.Debugf("Public key's hex: %s\n", hex.EncodeToString(publicKey.Serialize()))

		log.Debugf("\nWallet address: %s\n", walletAddr)
		fmt.Println(walletAddr)

		if password != "" {
			fmt.Println("Your private key had been encrypted by a passphrase, add -with-password in any further command")
		}

		return nil 

	},

}
