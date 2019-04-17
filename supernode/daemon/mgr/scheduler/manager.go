package scheduler

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

var _ mgr.SchedulerMgr = &Manager{}

// Manager is an implement of the interface of SchedulerMgr.
type Manager struct {
	progressMgr mgr.ProgressMgr
}

// NewManager returns a new Manager.
func NewManager(progressMgr mgr.ProgressMgr) *Manager {
	return &Manager{
		progressMgr: progressMgr,
	}
}

// Schedule gets scheduler result with specified taskID, clientID and peerID through some rules.
func (sm *Manager) Schedule(ctx context.Context, taskID, clientID, peerID string) ([]*mgr.PieceResult, error) {
	return nil, nil
}
