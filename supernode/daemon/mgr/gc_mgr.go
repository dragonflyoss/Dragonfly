package mgr

import (
	"context"
)

// GCMgr as an interface defines all operations about gc operation.
type GCMgr interface {
	// StartGC start to execute GC with a new goroutine.
	StartGC(ctx context.Context)

	// GCTask to do the gc task job with specified taskID.
	GCTask(ctx context.Context, taskID string)

	// GCPeer to do the gc peer job when a peer offline.
	GCPeer(ctx context.Context, peerID string)
}
