package net

import (
	//"errors"
	//"fmt"
	//"math/rand"
	//"sync"
	//"strings" 

	"github.com/siegfried415/go-crawling-bazaar/conf"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	//"github.com/siegfried415/go-crawling-bazaar/utils"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"

	//wyong, 20201113 
	"github.com/siegfried415/go-crawling-bazaar/kms"

	//wyong, 20201116 
	//"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"

        //wyong, 20201116
        //"github.com/ugorji/go/codec"


)

// InitKMS inits nasty stuff, only for testing.
func InitKMS(host RoutedHost, PubKeyStoreFile string) {
	//initResolver()
	kms.InitPublicKeyStore(PubKeyStoreFile, nil)
	if conf.GConf.KnownNodes != nil {
		for _, n := range conf.GConf.KnownNodes {
			//todo, wyong, 20201114 
			//rawNodeID := n.ID.ToRawNodeID()
			rawNodeID := &n.ID 

			log.WithFields(log.Fields{
				"node": rawNodeID.String(),
				"addr": n.Addr,
			}).Debug("set node addr")

			//wyong, 20201113 
			router := host.Router()
			router.SetNodeAddrCache(rawNodeID, n.Addr)

			/*
			//todo, just for test, wyong, 20201116 
			testNodeID := proto.TestNodeID(n.ID)
			testNode := &TestNode {
				ID:         testNodeID, 
				Role: 		n.Role,
				Addr:       n.Addr,
				DirectAddr: "",
				PublicKey : n.PublicKey,

			}
			var encNodeInfo []byte
			h := new(codec.MsgpackHandle)
			enc := codec.NewEncoderBytes(&encNodeInfo, h )
			if err := enc.Encode(testNode); err != nil {
				return
			}
			log.Debugf("encode test node : %s\n", string(encNodeInfo)) 
			*/

			node := &proto.Node{
				ID:         n.ID,
				Addr:       n.Addr,
				DirectAddr: n.DirectAddr,
				PublicKey:  n.PublicKey,

				//wyong, 20201116 
				//Nonce:      n.Nonce,

				Role:       n.Role,
			}
			log.WithField("node", node).Debug("known node to set")
			err := kms.SetNode(node)
			if err != nil {
				log.WithField("node", node).WithError(err).Error("set node failed")
			}

			if n.ID == conf.GConf.ThisNodeID {
				//todo, wyong, 20201114 
				//kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &n.Nonce)
				kms.SetLocalNodeID([]byte(rawNodeID.String()))
			}
		}
	}
	log.Debugf("AllNodes:\n %#v\n", conf.GConf.KnownNodes)
}
