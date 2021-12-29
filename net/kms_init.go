package net

import (
	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/kms"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/proto"
)

// InitKMS inits nasty stuff, only for testing.
func InitKMS(host RoutedHost, PubKeyStoreFile string) {
	//initResolver()
	kms.InitPublicKeyStore(PubKeyStoreFile, nil)
	if conf.GConf.KnownNodes != nil {
		for _, n := range conf.GConf.KnownNodes {
			rawNodeID := &n.ID 
			log.WithFields(log.Fields{
				"node": rawNodeID.String(),
				"addr": n.Addr,
			}).Debug("set node addr")

			router := host.Router()
			router.SetNodeAddrCache(rawNodeID, n.Addr)

			node := &proto.Node{
				ID:         n.ID,
				Addr:       n.Addr,
				DirectAddr: n.DirectAddr,
				PublicKey:  n.PublicKey,
				Role:       n.Role,
			}
			log.WithField("node", node).Debug("known node to set")
			err := kms.SetNode(node)
			if err != nil {
				log.WithField("node", node).WithError(err).Error("set node failed")
			}

			if n.ID == conf.GConf.ThisNodeID {
				kms.SetLocalNodeID([]byte(rawNodeID.String()))
			}
		}
	}
	log.Debugf("AllNodes:\n %#v\n", conf.GConf.KnownNodes)
}
