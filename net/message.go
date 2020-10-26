
package net

import (
	"context"
	//"sync"
	"time"
	//"fmt"
	"bufio" 
	"io/ioutil" 

        //"github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"

	//wyong, 20200928
	"github.com/ugorji/go/codec"
)

var sendMessageTimeout = time.Minute * 10

//wyong, 20200925 
func SendMsg(ctx context.Context, s inet.Stream, msg interface{} ) (*bufio.Writer,  error) {
	//log.Debugf("msgToStream called...")

	deadline := time.Now().Add(sendMessageTimeout)
	if dl, ok := ctx.Deadline(); ok {
		deadline = dl
	}
	if err := s.SetWriteDeadline(deadline); err != nil {
		//log.Warningf("error setting deadline: %s", err)
	}

	w := bufio.NewWriter(s)

	//Use Messagepack encoder, wyong, 20200928 
	var encMsg []byte
	h := new(codec.MsgpackHandle) 
	enc := codec.NewEncoderBytes(&encMsg, h )
	if err := enc.Encode(msg); err != nil {
		return nil, err
	}

	if _, err := w.Write(encMsg); err != nil { 
		//log.Debugf("error: %s", err)
		return nil, err
	}


	if err := w.Flush(); err != nil {
		//log.Debugf("error: %s", err)
		return nil, err
	}

	if err := s.SetWriteDeadline(time.Time{}); err != nil {
		//log.Warningf("error resetting deadline: %s", err)
	}
	return w, nil
}

//wyong, 20201018 
func RecvMsg(ctx context.Context, s inet.Stream, resp interface{} ) error {
	b, err := ioutil.ReadAll(s)
	if err != nil {
		return err 	
	} 

	h := new(codec.MsgpackHandle) 
	dec := codec.NewDecoderBytes(b, h )

	return dec.Decode(&resp)
}
