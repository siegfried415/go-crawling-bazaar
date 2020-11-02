package types

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z *Ack) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	if oTemp, err := z.Envelope.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	// map header, size 2
	o = append(o, 0x82)
	if oTemp, err := z.Header.AckHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.Header.DefaultHashSignVerifierImpl.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Ack) Msgsize() (s int) {
	s = 1 + 9 + z.Envelope.Msgsize() + 7 + 1 + 10 + z.Header.AckHeader.Msgsize() + 28 + z.Header.DefaultHashSignVerifierImpl.Msgsize()
	return
}

// MarshalHash marshals for hash
func (z *AckHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 4
	o = append(o, 0x84)
	if oTemp, err := z.NodeID.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.Response.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.ResponseHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = hsp.AppendTime(o, z.Timestamp)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *AckHeader) Msgsize() (s int) {
	s = 1 + 7 + z.NodeID.Msgsize() + 9 + z.Response.Msgsize() + 13 + z.ResponseHash.Msgsize() + 10 + hsp.TimeSize
	return
}

// MarshalHash marshals for hash
func (z AckResponse) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 0
	o = append(o, 0x80)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z AckResponse) Msgsize() (s int) {
	s = 1
	return
}

// MarshalHash marshals for hash
func (z *SignedAckHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	if oTemp, err := z.AckHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.DefaultHashSignVerifierImpl.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *SignedAckHeader) Msgsize() (s int) {
	s = 1 + 10 + z.AckHeader.Msgsize() + 28 + z.DefaultHashSignVerifierImpl.Msgsize()
	return
}