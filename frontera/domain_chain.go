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

package frontera

import (
	"context"

	//wyong, 20210528 
	//"os"
	//"path/filepath"

	//"sync"
	"sync/atomic"
	"time"
	"fmt"
	"math/big" 

	//wyong, 20210125 
	//"sort" 

	"github.com/pkg/errors"

	//wyong, 20201018
	//"github.com/libp2p/go-libp2p-core/host"

	//wyong, 20200929 
	//"github.com/siegfried415/go-crawling-bazaar/conf"


	//wyong, 20201002 
	//"github.com/siegfried415/go-crawling-bazaar/crypto" 
	//"github.com/siegfried415/go-crawling-bazaar/crypto/asymmetric"
	//"github.com/siegfried415/go-crawling-bazaar/kms"

	//"github.com/siegfried415/go-crawling-bazaar/kayak"
	//kt "github.com/siegfried415/go-crawling-bazaar/kayak/types"
	//kl "github.com/siegfried415/go-crawling-bazaar/kayak/wal"
	//"github.com/siegfried415/go-crawling-bazaar/proto"
	//urlchain "github.com/siegfried415/go-crawling-bazaar/urlchain"
	//"github.com/siegfried415/go-crawling-bazaar/storage"
	"github.com/siegfried415/go-crawling-bazaar/types"
	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	s "github.com/siegfried415/go-crawling-bazaar/state"
	//net "github.com/siegfried415/go-crawling-bazaar/net"

	//wyong, 20200805 
	//lru "github.com/hashicorp/golang-lru"

	//wyong, 20200818
	cid "github.com/ipfs/go-cid"

	//wyong, 20210122
        cache "github.com/patrickmn/go-cache"

)

/*todo 
const (
	// StorageFileName defines storage file name of database instance.
	StorageFileName = "storage.db3"

	// KayakWalFileName defines log pool name of database instance.
	KayakWalFileName = "kayak.ldb"

	// SQLChainFileName defines sqlchain storage file name.
	SQLChainFileName = "chain.db"

	// MaxRecordedConnectionSequences defines the max connection slots to anti reply attack.
	MaxRecordedConnectionSequences = 1000

	// PrepareThreshold defines the prepare complete threshold.
	PrepareThreshold = 1.0

	// CommitThreshold defines the commit complete threshold.
	CommitThreshold = 0.0

	// PrepareTimeout defines the prepare timeout config.
	PrepareTimeout = 10 * time.Second

	// CommitTimeout defines the commit timeout config.
	CommitTimeout = time.Minute

	// LogWaitTimeout defines the missing log wait timeout config.
	LogWaitTimeout = 10 * time.Second

	// SlowQuerySampleSize defines the maximum slow query log size (default: 1KB).
	SlowQuerySampleSize = 1 << 10

	//wyong, 20200805 
	activeBiddingsLimit = 16
)
*/


//wyong, 20210122 
func (domain *Domain) InitTables() (err error) {
	domain.InitUrlNodeTable()
	domain.InitUrlGraphTable() 
	domain.InitUrlCidTable() 
	return nil 
}

//todo, wyong, 20210122 
func (domain *Domain) InitUrlNodeTable() (err error) {

	//create urlNode cache, wyong, 20210206
	domain.urlNodeCache = cache.New(5*time.Minute, 10*time.Minute)

	// build query 
	query := types.Query {
		Pattern : "CREATE TABLE urlnode (url TEXT PRIMARY KEY NOT NULL, LastRequestedHeight int, LastCrawledHeight int, RetrivedCount int, CrawlInterval int )",
	}

        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,

				//todo, wyong, 20200817 
                                //ConnectionID: connID,
                                //SeqNo:        seqNo,
                                //Timestamp:    getLocalTime(),
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	request.SetContext(context.Background())
	if _, _, err := domain.chain.Query(request, true); err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
	}

	return 
}

