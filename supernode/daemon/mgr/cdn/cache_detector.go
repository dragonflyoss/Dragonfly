package cdn

import (
	"context"
	"hash"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

type cacheResult struct {
	startPieceNum int
	pieceMd5s     []string
	fileMD5       hash.Hash
}

// detectCache detects whether there is a corresponding file in the local.
// If any, check whether the entire file has been completely downloaded.
//
// If so, return the md5 of task file and return startPieceNum as -1.
// And if not, return the lastest piece num that has been downloaded.
func (cm *Manager) detectCache(ctx context.Context, task *types.TaskInfo) (*cacheResult, error) {
	// TODO: implement it.
	return nil, nil
}
