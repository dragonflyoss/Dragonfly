package progress

import (
	"fmt"
	"strconv"

	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/willf/bitset"
)

// updatePieceProgress added a new peer for the pieceNum when the srcPID successfully downloads the piece.
func (pm *Manager) updatePieceProgress(taskID, srcPID string, pieceNum int) error {
	key, err := generatePieceProgressKey(taskID, pieceNum)
	if err != nil {
		return err
	}

	pstate, err := pm.pieceProgress.getAsPieceState(key)
	if err != nil {
		if !errorType.IsDataNotFound(err) {
			return err
		}

		// initialize a PieceState if not found.
		if err := pm.pieceProgress.add(key, newPieceState()); err != nil {
			return err
		}

		// reacquisition after initialization.
		if pstate, err = pm.pieceProgress.getAsPieceState(key); err != nil {
			return err
		}
	}

	return pstate.add(srcPID)
}

// updateClientProgress update the client progress when clientID is not a supernode,
// otherwise update the super progress.
func (pm *Manager) updateClientProgress(taskID, srcCID, dstPID string, pieceNum, pieceStatus int) (bool, error) {
	// update piece bitSet
	if pm.cfg.IsSuperCID(srcCID) {
		ss, err := pm.superProgress.getAsSuperState(taskID)
		if err != nil {
			return false, err
		}
		return updatePieceBitSet(ss.pieceBitSet, pieceNum, pieceStatus), nil
	}

	cs, err := pm.clientProgress.getAsClientState(srcCID)
	if err != nil {
		return false, err
	}

	// update running piece
	err = updateRunningPiece(cs.runningPiece, srcCID, dstPID, pieceNum, pieceStatus)
	if err != nil {
		return false, err
	}

	return updatePieceBitSet(cs.pieceBitSet, pieceNum, pieceStatus), nil
}

// updateRunningPiece update the relationship between the running piece and srcCID and dstPID,
// which means the info that records the pieces being downloaded from dstPID to srcCID.
func updateRunningPiece(dstPIDMap *cutil.SyncMap, srcCID, dstPID string, pieceNum, pieceStatus int) error {
	pieceNumString := strconv.Itoa(pieceNum)
	if pieceStatus == config.PieceRUNNING && !cutil.IsEmptyStr(dstPID) {
		return dstPIDMap.Add(pieceNumString, dstPID)
	}

	if _, err := dstPIDMap.Get(pieceNumString); err != nil {
		return err
	}

	return dstPIDMap.Remove(pieceNumString)
}

// updatePieceBitSet adds a new piece for srcCID when it successfully downloads the piece.
func updatePieceBitSet(pieceBitSet *bitset.BitSet, pieceNum, pieceStatus int) bool {
	if pieceBitSet.Test(uint(getStartIndexByPieceNum(pieceNum) + config.PieceSUCCESS)) {
		return false
	}

	// clear the bits from pieceNum * 8 to (pieceNum+1)*8 at first.
	for i := getStartIndexByPieceNum(pieceNum); i < getStartIndexByPieceNum(pieceNum+1); i++ {
		pieceBitSet.Clear(uint(i))
	}
	// if the pieceStatus equals waiting,
	// keep bits from pieceNum * 8 to (pieceNum+1)*8 equals to 0.
	if pieceStatus == config.PieceWAITING {
		return true
	}
	if pieceStatus == config.PieceSEMISUC {
		pieceStatus = config.PieceSUCCESS
	}
	pieceBitSet.Set(uint(pieceNum*8 + pieceStatus))
	return true
}

// updatePeerProgress update the peer progress.
func (pm *Manager) updatePeerProgress(taskID, srcPID, dstPID string, pieceNum, pieceStatus int) error {
	var dstPeerState *peerState

	// update producerLoad of dstPID
	if !cutil.IsEmptyStr(dstPID) {
		dstPeerState, err := pm.peerProgress.getAsPeerState(dstPID)
		if err != nil {
			return err
		}
		updateProducerLoad(dstPeerState.producerLoad, taskID, dstPID, pieceNum, pieceStatus)
	}

	if !pm.needUpdatePeerInfo(srcPID, dstPID) {
		return nil
	}

	srcPeerState, err := pm.peerProgress.getAsPeerState(srcPID)
	if err != nil {
		return err
	}

	// update ClientErrorInfo/serviceErrorInfo
	if pieceStatus == config.PieceSUCCESS || pieceStatus == config.PieceSEMISUC {
		processPeerSucInfo(srcPeerState, dstPeerState)
	}
	if pieceStatus == config.PieceFAILED {
		if err := pm.updateBlackInfo(srcPID, dstPID); err != nil {
			return err
		}
		processPeerFailInfo(srcPeerState, dstPeerState)
	}
	return nil
}

