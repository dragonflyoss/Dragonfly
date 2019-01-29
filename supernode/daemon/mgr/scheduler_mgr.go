package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// SchedulerMgr is responsible for calculating scheduling results according to certain rules.
type SchedulerMgr interface {
	// Schedule get scheduler result with specified taskID, clientID through some rules.
	Schedule(ctx context.Context, taskID, clientID string) ([]*types.PieceInfo, error)
}
