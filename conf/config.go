/*
 * Copyright 2018 The CovenantSQL Authors.
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

package conf

import (
	"io/ioutil"
	"path"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	"github.com/siegfried415/go-crawling-bazaar/crypto/hash"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

// these const specify the role of this app, which can be "miner", "presbyterian".
const (
	MinerBuildTag         = "M"
	PresbyterianBuildTag = "P"
	ClientBuildTag        = "C"
	UnknownBuildTag       = "U"
)

// StartSucceedMessage is printed when CovenantSQL started successfully.
const StartSucceedMessage = "CovenantSQL Started Successfully"

// RoleTag indicate which role the daemon is playing.
var RoleTag = UnknownBuildTag

// BaseAccountInfo defines base info to build a BaseAccount.
type BaseAccountInfo struct {
	Address             hash.Hash `yaml:"Address"`
	StableCoinBalance   uint64    `yaml:"StableCoinBalance"`
	CovenantCoinBalance uint64    `yaml:"CovenantCoinBalance"`
}

// PBGenesisInfo hold all genesis info fields.
type PBGenesisInfo struct {
	// Version defines the block version
	Version int32 `yaml:"Version"`
	// Timestamp defines the initial time of chain
	Timestamp time.Time `yaml:"Timestamp"`
	// BaseAccounts defines the base accounts for testnet
	BaseAccounts []BaseAccountInfo `yaml:"BaseAccounts"`
}

// PresbyterianInfo hold all PB info fields.
type PresbyterianInfo struct {
	// PublicKey point to Presbyterian public key
	PublicKey *asymmetric.PublicKey `yaml:"PublicKey"`
	// NodeID is the node id of Presbyterian  
	NodeID proto.NodeID `yaml:"NodeID"`
	// RawNodeID
	RawNodeID proto.RawNodeID `yaml:"-"`

	// Nonce is the nonce, SEE: cmd/cql for more
	//Nonce cpuminer.Uint256 `yaml:"Nonce"`

	// ChainFileName is the chain db's name
	ChainFileName string `yaml:"ChainFileName"`
	// PBGenesis is the genesis block filed
	PBGenesis PBGenesisInfo `yaml:"PBGenesisInfo,omitempty"`
}

// MinerDatabaseFixture config.
type MinerDatabaseFixture struct {
	DatabaseID               proto.DatabaseID `yaml:"DatabaseID"`
	Term                     uint64           `yaml:"Term"`
	Leader                   proto.NodeID     `yaml:"Leader"`
	Servers                  []proto.NodeID   `yaml:"Servers"`
	GenesisBlockFile         string           `yaml:"GenesisBlockFile"`
	AutoGenerateGenesisBlock bool             `yaml:"AutoGenerateGenesisBlock,omitempty"`
}

// MinerInfo for miner config.
type MinerInfo struct {
	// node basic config.
	RootDir                string                 `yaml:"RootDir"`
	MaxReqTimeGap          time.Duration          `yaml:"MaxReqTimeGap,omitempty"`
	ProvideServiceInterval time.Duration          `yaml:"ProvideServiceInterval,omitempty"`
	DiskUsageInterval      time.Duration          `yaml:"DiskUsageInterval,omitempty"`
	TargetUsers            []proto.AccountAddress `yaml:"TargetUsers,omitempty"`
}

// DNSSeed defines seed DNS info.
type DNSSeed struct {
	EnforcedDNSSEC bool     `yaml:"EnforcedDNSSEC"`
	DNSServers     []string `yaml:"DNSServers"`
	Domain         string   `yaml:"Domain"`
	PBCount        int      `yaml:"PBCount"`
}

// APIConfig holds all configuration options related to the api.
//type APIConfig struct {
//	Address                       string   `yaml:"address"`
//	AccessControlAllowOrigin      []string `yaml:"accessControlAllowOrigin"`
//	AccessControlAllowCredentials bool     `yaml:"accessControlAllowCredentials"`
//	AccessControlAllowMethods     []string `yaml:"accessControlAllowMethods"`
//}

// SwarmConfig holds all configuration options related to the swarm.
type SwarmConfig struct {
	Address            string `yaml:"address"`
	PublicRelayAddress string `yaml:"public_relay_address,omitempty"`
}

// DatastoreConfig holds all the configuration options for the datastore.
// TODO: use the advanced datastore configuration from ipfs
type DatastoreConfig struct {
        Type string `yaml:"type"`
        Path string `yaml:"path"`
}


// Config holds all the config read from yaml config file.
type Config struct {
	//API		*APIConfig	`yaml:"API"`	

	UseTestMasterKey bool `yaml:"UseTestMasterKey,omitempty"` // when UseTestMasterKey use default empty masterKey
	// StartupSyncHoles indicates synchronizing hole blocks from other peers on Presbyterian  
	// startup/reloading.
	StartupSyncHoles bool `yaml:"StartupSyncHoles,omitempty"`
	GenerateKeyPair  bool `yaml:"-"`
	//TODO(auxten): set yaml key for config
	WorkingRoot        string            `yaml:"WorkingRoot"`
	PubKeyStoreFile    string            `yaml:"PubKeyStoreFile"`
	PrivateKeyFile     string            `yaml:"PrivateKeyFile"`
	WalletAddress      string            `yaml:"WalletAddress"`
	DHTFileName        string            `yaml:"DHTFileName"`
	ListenAddr         string            `yaml:"ListenAddr"`

	PublicRelayAddress	string            `yaml:"PublicRelayAddress"`

	AdapterAddr	   string	     `yaml:"AdapterAddr"` 

	//SwarmAddr 	   string		`yaml:"SwarmAddr"`
	//Swarm		   *SwarmConfig 	`yaml:"Swarm"`

	Datastore     *DatastoreConfig     `yaml:"datastore"`


	ListenDirectAddr   string            `yaml:"ListenDirectAddr,omitempty"`
	ExternalListenAddr string            `yaml:"-"` // for metric purpose
	ThisNodeID         proto.NodeID      `yaml:"ThisNodeID"`
	ValidDNSKeys       map[string]string `yaml:"ValidDNSKeys"` // map[DNSKEY]domain
	// Check By PB DHT.Ping
	MinNodeIDDifficulty int `yaml:"MinNodeIDDifficulty"`

	DNSSeed DNSSeed `yaml:"DNSSeed"`

	PB    *PresbyterianInfo    `yaml:"Presbyterian"`
	Miner *MinerInfo `yaml:"Miner,omitempty"`

	KnownNodes  []proto.Node `yaml:"KnownNodes"`
	SeedPBNodes []proto.Node `yaml:"-"`

	QPS                uint32        `yaml:"QPS"`
	ChainBusPeriod     time.Duration `yaml:"ChainBusPeriod"`
	BillingBlockCount  uint64        `yaml:"BillingBlockCount"` // BillingBlockCount is for sql chain miners syncing billing with main chain
	PBPeriod           time.Duration `yaml:"PBPeriod"`
	PBTick             time.Duration `yaml:"PBTick"`
	SQLChainPeriod     time.Duration `yaml:"SQLChainPeriod"`
	SQLChainTick       time.Duration `yaml:"SQLChainTick"`
	SQLChainTTL        int32         `yaml:"SQLChainTTL"`
	MinProviderDeposit uint64        `yaml:"MinProviderDeposit"`
}

// GConf is the global config pointer.
var GConf *Config


//todo
func (c *Config )WriteFile(configFilePath string) error {
       	out, err := yaml.Marshal(c)
	if err != nil {
		//ConsoleLog.WithError(err).Error("unexpected error")
		//SetExitStatus(1)
		return err 
	}

	//configFilePath := path.Join(repoDir, "config.yaml")
	err = ioutil.WriteFile(configFilePath, out, 0644)
	if err != nil {
		//ConsoleLog.WithError(err).Error("unexpected error")
		//SetExitStatus(1)
		return nil 
	}

	return nil 

}

//todo
func NewDefaultConfig() (config *Config, err error) {
	return nil, nil 
}


// LoadConfig loads config from configPath.
func LoadConfig(configPath string) (config *Config, err error) {
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.WithError(err).Error("read config file failed")
		return
	}
	config = &Config{}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		log.WithError(err).Error("unmarshal config file failed")
		return
	}

	if config.PBPeriod == time.Duration(0) {
		config.PBPeriod = 10 * time.Second
	}

	if config.WorkingRoot == "" {
		config.WorkingRoot = "./"
	}

	if config.PrivateKeyFile == "" {
		config.PrivateKeyFile = "private.key"
	}

	if config.PubKeyStoreFile == "" {
		config.PubKeyStoreFile = "public.keystore"
	}
	if config.DHTFileName == "" {
		config.DHTFileName = "dht.db"
	}

	configDir := path.Dir(configPath)
	if !path.IsAbs(config.PubKeyStoreFile) {
		config.PubKeyStoreFile = path.Join(configDir, config.PubKeyStoreFile)
	}

	if !path.IsAbs(config.PrivateKeyFile) {
		config.PrivateKeyFile = path.Join(configDir, config.PrivateKeyFile)
	}

	if !path.IsAbs(config.DHTFileName) {
		config.DHTFileName = path.Join(configDir, config.DHTFileName)
	}

	if !path.IsAbs(config.WorkingRoot) {
		config.WorkingRoot = path.Join(configDir, config.WorkingRoot)
	}

	if config.PB != nil && !path.IsAbs(config.PB.ChainFileName) {
		config.PB.ChainFileName = path.Join(configDir, config.PB.ChainFileName)
	}

	if config.Miner != nil && !path.IsAbs(config.Miner.RootDir) {
		config.Miner.RootDir = path.Join(configDir, config.Miner.RootDir)
	}

	/*todo
	if len(config.KnownNodes) > 0 {
		for _, node := range config.KnownNodes {
			if node.ID == config.ThisNodeID {
				if config.WalletAddress == "" && node.PublicKey != nil {
					var walletHash proto.AccountAddress

					if walletHash, err = crypto.PubKeyHash(node.PublicKey); err != nil {
						return
					}

					config.WalletAddress = walletHash.String()
				}

				if config.ExternalListenAddr == "" {
					config.ExternalListenAddr = node.Addr
				}

				break
			}
		}
	}
	*/

	return
}
