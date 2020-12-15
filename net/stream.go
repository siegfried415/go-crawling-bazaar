
package net

import (
	"context"
	//"sync"
	"time"
	//"fmt"
	"bufio" 
	//"io/ioutil" 
	"io"

        //"github.com/libp2p/go-libp2p-host"

	//go-libp2p-net -> go-libp2p-core/network
	//wyong, 20201029 
	network "github.com/libp2p/go-libp2p-core/network"

	//wyong, 20200928
	//"github.com/ugorji/go/codec"

	//wyong, 20201119
	"github.com/siegfried415/gdf-rebuild/utils"

	//wyong, 20201202
	"github.com/siegfried415/gdf-rebuild/utils/log"
)

var sendMessageTimeout = time.Minute * 10


type Stream struct {
	network.Stream
}

//wyong, 20201118 
//func NewStream(s network.Stream ) Stream {
//	return Stream {
//		Stream : s 
//	}
//}

//wyong, 20200925 
func (s Stream)SendMsg(ctx context.Context, msg interface{} ) (*bufio.Writer,  error) {
	//log.Debugf("Stream/SendMsg called...")
	
	//deadline := time.Now().Add(sendMessageTimeout)
	//if dl, ok := ctx.Deadline(); ok {
	//	deadline = dl
	//}

	//log.Debugf("Stream/SendMsg(20)\n") 
	//if err := s.Stream.SetWriteDeadline(deadline); err != nil {
	//	log.Warningf("error setting deadline: %s", err)
	//}

	//log.Debugf("Stream/SendMsg(30), input msg=%s\n", msg ) 
	w := bufio.NewWriter(s.Stream)

	//wyong, 20201119 
	encMsg, err := utils.EncodeMsgPack(msg)
	if err != nil {
		log.Debugf("Stream/SendMsg(35), err=%s\n", err ) 
		return nil, err
	}

	//log.Debugf("Stream/SendMsg(40), encMsg=%s\n", encMsg ) 


	if _, err := w.Write(encMsg.Bytes()); err != nil { 
		log.Debugf("Stream/SendMsg(45), error: %s", err)
		return nil, err
	}


	//log.Debugf("Stream/SendMsg(50)\n") 
	if err := w.Flush(); err != nil {
		log.Debugf("Stream/SendMsg(55), error: %s", err)
		return nil, err
	}

	//log.Debugf("Stream/SendMsg(60)\n") 
	//if err := s.Stream.SetWriteDeadline(time.Time{}); err != nil {
	//	log.Warningf("Stream/SendMsg(65), error resetting deadline: %s", err)
	//	//log.Debugf("Stream/SendMsg(65), err=%s\n", err ) 
	//}

	//log.Debugf("Stream/SendMsg(70)\n") 
	return w, nil
}

//wyong, 20201018 
func (s Stream)RecvMsg(ctx context.Context, resp interface{} ) error {
	//log.Debugf("Stream/RecvMsg(10)\n") 
	reader := bufio.NewReader(s.Stream) 

	//todo, is 1024 large enough?  wyong, 20201202 
	var buf[8192]byte
	n, err := reader.Read(buf[:]) //reader.ReadBytes()	
	if err == io.EOF {
		log.Debugf("Stream/RecvMsg(15), err=%s \n", err) 
		return err 	
	} 
	if err != nil {
		log.Debugf("Stream/RecvMsg(17), err=%s \n", err) 
		return err 	
	}

	//todo, what shoud we do if n == 8192, wyong, 20201202 
	//log.Debugf("Stream/RecvMsg(20), buf[:%d]=%s\n", n, buf[:n] ) 

	//todo, wyong, 20201202
	err = utils.DecodeMsgPack(buf[:n], resp)
	if err != nil {
		log.Debugf("Stream/RecvMsg(25), err=%s \n", err) 
		return err 	
	} 

	//log.Debugf("Stream/RecvMsg(30), resp=%s\n", resp ) 
	return err 
}
