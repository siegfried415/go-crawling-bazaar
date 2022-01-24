/*
 * Copyright (c) 2018 Filecoin Project
 * Copyright (c) 2022 https://github.com/siegfried415
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); 
 * you may not use this file except in compliance with the License. 
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software 
 * distributed under the License is distributed on an "AS IS" BASIS, 
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. 
 * See the License for the specific language governing permissions and 
 * limitations under the License.
 */


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
