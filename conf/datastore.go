/*
 * Copyright 2022 https://github.com/siegfried415
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

package conf 

import (
	"fmt" 
	"path/filepath"

	badgerds "github.com/ipfs/go-ds-badger"
)

func badgerOptions() *badgerds.Options {
	result := &badgerds.DefaultOptions
	result.Truncate = true
	return result
}

func NewDatastore(repoPath string ) (*badgerds.Datastore, error) {
	switch GConf.Datastore.Type {
	case "badgerds":
		ds, err := badgerds.NewDatastore(filepath.Join(repoPath, GConf.Datastore.Path), badgerOptions())
		if err != nil {
			return nil, err
		}
		return ds, nil 

	default:
		return nil, fmt.Errorf("unknown datastore type in config: %s", GConf.Datastore.Type)
	}

	return nil, nil 
}

