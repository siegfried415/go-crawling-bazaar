/*
 * Copyright 2018 The CovenantSQL Authors.
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

package frontera

import (
	"context"
	"sync/atomic"
	"time"
	"fmt"
	"math/big" 

	"github.com/pkg/errors"
        cache "github.com/patrickmn/go-cache"
	cid "github.com/ipfs/go-cid"

	"github.com/siegfried415/go-crawling-bazaar/utils/log"
	"github.com/siegfried415/go-crawling-bazaar/types"
	s "github.com/siegfried415/go-crawling-bazaar/state"



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

	activeBiddingsLimit = 16
)
*/


func (domain *Domain) InitTables() (err error) {
	domain.InitUrlNodeTable()
	domain.InitUrlGraphTable() 
	domain.InitUrlCidTable() 
	return nil 
}

//todo
func (domain *Domain) InitUrlNodeTable() (err error) {

	//create urlNode cache
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

//todo
func (domain *Domain) InitUrlGraphTable() (err error) {

	//create urlNode cache
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

func (domain *Domain) InitUrlCidTable() (err error) {

	//create urlNode cache
	domain.urlCidCache = cache.New(5*time.Minute, 10*time.Minute)

	// build query 
	query := types.Query {
		Pattern : "CREATE TABLE urlcid (url TEXT PRIMARY KEY NOT NULL, cid CHAR(64))",
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

func (domain *Domain) CreateUrlNode(url string, lastRequestedHeight uint32, lastCrawledHeight uint32, retrivedCount uint32, crawlInterval uint32  ) (*types.UrlNode, error ) {
	//build query 
	q := fmt.Sprintf( "INSERT INTO urlnode VALUES('%s',%d, %d, %d, %d )", url, lastRequestedHeight, lastCrawledHeight, retrivedCount, crawlInterval  ) 
	query := types.Query {
		Pattern : q ,
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
	_, _, err := domain.chain.Query(request, true)
	if  err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return nil, err
	}

	urlNode := &types.UrlNode {
		Url: url, 
		LastRequestedHeight: lastRequestedHeight,
		LastCrawledHeight : lastCrawledHeight, 
		RetrivedCount : retrivedCount, 
		CrawlInterval : crawlInterval, 
	}

	domain.urlNodeCache.Set(url, urlNode, cache.DefaultExpiration) 
	return urlNode, nil  
}

func (domain *Domain) UpdateUrlNode(urlNode *types.UrlNode ) error {
	//build query 
	q := fmt.Sprintf( "UPDATE urlnode SET LastRequestedHeight = '%d', LastCrawledHeight = '%d', RetrivedCount = '%d', CrawlInterval = '%d' WHERE Url = '%s' ", urlNode.LastRequestedHeight, urlNode.LastCrawledHeight, urlNode.RetrivedCount, urlNode.CrawlInterval , urlNode.Url  ) 
	query := types.Query {
		Pattern : q ,
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
	_, _, err := domain.chain.Query(request, true)
	if  err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return err
	}

	domain.urlNodeCache.Set(urlNode.Url, urlNode, cache.DefaultExpiration) 
	return nil 
}


func (domain *Domain) SetLastRequestedHeight(url string, height uint32) error {
        urlNode, err := domain.GetUrlNode(url)
        if err != nil {
		return err
        }

	urlNode.LastCrawledHeight = height  
	err = domain.UpdateUrlNode(urlNode) 
	if err != nil { 
		return err
	}

	return nil  
}

func (domain *Domain) SetLastCrawledHeight(url string, height uint32) error {
        urlNode, err := domain.GetUrlNode(url)
        if err != nil {
		return err
        }

	urlNode.LastCrawledHeight = height  
	err = domain.UpdateUrlNode(urlNode) 
	if err != nil {
		return err 
	}

	return nil 
}


func (domain *Domain) SetRetrivedCount(url string, count uint32 ) error  {
        urlNode, err := domain.GetUrlNode(url)
        if err != nil {
		return err
        }
	
	urlNode.RetrivedCount = count  
	err = domain.UpdateUrlNode(urlNode) 
	if err != nil { 
		return err
	} 

	return nil 
}

func (domain *Domain) SetCrawlInterval(url string, interval uint32 ) error {
        urlNode, err := domain.GetUrlNode(url)
        if err != nil {
		return err
        }
	
	urlNode.CrawlInterval = interval 
	err = domain.UpdateUrlNode(urlNode ) 
	if err != nil { 
		return err 
	} 

	return nil 
}

func ToU32( value interface{}) uint32 {
	return uint32(value.(int64))
}

func (domain *Domain) GetUrlNode(url string ) (*types.UrlNode, error ) {
	//get UlrNode from domain.urlNodeCache first. 
        urlNodeCache, found := domain.urlNodeCache.Get(url)
        if found {
		return urlNodeCache.(*types.UrlNode), nil 
        }

	//if not found, get it from urlchain
	q := fmt.Sprintf( "SELECT LastRequestedHeight, LastCrawledHeight, RetrivedCount, CrawlInterval FROM urlnode where url='%s'" , url ) 
	query := types.Query {
		Pattern : q ,
	}

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

	request.SetContext(context.Background())
	_, resp, err := domain.chain.Query(request, true)
	if  err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return nil, err 
	}

	//result process 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		return nil, err
	}

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

	//update cache
	domain.urlNodeCache.Set(url, urlNode, cache.DefaultExpiration) 

	return urlNode, nil 
}

func (domain *Domain) SetUrlLinksCount(parenturl string, url string,  linksCount uint32 ) error {
	//build query 
	q := fmt.Sprintf( "UPDATE urlgraph SET count = %d WHERE url ='%s' and child='%s' ", linksCount, parenturl, url ) 

	query := types.Query {
		Pattern : q ,
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
	_, _, err := domain.chain.Query(request, true)
	if  err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return err 
	}

	domain.urlGraphCache.Set(parenturl+url, linksCount, cache.DefaultExpiration) 
	return nil 
}

func (domain *Domain) GetUrlLinksCount(parentUrl string , url string) (uint32, error ) {
	//links count are forward count from parent ulr Node to url node ,
	//get it from domain.urlGraphCache first
        linksCount, found := domain.urlGraphCache.Get(parentUrl+url)
        if found {
                //fmt.Println(foo)
		return linksCount.(uint32), nil 
        }


	//if not found, get it from urlchain
	q := fmt.Sprintf( "SELECT count FROM urlgraph where url = '%s' and child = '%s'" , parentUrl, url ) 
	query := types.Query {
		Pattern : q ,
	}

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

	request.SetContext(context.Background())
	_, resp, err := domain.chain.Query(request, true)
	if  err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 0, err 
	}

	//result process 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		//return 0, errors.New("result error") 
		return 0, nil 
	}

	r0 := rows[0]		//first row
	if len(r0.Values) <=0 {
		return 0, errors.New("result error") 
	} 

	//todo, get linkscount from fr.Values.  
	//reference client/rows.go 
	result := ToU32(r0.Values[0])	

	domain.urlGraphCache.Set(parentUrl+url, result, cache.DefaultExpiration ) 
	return result, nil 
}

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

