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
	"context"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/sirupsen/logrus"
)

// DeleteTaskID deletes the super progress with specified taskID.
func (pm *Manager) DeleteTaskID(ctx context.Context, taskID string, pieceTotal int) (err error) {
	pm.superLoad.remove(taskID)
	pm.superProgress.remove(taskID)

	for i := 0; i < pieceTotal; i++ {
		key, err := generatePieceProgressKey(taskID, i)
		if err != nil {
			return err
		}
		pm.pieceProgress.remove(key)
	}
	return nil
}

// DeleteCID deletes the client progress with specified clientID.
func (pm *Manager) DeleteCID(ctx context.Context, clientID string) (err error) {
	return pm.clientProgress.remove(clientID)
}

// DeletePeerID deletes the related info with specified PeerID.
func (pm *Manager) DeletePeerID(ctx context.Context, peerID string) error {
	// NOTE: we should delete the peerID from the pieceProgress.
	// However, it will cost a lot of time to find which one refers to this peerID.
	// So we leave it to be deleted when scheduled.
	pm.deletePeerIDFromPeerProgress(ctx, peerID)
	pm.deletePeerIDFromBlackInfo(ctx, peerID)

	return nil
}

func (pm *Manager) deletePeerIDFromPeerProgress(ctx context.Context, peerID string) bool {
	if err := pm.peerProgress.remove(peerID); err != nil {
		logrus.Errorf("failed to delete peerID(%s) from peerProgress: %v", peerID, err)
		return false
	}
	return true
}

func (pm *Manager) deletePeerIDFromBlackInfo(ctx context.Context, peerID string) bool {
	result := true
	// delete the black info which use peerID as the key
	pm.clientBlackInfo.Delete(peerID)

	// TODO: delete the black info which refers to the specified peerID
	return result
}

// DeletePeerIDByPieceNum deletes the peerID which means that
// the peer no longer provides the service for the pieceNum of taskID.
func (pm *Manager) DeletePeerIDByPieceNum(ctx context.Context, taskID string, pieceNum int, peerID string) error {
	return pm.deletePeerIDByPieceNum(ctx, taskID, pieceNum, peerID)
}

// deletePeerIDByPieceNum deletes the peerID which means that
// the peer no longer provides the service for the pieceNum of taskID.
func (pm *Manager) deletePeerIDByPieceNum(ctx context.Context, taskID string, pieceNum int, peerID string) error {
	key, err := generatePieceProgressKey(taskID, pieceNum)
	if err != nil {
		return err
	}
	return pm.deletePeerIDByPieceProgressKey(ctx, key, peerID)
}

// deletePeerIDByPieceProgressKey deletes the peerID which means that
// the peer no longer provides the service for the pieceNum of taskID.
func (pm *Manager) deletePeerIDByPieceProgressKey(ctx context.Context, pieceProgressKey string, peerID string) error {
	ps, err := pm.pieceProgress.getAsPieceState(pieceProgressKey)
	if err != nil && !errortypes.IsDataNotFound(err) {
		return err
	}

	ps.delete(peerID)
	return nil
}
