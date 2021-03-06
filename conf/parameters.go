/*
 * Copyright 2018 The CovenantSQL Authors.
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

// This parameters should be kept consistent in all Presbyterians.
const (
	DefaultConfirmThreshold = float64(2) / 3.0
)

// These parameters will not cause inconsistency within certain range.
const (
	PBStartupRequiredReachableCount = 2 // NOTE: this includes myself
)

// Presbyterian chain improvements proposal heights.
const (
	PBHeightCIPFixProvideService = 675550 // inclusive, in 2019-5-15 16:11:40 +08:00
)
