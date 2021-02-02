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
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"fmt"

	//wyong, 20210125 
	"sort" 

	"github.com/pkg/errors"

	//wyong, 20201018
	//"github.com/libp2p/go-libp2p-core/host"

	//wyong, 20200929 
	"github.com/siegfried415/gdf-rebuild/conf"


	//wyong, 20201002 
	"github.com/siegfried415/gdf-rebuild/crypto" 
	"github.com/siegfried415/gdf-rebuild/crypto/asymmetric"
	"github.com/siegfried415/gdf-rebuild/kms"

	"github.com/siegfried415/gdf-rebuild/kayak"
	kt "github.com/siegfried415/gdf-rebuild/kayak/types"
	kl "github.com/siegfried415/gdf-rebuild/kayak/wal"
	"github.com/siegfried415/gdf-rebuild/proto"
	urlchain "github.com/siegfried415/gdf-rebuild/urlchain"
	"github.com/siegfried415/gdf-rebuild/storage"
	"github.com/siegfried415/gdf-rebuild/types"
	"github.com/siegfried415/gdf-rebuild/utils/log"
	x "github.com/siegfried415/gdf-rebuild/xenomint"
	net "github.com/siegfried415/gdf-rebuild/net"

	//wyong, 20200805 
	lru "github.com/hashicorp/golang-lru"

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
func (domain *Domain) InitTable() (err error) {
	InitUrlNodeTable()
	InitUrlGraphTable() 
	InitUrlCidTable() 
}

