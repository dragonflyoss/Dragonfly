package progress

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/willf/bitset"
)

// PieceStatus which is used for GetPieceByCID.
const (
	// PieceRunning means that the pieces is being downloaded.
	PieceRunning = "running"

	// PieceSuccess means that the piece has been downloaded successful.
	PieceSuccess = "success"

	// PieceAvailable means that the piece has neither been downloaded successfully
	// nor being downloaded and supernode has downloaded it successfully.
	PieceAvailable = "available"
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
	// key:srcPID,value:map[dstPID]*Atomic
	clientBlackInfo *cutil.SyncMap

	cfg *config.Config
}

// NewManager returns a new Manager.
func NewManager(cfg *config.Config) (*Manager, error) {
	return &Manager{
		cfg:             cfg,
		superProgress:   newStateSyncMap(),
		clientProgress:  newStateSyncMap(),
		peerProgress:    newStateSyncMap(),
		pieceProgress:   newStateSyncMap(),
		clientBlackInfo: cutil.NewSyncMap(),
	}, nil
}

// InitProgress init the correlation information between peers and pieces, etc.
func (pm *Manager) InitProgress(ctx context.Context, taskID, peerID, clientID string) (err error) {
	// validate the param
	if cutil.IsEmptyStr(taskID) {
		return errors.Wrap(errorType.ErrEmptyValue, "taskID")
	}
	if cutil.IsEmptyStr(clientID) {
		return errors.Wrap(errorType.ErrEmptyValue, "clientID")
	}
	if cutil.IsEmptyStr(peerID) {
		return errors.Wrap(errorType.ErrEmptyValue, "peerID")
	}

	// init cdn node if the clientID represents a supernode.
	if pm.cfg.IsSuperCID(clientID) {
		return pm.superProgress.add(taskID, newSuperState())
	}

	// init peer node if the clientID represents a ordinary peer node.
	if err := pm.clientProgress.add(clientID, newClientState()); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if err := pm.clientProgress.remove(clientID); err != nil {
				logrus.Errorf("failed to delete clientProgress for clientID: %s", clientID)
			}
		}
	}()

	return pm.peerProgress.add(peerID, newPeerState())
}

// UpdateProgress update the correlation information between peers and pieces.
// NOTE: What if the update failed?
func (pm *Manager) UpdateProgress(ctx context.Context, taskID, srcCID, srcPID, dstPID string, pieceNum, pieceStatus int) error {
	if cutil.IsEmptyStr(taskID) {
		return errors.Wrap(errorType.ErrEmptyValue, "taskID")
	}
	if cutil.IsEmptyStr(srcCID) {
		return errors.Wrapf(errorType.ErrEmptyValue, "srcCID for taskID:%s", taskID)
	}
	if cutil.IsEmptyStr(srcPID) {
		return errors.Wrapf(errorType.ErrEmptyValue, "srcPID for taskID:%s", taskID)
	}

	// Step1: update the PieceProgress
	// Add one more peer for this piece when the srcPID successfully downloads the piece.
	if pieceStatus == config.PieceSUCCESS {
		if err := pm.updatePieceProgress(taskID, srcPID, pieceNum); err != nil {
			logrus.Errorf("failed to update PieceProgress taskID(%s) srcPID(%s) pieceNum(%d): %v",
				taskID, srcPID, pieceNum, err)
			return err
		}
		logrus.Debugf("success to update PieceProgress taskID(%s) srcPID(%s) pieceNum(%d)",
			taskID, srcPID, pieceNum)
	}

	// Step2: update the clientProgress and superProgress
	result, err := pm.updateClientProgress(taskID, srcCID, dstPID, pieceNum, pieceStatus)
	if err != nil {
		logrus.Errorf("failed to update ClientProgress taskID(%s) srcCID(%s) dstPID(%s) pieceNum(%d) pieceStatus(%d): %v",
			taskID, srcCID, dstPID, pieceNum, pieceStatus, err)
		return err
	}
	logrus.Debugf("success to update ClientProgress taskID(%s) srcCID(%s) dstPID(%s) pieceNum(%d) pieceStatus(%d) with result: %t",
		taskID, srcCID, dstPID, pieceNum, pieceStatus, result)
	// It means that it's already successful and
	// there is no need to perform subsequent updates
	// when err==nil and result ==false.
	if !result {
		return nil
	}

	// Step3: update the peerProgress
	if err := pm.updatePeerProgress(taskID, srcPID, dstPID, pieceNum, pieceStatus); err != nil {
		logrus.Errorf("failed to update PeerProgress taskID(%s) srcCID(%s) dstPID(%s) pieceNum(%d) pieceStatus(%d): %v",
			taskID, srcCID, dstPID, pieceNum, pieceStatus, err)
		return err
	}
	logrus.Debugf("success to update PeerProgress taskID(%s) srcCID(%s) dstPID(%s) pieceNum(%d) pieceStatus(%d)",
		taskID, srcCID, dstPID, pieceNum, pieceStatus)
	return nil
}

