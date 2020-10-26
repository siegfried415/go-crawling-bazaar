package message

import (
	//"fmt"
	"io"

	//wyong, 20190124
	//"bufio"
        //"encoding/json"
        //"encoding/binary"

	pb "github.com/siegfried415/gdf-rebuild/frontera/message/pb"
	wantlist "github.com/siegfried415/gdf-rebuild/frontera/wantlist"
	//bidlist "github.com/siegfried415/gdf-rebuild/frontera/bidlist"
	//blocks "github.com/ipfs/go-block-format"
        //peer "github.com/libp2p/go-libp2p-peer"
	"github.com/siegfried415/gdf-rebuild/proto"

	cid "github.com/ipfs/go-cid"
	ggio "github.com/gogo/protobuf/io"
	inet "github.com/libp2p/go-libp2p-net"
	
	//wyong, 20190124
	logging "github.com/ipfs/go-log"
)


//wyong, 20190124
var log = logging.Logger("message")

// TODO move message.go into the bitswap package
// TODO move bs/msg/internal/pb to bs/internal/pb and rename pb package to bitswap_pb

type BiddingMessage interface {
	// Wantlist returns a slice of unique keys that represent data wanted by
	// the sender.
	Wantlist() []BiddingEntry

	// Cids returns a slice of unique cids of urls
	//wyong, 20190114
	Biddings() []BiddingEntry

	// AddEntry adds an entry to the Wantlist.
	AddBidding(url string, priority int, cids map[proto.NodeID]cid.Cid )

	Cancel(url string)

	Empty() bool

	// A full wantlist is an authoritative copy, a 'non-full' wantlist is a patch-set
	Full() bool

	//AddBidEntry(bid *BidEntry)
	Exportable

	Loggable() map[string]interface{}
}

type Exportable interface {
	/*
	ToProtoV0() *pb.Message
	ToProtoV1() *pb.Message
	ToNetV0(w io.Writer) error
	ToNetV1(w io.Writer) error
	*/

	//wyong, 20190124 
	//ToProto() *pb.Message

	ToNet(w io.Writer) error
}

type impl struct {
	full     bool
	wantlist map[string]*BiddingEntry
	//bids	 map[string]*BidEntry

	from 	string 	//wyong, 20200826 
}

func New(full bool, from string ) BiddingMessage {
	return newMsg(full, from )
}

func newMsg(full bool, from string ) *impl {
	return &impl{
		//bids:   make(map[string]*BidEntry),
		wantlist: make(map[string]*BiddingEntry),
		full:     full,
		from : 	 from , //wyong, 20200826  
	}
}

type BiddingEntry struct {
	*wantlist.BiddingEntry
	Cancel bool
}

/*
type BidEntry struct {
	*bidlist.BidEntry 
}
*/


//wyong, 20190124 
func newMessageFromProto(pbm pb.Message) (BiddingMessage, error) {
        log.Debugf("newMessageFromProto called...")

	m := newMsg(pbm.Wantlist.Full, pbm.From )
	for _, e := range pbm.Wantlist.Entries {
		url := e.GetUrl()
        	log.Debugf("get bidding with url %s", url )
		//if err != nil {
		//	return nil, fmt.Errorf("incorrectly formatted url in wantlist: %s", err)
		//}

		//wyong, 20190115 
		bids := make(map[proto.NodeID]*wantlist.BidEntry)
		for _, bid := range e.GetBids() {
			//TODO, check url and cid, wyong, 20181220 
			cid, err := cid.Cast(bid.GetCid())
			if err != nil {
				return nil, err
			}

        		log.Debugf("get bid with cid %s", cid )

			from,err := proto.IDFromBytes(bid.GetFrom())
			if err != nil {
				return nil, err
			}
			
        		log.Debugf("get bid with from %s", from)

			//bids = append(bids, &wantlist.BidEntry{cid: cid})
			bids[from] = wantlist.NewBidEntry(cid, from )

		}

		m.addBidding(url, int(e.Priority), e.Cancel, bids)
	}

	return m, nil
}


/*
func NewBid(url string, cid cid.Cid) (*BidEntry, error) {
	return &BidEntry {
			BidEntry: &bidlist.BidEntry{
				Url:      url,
				Cid : cid,
			},
	},nil  
}
*/


func (m *impl) Full() bool {
	return m.full
}

func (m *impl) Empty() bool {
	return len(m.wantlist) == 0
}

func (m *impl) Wantlist() []BiddingEntry {
	out := make([]BiddingEntry, 0, len(m.wantlist))
	for _, e := range m.wantlist {
		out = append(out, *e)
	}
	return out
}


/* wyong, 20190114 
func (m *impl) Bids() []BidEntry {
	bs := make([]BidEntry, 0, len(m.bids))
	for _, bid := range m.bids {
		bs = append(bs, *bid)
	}
	return bs
}
*/

// wyong, 20190114 
func (m *impl) Biddings() []BiddingEntry {
	biddings := make([]BiddingEntry, 0, len(m.wantlist))
	for _, bidding := range m.wantlist {
		biddings = append(biddings, *bidding)
	}
	return biddings
}