//todo, wyong, 20210122 
func (domain *Domain) InitUrlNodeTable() (err error) {

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

	// build query 
	query := types.Query {
		//Pattern : "CREATE TABLE urlgraph (url TEXT PRIMARY KEY NOT NULL, cid CHAR(64))",
		//todo, check max width of from/vhash/proof/chash , wyong, 20210122 
		Pattern : "CREATE TABLE urlcid (url TEXT PRIMARY KEY NOT NULL, cid CHAR(64), from CHAR(?), vhash CHAR(?), proof CHAR(?), chash CHAR(?))",
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
func (domain *Domain) CreateUrlNode(url string, lastRequestedHeight, lastCrawledHeight, RetrivedCount , CrawlInterval int  ) (*types.UrlNode, error ) {
	//todo, create and insert it into urlnode . wyong, 20210126
	log.Debugf("Domain/SetCid(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "INSERT INTO urlnode VALUES('%s','%s', '%s', '%s', '%s' )", url, lastRequestedHeight, lastCrawledHeight, retrivedCount, crawlInterval  ) 
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
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/SetCid(30)\n") 
	request.SetContext(context.Background())
	_, _, err = domain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/SetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/SetCid(40)\n" ) 
	urlNode := &types.UrlNode {
		Url: url, 
		LastRetrivedHeight: lastRetrivedHeight,
		LastCrawledHeight : lastCrawledHeight, 
		RetrivedCount : retrivedCount, 
		CrawlInterval : crawlInterval, 
	}

	domain.urlNodeCache.Set(url, urlNode) 

	return urlNode, nil  
}

//wyong, 20210130
func (domain *Domain) SetLastRequestedHeight(url string, height int) error {
	//todo, create and insert it into urlnode . wyong, 20210126
	log.Debugf("Domain/SetCid(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "UPDATE urlnode SET LastRequestedHeight ='%s' WHERE Url='%s' ", height, url ) 
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
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/SetCid(30)\n") 
	request.SetContext(context.Background())
	_, _, err = domain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/SetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/SetCid(40)\n" ) 
	domain.urlNodeCache.Set(url, &types.UrlNode {
		Url: url, 
		LastRequestedHeight : height ,
	}) 

	return nil 
}

//wyong, 20210130
func (domain *Domain) SetLastCrawledHeight(url string, height int) error {
	//todo, create and insert it into urlnode . wyong, 20210126
	log.Debugf("Domain/SetCid(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "UPDATE urlnode SET LastCrawledHeight ='%s' WHERE Url='%s' ", height, url ) 
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
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/SetCid(30)\n") 
	request.SetContext(context.Background())
	_, _, err = domain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/SetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/SetCid(40)\n" ) 
	domain.urlNodeCache.Set(url, &types.UrlNode {
		Url: url, 
		LastCrawledHeight : height ,
	}) 

	return nil 
}


//wyong, 20210130
func (domain *Domain) SetRetrivedCount(urls string, count int  ) (types.UrlNode, error ) {
	//todo, create and insert it into urlnode . wyong, 20210126
	log.Debugf("Domain/SetCid(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "UPDATE urlnode SET RetrivedCount='%s' WHERE Url='%s' ", count, url ) 
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
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/SetCid(30)\n") 
	request.SetContext(context.Background())
	_, _, err = domain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/SetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/SetCid(40)\n" ) 
	domain.urlNodeCache.Set(url, &types.UrlNode {
		Url: url, 
		RetrivedCount: count,
	}) 

	return nil 
}

//todo, wyong, 20210130
func (domain *Domain) SetCrawlInterval(urls string, interval int ) (types.UrlNode, error ) {
	//todo, create and insert it into urlnode . wyong, 20210126
	log.Debugf("Domain/SetCid(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "UPDATE urlnode SET CrawlInterval = '%s' WHERE Url='%s' ", interval, url ) 
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
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/SetCid(30)\n") 
	request.SetContext(context.Background())
	_, _, err = domain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/SetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/SetCid(40)\n" ) 
	domain.urlNodeCache.Set(url, &types.UrlNode {
		Url: url, 
		CrawlInterval: interval,
	}) 

	return nil 
}

//todo, wyong, 20210117
func (domain *Domain) GetUrlNode(url string ) (types.UrlNode, error ) {
	//get UlrNode from domain.urlNodeCache first. wyong, 20210122
        urlNode, found := domain.urlNodeCache.Get(url)
        if found {
                //fmt.Println(foo)
		return urlNode 
        }

	//if not found, get it from urlchain, wyong, 20210122 
	q := fmt.Sprintf( "SELECT * FROM urlnode where url='%s'" , url ) 
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
			},
		},
		Payload: types.RequestPayload{
			Queries: []types.Query{query},
		},
	}

	log.Debugf("Domain/GetCid(30)\n") 
	request.SetContext(context.Background())
	_, resp, err := domain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/GetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/GetCid(40)\n") 
	//result process , wyong, 20200817 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		return 
	}

	log.Debugf("Domain/GetCid(50), first row =%s\n", rows[0]) 
	fr := rows[0]		//first row
	if len(fr.Values) <=0 {
		return 
	} 

	//todo, get UrlNode from fr.Values.  
	//reference client/rows.go 
	return type.UrlNode {
		LastRequestedHeight : fr.Values[1], 
		LastCrawledHeight : fr.Values[2], 
		RetrivedCount : fr.Values[3], 
		CrawlInterval : fr.Values[4], 
	}

	//todo, update cache, wyong, 20210130 
}

//wyong, 20210130
func (domain *Domain) SetUrlLinksCount(parenturl string, url string,  linksCount int ) error {
	//todo, create and insert it into urlnode . wyong, 20210126
	log.Debugf("Domain/SetCid(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "UPDATE urlgraph SET LinksCount = '%s' WHERE partent ='%s' and child='%s' ", linksCount, parenturl, url ) 

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
                        },
                },
                Payload: types.RequestPayload{
                        Queries: []types.Query{query},
                },
        }

	log.Debugf("Domain/SetCid(30)\n") 
	request.SetContext(context.Background())
	_, _, err = domain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/SetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/SetCid(40)\n" ) 
	domain.urlGraphCache.Set(parenturl+url, linksCount) 

	return nil 
}

//wyong, 20210117
func (domain *Domain) GetUrlLinksCount(parenturl string , url string) (int, error ) {
	//links count are forward count from parent ulr Node to url node ,
	//get it from domain.urlGraphCache first. wyong, 20210122
        linksCount, found := domain.urlGraphCache.Get(parentUrl+url)
        if found {
                //fmt.Println(foo)
		return linksCount  
        }


	//if not found, get it from urlchain, wyong, 20210122 
	q := fmt.Sprintf( "SELECT * FROM urlgraph where parent='%s' and child='%s'" , parenturl, url ) 
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
			},
		},
		Payload: types.RequestPayload{
			Queries: []types.Query{query},
		},
	}

	log.Debugf("Domain/GetCid(30)\n") 
	request.SetContext(context.Background())
	_, resp, err := domain.Query(request, true)
	if  err != nil {
		log.Debugf("Domain/GetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/GetCid(40)\n") 
	//result process , wyong, 20200817 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		return 
	}

	log.Debugf("Domain/GetCid(50), first row =%s\n", rows[0]) 
	fr := rows[0]		//first row
	if len(fr.Values) <=0 {
		return 
	} 

	//todo, get linkscount from fr.Values.  
	//reference client/rows.go 
	return fr.Values[0]
}

