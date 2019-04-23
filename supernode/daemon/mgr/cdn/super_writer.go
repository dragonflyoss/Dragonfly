package cdn

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
)

// startWriter writes the stream data from the reader to the underlying storage.
func (cm *Manager) startWriter(ctx context.Context, cfg *config.Config, reader *cutil.LimitReader,
	task *types.TaskInfo, startPieceNum int, httpFileLength int64, pieceContSize int32) error {
	return nil
}
