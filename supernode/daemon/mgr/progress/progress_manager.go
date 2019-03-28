package progress

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ mgr.ProgressMgr = &Manager{}

// Manager is an implementation of the interface of ProgressMgr.
type Manager struct {
	// superProgress maintains the super progress.
	// key:taskID,value:*superState
	superProgress *stateSyncMap

	// clientProgress maintains the client progress.
	// key:CID,value:*clientState
	clientProgress *stateSyncMap

	// peerProgress maintains the peer progress.
	// key:PeerID,value:*peerState
	peerProgress *stateSyncMap

	// pieceProgress maintains the information about
	// which peers the piece currently exists on
	// key:pieceNum@taskID,value:*pieceState
	pieceProgress *stateSyncMap

	// clientBlackInfo maintains the blacklist of the PID.
	// key:srcPID,value:map[dstPID]bool
	clientBlackInfo *cutil.SyncMap
}

// NewManager returns a new Manager.
func NewManager() (*Manager, error) {
	return &Manager{
		superProgress:   newStateSyncMap(),
		clientProgress:  newStateSyncMap(),
		peerProgress:    newStateSyncMap(),
		pieceProgress:   newStateSyncMap(),
		clientBlackInfo: cutil.NewSyncMap(),
	}, nil
}

// InitProgress init the correlation information between peers and pieces, etc.
func (pm *Manager) InitProgress(ctx context.Context, taskID, peerID, clientID string) error {
	// validate the param
	if cutil.IsEmptyStr(taskID) {
		return errors.Wrap(errorType.ErrEmptyValue, "taskID")
	}
	if cutil.IsEmptyStr(clientID) {
		return errors.Wrap(errorType.ErrEmptyValue, "clientID")
	}

	// init cdn node if the clientID represents a supernode.
	if isSuperCID(clientID) {
		return pm.superProgress.add(taskID, newSuperState())
	}
	defer func() {
		if err := pm.superProgress.remove(taskID); err != nil {
			logrus.Errorf("failed to delete superProgress for taskID: %s", taskID)
		}
	}()

	// init peer node if the clientID represents a ordinary peer node.
	if err := pm.clientProgress.add(clientID, newClientState()); err != nil {
		return err
	}
	defer func() {
		if err := pm.clientProgress.remove(clientID); err != nil {
			logrus.Errorf("failed to delete clientProgress for clientID: %s", clientID)
		}
	}()

	return pm.peerProgress.add(peerID, newPeerState())
}

// UpdateProgress update the correlation information between peers and pieces.
// NOTE: What if the update failed?
func (pm *Manager) UpdateProgress(ctx context.Context, taskID, srcCID, dstCID, srcPID, dstPID, pieceRange string, pieceStatus int) error {
	// Step1: validate
	if cutil.IsEmptyStr(taskID) {
		return errors.Wrap(errorType.ErrEmptyValue, "taskID")
	}
	if cutil.IsEmptyStr(srcCID) {
		return errors.Wrapf(errorType.ErrEmptyValue, "srcCID for taskID:%s", taskID)
	}

	// Step2: calculate pieceNum
	pieceNum := util.CalculatePieceNum(pieceRange)
	if pieceNum == -1 {
		return errors.Wrapf(errorType.ErrInvalidValue, "pieceRange: %s", pieceRange)
	}

	// Step3: update the PieceProgress
	// Add one more peer for this piece when the srcPID successfully downloads the piece.
	if pieceStatus == config.PieceSUCCESS {
		if err := pm.updatePieceProgress(taskID, srcPID, pieceNum); err != nil {
			return err
		}
	}

	// Step4: update the clientProgress and superProgress
	result, err := pm.updateClientProgress(taskID, srcCID, dstCID, pieceNum, pieceStatus)
	if err != nil {
		return err
	}
	// It means that it's already successful and
	// there is no need to perform subsequent updates
	// when err==nil and result ==false.
	if !result {
		return nil
	}

	// Step5: update the peerProgress
	return pm.updatePeerProgress(taskID, srcPID, dstPID, pieceRange, pieceStatus)
}

// GetPieceByCID get all pieces with specified clientID.
func (pm *Manager) GetPieceByCID(ctx context.Context, taskID, clientID, pieceStatus string) (pieceNums []int, err error) {
	return nil, nil
}

// GetPeersByPieceNum get all peers ID with specified taskID and pieceNum.
func (pm *Manager) GetPeersByPieceNum(ctx context.Context, taskID string, pieceNum int) (peerIDs []string, err error) {
	return nil, nil
}

// GetPeersByTaskID get all peers info with specified taskID.
func (pm *Manager) GetPeersByTaskID(ctx context.Context, taskID string) (peersInfo []*types.PeerInfo, err error) {
	return nil, nil
}
