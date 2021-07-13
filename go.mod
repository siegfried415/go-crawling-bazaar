module go-crawling-bazaar

go 1.13

require (
	github.com/CovenantSQL/HashStablePack v2.0.0+incompatible
	github.com/CovenantSQL/beacon v0.0.0-20190521023351-8402bfe07ece
	github.com/cyberdelia/go-metrics-graphite v0.0.0-20161219230853-39f87cc3b432
	github.com/davecgh/go-spew v1.1.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-bitswap v0.1.8
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-blockservice v0.1.3
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.4
	github.com/ipfs/go-ds-badger v0.2.3
	github.com/ipfs/go-graphsync v0.0.0-00010101000000-000000000000
	github.com/ipfs/go-ipfs-blockstore v0.1.4
	github.com/ipfs/go-ipfs-cmdkit v0.0.1
	github.com/ipfs/go-ipfs-cmds v0.4.0
	github.com/ipfs/go-ipfs-exchange-interface v0.0.1
	github.com/ipfs/go-ipfs-files v0.0.1
	github.com/ipfs/go-ipfs-keystore v0.0.1
	github.com/ipfs/go-ipfs-routing v0.1.0
	github.com/ipfs/go-ipld-cbor v0.0.1
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v1.0.4
	github.com/ipfs/go-merkledag v0.0.6
	github.com/ipld/go-ipld-prime v0.5.0
	github.com/libp2p/go-libp2p v0.11.0
	github.com/libp2p/go-libp2p-autonat-svc v0.2.0
	github.com/libp2p/go-libp2p-circuit v0.3.1
	github.com/libp2p/go-libp2p-core v0.7.0
	github.com/libp2p/go-libp2p-host v0.1.0
	github.com/libp2p/go-libp2p-kad-dht v0.0.0-00010101000000-000000000000
	github.com/libp2p/go-libp2p-loggables v0.1.0
	github.com/libp2p/go-libp2p-pubsub v0.0.0-00010101000000-000000000000
	github.com/libp2p/go-libp2p-swarm v0.2.8
	github.com/mfonda/simhash v0.0.0-20151007195837-79f94a1100d6
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multiaddr-net v0.2.0
	github.com/multiformats/go-multihash v0.0.14
	github.com/pkg/errors v0.9.1
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/siegfried415/go-crawling-bazaar v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1

	github.com/ugorji/go v1.1.2
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8
	github.com/whyrusleeping/go-logging v0.0.1
	github.com/zserge/metric v0.1.0
	golang.org/x/crypto v0.0.0-20200423211502-4bdfaf469ed5
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	gopkg.in/yaml.v2 v2.2.4

)

replace github.com/siegfried415/go-crawling-bazaar => ./

//replace github.com/libp2p/go-libp2p => github.com/libp2p/go-libp2p v0.11.0

replace github.com/libp2p/go-libp2p-core => github.com/libp2p/go-libp2p-core v0.6.0

replace github.com/libp2p/go-libp2p-kad-dht => github.com/libp2p/go-libp2p-kad-dht v0.1.1

replace github.com/libp2p/go-libp2p-pubsub => github.com/libp2p/go-libp2p-pubsub v0.1.0

replace github.com/libp2p/go-libp2p-swarm => github.com/libp2p/go-libp2p-swarm v0.2.8

replace github.com/libp2p/go-libp2p-circuit => github.com/libp2p/go-libp2p-circuit v0.3.1

replace github.com/libp2p/go-libp2p-autonat => github.com/libp2p/go-libp2p-autonat v0.3.0

//replace github.com/libp2p/go-libp2p-autonat-svc => github.com/libp2p/go-libp2p-autonat-svc v0.1.0

replace github.com/ipfs/go-bitswap => github.com/ipfs/go-bitswap v0.1.5

replace github.com/ipfs/go-car => github.com/ipfs/go-car v0.0.1

replace github.com/ipfs/go-datastore => github.com/ipfs/go-datastore v0.0.5

replace github.com/ipfs/go-ds-badger => github.com/ipfs/go-ds-badger v0.0.5

replace github.com/ipfs/go-fs-lock => github.com/ipfs/go-fs-lock v0.0.1

replace github.com/ipfs/go-graphsync => github.com/ipfs/go-graphsync v0.0.3

replace github.com/ipfs/go-hamt-ipld => github.com/ipfs/go-hamt-ipld v0.0.13

replace github.com/ipfs/go-ipfs-blockstore => github.com/ipfs/go-ipfs-blockstore v0.0.1

replace github.com/ipfs/go-ipfs-cmdkit => github.com/ipfs/go-ipfs-cmdkit v0.0.1

replace github.com/ipfs/go-ipfs-cmds => github.com/ipfs/go-ipfs-cmds v0.0.1

replace github.com/ipfs/go-merkledag => github.com/ipfs/go-merkledag v0.0.2

replace github.com/ipfs/go-unixfs => github.com/ipfs/go-unixfs v0.0.1

replace github.com/multiformats/go-multiaddr => github.com/multiformats/go-multiaddr v0.3.1

//replace github.com/multiformats/go-multiaddr-dns => github.com/multiformats/go-multiaddr-dns v0.0.3

//replace github.com/multiformats/go-multiaddr-net => github.com/multiformats/go-multiaddr-net v0.0.1

replace github.com/multiformats/go-multiaddr-fmt => github.com/multiformats/go-multiaddr-fmt v0.0.1

replace github.com/libp2p/go-libp2p-record => github.com/libp2p/go-libp2p-record v0.0.1

replace github.com/libp2p/go-eventbus => github.com/libp2p/go-eventbus v0.2.1

//replace github.com/libp2p/go-libp2p-transport-upgrader => github.com/libp2p/go-libp2p-transport-upgrader v0.2.0
