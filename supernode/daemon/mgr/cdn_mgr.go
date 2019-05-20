package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// CDNMgr as an interface defines all operations against CDN and
// operates on the underlying files stored on the local disk, etc.
type CDNMgr interface {
	// TriggerCDN will trigger CDN to download the file from sourceUrl.
	// In common, it will including the following steps:
	// 1). download the source file
	// 2). update the taskInfo
	// 3). write the file to disk
	TriggerCDN(ctx context.Context, taskInfo *types.TaskInfo) error

	// GetStatus get the status of the file.
	GetStatus(ctx context.Context, taskID string) (cdnStatus string, err error)

	// Delete the file from disk with specified taskID.
	Delete(ctx context.Context, taskID string) error
}

// CDNManager is an implementation of the interface of CDNMgr.
type CDNManager struct{}

// NewCDNManager returns a new CDNManager.
func NewCDNManager() (*CDNManager, error) {
	return &CDNManager{}, nil
}

// TriggerCDN will trigger CDN to download the file from sourceUrl.
func (cm *CDNManager) TriggerCDN(ctx context.Context, taskInfo *types.TaskInfo) error {
	return nil
}

// GetStatus get the status of the file.
func (cm *CDNManager) GetStatus(ctx context.Context, taskID string) (cdnStatus string, err error) {
	return "", nil
}

// Delete the file from disk with specified taskID.
func (cm *CDNManager) Delete(ctx context.Context, taskID string) error {
	return nil
}