func (domain *Domain) GetProbability(parentUrl string, parentProb float64, url string ) (float64, error ) {
        forwardProbability := big.NewFloat(0)
        parentProbability := big.NewFloat(parentProb)

        parentRetrivedCount := big.NewFloat(2)
        forwardCount := big.NewFloat(1)

        parentUrlNode, err := domain.GetUrlNode(parentUrl)
        if err != nil  {
		prc := big.NewInt(int64(parentUrlNode.RetrivedCount)) 
                parentRetrivedCount.Add(parentRetrivedCount,  new(big.Float).SetInt(prc))

                linksCount, err := domain.GetUrlLinksCount(parentUrl, url )
                if err != nil  {
			lc := big.NewInt(int64(linksCount)) 
                        forwardCount.Add(forwardCount, new(big.Float).SetInt(/* int(linksCount) */ lc ))
                }
        }

        forwardProbability.Quo(forwardCount, parentRetrivedCount)
        forwardProbability.Mul(forwardProbability, parentProbability)

	result, _ := forwardProbability.Float64()
        return result , nil

}


func (domain *Domain) SetCid( url string, c cid.Cid) (err error) {
	//build query
	q := fmt.Sprintf( "INSERT INTO urlcid (url, cid) VALUES('%s', '%s')", url, c.String()) 
	query := types.Query {
		Pattern : q ,
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
	_, _, err = domain.chain.Query(request, true)
	if  err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return 
	}

	//update cache
	domain.urlCidCache.Set(url, c, cache.DefaultExpiration ) 

	return 
}

func B2S(bs []uint8) string {
	ba := []byte{}
	for _, b := range bs {
		ba = append(ba, byte(b))
	}
	return string(ba)
}


func (domain *Domain) GetCid(url string) (c cid.Cid, err error) {

	if urlcid, found := domain.urlCidCache.Get(url); found {
		return urlcid.(cid.Cid), nil 
	} 

	q := fmt.Sprintf( "SELECT cid FROM urlcid where url='%s'" , url ) 
	query := types.Query {
		Pattern : q ,
	}

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

	request.SetContext(context.Background())
	_, resp, err := domain.chain.Query(request, true)
	if  err != nil {
		err = errors.Wrap(err, "failed to execute with eventual consistency")
		return cid.Cid{}, err 
	}

	//result process 
	rows := resp.Payload.Rows //all the rows 
	if rows==nil  || len(rows) <= 0 {
		return cid.Cid{}, errors.New("there are errors in url chain result")
	}

	r0 := rows[0]		//first row
	if len(r0.Values) <=0 {
		return cid.Cid{}, errors.New("there are errors in url chain result")
	} 


	r0c0 := B2S((r0.Values[0]).([]uint8))	//first column of first row 
	c, err  = cid.Decode(r0c0)	//convert it to cid 
	if  err != nil {
		return cid.Cid{}, err  
	}

	//put urlcid to urlCidCache
	domain.urlCidCache.Set(url, c, cache.DefaultExpiration) 
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
		//we use EventualConsistency only.  
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

	//todo
	response.Header.ResponseAccount = domain.accountAddr

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


