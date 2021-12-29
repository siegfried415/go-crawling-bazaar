
package types

//go:generate hsp

type UrlNode struct {
        Url     string          
        RetrivedCount           uint32	
	LastCrawledHeight	uint32	
        LastRequestedHeight    	uint32  
        CrawlInterval           uint32	
}

