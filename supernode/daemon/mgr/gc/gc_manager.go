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

	"github.com/dragonflyoss/Dragonfly/pkg/metricsutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var _ mgr.GCMgr = &Manager{}

type metrics struct {
	gcTasksCount    *prometheus.CounterVec
	gcPeersCount    *prometheus.CounterVec
	gcDisksCount    *prometheus.CounterVec
	lastGCDisksTime *prometheus.GaugeVec
}

func newMetrics(register prometheus.Registerer) *metrics {
	return &metrics{
		gcTasksCount: metricsutils.NewCounter(config.SubsystemSupernode, "gc_tasks_total",
			"Total number of tasks that have been garbage collected", []string{}, register),

		gcPeersCount: metricsutils.NewCounter(config.SubsystemSupernode, "gc_peers_total",
			"Total number of peers that have been garbage collected", []string{}, register),

		gcDisksCount: metricsutils.NewCounter(config.SubsystemSupernode, "gc_disks_total",
			"Total number of garbage collecting the task data in disks", []string{}, register),

		lastGCDisksTime: metricsutils.NewGauge(config.SubsystemSupernode, "last_gc_disks_timestamp_seconds",
			"Timestamp of the last disk gc", []string{}, register),
	}
}

// Manager is an implementation of the interface of DfgetTaskMgr.
type Manager struct {
	cfg *config.Config

	// mgr objects
	taskMgr      mgr.TaskMgr
	peerMgr      mgr.PeerMgr
	dfgetTaskMgr mgr.DfgetTaskMgr
	progressMgr  mgr.ProgressMgr
	cdnMgr       mgr.CDNMgr
	metrics      *metrics
}

// NewManager returns a new Manager.
func NewManager(cfg *config.Config, taskMgr mgr.TaskMgr, peerMgr mgr.PeerMgr, dfgetTaskMgr mgr.DfgetTaskMgr,
	progressMgr mgr.ProgressMgr, cdnMgr mgr.CDNMgr, register prometheus.Registerer) (*Manager, error) {
	return &Manager{
		cfg:          cfg,
		taskMgr:      taskMgr,
		peerMgr:      peerMgr,
		dfgetTaskMgr: dfgetTaskMgr,
		progressMgr:  progressMgr,
		cdnMgr:       cdnMgr,
		metrics:      newMetrics(register),
	}, nil
}

// StartGC starts to do the gc jobs.
func (gcm *Manager) StartGC(ctx context.Context) {
	logrus.Debugf("start the gc job")

	// start a goroutine to gc the tasks
	go func() {
		// delay to execute GC after gcm.initialDelay
		time.Sleep(gcm.cfg.GCInitialDelay)

		// execute the GC by fixed delay
		ticker := time.NewTicker(gcm.cfg.GCMetaInterval)
		for range ticker.C {
			gcm.gcTasks(ctx)
		}
	}()

	// start a goroutine to gc the peers
	go func() {
		// delay to execute GC after gcm.initialDelay
		time.Sleep(gcm.cfg.GCInitialDelay)

		// execute the GC by fixed delay
		ticker := time.NewTicker(gcm.cfg.GCMetaInterval)
		for range ticker.C {
			gcm.gcPeers(ctx)
		}
	}()

	// start a goroutine to gc the disks
	go func() {
		// delay to execute GC after gcm.initialDelay
		time.Sleep(gcm.cfg.GCInitialDelay)

		// execute the GC by fixed delay
		ticker := time.NewTicker(gcm.cfg.GCDiskInterval)
		for range ticker.C {
			gcm.gcDisk(ctx)
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
