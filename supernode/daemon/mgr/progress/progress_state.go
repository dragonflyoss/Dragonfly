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

package progress

import (
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/atomiccount"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"

	"github.com/willf/bitset"
)

type superState struct {
	// pieceBitSet maintains the piece bitSet of CID
	// which means that the status of each pieces of the task corresponding to taskID on the supernode.
	pieceBitSet *bitset.BitSet
}

type clientState struct {
	// pieceBitSet maintains the piece bitSet of CID
	// which means that the status of each pieces of the task on the peer corresponding to cid.
	pieceBitSet *bitset.BitSet

	// runningPiece maintains the pieces currently being downloaded from dstCID to srcCID.
	// key:pieceNum,value:dstPID
	runningPiece *syncmap.SyncMap
}

type peerState struct {
	// loadNum is the load of download services provided by the current node.
	//
	// This filed should be initialized in advance. If not, it will return an error.
	producerLoad *atomiccount.AtomicInt

	// clientErrorCount maintains the number of times that PeerID failed to downloaded from the other peer nodes.
	//
	// When this field is used, it will be initialized automatically with new AtomicInteger(0)
	// if it is not initialized.
	clientErrorCount *atomiccount.AtomicInt

	// serviceErrorCount maintains the number of times that the other peer nodes failed to downloaded from the PeerID.
	//
	// When this field is used, it will be initialized automatically with new AtomicInteger(0)
	// if it is not initialized.
	serviceErrorCount *atomiccount.AtomicInt

	// serviceDownTime the down time of the peer service.
	serviceDownTime int64
}

type superLoadState struct {
	// superLoad maintains the load num downloaded from the supernode for each task.
	loadValue *atomiccount.AtomicInt

	// loadModTime will record the time when the load be modified.
	loadModTime time.Time
}

func newSuperState() *superState {
	return &superState{
		pieceBitSet: &bitset.BitSet{},
	}
}

func newClientState() *clientState {
	return &clientState{
		pieceBitSet:  &bitset.BitSet{},
		runningPiece: syncmap.NewSyncMap(),
	}
}

func newPeerState() *peerState {
	return &peerState{
		producerLoad:      atomiccount.NewAtomicInt(0),
		clientErrorCount:  atomiccount.NewAtomicInt(0),
		serviceErrorCount: atomiccount.NewAtomicInt(0),
	}
}

func newSuperLoadState() *superLoadState {
	return &superLoadState{
		loadValue:   atomiccount.NewAtomicInt(0),
		loadModTime: time.Now(),
	}
}
