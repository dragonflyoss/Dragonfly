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

package scheduler

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var _ mgr.SchedulerMgr = &Manager{}

// Manager is an implement of the interface of SchedulerMgr.
type Manager struct {
	cfg         *config.Config
	progressMgr mgr.ProgressMgr
}

// NewManager returns a new Manager.
func NewManager(cfg *config.Config, progressMgr mgr.ProgressMgr) (*Manager, error) {
	return &Manager{
		cfg:         cfg,
		progressMgr: progressMgr,
	}, nil
}

// Schedule gets scheduler result with specified taskID, clientID and peerID through some rules.
func (sm *Manager) Schedule(ctx context.Context, taskID, clientID, peerID string) ([]*mgr.PieceResult, error) {
	// get available pieces
	pieceAvailable, err := sm.progressMgr.GetPieceProgressByCID(ctx, taskID, clientID, "available")
	if err != nil {
		return nil, err
	}
	if len(pieceAvailable) == 0 {
		return nil, errors.Wrapf(errortypes.ErrPeerWait, "taskID(%s) clientID(%s)", taskID, clientID)
	}
	logrus.Debugf("scheduler get available pieces %v for taskID(%s) clientID(%s)", pieceAvailable, taskID, clientID)

	// get running pieces
	pieceRunning, err := sm.progressMgr.GetPieceProgressByCID(ctx, taskID, clientID, "running")
	if err != nil {
		return nil, err
	}
	logrus.Debugf("scheduler get running pieces %v for taskID(%s) clientID(%s)", pieceRunning, taskID, clientID)
	runningCount := len(pieceRunning)
	if runningCount >= sm.cfg.PeerDownLimit {
		return nil, errors.Wrapf(errortypes.PeerContinue, "taskID: %s,clientID: %s", taskID, clientID)
	}

	// prioritize pieces
	pieceNums, err := sm.sort(ctx, pieceAvailable, pieceRunning, taskID)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("scheduler get pieces %v with prioritize for taskID(%s) clientID(%s)", pieceNums, taskID, clientID)

	return sm.getPieceResults(ctx, taskID, clientID, peerID, pieceNums, runningCount)
}

func (sm *Manager) sort(ctx context.Context, pieceNums, runningPieces []int, taskID string) ([]int, error) {
	pieceCountMap, err := sm.getPieceCountMap(ctx, pieceNums, taskID)
	if err != nil {
		return nil, err
	}

	sm.sortExecutor(ctx, pieceNums, getCenterNum(runningPieces), pieceCountMap)
	return pieceNums, nil
}

func (sm *Manager) getPieceCountMap(ctx context.Context, pieceNums []int, taskID string) (map[int]int, error) {
	pieceCountMap := make(map[int]int)
	for i := 0; i < len(pieceNums); i++ {
		// NOTE: should we return errors here or just record an error log?
		peerIDs, err := sm.progressMgr.GetPeerIDsByPieceNum(ctx, taskID, pieceNums[i])
		if err != nil {
			return nil, err
		}
		pieceCountMap[pieceNums[i]] = len(peerIDs)
	}
	return pieceCountMap, nil
}

// sortExecutor sorts the pieces by distributedCount and the distance to center value of running piece nums.
func (sm *Manager) sortExecutor(ctx context.Context, pieceNums []int, centerNum int, pieceCountMap map[int]int) {
	if len(pieceNums) == 0 || len(pieceCountMap) == 0 {
		return
	}

	sort.Slice(pieceNums, func(i, j int) bool {
		// sort by distributedCount to ensure that
		// the least distributed pieces in the network are prioritized
		if pieceCountMap[pieceNums[i]] < pieceCountMap[pieceNums[j]] {
			return true
		}

		if pieceCountMap[pieceNums[i]] > pieceCountMap[pieceNums[j]] {
			return false
		}

		// sort by piece distance when multiple pieces have the same distributedCount
		if abs(pieceNums[i]-centerNum) < abs(pieceNums[j]-centerNum) {
			return true
		}

		// randomly choose whether to exchange when the distance to center value is equal
		if abs(pieceNums[i]-centerNum) == abs(pieceNums[j]-centerNum) {
			randNum := rand.Intn(2)
			if randNum == 0 {
				return true
			}
		}
		return false
	})
}