// GetPieceProgressByCID get all pieces with specified clientID.
//
// And the pieceStatus should be one of the `PieceRunning`,`PieceSuccess` and `PieceAvailable`.
// If not, the `PieceAvailable` will be as the default value.
func (pm *Manager) GetPieceProgressByCID(ctx context.Context, taskID, clientID, pieceStatus string) (pieceNums []int, err error) {
	cs, err := pm.clientProgress.getAsClientState(clientID)
	if err != nil {
		return nil, err
	}

	// get running pieces
	runningPieces := cs.runningPiece.ListKeyAsIntSlice()
	if pieceStatus == PieceRunning {
		return runningPieces, nil
	}

	// get bitset
	ss, err := pm.superProgress.getAsSuperState(taskID)
	if err != nil {
		return nil, err
	}
	clientBitset := cs.pieceBitSet.Clone()
	cdnBitset := ss.pieceBitSet.Clone()

	// get successful pieces
	if pieceStatus == PieceSuccess {
		return getSuccessfulPieces(clientBitset, cdnBitset)
	}

	// get available pieces
	return getAvailablePieces(clientBitset, cdnBitset, runningPieces)
}

// DeletePieceProgressByCID delete the pieces progress with specified clientID.
func (pm *Manager) DeletePieceProgressByCID(ctx context.Context, taskID, clientID string) (err error) {
	if pm.cfg.IsSuperCID(clientID) {
		return pm.superProgress.remove(taskID)
	}

	return pm.clientProgress.remove(clientID)
}

// GetPeerIDsByPieceNum gets all peerIDs with specified taskID and pieceNum.
// It will return nil when no peers is available.
func (pm *Manager) GetPeerIDsByPieceNum(ctx context.Context, taskID string, pieceNum int) (peerIDs []string, err error) {
	key, err := generatePieceProgressKey(taskID, pieceNum)
	if err != nil {
		return nil, err
	}
	ps, err := pm.pieceProgress.getAsPieceState(key)
	if err != nil {
		return nil, err
	}

	return ps.getAvailablePeers(), nil
}

// DeletePeerIDByPieceNum deletes the peerID which means that
// the peer no longer provides the service for the pieceNum of taskID.
func (pm *Manager) DeletePeerIDByPieceNum(ctx context.Context, taskID string, pieceNum int, peerID string) error {
	key, err := generatePieceProgressKey(taskID, pieceNum)
	if err != nil {
		return err
	}
	ps, err := pm.pieceProgress.getAsPieceState(key)
	if err != nil {
		return err
	}

	return ps.delete(peerID)
}

// GetPeerStateByPeerID gets peer state with specified peerID.
func (pm *Manager) GetPeerStateByPeerID(ctx context.Context, peerID string) (*mgr.PeerState, error) {
	peerState, err := pm.peerProgress.getAsPeerState(peerID)
	if err != nil {
		return nil, err
	}

	return &mgr.PeerState{
		PeerID:            peerID,
		ServiceDownTime:   peerState.serviceDownTime,
		ClientErrorCount:  peerState.clientErrorCount.Get(),
		ServiceErrorCount: peerState.serviceErrorCount.Get(),
		ProducerLoad:      peerState.producerLoad.Get(),
	}, nil
}

// DeletePeerStateByPeerID deletes the peerState by PeerID.
func (pm *Manager) DeletePeerStateByPeerID(ctx context.Context, peerID string) error {
	// delete client blackinfo
	// TODO: delete the blackinfo that refer to peerID
	pm.clientBlackInfo.Delete(peerID)

	// delete peer progress
	return pm.peerProgress.remove(peerID)
}

// GetPeersByTaskID get all peers info with specified taskID.
func (pm *Manager) GetPeersByTaskID(ctx context.Context, taskID string) (peersInfo []*types.PeerInfo, err error) {
	return nil, nil
}

// GetBlackInfoByPeerID get black info with specified peerID.
func (pm *Manager) GetBlackInfoByPeerID(ctx context.Context, peerID string) (dstPIDMap *cutil.SyncMap, err error) {
	return pm.clientBlackInfo.GetAsMap(peerID)
}

// getSuccessfulPieces gets pieces that the piece has been downloaded successful.
func getSuccessfulPieces(clientBitset, cdnBitset *bitset.BitSet) ([]int, error) {
	successPieces := make([]int, 0)
	clientBitset.InPlaceIntersection(cdnBitset)
	for i, e := clientBitset.NextSet(0); e; i, e = clientBitset.NextSet(i + 1) {
		if getPieceStatusByIndex(i) == config.PieceSUCCESS {
			successPieces = append(successPieces, getPieceNumByIndex(i))
		}
	}

	return successPieces, nil
}

// getAvailablePieces gets pieces that has neither been downloaded successfully
// nor being downloaded and supernode has downloaded it successfully.
func getAvailablePieces(clientBitset, cdnBitset *bitset.BitSet, runningPieceNums []int) ([]int, error) {
	cdnBitset.InPlaceDifference(clientBitset)
	availablePieces := make(map[int]bool)
	for i, e := cdnBitset.NextSet(0); e; i, e = cdnBitset.NextSet(i + 1) {
		pieceStatus := getPieceStatusByIndex(i)
		if pieceStatus == config.PieceSUCCESS {
			availablePieces[getPieceNumByIndex(i)] = true
		}

		if pieceStatus == config.PieceFAILED {
			return nil, errors.Wrapf(errorType.ErrCDNFail, "pieceNum: %d", getPieceNumByIndex(i))
		}
	}

	if len(availablePieces) == 0 {
		return nil, nil
	}

	for _, v := range runningPieceNums {
		if availablePieces[v] {
			delete(availablePieces, v)
		}
	}

	return parseMapKeyToIntSlice(availablePieces), nil
}

func parseMapKeyToIntSlice(mmap map[int]bool) (result []int) {
	for k, v := range mmap {
		if v {
			result = append(result, k)
		}
	}

	return
}
