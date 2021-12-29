
package net

import (
	"context"
	"time"
	"bufio" 
	"io"

	//"github.com/siegfried415/go-crawling-bazaar/utils/log"
	network "github.com/libp2p/go-libp2p-core/network"
	"github.com/siegfried415/go-crawling-bazaar/utils"
)

var sendMessageTimeout = time.Minute * 10


type Stream struct {
	network.Stream
}

func (s Stream)SendMsg(ctx context.Context, msg interface{} ) (*bufio.Writer,  error) {
	w := bufio.NewWriter(s.Stream)

	encMsg, err := utils.EncodeMsgPack(msg)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(encMsg.Bytes()); err != nil { 
		return nil, err
	}

	if err := w.Flush(); err != nil {
		return nil, err
	}

	return w, nil
}

func (s Stream)RecvMsg(ctx context.Context, resp interface{} ) error {
	reader := bufio.NewReader(s.Stream) 

	//todo, is this large enough?  
	var buf[8192]byte
	n, err := reader.Read(buf[:]) //reader.ReadBytes()	
	if err == io.EOF {
		return err 	
	} 
	if err != nil {
		return err 	
	}

	//todo, what shoud we do if n == 8192, 
	err = utils.DecodeMsgPack(buf[:n], resp)
	if err != nil {
		return err 	
	} 

	return err 
}