//todo, wyong, 20210122 
func (domain *Domain) InitUrlGraphTable() (err error) {

	//create urlNode cache, wyong, 20210206
	domain.urlGraphCache = cache.New(5*time.Minute, 10*time.Minute)

	// build query 
	query := types.Query {
		Pattern : "CREATE TABLE urlgraph (url TEXT PRIMARY KEY NOT NULL, child TEXT, count int )",
	}

        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	request.SetContext(context.Background())
	if _, _, err := domain.chain.Query(request, true); err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
	}

	return 
}

//wyong, 20200817 
func (domain *Domain) InitUrlCidTable() (err error) {

	//create urlNode cache, wyong, 20210206
	domain.urlCidCache = cache.New(5*time.Minute, 10*time.Minute)

	// build query 
	query := types.Query {
		Pattern : "CREATE TABLE urlcid (url TEXT PRIMARY KEY NOT NULL, cid CHAR(64))",
		//todo, check max width of from/vhash/proof/chash , wyong, 20210122 
		//Pattern : "CREATE TABLE urlcid (url TEXT PRIMARY KEY NOT NULL, cid CHAR(64), from CHAR(?), vhash CHAR(?), proof CHAR(?), chash CHAR(?))",
	}

        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,

				//todo, wyong, 20200817 
                                //ConnectionID: connID,
                                //SeqNo:        seqNo,
                                //Timestamp:    getLocalTime(),
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	request.SetContext(context.Background())
	if _, _, err := domain.chain.Query(request, true); err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
	}

	return 
}

