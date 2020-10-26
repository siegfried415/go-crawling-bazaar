/*
 * Copyright (c) 2018 Filecoin Project
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

package commands

import (
	"fmt"
	"io"

	cmdkit "github.com/ipfs/go-ipfs-cmdkit"
	cmds "github.com/ipfs/go-ipfs-cmds"

	"github.com/siegfried415/go-crawling-bazaar/flags"
	//log "github.com/siegfried415/go-crawling-bazaar/utils/log"
)

type versionInfo struct {
	// Commit, is the git sha that was used to build this version of go-filecoin.
	Commit string

	//20220126  
	Version string 
}

var versionCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Show go-crawling-bazaar version information",
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		return re.Emit(&versionInfo{
			Commit : flags.Commit,
			Version : flags.Version, 	
		})
	},
	Type: versionInfo{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, vo *versionInfo) error {
			_, err := fmt.Fprintf(w, "version: %s\n", vo.Version)
			return err
		}),
	},
}
