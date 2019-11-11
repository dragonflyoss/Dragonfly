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

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/util"
)

// PeerMgr as an interface defines all operations against Peer.
// A Peer represents a web server that provides file downloads for others.
type PeerMgr interface {
	// Register a peer with specified peerInfo.
	// Supernode will generate a unique peerID for every Peer with PeerInfo provided.
	Register(ctx context.Context, peerCreateRequest *types.PeerCreateRequest) (peerCreateResponse *types.PeerCreateResponse, err error)

	// DeRegister offline a peer service and
	// NOTE: update the info related for scheduler.
	DeRegister(ctx context.Context, peerID string) error

	// Get the peer Info with specified peerID.
	Get(ctx context.Context, peerID string) (*types.PeerInfo, error)

	// GetAllPeerIDs returns all peerIDs.
	GetAllPeerIDs(ctx context.Context) (peerIDs []string)

	// List returns a list of peers info with filter.
	List(ctx context.Context, filter *util.PageFilter) (peerList []*types.PeerInfo, err error)
}