//todo, wyong, 20210126
func (domain *Domain) CreateUrlNode(url string, lastRequestedHeight uint32, lastCrawledHeight uint32, retrivedCount uint32, crawlInterval uint32  ) (*types.UrlNode, error ) {
	//todo, create and insert it into urlnode . wyong, 20210126
	log.Debugf("Domain/CreateUrlNode(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "INSERT INTO urlnode VALUES('%s',%d, %d, %d, %d )", url, lastRequestedHeight, lastCrawledHeight, retrivedCount, crawlInterval  ) 
	query := types.Query {
		Pattern : q ,
	}

	log.Debugf("Domain/CreateUrlNode(20), q=%s\n", q ) 
        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/CreateUrlNode(30)\n") 
	request.SetContext(context.Background())

	//_, err := domain.Query(request)
	_, _, err := domain.chain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/CreateUrlNode(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return nil, err
	}

	log.Debugf("Domain/CreateUrlNode(40)\n" ) 
	urlNode := &types.UrlNode {
		Url: url, 
		LastRequestedHeight: lastRequestedHeight,
		LastCrawledHeight : lastCrawledHeight, 
		RetrivedCount : retrivedCount, 
		CrawlInterval : crawlInterval, 
	}

	//wyong, 20210205 
	domain.urlNodeCache.Set(url, urlNode, cache.DefaultExpiration) 

	return urlNode, nil  
}

func (domain *Domain) UpdateUrlNode(urlNode *types.UrlNode ) error {
	log.Debugf("Domain/updateUrlNode(10), urlNode=%s\n", urlNode) 

	//build query , wyong, 20210223
	q := fmt.Sprintf( "UPDATE urlnode SET LastRequestedHeight = '%d', LastCrawledHeight = '%d', RetrivedCount = '%d', CrawlInterval = '%d' WHERE Url = '%s' ", urlNode.LastRequestedHeight, urlNode.LastCrawledHeight, urlNode.RetrivedCount, urlNode.CrawlInterval , urlNode.Url  ) 
	query := types.Query {
		Pattern : q ,
	}

	log.Debugf("Domain/updateUrlNode(20), q = %s \n", q ) 
        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/updateUrlNode(30)\n") 

	//todo, wyong, 20210223 
	//_, err := domain.Query(request)
	request.SetContext(context.Background())
	_, _, err := domain.chain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/updateUrlNode(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return err
	}

	log.Debugf("Domain/updateUrlNode(40)\n" ) 
	domain.urlNodeCache.Set(urlNode.Url, urlNode, cache.DefaultExpiration) 
	log.Debugf("Domain/updateUrlNode(50)\n" ) 
	return nil 
}


//wyong, 20210130
func (domain *Domain) SetLastRequestedHeight(url string, height uint32) error {
	log.Debugf("Domain/SetLastRequestedHeight(10), url=%s, LastRequestedHeight=%d\n", url, height ) 
        urlNode, err := domain.GetUrlNode(url)
        if err != nil {
		return err
        }

	log.Debugf("Domain/SetLastRequestedHeight(20), urlNode.LastRequestedHeight(%d) is set to %d\n", urlNode.LastRequestedHeight, height) 
	urlNode.LastCrawledHeight = height  

	err = domain.UpdateUrlNode(urlNode) 
	if err != nil { 
		log.Debugf("Domain/SetLastRequestedHeight(30), err=%s\n", err.Error) 
		return err
	}

	log.Debugf("Domain/SetLastRequestedHeight(40)\n") 
	return nil  
}

//wyong, 20210130
func (domain *Domain) SetLastCrawledHeight(url string, height uint32) error {
	log.Debugf("Domain/SetLastCrawledHeight(10), url=%s, LastCrawledHeight=%d \n", url, height ) 
        urlNode, err := domain.GetUrlNode(url)
        if err != nil {
		return err
        }

	log.Debugf("Domain/SetLastCrawledHeight(20), urlNode.LastCrawledHeight(%d) is set to %d\n", urlNode.LastCrawledHeight, height) 
	urlNode.LastCrawledHeight = height  

	err = domain.UpdateUrlNode(urlNode) 
	if err != nil {
		log.Debugf("Domain/SetLastCrawledHeight(30), err = %s \n", err.Error()) 
		return err 
	}

	log.Debugf("Domain/SetLastCrawledHeight(40)\n" ) 
	return nil 
}


//wyong, 20210130
func (domain *Domain) SetRetrivedCount(url string, count uint32 ) error  {
	log.Debugf("Domain/SetRetrivedCount(10), url=%s, RetrivedCount=%d\n", url, count ) 
        urlNode, err := domain.GetUrlNode(url)
        if err != nil {
		return err
        }
	
	log.Debugf("Domain/SetRetrivedCount(20), urlNode.RetrivedCount(%d) is set to %d\n", urlNode.RetrivedCount, count) 
	urlNode.RetrivedCount = count  

	err = domain.UpdateUrlNode(urlNode) 
	if err != nil { 
		log.Debugf("Domain/SetRetrivedCount(30), err=%s\n", err.Error()) 
		return err
	} 

	log.Debugf("Domain/SetRetrivedCount(40)\n") 
	return nil 
}

//todo, wyong, 20210130
func (domain *Domain) SetCrawlInterval(url string, interval uint32 ) error {
	log.Debugf("Domain/SetCrawlInterval(10), url=%s, Interval=%d\n", url, interval ) 
        urlNode, err := domain.GetUrlNode(url)
        if err != nil {
		return err
        }
	
	log.Debugf("Domain/SetCrawlInterval(20), urlNode.CrawlInterval(%d) is set to %d\n", urlNode.CrawlInterval, interval ) 
	urlNode.CrawlInterval = interval 

	err = domain.UpdateUrlNode(urlNode ) 
	if err != nil { 
		log.Debugf("Domain/SetCrawlInterval(30), err=%s\n", err.Error()) 
		return err 
	} 

	log.Debugf("Domain/SetCrawlInterval(40)\n" ) 
	return nil 
}

//wyong, 20210224 
func ToU32( value interface{}) uint32 {
	return uint32(value.(int64))
}

//todo, wyong, 20210117
func (domain *Domain) GetUrlNode(url string ) (*types.UrlNode, error ) {
	//get UlrNode from domain.urlNodeCache first. wyong, 20210122
        urlNodeCache, found := domain.urlNodeCache.Get(url)
        if found {
		return urlNodeCache.(*types.UrlNode), nil 
        }

	//if not found, get it from urlchain, wyong, 20210122 
	q := fmt.Sprintf( "SELECT LastRequestedHeight, LastCrawledHeight, RetrivedCount, CrawlInterval FROM urlnode where url='%s'" , url ) 
	query := types.Query {
		Pattern : q ,
	}

	log.Debugf("Domain/GetUrlNode(20), q=%s\n", q ) 
	// build request
	request := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    types.ReadQuery,
				NodeID:       domain.nodeID,
				DomainID:     domain.domainID,
			},
		},
		Payload: types.RequestPayload{
			Queries: []types.Query{query},
		},
	}

	log.Debugf("Domain/GetUrlNode(30)\n") 
	request.SetContext(context.Background())

	//resp, err := domain.Query(request)
	_, resp, err := domain.chain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/GetUrlNode(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return nil, err 
	}

	log.Debugf("Domain/GetUrlNode(40)\n") 
	//result process , wyong, 20200817 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		return nil, err
	}

	log.Debugf("Domain/GetUrlNode(50), rows =%s\n", rows) 
	r0 := rows[0] 
	if len(r0.Values) <=0 {
		return nil, errors.New("there are errors in url chain result")
	} 

	//todo, get UrlNode from fr.Values.  
	//reference client/rows.go 
	urlNode := &types.UrlNode {
		Url : url , 
		LastRequestedHeight : ToU32(r0.Values[0]), 
		LastCrawledHeight : ToU32(r0.Values[1]),
		RetrivedCount : ToU32(r0.Values[2]), 
		CrawlInterval : ToU32(r0.Values[3]),
	}

	log.Debugf("Domain/GetUrlNode(60), urlNode =%s\n", urlNode) 

	//update cache, wyong, 20210130 
	domain.urlNodeCache.Set(url, urlNode, cache.DefaultExpiration) 

	return urlNode, nil 
}

