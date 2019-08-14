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

package mgr

import (
	"context"
)

// PieceResult contains the information about which piece to download from which node.
type PieceResult struct {
	TaskID   string
	PieceNum int
	DstPID   string
}

// SchedulerMgr is responsible for calculating scheduling results according to certain rules.
type SchedulerMgr interface {
	// Schedule gets scheduler result with specified taskID, clientID and peerID through some rules.
	Schedule(ctx context.Context, taskID, clientID, peerID string) ([]*PieceResult, error)
}
