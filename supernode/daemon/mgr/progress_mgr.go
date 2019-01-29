package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// ProgressMgr is responsible for maintaining the correspondence between peer and pieces.
type ProgressMgr interface {
	// InitProgress init the correlation information between peers and pieces, etc.
	InitProgress(ctx context.Context, taskID, clientID string) error

	// UpdateProgress update the correlation information between peers and pieces.
	// 1. update the info about srcCid to tell the scheduler that corresponding peer has the piece now.
	// 2. update the info about dstCid to tell the scheduler that someone has downloaded the piece form here.
	// Scheduler will calculate the load and times of error/success for every peer to make better decisions.
	UpdateProgress(ctx context.Context, taskID, srcCid, dstCid string, pieceRange string, pieceStatus string) error

	// GetPieceByCid get all pieces with specified clientID.
	GetPiecesByCid(ctx context.Context, clientID string) (pieceRanges []string, err error)

	// GetPeersByPieceRange get all peers info with specified taskID and pieceRange.
	GetPeersByPieceRange(ctx context.Context, taskID, pieceRange string) (peersInfo []*types.PeerInfo, err error)

	// GetPeersBytaskID get all peers info with specified taskID.
	GetPeersBytaskID(ctx context.Context, taskID string) (peersInfo []*types.PeerInfo, err error)
}