//wyong, 20210130
func (domain *Domain) SetUrlLinksCount(parenturl string, url string,  linksCount uint32 ) error {
	//todo, create and insert it into urlnode . wyong, 20210126
	log.Debugf("Domain/SetUrlLinksCount(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "UPDATE urlgraph SET count = %d WHERE url ='%s' and child='%s' ", linksCount, parenturl, url ) 

	query := types.Query {
		Pattern : q ,
	}

	log.Debugf("Domain/SetUrlLinksCount(20), q=%s\n", q ) 
        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/SetUrlLinksCount(30)\n") 
	request.SetContext(context.Background())

	//_, err := domain.Query(request)
	_, _, err := domain.chain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/SetUrlLinksCount(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return err 
	}

	log.Debugf("Domain/SetUrlLinksCount(40)\n" ) 

	//todo, wyong, 20210205 
	domain.urlGraphCache.Set(parenturl+url, linksCount, cache.DefaultExpiration) 

	return nil 
}

//wyong, 20210117
func (domain *Domain) GetUrlLinksCount(parentUrl string , url string) (uint32, error ) {
	//links count are forward count from parent ulr Node to url node ,
	//get it from domain.urlGraphCache first. wyong, 20210122
        linksCount, found := domain.urlGraphCache.Get(parentUrl+url)
        if found {
                //fmt.Println(foo)
		return linksCount.(uint32), nil 
        }


	//if not found, get it from urlchain, wyong, 20210122 
	q := fmt.Sprintf( "SELECT count FROM urlgraph where url = '%s' and child = '%s'" , parentUrl, url ) 
	query := types.Query {
		Pattern : q ,
	}

	log.Debugf("Domain/GetUrlLinksCount(20), q=%s\n", q ) 
	// build request
	request := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    types.ReadQuery,
				NodeID:       domain.nodeID,
				DomainID:     domain.domainID,
			},
		},
		Payload: types.RequestPayload{
			Queries: []types.Query{query},
		},
	}

	log.Debugf("Domain/GetUrlLinksCount(30)\n") 
	request.SetContext(context.Background())

	//resp, err := domain.Query(request)
	_, resp, err := domain.chain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/GetUrlLinksCount(35), err = %s\n", err.Error()) 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 0, err 
	}

	log.Debugf("Domain/GetUrlLinksCount(40)\n") 
	//result process , wyong, 20200817 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		//return 0, errors.New("result error") 
		return 0, nil 
	}

	log.Debugf("Domain/GetUrlLinksCount(50), first row =%s\n", rows[0]) 
	r0 := rows[0]		//first row
	if len(r0.Values) <=0 {
		return 0, errors.New("result error") 
	} 

	//todo, get linkscount from fr.Values.  
	//reference client/rows.go 
	result := ToU32(r0.Values[0])	//uint32((fr.Values[0]).(int64))

	//todo, wyong, 20210205 
	domain.urlGraphCache.Set(parentUrl+url, result, cache.DefaultExpiration ) 

	return result, nil 
}