func (sm *Manager) getPieceResults(ctx context.Context, taskID, clientID, srcPID string, pieceNums []int, runningCount int) ([]*mgr.PieceResult, error) {
	// validate ClientErrorCount
	var useSupernode bool
	srcPeerState, err := sm.progressMgr.GetPeerStateByPeerID(ctx, srcPID)
	if err != nil {
		return nil, err
	}
	if srcPeerState.ClientErrorCount.Get() > int32(sm.cfg.FailureCountLimit) {
		logrus.Warnf("scheduler: peerID: %s got errors for %d times which reaches error limit: %d for taskID(%s)",
			srcPID, srcPeerState.ClientErrorCount.Get(), sm.cfg.FailureCountLimit, taskID)
		useSupernode = true
	}

	pieceResults := make([]*mgr.PieceResult, 0)
	for i := 0; i < len(pieceNums); i++ {
		var dstPID string
		if useSupernode {
			dstPID = sm.cfg.GetSuperPID()
		} else {
			// get peerIDs by pieceNum
			peerIDs, err := sm.progressMgr.GetPeerIDsByPieceNum(ctx, taskID, pieceNums[i])
			logrus.Debugf("scheduler: success to get peerIDs(%v) pieceNum(%d) taskID(%s), clientID(%s)", peerIDs, pieceNums[i], taskID, clientID)
			if err != nil {
				return nil, errors.Wrapf(errortypes.ErrUnknownError, "failed to get peerIDs for pieceNum: %d of taskID: %s", pieceNums[i], taskID)
			}
			dstPID = sm.tryGetPID(ctx, taskID, pieceNums[i], srcPID, peerIDs)
		}

		if dstPID == "" {
			continue
		}

		// We limit the number of simultaneous connections that supernode can accept for each task.
		if sm.cfg.IsSuperPID(dstPID) {
			updated, err := sm.progressMgr.UpdateSuperLoad(ctx, taskID, 1, int32(sm.cfg.PeerDownLimit))
			if err != nil {
				logrus.Warnf("failed to update super load taskID(%s) clientID(%s): %v", taskID, clientID, err)
				continue
			}
			if !updated {
				continue
			}
		}

		if err := sm.progressMgr.UpdateClientProgress(ctx, taskID, clientID, dstPID, pieceNums[i], config.PieceRUNNING); err != nil {
			logrus.Warnf("scheduler: failed to update client progress running for pieceNum(%d) taskID(%s) clientID(%s) dstPID(%s)", pieceNums[i], taskID, clientID, dstPID)
			continue
		}

		pieceResults = append(pieceResults, &mgr.PieceResult{
			TaskID:   taskID,
			PieceNum: pieceNums[i],
			DstPID:   dstPID,
		})

		runningCount++
		if runningCount >= sm.cfg.PeerDownLimit {
			break
		}
	}

	return pieceResults, nil
}

// tryGetPID returns an available dstPID from ps.pieceContainer.
func (sm *Manager) tryGetPID(ctx context.Context, taskID string, pieceNum int, srcPID string, peerIDs []string) (dstPID string) {
	defer func() {
		if dstPID == "" {
			dstPID = sm.cfg.GetSuperPID()
		}
	}()

	for i := 0; i < len(peerIDs); i++ {
		// if failed to get peerState, and then it should not be needed.
		peerState, err := sm.progressMgr.GetPeerStateByPeerID(ctx, peerIDs[i])
		if err != nil {
			logrus.Warnf("scheduler: failed to GetPeerStateByPeerID taskID(%s) peerID(%s): %v", taskID, peerIDs[i], err)
			sm.deletePeerIDByPieceNum(ctx, taskID, pieceNum, peerIDs[i])
			continue
		}

		// if the service has been down, and then it should not be needed.
		if peerState.ServiceDownTime > 0 {
			logrus.Warnf("scheduler: the peer(%s) has been offline and will delete it from piece state", peerIDs[i])
			sm.deletePeerIDByPieceNum(ctx, taskID, pieceNum, peerIDs[i])
			continue
		}

		// if service has failed for EliminationLimit times, and then it should not be needed.
		if peerState.ServiceErrorCount != nil {
			serviceErrorCount := peerState.ServiceErrorCount.Get()
			if int(serviceErrorCount) >= sm.cfg.EliminationLimit {
				logrus.Warnf("scheduler: the peer(%s) has been eliminated to because of too many errors(%d) occurred as a peer server", peerIDs[i], serviceErrorCount)
				sm.deletePeerIDByPieceNum(ctx, taskID, pieceNum, peerIDs[i])
				continue
			}
		}

		// if the v is in the blackList, try the next one.
		blackInfo, err := sm.progressMgr.GetBlackInfoByPeerID(ctx, srcPID)
		if err != nil && !errortypes.IsDataNotFound(err) {
			logrus.Warnf("scheduler: failed to get blackInfo for peerID %s: %v", peerIDs[i], err)
			continue
		}
		if blackInfo != nil && isExistInMap(blackInfo, peerIDs[i]) {
			continue
		}

		if peerState.ProducerLoad != nil {
			if peerState.ProducerLoad.Add(1) <= int32(sm.cfg.PeerUpLimit) {
				return peerIDs[i]
			}
			peerState.ProducerLoad.Add(-1)
		}
	}
	return
}

func (sm *Manager) deletePeerIDByPieceNum(ctx context.Context, taskID string, pieceNum int, peerID string) {
	if err := sm.progressMgr.DeletePeerIDByPieceNum(ctx, taskID, pieceNum, peerID); err != nil {
		logrus.Warnf("scheduler: failed to delete the peerID %s for pieceNum %d of taskID: %s: %v", peerID, pieceNum, taskID, err)
	}
}

// isExistInMap returns whether the key exists in the mmap.
func isExistInMap(mmap *syncmap.SyncMap, key string) bool {
	if mmap == nil {
		return false
	}
	_, err := mmap.Get(key)
	return err == nil
}

// get the center value of the piece num being downloaded
func getCenterNum(runningPieces []int) int {
	if len(runningPieces) == 0 {
		return 0
	}

	totalDistance := 0
	for i := 0; i < len(runningPieces); i++ {
		totalDistance += runningPieces[i]
	}
	return totalDistance / (len(runningPieces))
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}
