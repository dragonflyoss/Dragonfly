/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package basic

import (
	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// SchedulePeerInfo defines how to get resource from peer info.
type SchedulePeerInfo struct {
	// basic peer info
	*types.PeerInfo
	// Path represents the path to get resource from the peer.
	Path string
}

// SchedulerResult defines the result of schedule of range data.
type SchedulePieceDataResult struct {
	Off  int64
	Size int64

	// PeerInfos represents the schedule peers which to get the range data.
	PeerInfos []*SchedulePeerInfo
}