//wyong, 20210224
func (domain *Domain) AddUrlLinksCount(parentUrl string, url string ) error {

	linksCount, err := domain.GetUrlLinksCount(parentUrl, url) 
	if err != nil {
		return err 
	}

	err = domain.SetUrlLinksCount(parentUrl, url, linksCount + 1 ) 
        if err != nil {
                return err 
        }

	return nil 
}

//todo, is math/big is necessary? wyong, 20210117
func (domain *Domain) GetProbability(parentUrl string, parentProb float64, url string ) (float64, error ) {
	log.Debugf("GetProbability(10)\n")

        //wyong, 20200416
        forwardProbability := big.NewFloat(0)

        //wyong, 20210125
        parentProbability := big.NewFloat(parentProb)

        parentRetrivedCount := big.NewFloat(2)
        forwardCount := big.NewFloat(1)

        //parentUrlNode, ok := state.UrlNodes[parentUrl]
        parentUrlNode, err := domain.GetUrlNode(parentUrl)
        if err != nil  {
                log.Debugf("GetProbability(20)\n")
		prc := big.NewInt(int64(parentUrlNode.RetrivedCount)) 
                parentRetrivedCount.Add(parentRetrivedCount,  new(big.Float).SetInt( /* int(parentUrlNode.RetrivedCount)*/ prc ))

                //todo, wyong, 20210122
                //linkCount, ok := parentUrlNode.LinksCount[url]
                linksCount, err := domain.GetUrlLinksCount(parentUrl, url )

                //bugfix, wyong, 20200313
                if err != nil  {
			lc := big.NewInt(int64(linksCount)) 
                        forwardCount.Add(forwardCount, new(big.Float).SetInt(/* int(linksCount) */ lc ))
                }
        }
        log.Debugf("GetProbability(30), parentRetrivedCount=%s, forwardCount=%s \n",
                                        parentRetrivedCount.String(), forwardCount.String())

        forwardProbability.Quo(forwardCount, parentRetrivedCount)
        forwardProbability.Mul(forwardProbability, parentProbability)

        log.Debugf("GetProbability(40), forwardProbability=%s\n", forwardProbability.String())
	result, _ := forwardProbability.Float64()

        return result , nil

}


