package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// ProgressMgr is responsible for maintaining the correspondence between peer and pieces.
type ProgressMgr interface {
	// InitProgress init the correlation information between peers and pieces, etc.
	InitProgress(ctx context.Context, taskID, peerID, clientID string) error

	// UpdateProgress update the correlation information between peers and pieces.
	// 1. update the info about srcCid to tell the scheduler that corresponding peer has the piece now.
	// 2. update the info about dstCid to tell the scheduler that someone has downloaded the piece form here.
	// Scheduler will calculate the load and times of error/success for every peer to make better decisions.
	UpdateProgress(ctx context.Context, taskID, srcCID, dstCID, srcPID, dstPID, pieceRange string, pieceStatus int) error

	// GetPieceByCID get all pieces with specified clientID.
	GetPieceByCID(ctx context.Context, taskID, clientID, pieceStatus string) (pieceNums []int, err error)

	// GetPeersByPieceNum get all peers ID with specified taskID and pieceNum.
	GetPeersByPieceNum(ctx context.Context, taskID string, pieceNum int) (peerIDs []string, err error)

	// GetPeersByTaskID get all peers info with specified taskID.
	GetPeersByTaskID(ctx context.Context, taskID string) (peersInfo []*types.PeerInfo, err error)
}
