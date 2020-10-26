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


package version

import (
	"strconv"
	"strings"
)

// Check ensures that we are using the correct version of go
func Check(version string) bool {
	pieces := strings.Split(version, ".")

	if pieces[0] != "go1" {
		return false
	}

	minorVersion, _ := strconv.Atoi(pieces[1])

	if minorVersion > 12 {
		return true
	}

	if minorVersion < 12 {
		return false
	}

	if len(pieces) < 3 {
		return false
	}

	patchVersion, _ := strconv.Atoi(pieces[2])
	return patchVersion >= 1
}
