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

package node

import (
	"context"
	"database/sql"
	//"os"
	//"sync"
	//"time"

	"github.com/pkg/errors"

	"github.com/siegfried415/go-crawling-bazaar/net/consistent"
	"github.com/siegfried415/go-crawling-bazaar/kms"
	"github.com/siegfried415/go-crawling-bazaar/proto"
	//"github.com/siegfried415/go-crawling-bazaar/route"
	//rpc "github.com/siegfried415/go-crawling-bazaar/rpc/mux"
	"github.com/siegfried415/go-crawling-bazaar/storage"
	"github.com/siegfried415/go-crawling-bazaar/utils"
	//"github.com/siegfried415/go-crawling-bazaar/utils/log"
)

// LocalStorage holds consistent and storage struct.
type LocalStorage struct {
	consistent *consistent.Consistent
	*storage.Storage
}

func initStorage(dbFile string) (stor *LocalStorage, err error) {
	var st *storage.Storage
	if st, err = storage.New(dbFile); err != nil {
		return
	}
	//TODO(auxten): try BLOB for `id`, to performance test
	_, err = st.Exec(context.Background(), []storage.Query{
		{
			Pattern: "CREATE TABLE IF NOT EXISTS `dht` (`id` TEXT NOT NULL PRIMARY KEY, `node` BLOB);",
		},
	})
	if err != nil {
		//wd, _ := os.Getwd()
		//log.WithField("file", utils.FJ(wd, dbFile)).WithError(err).Error("create dht table failed")
		return
	}

	stor = &LocalStorage{
		Storage: st,
	}

	return
}

// SetNode handles dht storage node update.
func (s *LocalStorage) SetNode(node *proto.Node) (err error) {
	query := "INSERT OR REPLACE INTO `dht` (`id`, `node`) VALUES (?, ?);"
	//log.Debugf("sql: %#v", query)

	nodeBuf, err := utils.EncodeMsgPack(node)
	if err != nil {
		err = errors.Wrap(err, "encode node failed")
		return
	}

	//wyong, 20201112 
	//err = route.SetNodeAddrCache(node.ID.ToRawNodeID(), node.Addr)
	//if err != nil {
	//	//log.WithFields(log.Fields{
	//	//	"id":   node.ID,
	//	//	"addr": node.Addr,
	//	//}).WithError(err).Error("set node addr cache failed")
	//}

	err = kms.SetNode(node)
	if err != nil {
		//log.WithField("node", node).WithError(err).Error("kms set node failed")
	}

	// if s.consistent == nil, it is called during Init. and AddCache will be called by consistent.InitConsistent
	if s.consistent != nil {
		s.consistent.AddCache(*node)
	}

	// execute query
	if _, err = s.Storage.Exec(context.Background(), []storage.Query{
		{
			Pattern: query,
			Args: []sql.NamedArg{
				sql.Named("", node.ID),
				sql.Named("", nodeBuf.Bytes()),
			},
		},
	}); err != nil {
		err = errors.Wrap(err, "execute query in dht database failed")
	}

	return
}

