package types

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z *DagCatResponse) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	// map header, size 2
	o = append(o, 0x82, 0x82)
	if oTemp, err := z.Header.DagCatResponseHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.Header.DagCatResponseHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	// map header, size 1
	o = append(o, 0x81)
	o = hsp.AppendBytes(o, z.Payload.Data)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *DagCatResponse) Msgsize() (s int) {
	s = 1 + 7 + 1 + 21 + z.Header.DagCatResponseHeader.Msgsize() + 19 + z.Header.DagCatResponseHash.Msgsize() + 8 + 1 + 5 + hsp.BytesPrefixSize + len(z.Payload.Data)
	return
}

// MarshalHash marshals for hash
func (z *DagCatResponseHeader) MarshalHash() (o []byte, err error) {
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
func (z *DagCatResponseHeader) Msgsize() (s int) {
	s = 1 + 13 + hsp.Int64Size + 13 + hsp.Int64Size + 10 + hsp.Uint64Size + 7 + z.NodeID.Msgsize() + 12 + z.PayloadHash.Msgsize() + 8 + z.Request.Msgsize() + 12 + z.RequestHash.Msgsize() + 16 + z.ResponseAccount.Msgsize() + 9 + hsp.Uint64Size + 10 + hsp.TimeSize
	return
}

// MarshalHash marshals for hash
func (z *DagCatResponsePayload) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 1
	o = append(o, 0x81)
	o = hsp.AppendBytes(o, z.Data)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *DagCatResponsePayload) Msgsize() (s int) {
	s = 1 + 5 + hsp.BytesPrefixSize + len(z.Data)
	return
}

// MarshalHash marshals for hash
func (z *DagCatSignedResponseHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	if oTemp, err := z.DagCatResponseHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.DagCatResponseHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *DagCatSignedResponseHeader) Msgsize() (s int) {
	s = 1 + 19 + z.DagCatResponseHash.Msgsize() + 21 + z.DagCatResponseHeader.Msgsize()
	return
}
