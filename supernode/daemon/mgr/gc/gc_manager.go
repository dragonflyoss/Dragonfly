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

// StartGC start to do the gc jobs.
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

// GCTask to do the gc job with specified taskID.
func (gcm *Manager) GCTask(ctx context.Context, taskID string) {
	gcm.gcTask(ctx, taskID)
}

// GCPeer to do the gc job when a peer offline.
func (gcm *Manager) GCPeer(ctx context.Context, peerID string) {
	gcm.gcPeer(ctx, peerID)
}
