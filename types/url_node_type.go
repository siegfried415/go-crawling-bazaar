
package types


import (
	//"fmt"
	//"time"
	//"math/big" 

	//"github.com/siegfried415/gdf-rebuild/crypto/asymmetric"
	//"github.com/siegfried415/gdf-rebuild/crypto/hash"
	//"github.com/siegfried415/gdf-rebuild/crypto/verifier"
	//"github.com/siegfried415/gdf-rebuild/proto"
)

//go:generate hsp

type UrlNode struct {
        Url     string          //wyong, 20191113
        //Cid     []byte  //cid.Cid
        //ID      *big.Int        //wyong, 20200126

        //wyong, 20191128
        //Owners        []address.Address
        //Revisions       []*Revision
        //Head            uint8

        //Links   map[string]*UrlNode     //change array to map,  wyong, 20191107
        //LinksCount map[string]*big.Int  //wyong, 20191107

        //wyong, 20191107
        //CrawledCount            *big.Int
        //RequestedCount          *big.Int

        RetrivedCount           uint32	//wyong, 20191115

	//wyong, 20210203 
	LastCrawledHeight	uint32	

        //use blockheight rather then time, wyong, 20200217
        //wyong, 20191119
        LastRequestedHeight    	uint32  

        //int64->uint64, wyong, 20200217
        CrawlInterval           uint32	//int64

}

