package types

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z *MinerIncome) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	o = hsp.AppendUint64(o, z.Income)
	if oTemp, err := z.Miner.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *MinerIncome) Msgsize() (s int) {
	s = 1 + 7 + hsp.Uint64Size + 6 + z.Miner.Msgsize()
	return
}

// MarshalHash marshals for hash
func (z Range) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 2
	o = append(o, 0x82)
	o = hsp.AppendUint32(o, z.From)
	o = hsp.AppendUint32(o, z.To)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Range) Msgsize() (s int) {
	s = 1 + 5 + hsp.Uint32Size + 3 + hsp.Uint32Size
	return
}

// MarshalHash marshals for hash
func (z *UpdateBilling) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 3
	o = append(o, 0x83)
	if oTemp, err := z.DefaultHashSignVerifierImpl.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.TransactionTypeMixin.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	if oTemp, err := z.UpdateBillingHeader.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *UpdateBilling) Msgsize() (s int) {
	s = 1 + 28 + z.DefaultHashSignVerifierImpl.Msgsize() + 21 + z.TransactionTypeMixin.Msgsize() + 20 + z.UpdateBillingHeader.Msgsize()
	return
}

// MarshalHash marshals for hash
func (z *UpdateBillingHeader) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 5
	o = append(o, 0x85)
	if oTemp, err := z.Nonce.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	// map header, size 2
	o = append(o, 0x82)
	o = hsp.AppendUint32(o, z.Range.From)
	o = hsp.AppendUint32(o, z.Range.To)
	if oTemp, err := z.Receiver.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	o = hsp.AppendArrayHeader(o, uint32(len(z.Users)))
	for za0001 := range z.Users {
		if z.Users[za0001] == nil {
			o = hsp.AppendNil(o)
		} else {
			if oTemp, err := z.Users[za0001].MarshalHash(); err != nil {
				return nil, err
			} else {
				o = hsp.AppendBytes(o, oTemp)
			}
		}
	}
	o = hsp.AppendInt32(o, z.Version)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *UpdateBillingHeader) Msgsize() (s int) {
	s = 1 + 6 + z.Nonce.Msgsize() + 6 + 1 + 5 + hsp.Uint32Size + 3 + hsp.Uint32Size + 9 + z.Receiver.Msgsize() + 6 + hsp.ArrayHeaderSize
	for za0001 := range z.Users {
		if z.Users[za0001] == nil {
			s += hsp.NilSize
		} else {
			s += z.Users[za0001].Msgsize()
		}
	}
	s += 2 + hsp.Int32Size
	return
}

// MarshalHash marshals for hash
func (z *UserCost) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 3
	o = append(o, 0x83)
	o = hsp.AppendUint64(o, z.Cost)
	o = hsp.AppendArrayHeader(o, uint32(len(z.Miners)))
	for za0001 := range z.Miners {
		if z.Miners[za0001] == nil {
			o = hsp.AppendNil(o)
		} else {
			// map header, size 2
			o = append(o, 0x82)
			if oTemp, err := z.Miners[za0001].Miner.MarshalHash(); err != nil {
				return nil, err
			} else {
				o = hsp.AppendBytes(o, oTemp)
			}
			o = hsp.AppendUint64(o, z.Miners[za0001].Income)
		}
	}
	if oTemp, err := z.User.MarshalHash(); err != nil {
		return nil, err
	} else {
		o = hsp.AppendBytes(o, oTemp)
	}
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *UserCost) Msgsize() (s int) {
	s = 1 + 5 + hsp.Uint64Size + 7 + hsp.ArrayHeaderSize
	for za0001 := range z.Miners {
		if z.Miners[za0001] == nil {
			s += hsp.NilSize
		} else {
			s += 1 + 6 + z.Miners[za0001].Miner.Msgsize() + 7 + hsp.Uint64Size
		}
	}
	s += 5 + z.User.Msgsize()
	return
}
