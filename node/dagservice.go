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

package node

import (
        "context"
	"io/ioutil" 

        cid "github.com/ipfs/go-cid"
        protocol "github.com/libp2p/go-libp2p-core/protocol"

	dag "github.com/siegfried415/go-crawling-bazaar/dag" 
	//log "github.com/siegfried415/go-crawling-bazaar/utils/log" 
        net "github.com/siegfried415/go-crawling-bazaar/net"
        "github.com/siegfried415/go-crawling-bazaar/types"
)

// GossipRequest defines the gossip request payload.
//type GossipRequest struct {
//	proto.Envelope
//	Node *proto.Node
//	TTL  uint32
//}

// DagService defines the dag service instance.
type DagService struct {
	d *dag.DAG 
}

// NewDagService returns new dag service.
func NewDagService(serviceName string, h net.RoutedHost,  d *dag.DAG) (*DagService, error) {
	ds := &DagService{
		d: d,
	}

	//h.SetStreamHandlerExt( protocol.ID("DAG.Get"), ds.DagCetHandler)
	h.SetStreamHandlerExt( protocol.ID("DAG.Cat"), ds.DagCatHandler)
	//h.SetStreamHandlerExt( protocol.ID("DAG.ImportData"), ds.DagImportDataHandler)

	return ds, nil 
}

func (ds *DagService) DagCatHandler (s net.Stream) {

        ctx := context.Background()
        var req types.DagCatRequestMessage

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	//todo, only the first request handled. 
	Cid, err := cid.Decode(req.Payload.Requests[0].Cid)
	if err != nil {
		return
	}

        dagReader, err := ds.d.Cat(ctx, Cid)
	if err != nil {
                return
        }

	//20220116 
	out, err := ioutil.ReadAll(dagReader)  	
	if err != nil {
                return
        }

        res := &types.DagCatResponse{
                Header: types.DagCatSignedResponseHeader{
                        DagCatResponseHeader: types.DagCatResponseHeader{
                                Request:     req.Header.DagCatRequestHeader,
                                RequestHash: req.Header.Hash(),
                                //NodeID:      rpc.frontera.nodeID, //f.nodeID,
                                //Timestamp:   getLocalTime(),
                        },
                },
                Payload: types.DagCatResponsePayload{
                        Data: out,
                },

        }

        s.SendMsg(ctx, res)
}


/*
func (ds *DagService) DagImportDataHandler (s net.Stream) {
        ctx := context.Background()
        var req types.DagImportDataMessage

        err := s.RecvMsg(ctx, &req)
        if err != nil {
                return
        }

	//todo, create io.Reader from req.Payload.Data. 
        if out, err = ds.d.ImportData(ctx, reader ); err != nil {
                return
        }

	data = out.Read()
        res = &types.DagImportDataResponse{
                Header: types.DagImportDataSignedResponseHeader{
                        DagImportDataResponseHeader: types.DagImportDataResponseHeader{
                                Request:     req.Header.DagRequestHeader,
                                RequestHash: req.Header.Hash(),
                                //NodeID:      rpc.frontera.nodeID, //f.nodeID,
                                Timestamp:   getLocalTime(),
                        },
                },
                Payload: types.DagImportDataResponsePayload{
                        Data: data,
                },

        }

        s.SendMsg(ctx, res)
}
*/
