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
	"github.com/dragonflyoss/Dragonfly/pkg/atomiccount"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
)

// PeerState maintains peer related information.
type PeerState struct {
	// PeerID identifies a peer uniquely.
	PeerID string

	// ProducerLoad is the load of download services provided by the current node.
	ProducerLoad *atomiccount.AtomicInt

	// ClientErrorCount maintains the number of times that PeerID failed to downloaded from the other peer nodes.
	ClientErrorCount *atomiccount.AtomicInt

	// ServiceErrorCount maintains the number of times that the other peer nodes failed to downloaded from the PeerID.
	ServiceErrorCount *atomiccount.AtomicInt

	// ServiceDownTime the down time of the peer service.
	ServiceDownTime int64
}

// ProgressMgr is responsible for maintaining the correspondence between peer and pieces.
type ProgressMgr interface {
	// InitProgress inits the correlation information between peers and pieces, etc.
	InitProgress(ctx context.Context, taskID, peerID, clientID string) error

	// UpdateProgress updates the correlation information between peers and pieces.
	// 1. update the info about srcCID to tell the scheduler that corresponding peer has the piece now.
	// 2. update the info about dstPID to tell the scheduler that someone has downloaded the piece form here.
	// Scheduler will calculate the load and times of error/success for every peer to make better decisions.
	UpdateProgress(ctx context.Context, taskID, srcCID, srcPID, dstPID string, pieceNum, pieceStatus int) error

	// UpdateClientProgress updates the info when success to schedule peer srcCID to download from dstPID.
	UpdateClientProgress(ctx context.Context, taskID, srcCID, dstPID string, pieceNum, pieceStatus int) error

	// GetPieceProgressByCID gets all pieces progress with specified clientID.
	// The filter parameter depends on the specific implementation.
	GetPieceProgressByCID(ctx context.Context, taskID, clientID, filter string) (pieceNums []int, err error)

	// GetPeerIDsByPieceNum gets all peerIDs with specified taskID and pieceNum.
	GetPeerIDsByPieceNum(ctx context.Context, taskID string, pieceNum int) (peerIDs []string, err error)

	// DeletePeerIDByPieceNum deletes the peerID which means that
	// the peer no longer provides the service for the pieceNum of taskID.
	DeletePeerIDByPieceNum(ctx context.Context, taskID string, pieceNum int, peerID string) error

	// GetPeerStateByPeerID gets peer state with specified peerID.
	GetPeerStateByPeerID(ctx context.Context, peerID string) (peerState *PeerState, err error)

	// UpdateSuperLoad updates the superload of taskID by adding the delta.
	// The updated will be `false` if failed to do update operation.
	//
	// It's considered as a failure when then superload is greater than limit after adding delta.
	UpdatePeerServiceDown(ctx context.Context, peerID string) (err error)

	// GetPeersByTaskID gets all peers info with specified taskID.
	GetPeersByTaskID(ctx context.Context, taskID string) (peersInfo []*types.PeerInfo, err error)

	// GetBlackInfoByPeerID gets black info with specified peerID.
	GetBlackInfoByPeerID(ctx context.Context, peerID string) (dstPIDMap *syncmap.SyncMap, err error)

	// UpdateSuperLoad updates the superLoad with delta.
	//
	// The value will be rolled back if it exceeds the limit after updated and returns false.
	UpdateSuperLoad(ctx context.Context, taskID string, delta, limit int32) (updated bool, err error)

	// DeleteTaskID deletes the super progress with specified taskID.
	DeleteTaskID(ctx context.Context, taskID string, pieceTotal int) (err error)

	// DeleteCID deletes the super progress with specified clientID.
	DeleteCID(ctx context.Context, clientID string) (err error)

	// DeletePeerID deletes the peerState by PeerID.
	DeletePeerID(ctx context.Context, peerID string) (err error)
}
