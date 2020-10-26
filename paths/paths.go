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

package paths

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

// node repo path defaults
const filPathVar = "FIL_PATH"
const defaultRepoDir = "~/.gcb"

// node sector storage path defaults
const filSectorPathVar = "FIL_SECTOR_PATH"
const defaultSectorDir = ".filecoin_sectors"
const defaultSectorStagingDir = "staging"
const defaultSectorSealingDir = "sealed"

// GetRepoPath returns the path of the filecoin repo from a potential override
// string, the FIL_PATH environment variable and a default of ~/.filecoin/repo.
func GetRepoPath(override string) (string, error) {
	// override is first precedence
	if override != "" {
		return homedir.Expand(override)
	}
	// Environment variable is second precedence
	envRepoDir := os.Getenv(filPathVar)
	if envRepoDir != "" {
		return homedir.Expand(envRepoDir)
	}
	// Default is third precedence
	return homedir.Expand(defaultRepoDir)
}

// GetSectorPath returns the path of the filecoin sector storage from a
// potential override string, the FIL_SECTOR_PATH environment variable and a
// default of repoPath/../.filecoin_sectors.
func GetSectorPath(override, repoPath string) (string, error) {
	// override is first precedence
	if override != "" {
		return homedir.Expand(override)
	}
	// Environment variable is second precedence
	envRepoDir := os.Getenv(filSectorPathVar)
	if envRepoDir != "" {
		return homedir.Expand(envRepoDir)
	}
	// Default is third precedence: repoPath/../defaultSectorDir
	return homedir.Expand(filepath.Join(repoPath, "../", defaultSectorDir))
}

// StagingDir returns the path to the sector staging directory given the sector
// storage directory path.
func StagingDir(sectorPath string) (string, error) {
	return homedir.Expand(filepath.Join(sectorPath, defaultSectorStagingDir))
}

// SealedDir returns the path to the sector sealed directory given the sector
// storage directory path.
func SealedDir(sectorPath string) (string, error) {
	return homedir.Expand(filepath.Join(sectorPath, defaultSectorSealingDir))
}