//wyong, 20200821
func (domain *Domain) SetCid( url string, c cid.Cid) (err error) {
	log.Debugf("Domain/SetCid(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "INSERT INTO urlcid (url, cid) VALUES('%s', '%s')", url, c.String()) 
	query := types.Query {
		Pattern : q ,
	}

	log.Debugf("Domain/SetCid(20), q=%s\n", q ) 
        // build request
        request := &types.Request{
                Header: types.SignedRequestHeader{
                        RequestHeader: types.RequestHeader{
                                QueryType:    types.WriteQuery,
                                NodeID:       domain.nodeID,
                                DomainID:     domain.domainID,

				//todo, wyong, 20200817 
                                //ConnectionID: connID,
                                //SeqNo:        seqNo,
                                //Timestamp:    getLocalTime(),
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/SetCid(30)\n") 
	request.SetContext(context.Background())
	_, _, err = domain.chain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/SetCid(35), err = %s\n", err.Error()) 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/SetCid(40)\n" ) 

	//todo, update cache, wyong, 20210130 
	domain.urlCidCache.Set(url, c, cache.DefaultExpiration ) 

	return 
}

//wyong, 20201208 
func B2S(bs []uint8) string {
	ba := []byte{}
	for _, b := range bs {
		ba = append(ba, byte(b))
	}
	return string(ba)
}


//wyong, 20200817
func (domain *Domain) GetCid(url string) (c cid.Cid, err error) {

	log.Debugf("Domain/GetCid(10), url=%s\n", url) 
	if urlcid, found := domain.urlCidCache.Get(url); found {
		log.Debugf("Domain/GetCid(15), cid=%s\n", urlcid.(cid.Cid).String()) 
		return urlcid.(cid.Cid), nil 
	} 

	q := fmt.Sprintf( "SELECT cid FROM urlcid where url='%s'" , url ) 
	query := types.Query {
		Pattern : q ,
	}

	log.Debugf("Domain/GetCid(20), q=%s\n", q ) 
	// build request
	request := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    types.ReadQuery,
				NodeID:       domain.nodeID,
				DomainID:     domain.domainID,

				//todo, wyong, 20200817 
				//ConnectionID: connID,
				//SeqNo:        seqNo,
				//Timestamp:    getLocalTime(),
			},
		},
		Payload: types.RequestPayload{
			Queries: []types.Query{query},
		},
	}

	log.Debugf("Domain/GetCid(30)\n") 
	request.SetContext(context.Background())
	_, resp, err := domain.chain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/GetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return cid.Cid{}, err 
	}

	log.Debugf("Domain/GetCid(40)\n") 
	//result process , wyong, 20200817 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		return cid.Cid{}, errors.New("there are errors in url chain result")
	}

	log.Debugf("Domain/GetCid(50)\n") 
	r0 := rows[0]		//first row
	if len(r0.Values) <=0 {
		return cid.Cid{}, errors.New("there are errors in url chain result")
	} 

	log.Debugf("Domain/GetCid(60), first row =%s\n", r0) 

	//bugfix, wyong, 20201208 
	//r0c0 := r0.Values[0].(string)	//first column of first row 
	r0c0 := B2S((r0.Values[0]).([]uint8))	//first column of first row 

	log.Debugf("Domain/GetCid(70), first column of first row =%s\n", r0c0 ) 
	c, err  = cid.Decode(r0c0)	//convert it to cid 
	if  err != nil {
		log.Debugf("Domain/GetCid(75)\n") 
		return cid.Cid{}, err  
	}

	log.Debugf("Domain/GetCid(80), cid=%s\n", c.String()) 

	//todo, put urlcid to urlCidCache, wyong, 20210122 
	domain.urlCidCache.Set(url, c, cache.DefaultExpiration) 

	log.Debugf("Domain/GetCid(90)\n") 
	return c, nil 

}

func (domain *Domain) logSlow(request *types.Request, isFinished bool, tmStart time.Time) {
        if request == nil {
                return
        }

        // sample the queries
        querySample := ""

        for _, q := range request.Payload.Queries {
                if len(querySample) < SlowQuerySampleSize {
                        querySample += "; "
                        querySample += q.Pattern
                } else {
                        break
                }
        }

        if len(querySample) >= SlowQuerySampleSize {
                querySample = querySample[:SlowQuerySampleSize-3]
                querySample += "..."
        }

        log.WithFields(log.Fields{
                "finished": isFinished,
                "domain":       request.Header.DomainID,
                "req_time": request.Header.Timestamp.String(),
                "req_node": request.Header.NodeID,
                "count":    request.Header.BatchCount,
                "type":     request.Header.QueryType.String(),
                "sample":   querySample,
                "start":    tmStart.String(),
                "elapsed":  time.Now().Sub(tmStart).String(),
        }).Error("slow query detected")
}