//todo, wyong, 20210117
func (domain *Domain) GetProbability(parentUrl string, parentProb float, url string ) (float, error ) {
        //wyong, 20200416
        forwardProbability := big.NewFloat(0)

        //wyong, 20210125
        parentProbability := big.NewFloat(parentProb)

        parentRetrivedCount := big.NewFloat(2)
        forwardCount := big.NewFloat(1)

        //parentUrlNode, ok := state.UrlNodes[parentUrl]
        parentUrlNode, ok := domain.GetUrlNode(parentUrl)
        if ok == true {
                fmt.Printf("GetProbability(30)\n")
                parentRetrivedCount.Add(parentRetrivedCount, new(big.Float).SetInt(parentUrlNode.RetrivedCount))

                //todo, wyong, 20210122
                //linkCount, ok := parentUrlNode.LinksCount[url]
                linksCount, ok := domain.GetUrlLinksCount(parentUrl, url )

                //bugfix, wyong, 20200313
                if ok != false  {
                        forwardCount.Add(forwardCount, new(big.Float).SetInt( linksCount))
                }
        }
        fmt.Printf("GetProbability(40), parentRetrivedCount=%s, forwardCount=%s \n",
                                        parentRetrivedCount.String(), forwardCount.String())

        forwardProbability.Quo(forwardCount, parentRetrivedCount)
        forwardProbability.Mul(forwardProbability, parentProbability)

        fmt.Printf("GetProbability(50), forwardProbability=%s\n", forwardProbability.String())
        return forwardProbability, nil

}


//wyong, 20200821
func (domain *Domain) SetCid( url string, c cid.Cid) (err error) {

	//todo, wyong, 20210122 
	domain.urlCidCache.Set(url, &types.UrlCid {
		Cid : c, 
		From :
		Vhash :
		Proof :
		Chash :
	}) 

	log.Debugf("Domain/SetCid(10), url=%s\n", url) 

	//todo, build query , wyong, 20210122 
	q := fmt.Sprintf( "INSERT INTO urlcid VALUES('%s', '%s')", url, c.String(), ... ) 
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
		log.Debugf("Domain/SetCid(35)\n") 
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	log.Debugf("Domain/SetCid(40)\n" ) 
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
func (domain *Domain) GetCid(parenturl string,  url string) (c cid.Cid, err error) {

	log.Debugf("Domain/GetCid(10), url=%s\n", url) 
	if urlcids, found := domain.urlCidCache.Get(url) ; found {
		
	} else {

		//todo, get cids by multiple crawlers, wyong, 20210120 
		q := fmt.Sprintf( "SELECT cid FROM urlcid where url='%s'" , url ) 
		query := types.Query {
			Pattern : q ,
		}

		//todo, check those cids, and find out the best cid, wyong, 20210120 

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
			return 
		}

		log.Debugf("Domain/GetCid(40)\n") 
		//result process , wyong, 20200817 
		rows := resp.Payload.Rows //all the rows 
		if rows==nil  || len(rows) <= 0 {
			return 
		}

		log.Debugf("Domain/GetCid(50), first row =%s\n", rows[0]) 
		fr := rows[0]		//first row
		if len(fr.Values) <=0 {
			return 
		} 

		log.Debugf("Domain/GetCid(60), first column of first rows=%s \n", fr.Values[0] ) 

		//bugfix, wyong, 20201208 
		//fcfr:= fr.Values[0].(string)	//first column of first row 
		fcfr:= B2S((fr.Values[0]).([]uint8))	//first column of first row 

		log.Debugf("Domain/GetCid(70), raw cid =%s\n", fcfr ) 
		c, err  = cid.Decode(fcfr)	//convert it to cid 
		if  err != nil {
			log.Debugf("Domain/GetCid(75)\n") 
			return 
		}

		//todo, put urlcids to urlCidCache, wyong, 20210122 

	}

	//todo, process urlcids to get cid, wyong, 20210122 

	log.Debugf("Domain/GetCid(80), cid = %s\n", c.String()) 
	return 
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
		tracker     *x.QueryTracker
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
		if domain.cfg.UseEventualConsistency {
			// reset context
			request.SetContext(context.Background())
			if tracker, response, err = domain.chain.Query(request, true); err != nil {
				err = errors.Wrap(err, "failed to execute with eventual consistency")
				return
			}
		} else {
			if tracker, response, err = domain.writeQuery(request); err != nil {
				err = errors.Wrap(err, "failed to execute")
				return
			}
		}
	default:
		// TODO(xq262144): verbose errors with custom error structure
		return nil, errors.Wrap(ErrInvalidRequest, "invalid query type")
	}

	//todo, wyong, 20200930 
	//response.Header.ResponseAccount = domain.accountAddr

	// build hash
	if err = response.BuildHash(); err != nil {
		err = errors.Wrap(err, "failed to build response hash")
		return
	}

	if err = domain.chain.AddResponse(&response.Header); err != nil {
		log.WithError(err).Debug("failed to add response to index")
		return
	}
	tracker.UpdateResp(response)

	return
}

func (domain *Domain) writeQuery(request *types.Request) (tracker *x.QueryTracker, response *types.Response, err error) {
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


