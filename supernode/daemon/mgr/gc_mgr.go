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

// GCMgr as an interface defines all operations about gc operation.
type GCMgr interface {
	// StartGC starts to execute GC with a new goroutine.
	StartGC(ctx context.Context)

	// GCTask is used to do the gc task job with specified taskID.
	// The CDN file will be deleted when the full is true.
	GCTask(ctx context.Context, taskID string, full bool)

	// GCPeer is used to do the gc peer job when a peer offline.
	GCPeer(ctx context.Context, peerID string)
}
