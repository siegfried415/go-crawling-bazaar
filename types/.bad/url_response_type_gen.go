package types

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z *UrlResponse) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	// map header, size 2
	o = append(o, 0x82, 0x82)
	if oTemp, err := z.Header.UrlResponseHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.Header.UrlResponseHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}

	/* wyong, 20200824 
	if oTemp, err := z.Payload.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	*/
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *UrlResponse) Msgsize() (s int) {
	s = 1 + 7 + 1 + 15 + z.Header.UrlResponseHeader.Msgsize() + 13 + z.Header.UrlResponseHash.Msgsize() /* wyong, 20200824  + 8 + z.Payload.Msgsize() */ 
	return
}

// MarshalHash marshals for hash
func (z *UrlResponseHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 10
	o = append(o, 0x8a)
	o = hsp.AppendInt64(o, z.AffectedRows)
	o = hsp.AppendInt64(o, z.LastInsertID)
	o = hsp.AppendUint64(o, z.LogOffset)
	if oTemp, err := z.NodeID.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.PayloadHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.Request.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.RequestHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.ResponseAccount.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = hsp.AppendUint64(o, z.RowCount)
	o = hsp.AppendTime(o, z.Timestamp)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *UrlResponseHeader) Msgsize() (s int) {
	s = 1 + 13 + hsp.Int64Size + 13 + hsp.Int64Size + 10 + hsp.Uint64Size + 7 + z.NodeID.Msgsize() + 12 + z.PayloadHash.Msgsize() + 8 + z.Request.Msgsize() + 12 + z.RequestHash.Msgsize() + 16 + z.ResponseAccount.Msgsize() + 9 + hsp.Uint64Size + 10 + hsp.TimeSize
	return
}

/* wyong, 20200824 
// MarshalHash marshals for hash
func (z *UrlResponsePayload) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 3
	o = append(o, 0x83)

	//o = hsp.AppendArrayHeader(o, uint32(len(z.Columns)))
	//for za0001 := range z.Columns {
	//	o = hsp.AppendString(o, z.Columns[za0001])
	//}
	//o = hsp.AppendArrayHeader(o, uint32(len(z.DeclTypes)))
	//for za0002 := range z.DeclTypes {
	//	o = hsp.AppendString(o, z.DeclTypes[za0002])
	//}
	//o = hsp.AppendArrayHeader(o, uint32(len(z.Rows)))
	//for za0003 := range z.Rows {
	//	// map header, size 1
	//	o = append(o, 0x81)
	//	o = hsp.AppendArrayHeader(o, uint32(len(z.Rows[za0003].Values)))
	//	for za0004 := range z.Rows[za0003].Values {
	//		o, err = hsp.AppendIntf(o, z.Rows[za0003].Values[za0004])
	//		if err != nil {
	//			return
	//		}
	//	}
	//}

	//o = hsp.AppendBytes(o, z.Err.Bytes() )

	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *UrlResponsePayload) Msgsize() (s int) {
	s = 1 + 8 + hsp.ArrayHeaderSize
	//for za0001 := range z.Columns {
	//	s += hsp.StringPrefixSize + len(z.Columns[za0001])
	//}
	//s += 10 + hsp.ArrayHeaderSize
	//for za0002 := range z.DeclTypes {
	//	s += hsp.StringPrefixSize + len(z.DeclTypes[za0002])
	//}
	//s += 5 + hsp.ArrayHeaderSize
	//for za0003 := range z.Rows {
	//	s += 1 + 7 + hsp.ArrayHeaderSize
	//	for za0004 := range z.Rows[za0003].Values {
	//		s += hsp.GuessSize(z.Rows[za0003].Values[za0004])
	//	}
	//}

	//for za0001 := range z.Cids {
	//	s += hsp.StringPrefixSize + len(z.Cids[za0001].Bytes())
	//}
	return
}

// MarshalHash marshals for hash
func (z *ResponseRow) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 1
	o = append(o, 0x81)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Values)))
	for za0001 := range z.Values {
		o, err = hsp.AppendIntf(o, z.Values[za0001])
		if err != nil {
			return
		}
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *ResponseRow) Msgsize() (s int) {
	s = 1 + 7 + hsp.ArrayHeaderSize
	for za0001 := range z.Values {
		s += hsp.GuessSize(z.Values[za0001])
	}
	return
}
*/

// MarshalHash marshals for hash
func (z *UrlSignedResponseHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	if oTemp, err := z.UrlResponseHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.UrlResponseHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *UrlSignedResponseHeader) Msgsize() (s int) {
	s = 1 + 13 + z.UrlResponseHash.Msgsize() + 15 + z.UrlResponseHeader.Msgsize()
	return
}
