package mgr

import (
	"context"
)

// PieceResult contains the information about which piece to download from which node.
type PieceResult struct {
	TaskID   string
	PieceNum int
	DstPID   string
}

// SchedulerMgr is responsible for calculating scheduling results according to certain rules.
type SchedulerMgr interface {
	// Schedule gets scheduler result with specified taskID, clientID and peerID through some rules.
	Schedule(ctx context.Context, taskID, clientID, peerID string) ([]*PieceResult, error)
}
