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
	"time"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"

	"github.com/sirupsen/logrus"
)

var _ mgr.GCMgr = &Manager{}

// Manager is an implementation of the interface of DfgetTaskMgr.
type Manager struct {
	cfg *config.Config

	// mgr objects
	taskMgr      mgr.TaskMgr
	peerMgr      mgr.PeerMgr
	dfgetTaskMgr mgr.DfgetTaskMgr
	progressMgr  mgr.ProgressMgr
	cdnMgr       mgr.CDNMgr
}

// NewManager returns a new Manager.
func NewManager(cfg *config.Config, taskMgr mgr.TaskMgr, peerMgr mgr.PeerMgr,
	dfgetTaskMgr mgr.DfgetTaskMgr, progressMgr mgr.ProgressMgr, cdnMgr mgr.CDNMgr) (*Manager, error) {
	return &Manager{
		cfg:          cfg,
		taskMgr:      taskMgr,
		peerMgr:      peerMgr,
		dfgetTaskMgr: dfgetTaskMgr,
		progressMgr:  progressMgr,
		cdnMgr:       cdnMgr,
	}, nil
}

// StartGC starts to do the gc jobs.
func (gcm *Manager) StartGC(ctx context.Context) {
	logrus.Debugf("start the gc job")

	go func() {
		// delay to execute GC after gcm.initialDelay
		time.Sleep(gcm.cfg.GCInitialDelay)

		// execute the GC by fixed delay
		ticker := time.NewTicker(gcm.cfg.GCMetaInterval)
		for range ticker.C {
			go gcm.gcTasks(ctx)
			go gcm.gcPeers(ctx)
		}
	}()
}

// GCTask is used to do the gc job with specified taskID.
func (gcm *Manager) GCTask(ctx context.Context, taskID string, full bool) {
	gcm.gcTask(ctx, taskID, full)
}

// GCPeer is used to do the gc job when a peer offline.
func (gcm *Manager) GCPeer(ctx context.Context, peerID string) {
	gcm.gcPeer(ctx, peerID)
}
