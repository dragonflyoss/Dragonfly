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

package gc

import (
	"context"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/timeutils"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/sirupsen/logrus"
)

const (
	// gcPeersTimeout specifies the timeout for peers gc.
	// If the actual execution time exceeds this threshold, a warning will be thrown.
	gcPeersTimeout = 2.0 * time.Second
)

func (gcm *Manager) gcPeers(ctx context.Context) {
	var gcPeerCount int
	startTime := time.Now()
	peerIDs := gcm.peerMgr.GetAllPeerIDs(ctx)

	for _, peerID := range peerIDs {
		if gcm.cfg.IsSuperPID(peerID) {
			continue
		}
		peerState, err := gcm.progressMgr.GetPeerStateByPeerID(ctx, peerID)
		if err != nil {
			logrus.Warnf("gc peers: failed to get peerState peerID(%s): %v", peerID, err)
			gcm.gcPeer(ctx, peerID)
			continue
		}

		if peerState.ServiceDownTime == 0 ||
			timeutils.GetCurrentTimeMillis()-peerState.ServiceDownTime < int64(gcm.cfg.PeerGCDelay/time.Millisecond) {
			continue
		}

		gcm.gcPeer(ctx, peerID)
		gcPeerCount++
	}

	// slow GC detected, report it with a log warning
	if timeDuring := time.Since(startTime); timeDuring > gcPeersTimeout {
		logrus.Warnf("gc peers:%d cost:%.3f", gcPeerCount, timeDuring.Seconds())
	}

	gcm.metrics.gcPeersCount.WithLabelValues().Add(float64(gcPeerCount))

	logrus.Infof("gc peers: success to gc peer count(%d), remainder count(%d)", gcPeerCount, len(peerIDs)-gcPeerCount)
}

func (gcm *Manager) gcPeer(ctx context.Context, peerID string) {
	logrus.Infof("gc peer: start to deal with peer: %s", peerID)

	util.GetLock(peerID, false)
	defer util.ReleaseLock(peerID, false)

	var wg sync.WaitGroup
	wg.Add(2)

	go func(wg *sync.WaitGroup) {
		gcm.gcCIDsByPeerID(ctx, peerID)
		wg.Done()
	}(&wg)

	go func(wg *sync.WaitGroup) {
		gcm.gcPeerByPeerID(ctx, peerID)
		wg.Done()
	}(&wg)

	wg.Wait()
}

func (gcm *Manager) gcCIDsByPeerID(ctx context.Context, peerID string) {
	// get related CIDs
	keys, err := gcm.dfgetTaskMgr.GetCIDAndTaskIDsByPeerID(ctx, peerID)
	if err != nil {
		logrus.Errorf("gc Peer: failed to get cids with corresponding taskID by specified peerID(%s): %v", peerID, err)
	}
	var cids []string
	for key := range keys {
		cids = append(cids, key)
	}

	if err := gcm.gcDfgetTasks(ctx, keys, cids); err != nil {
		logrus.Errorf("gc Peer: failed to gc dfgetTask info peerID(%s): %v", peerID, err)
	}
}

func (gcm *Manager) gcPeerByPeerID(ctx context.Context, peerID string) {
	if err := gcm.progressMgr.DeletePeerID(ctx, peerID); err != nil {
		logrus.Errorf("gc Peer: failed to gc progress peer info peerID(%s): %v", peerID, err)
	}
	if err := gcm.peerMgr.DeRegister(ctx, peerID); err != nil {
		logrus.Errorf("gc Peer: failed to gc peer info peerID(%s): %v", peerID, err)
	}
}