// Query defines database query interface.
func (domain *Domain) Query(request *types.Request) (response *types.Response, err error) {
	// Just need to verify signature in domain.saveAck
	//if err = request.Verify(); err != nil {
	//	return
	//}

	var (
		isSlowQuery uint32
		tracker     *s.QueryTracker
		tmStart     = time.Now()
	)

	// log the query if the underlying storage layer take too long to response
	slowQueryTimer := time.AfterFunc(domain.cfg.SlowQueryTime, func() {
		// mark as slow query
		atomic.StoreUint32(&isSlowQuery, 1)
		domain.logSlow(request, false, tmStart)
	})
	defer slowQueryTimer.Stop()
	defer func() {
		if atomic.LoadUint32(&isSlowQuery) == 1 {
			// slow query
			domain.logSlow(request, true, tmStart)
		}
	}()

	switch request.Header.QueryType {
	case types.ReadQuery:
		if tracker, response, err = domain.chain.Query(request, false); err != nil {
			err = errors.Wrap(err, "failed to query read query")
			return
		}
	case types.WriteQuery:
		//we use EventualConsistency only,  wyong, 20210528 
		//if domain.cfg.UseEventualConsistency {
			// reset context
			request.SetContext(context.Background())
			if tracker, response, err = domain.chain.Query(request, true); err != nil {
				err = errors.Wrap(err, "failed to execute with eventual consistency")
				return
			}
		//} else {
		//	if tracker, response, err = domain.writeQuery(request); err != nil {
		//		err = errors.Wrap(err, "failed to execute")
		//		return
		//	}
		//}

	default:
		// TODO(xq262144): verbose errors with custom error structure
		return nil, errors.Wrap(ErrInvalidRequest, "invalid query type")
	}

	//todo, wyong, 20200930 
	response.Header.ResponseAccount = domain.accountAddr

	// build hash
	if err = response.BuildHash(); err != nil {
		err = errors.Wrap(err, "failed to build response hash")
		return
	}

	//todo, got error here, wyong, 20210223  
	if err = domain.chain.AddResponse(&response.Header); err != nil {
		log.WithError(err).Debug("failed to add response to index")
		return
	}
	tracker.UpdateResp(response)

	return
}

/* wyong, 20210528 
func (domain *Domain) writeQuery(request *types.Request) (tracker *s.QueryTracker, response *types.Response, err error) {
	// check database size first, wal/kayak/chain database size is not included
	if domain.cfg.SpaceLimit > 0 {
		path := filepath.Join(domain.cfg.DataDir, StorageFileName)
		var statInfo os.FileInfo
		if statInfo, err = os.Stat(path); err != nil {
			if !os.IsNotExist(err) {
				return
			}
		} else {
			if uint64(statInfo.Size()) > domain.cfg.SpaceLimit {
				// rejected
				err = ErrSpaceLimitExceeded
				return
			}
		}
	}

	// call kayak runtime Process
	var result interface{}
	if result, _, err = domain.kayakRuntime.Apply(request.GetContext(), request); err != nil {
		err = errors.Wrap(err, "apply failed")
		return
	}

	var (
		tr *TrackerAndResponse
		ok bool
	)
	if tr, ok = (result).(*TrackerAndResponse); !ok {
		err = errors.Wrap(err, "invalid response type")
		return
	}
	tracker = tr.Tracker
	response = tr.Response
	return
}
*/


func (domain *Domain) saveAck(ackHeader *types.SignedAckHeader) (err error) {
        return domain.chain.VerifyAndPushAckedQuery(ackHeader)
}

// Ack defines client response ack interface.
func (domain *Domain) Ack(ack *types.Ack) (err error) {
        // Just need to verify signature in domain.saveAck
        //if err = ack.Verify(); err != nil {
        //      return
        //}

        return domain.saveAck(&ack.Header)
}


func getLocalTime() time.Time {
        return time.Now().UTC()
}


