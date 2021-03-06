package types

// Code generated by github.com/CovenantSQL/HashStablePack DO NOT EDIT.

import (
	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

// MarshalHash marshals for hash
func (z *UrlNode) MarshalHash() (o []byte, err error) {
	var b []byte
	o = hsp.Require(b, z.Msgsize())
	// map header, size 5
	o = append(o, 0x85)
	o = hsp.AppendUint32(o, z.CrawlInterval)
	o = hsp.AppendUint32(o, z.LastCrawledHeight)
	o = hsp.AppendUint32(o, z.LastRequestedHeight)
	o = hsp.AppendUint32(o, z.RetrivedCount)
	o = hsp.AppendString(o, z.Url)
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *UrlNode) Msgsize() (s int) {
	s = 1 + 14 + hsp.Uint32Size + 18 + hsp.Uint32Size + 20 + hsp.Uint32Size + 14 + hsp.Uint32Size + 4 + hsp.StringPrefixSize + len(z.Url)
	return
}
