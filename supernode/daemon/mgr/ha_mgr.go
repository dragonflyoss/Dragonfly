package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
)

// HaMgr is the interface to implement supernode Ha.
type HaMgr interface {

	// CloseHaManager closes the tool used to implement supernode ha.
	CloseHaManager(ctx context.Context) error

	// HADaemon is the etcd daemon progress to manager superodes cluster
	HADaemon(ctx context.Context) error

	// SendPostCopy sends post request to other supernode like dfget
	SendPostCopy(ctx context.Context, req interface{}, path string, node *config.SupernodeInfo) error

	// TriggerOtherSupernodeDownload triggers other superode only download file
	TriggerOtherSupernodeDownload(ctx context.Context, req *types.TaskRegisterRequest) error
}