func (pm *Manager) updateBlackInfo(srcPID, dstPID string) error {
	// update black List
	blackList, err := pm.clientBlackInfo.GetAsMap(srcPID)
	if err != nil {
		if !errorType.IsDataNotFound(err) {
			return err
		}

		blackList = cutil.NewSyncMap()
		if err := pm.clientBlackInfo.Add(srcPID, blackList); err != nil {
			return err
		}
	}

	v, err := blackList.GetAsAtomicInt(dstPID)
	if err == nil {
		v.Add(1)
		return nil
	}
	if errorType.IsDataNotFound(err) {
		return blackList.Add(dstPID, cutil.NewAtomicInt(1))
	}

	return err
}

// processPeerSucInfo sets the count of errors to 0
// when srcCID successfully downloads a piece from dstPID.
func processPeerSucInfo(srcPeerState, dstPeerState *peerState) {
	// update ClientErrorInfo
	if srcPeerState != nil && srcPeerState.clientErrorCount != nil {
		srcPeerState.clientErrorCount.Set(0)
	}

	// update ServiceErrorInfo
	if dstPeerState != nil && dstPeerState.serviceErrorCount != nil {
		dstPeerState.serviceErrorCount.Set(0)
	}
}

// processPeerFailInfo adds one to the count of errors
// when srcCID failed to download a piece from dstPID.
func processPeerFailInfo(srcPeerState, dstPeerState *peerState) {
	// update clientErrorInfo
	if srcPeerState != nil {
		if srcPeerState.clientErrorCount != nil {
			srcPeerState.clientErrorCount.Add(1)
		} else {
			srcPeerState.clientErrorCount = cutil.NewAtomicInt(1)
		}
	}

	// update serviceErrorInfo
	if dstPeerState != nil {
		if dstPeerState.serviceErrorCount != nil {
			dstPeerState.serviceErrorCount.Add(1)
		} else {
			dstPeerState.serviceErrorCount = cutil.NewAtomicInt(1)
		}
	}
}

// updateProducerLoad update the load of the clientID.
// TODO: avoid multiple calls
func updateProducerLoad(load *cutil.AtomicInt, taskID, peerID string, pieceNum, pieceStatus int) {
	if load == nil {
		logrus.Warnf("client load maybe not initialized,taskID: %s,peerID: %s,pieceNum: %d",
			taskID, peerID, pieceNum)
		load = cutil.NewAtomicInt(0)
	}

	// increase the load of peerID when pieceStatus equals PieceRUNNING
	if pieceStatus == config.PieceRUNNING {
		load.Add(1)
		return
	}

	// decrease the load of peerID when pieceStatus not equals PieceRUNNING
	loadNew := load.Add(-1)
	if loadNew < 0 {
		logrus.Warnf("client load maybe illegal,taskID: %s,peerID: %s,pieceNum: %d,load: %d",
			taskID, peerID, pieceNum, loadNew)
		load.Add(1)
	}
}

// needUpdatePeerInfo returns whether we should update the peer related info.
// It returns false when the PeerID is empty or represents a supernode.
func (pm *Manager) needUpdatePeerInfo(srcPID, dstPID string) bool {
	if cutil.IsEmptyStr(srcPID) || cutil.IsEmptyStr(dstPID) ||
		pm.cfg.IsSuperPID(srcPID) || pm.cfg.IsSuperPID(dstPID) {
		return false
	}
	return true
}

// generatePieceProgressKey returns a string as the key of PieceProgress.
func generatePieceProgressKey(taskID string, pieceNum int) (string, error) {
	if cutil.IsEmptyStr(taskID) || pieceNum < 0 {
		return "", errors.Wrapf(errorType.ErrInvalidValue,
			"failed to make piece progress key with taskID: %s and pieceNum: %d", taskID, pieceNum)
	}
	return fmt.Sprintf("%d@%s", pieceNum, taskID), nil
}

func getStartIndexByPieceNum(pieceNum int) int {
	return pieceNum * 8
}

func getPieceNumByIndex(index uint) int {
	return int(index / 8)
}

func getPieceStatusByIndex(index uint) int {
	return int(index % 8)
}
