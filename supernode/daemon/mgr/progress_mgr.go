package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
)

// PeerState maintains peer related information.
type PeerState struct {
	// PeerID identifies a peer uniquely.
	PeerID string

	// ProducerLoad is the load of download services provided by the current node.
	ProducerLoad int32

	// ClientErrorCount maintains the number of times that PeerID failed to downloaded from the other peer nodes.
	ClientErrorCount int32

	// ServiceErrorCount maintains the number of times that the other peer nodes failed to downloaded from the PeerID.
	ServiceErrorCount int32

	// ServiceDownTime the down time of the peer service.
	ServiceDownTime int64
}

// ProgressMgr is responsible for maintaining the correspondence between peer and pieces.
type ProgressMgr interface {
	// InitProgress init the correlation information between peers and pieces, etc.
	InitProgress(ctx context.Context, taskID, peerID, clientID string) error

	// UpdateProgress update the correlation information between peers and pieces.
	// 1. update the info about srcCID to tell the scheduler that corresponding peer has the piece now.
	// 2. update the info about dstPID to tell the scheduler that someone has downloaded the piece form here.
	// Scheduler will calculate the load and times of error/success for every peer to make better decisions.
	UpdateProgress(ctx context.Context, taskID, srcCID, srcPID, dstPID string, pieceNum, pieceStatus int) error

	// GetPieceProgressByCID get all pieces progress with specified clientID.
	// The filter parameter depends on the specific implementation.
	GetPieceProgressByCID(ctx context.Context, taskID, clientID, filter string) (pieceNums []int, err error)

	// DeletePieceProgressByCID delete the pieces progress with specified clientID.
	DeletePieceProgressByCID(ctx context.Context, taskID, clientID string) (err error)

	// GetPeerIDsByPieceNum gets all peerIDs with specified taskID and pieceNum.
	GetPeerIDsByPieceNum(ctx context.Context, taskID string, pieceNum int) (peerIDs []string, err error)

	// DeletePeerIDByPieceNum deletes the peerID which means that
	// the peer no longer provides the service for the pieceNum of taskID.
	DeletePeerIDByPieceNum(ctx context.Context, taskID string, pieceNum int, peerID string) error

	// GetPeerStateByPeerID gets peer state with specified peerID.
	GetPeerStateByPeerID(ctx context.Context, peerID string) (peerState *PeerState, err error)

	// DeletePeerStateByPeerID deletes the peerState by PeerID.
	DeletePeerStateByPeerID(ctx context.Context, peerID string) error

	// GetPeersByTaskID get all peers info with specified taskID.
	GetPeersByTaskID(ctx context.Context, taskID string) (peersInfo []*types.PeerInfo, err error)

	// GetBlackInfoByPeerID get black info with specified peerID.
	GetBlackInfoByPeerID(ctx context.Context, peerID string) (dstPIDMap *cutil.SyncMap, err error)
}
