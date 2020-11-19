
package net

import (
	"context"
	//"sync"
	"time"
	//"fmt"
	"bufio" 
	//"io/ioutil" 

        //"github.com/libp2p/go-libp2p-host"

	//go-libp2p-net -> go-libp2p-core/network
	//wyong, 20201029 
	network "github.com/libp2p/go-libp2p-core/network"

	//wyong, 20200928
	//"github.com/ugorji/go/codec"

	//wyong, 20201119
	"github.com/siegfried415/gdf-rebuild/utils"
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
	//log.Debugf("msgToStream called...")
	//fmt.Printf("Stream/SendMsg(10)\n") 
	
	deadline := time.Now().Add(sendMessageTimeout)
	if dl, ok := ctx.Deadline(); ok {
		deadline = dl
	}

	//fmt.Printf("Stream/SendMsg(20)\n") 
	if err := s.Stream.SetWriteDeadline(deadline); err != nil {
		//log.Warningf("error setting deadline: %s", err)
	}

	//fmt.Printf("Stream/SendMsg(30), input msg=%s\n", msg ) 
	w := bufio.NewWriter(s.Stream)

	//wyong, 20201119 
	encMsg, err := utils.EncodeMsgPack(msg)
	if err != nil {
		//fmt.Printf("Stream/SendMsg(35), err=%s\n", err ) 
		return nil, err
	}

	//fmt.Printf("Stream/SendMsg(40), encMsg=%s\n", encMsg ) 


	if _, err := w.Write(encMsg.Bytes()); err != nil { 
		//log.Debugf("error: %s", err)
		//fmt.Printf("Stream/SendMsg(45), err=%s\n", err ) 
		return nil, err
	}


	//fmt.Printf("Stream/SendMsg(50)\n") 
	if err := w.Flush(); err != nil {
		//log.Debugf("error: %s", err)
		//fmt.Printf("Stream/SendMsg(55), err=%s\n", err ) 
		return nil, err
	}

	//fmt.Printf("Stream/SendMsg(60)\n") 
	if err := s.Stream.SetWriteDeadline(time.Time{}); err != nil {
		//log.Warningf("error resetting deadline: %s", err)
		//fmt.Printf("Stream/SendMsg(65), err=%s\n", err ) 
	}

	//fmt.Printf("Stream/SendMsg(70)\n") 
	return w, nil
}

//wyong, 20201018 
func (s Stream)RecvMsg(ctx context.Context, resp interface{} ) error {
	//fmt.Printf("Stream/RecvMsg(10)\n") 

	reader := bufio.NewReader(s.Stream) 
	var buf[1024]byte
	n, err := reader.Read(buf[:]) //reader.ReadBytes()	
	if n <= 0  {
		//fmt.Printf("Stream/RecvMsg(15), err=%s \n", err) 
		return err 	
	} 

	//fmt.Printf("Stream/RecvMsg(20), buf[:n]=%s\n", buf[:n] ) 

	//wyong, 20201119 
	err = utils.DecodeMsgPackPlain(buf[:n], resp)
	if err != nil {
		//fmt.Printf("Stream/RecvMsg(25), err=%s \n", err) 
		return err 	
	} 

	//fmt.Printf("Stream/RecvMsg(30), resp=%s\n", resp ) 
	return err 
}
