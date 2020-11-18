package proto

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z *AddrAndGas) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 3
	o = append(o, 0x83)
	if oTemp, err := z.AccountAddress.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = hsp.AppendUint64(o, z.GasAmount)
	// map header, size 1
	o = append(o, 0x81)
	if oTemp, err := z.RawNodeID.Hash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *AddrAndGas) Msgsize() (s int) {
	s = 1 + 15 + z.AccountAddress.Msgsize() + 10 + hsp.Uint64Size + 10 + 1 + 5 + z.RawNodeID.Hash.Msgsize()
	return
}

// MarshalHash marshals for hash
func (z *Node) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 5
	o = append(o, 0x85)
	o = hsp.AppendString(o, z.Addr)
	o = hsp.AppendString(o, z.DirectAddr)
	o = hsp.AppendString(o, string(z.ID))
	if z.PublicKey == nil {
		o = hsp.AppendNil(o)
	} else {
		if oTemp, err := z.PublicKey.MarshalHash(); err != nil {
			return nil, err
		} else {
			o = hsp.AppendBytes(o, oTemp)
		}
	}
	o = hsp.AppendInt(o, int(z.Role))
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Node) Msgsize() (s int) {
	s = 1 + 5 + hsp.StringPrefixSize + len(z.Addr) + 11 + hsp.StringPrefixSize + len(z.DirectAddr) + 3 + hsp.StringPrefixSize + len(string(z.ID)) + 10
	if z.PublicKey == nil {
		s += hsp.NilSize
	} else {
		s += z.PublicKey.Msgsize()
	}
	s += 5 + hsp.IntSize
	return
}

// MarshalHash marshals for hash
func (z NodeID) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	o = hsp.AppendString(o, string(z))
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z NodeID) Msgsize() (s int) {
	s = hsp.StringPrefixSize + len(string(z))
	return
}

// MarshalHash marshals for hash
func (z *NodeKey) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 1
	o = append(o, 0x81)
	if oTemp, err := z.Hash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *NodeKey) Msgsize() (s int) {
	s = 1 + 5 + z.Hash.Msgsize()
	return
}

// MarshalHash marshals for hash
func (z *RawNodeID) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 1
	o = append(o, 0x81)
	if oTemp, err := z.Hash.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *RawNodeID) Msgsize() (s int) {
	s = 1 + 5 + z.Hash.Msgsize()
	return
}

// MarshalHash marshals for hash
func (z ServerRole) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	o = hsp.AppendInt(o, int(z))
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z ServerRole) Msgsize() (s int) {
	s = hsp.IntSize
	return
}

// MarshalHash marshals for hash
func (z ServerRoles) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	o = hsp.AppendArrayHeader(o, uint32(len(z)))
	for za0001 := range z {
		o = hsp.AppendInt(o, int(z[za0001]))
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z ServerRoles) Msgsize() (s int) {
	s = hsp.ArrayHeaderSize + (len(z) * (hsp.IntSize))
	return
}