func (m *impl) Cancel(url string) {
	delete(m.wantlist, url)
	m.addBidding(url, 0, true, nil )
}

//TODO, those cids are all have same from, is it corrent, wyong, 20190118 
func (m *impl) AddBidding(url string, priority int, cids map[proto.NodeID]cid.Cid ) {
	//wyong, 20190115 
	//bids := make([]wantlist.BidEntry, 0, len(cids))
	bids := make(map[proto.NodeID]*wantlist.BidEntry)

	for from, cid := range cids {
		//bids = append(bids, &wantlist.BidEntry{cid : cid } )
		bids[from] = wantlist.NewBidEntry(cid, from )
	}

	m.addBidding(url, priority, false, bids )
}

func (m *impl) addBidding(url string, priority int, cancel bool, bids map[proto.NodeID]*wantlist.BidEntry) {
	e, exists := m.wantlist[url]
	if exists {
		e.Priority = priority
		e.Cancel = cancel
	} else {
		m.wantlist[url] = &BiddingEntry{
			BiddingEntry: &wantlist.BiddingEntry{
				Url:      url,
				Priority: priority,
				Bids:	  bids, 		//wyong, 20190115 
			},
			Cancel: cancel,
		}
	}
}

/* wyong, 20190114
func (m *impl) AddBidEntry(b *BidEntry) {
	m.bids[b.Url] = b  
}
*/

func FromNet(r io.Reader) (BiddingMessage, error) {
        log.Debugf("FromNet called...")
	pbr := ggio.NewDelimitedReader(r, inet.MessageSizeMax)
	return FromPBReader(pbr)
}

func FromPBReader(pbr ggio.Reader) (BiddingMessage, error) {
	pb := new(pb.Message)
	if err := pbr.ReadMsg(pb); err != nil {
		return nil, err
	}

	return newMessageFromProto(*pb)
}

/*
//wyong, 20190124
func FromJsonReader(r io.Reader) (BiddingMessage,error) {
        log.Debugf("FromJsonReader called...")
	br := bufio.NewReader(r)

        length64, err := binary.ReadUvarint(br)
        if err != nil {
                return nil, err
        }

        length := int(length64)
        log.Debugf("FromJsonReader(10), length =%d", length )
        //if length < 0 || length > this.maxSize {
        //        return nil, io.ErrShortBuffer
        //}

	buf := make([]byte, length) 
	if _, err := io.ReadFull(br, buf); err != nil {
		return nil, err
	}

        log.Debugf("FromJsonReader(20), buf = %s", string(buf))

	pb := new(pb.Message)
	if err := json.Unmarshal(buf, pb); err!= nil {
		return nil, err
	}

        log.Debugf("FromJsonReader(30) ")
	//return msg, nil

	return newMessageFromProto(*pb)

}
*/


func (m *impl) ToProto() *pb.Message {
        log.Debugf("ToProto called...")
	pbm := new(pb.Message)
	pbm.Wantlist.Entries = make([]*pb.Message_Wantlist_Entry, 0, len(m.wantlist))
	for _, e := range m.wantlist {

        	log.Debugf("prepare sending bidding of url %s", e.GetUrl())
		//wyong, 20190115 
		bids := make([]*pb.Message_Wantlist_Entry_Bid, 0, len(e.GetBids()))
		for _, bid := range e.GetBids() {
        		log.Debugf("prepare sending bid with cid %s ", bid.Cid )
			bids = append(bids, &pb.Message_Wantlist_Entry_Bid{
				//Cid:   bid.Cid.Prefix().Bytes(), 		
				Cid:   bid.Cid.Bytes(), 		
				From : []byte(bid.From),
			})
		}
		
		pbm.Wantlist.Entries = append(pbm.Wantlist.Entries, &pb.Message_Wantlist_Entry{
			Url:	  e.Url, 
			Priority: int32(e.Priority),
			Cancel:   e.Cancel,
			Bids  :	    bids, 
		})
	}

	pbm.Wantlist.Full = m.full
	return pbm
}

func (m *impl) ToNet(w io.Writer) error {
        log.Debugf("ToNet called...")
	pbw := ggio.NewDelimitedWriter(w)
	return pbw.WriteMsg(m.ToProto())
}

/* wyong, 20190124 
func (m *impl) ToNet(w io.Writer) error {
        log.Debugf("ToNet called...")

	msg := m.ToProto()
	data, err := json.Marshal(*msg)
	if err != nil {
		return err
	}
        log.Debugf("ToNet(10), data = %s", string(data))

        length := uint64(len(data))
        log.Debugf("ToNet(20), length = %d", length )

	//TODO, how many byte can hold a uint64?  wyong, 20190124 
	lenBuf := make([]byte, 8) 

        n := binary.PutUvarint(lenBuf, length)
        _, err = w.Write(lenBuf[:n])
        if err != nil {
                return err
        }

        log.Debugf("ToNet(30) ")
	_, err = w.Write(data)

        log.Debugf("ToNet(40) ")
	return err 
	
}
*/


func (m *impl) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"wants":  m.Wantlist(),
	}
}
