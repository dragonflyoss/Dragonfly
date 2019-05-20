package cdn

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

var _ mgr.CDNMgr = &Manager{}

// Manager is an implementation of the interface of CDNMgr.
type Manager struct{}

// NewManager returns a new Manager.
func NewManager() (*Manager, error) {
	return &Manager{}, nil
}

// TriggerCDN will trigger CDN to download the file from sourceUrl.
func (cm *Manager) TriggerCDN(ctx context.Context, taskInfo *types.TaskInfo) error {
	return nil
}

// GetStatus get the status of the file.
func (cm *Manager) GetStatus(ctx context.Context, taskID string) (cdnStatus string, err error) {
	return "", nil
}

// Delete the file from disk with specified taskID.
func (cm *Manager) Delete(ctx context.Context, taskID string) error {
	return nil
}
