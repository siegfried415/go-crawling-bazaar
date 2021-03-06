package types

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z NamedArg) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	o = hsp.AppendString(o, z.Name)
	o, err = hsp.AppendIntf(o, z.Value)
	if err != nil {
		return
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z NamedArg) Msgsize() (s int) {
	s = 1 + 5 + hsp.StringPrefixSize + len(z.Name) + 6 + hsp.GuessSize(z.Value)
	return
}

// MarshalHash marshals for hash
func (z *Query) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Args)))
	for za0001 := range z.Args {
		// map header, size 2
		o = append(o, 0x82)
		o = hsp.AppendString(o, z.Args[za0001].Name)
		o, err = hsp.AppendIntf(o, z.Args[za0001].Value)
		if err != nil {
			return
		}
	}
	o = hsp.AppendString(o, z.Pattern)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Query) Msgsize() (s int) {
	s = 1 + 5 + hsp.ArrayHeaderSize
	for za0001 := range z.Args {
		s += 1 + 5 + hsp.StringPrefixSize + len(z.Args[za0001].Name) + 6 + hsp.GuessSize(z.Args[za0001].Value)
	}
	s += 8 + hsp.StringPrefixSize + len(z.Pattern)
	return
}

// MarshalHash marshals for hash
func (z *QueryKey) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 3
	o = append(o, 0x83)
	o = hsp.AppendUint64(o, z.ConnectionID)
	if oTemp, err := z.NodeID.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = hsp.AppendUint64(o, z.SeqNo)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *QueryKey) Msgsize() (s int) {
	s = 1 + 13 + hsp.Uint64Size + 7 + z.NodeID.Msgsize() + 6 + hsp.Uint64Size
	return
}

// MarshalHash marshals for hash
func (z QueryType) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	o = hsp.AppendInt32(o, int32(z))
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z QueryType) Msgsize() (s int) {
	s = hsp.Int32Size
	return
}

// MarshalHash marshals for hash
func (z *Request) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 3
	o = append(o, 0x83)
	if oTemp, err := z.Envelope.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	// map header, size 2
	o = append(o, 0x82)
	if oTemp, err := z.Header.RequestHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.Header.DefaultHashSignVerifierImpl.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	// map header, size 1
	o = append(o, 0x81)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Payload.Queries)))
	for za0001 := range z.Payload.Queries {
		if oTemp, err := z.Payload.Queries[za0001].MarshalHash(); err != nil {
			return nil, err
		} else {
			o = hsp.AppendBytes(o, oTemp)
		}
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Request) Msgsize() (s int) {
	s = 1 + 9 + z.Envelope.Msgsize() + 7 + 1 + 14 + z.Header.RequestHeader.Msgsize() + 28 + z.Header.DefaultHashSignVerifierImpl.Msgsize() + 8 + 1 + 8 + hsp.ArrayHeaderSize
	for za0001 := range z.Payload.Queries {
		s += z.Payload.Queries[za0001].Msgsize()
	}
	return
}

// MarshalHash marshals for hash
func (z *RequestHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 8
	o = append(o, 0x88)
	o = hsp.AppendUint64(o, z.BatchCount)
	o = hsp.AppendUint64(o, z.ConnectionID)
	if oTemp, err := z.DomainID.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.NodeID.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.QueriesHash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = hsp.AppendInt32(o, int32(z.QueryType))
	o = hsp.AppendUint64(o, z.SeqNo)
	o = hsp.AppendTime(o, z.Timestamp)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RequestHeader) Msgsize() (s int) {
	s = 1 + 11 + hsp.Uint64Size + 13 + hsp.Uint64Size + 9 + z.DomainID.Msgsize() + 7 + z.NodeID.Msgsize() + 12 + z.QueriesHash.Msgsize() + 10 + hsp.Int32Size + 6 + hsp.Uint64Size + 10 + hsp.TimeSize
	return
}

// MarshalHash marshals for hash
func (z *RequestPayload) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 1
	o = append(o, 0x81)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Queries)))
	for za0001 := range z.Queries {
		if oTemp, err := z.Queries[za0001].MarshalHash(); err != nil {
			return nil, err
		} else {
			o = hsp.AppendBytes(o, oTemp)
		}
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RequestPayload) Msgsize() (s int) {
	s = 1 + 8 + hsp.ArrayHeaderSize
	for za0001 := range z.Queries {
		s += z.Queries[za0001].Msgsize()
	}
	return
}

// MarshalHash marshals for hash
func (z *SignedRequestHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	if oTemp, err := z.DefaultHashSignVerifierImpl.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.RequestHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *SignedRequestHeader) Msgsize() (s int) {
	s = 1 + 28 + z.DefaultHashSignVerifierImpl.Msgsize() + 14 + z.RequestHeader.Msgsize()
	return
}
