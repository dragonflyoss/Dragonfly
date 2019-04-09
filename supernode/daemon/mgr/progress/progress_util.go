package progress

import (
	"fmt"
	"strconv"
	"strings"

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
func (pm *Manager) updateClientProgress(taskID, srcCID, dstCID string, pieceNum, pieceStatus int) (bool, error) {
	if isSuperCID(srcCID) {
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

	err = updateRunningPiece(cs.runningPiece, srcCID, dstCID, pieceNum, pieceStatus)
	if err != nil {
		return false, err
	}

	return updatePieceBitSet(cs.pieceBitSet, pieceNum, pieceStatus), nil
}

// updateRunningPiece update the relationship between the running piece and srcCID and dstCID,
// which means the info that records the pieces being downloaded from dstCID to srcCID.
func updateRunningPiece(dstCIDMap *cutil.SyncMap, srcCID, dstCID string, pieceNum, pieceStatus int) error {
	pieceNumString := strconv.Itoa(pieceNum)
	if pieceStatus == config.PieceRUNNING && !cutil.IsEmptyStr(dstCID) {
		return dstCIDMap.Add(pieceNumString, dstCID)
	}

	if _, err := dstCIDMap.Get(pieceNumString); err != nil {
		return err
	}

	return dstCIDMap.Remove(pieceNumString)
}

// updatePieceBitSet adds a new piece for srcCID when it successfully downloads the piece.
func updatePieceBitSet(pieceBitSet *bitset.BitSet, pieceNum, pieceStatus int) bool {
	if pieceBitSet.Test(uint(pieceNum*8 + config.PieceSUCCESS)) {
		return false
	}

	// clear the bits from pieceNum * 8 to (pieceNum+1)*8 at first.
	for i := pieceNum * 8; i < (pieceNum+1)*8; i++ {
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
func (pm *Manager) updatePeerProgress(taskID, srcPID, dstPID, pieceRange string, pieceStatus int) error {
	// update producerLoad of dstPID
	if cutil.IsEmptyStr(dstPID) {
		ps, err := pm.peerProgress.getAsPeerState(srcPID)
		if err != nil {
			return err
		}
		updateProducerLoad(ps.producerLoad, taskID, dstPID, pieceRange)
	}

	ps, err := pm.peerProgress.getAsPeerState(srcPID)
	if err != nil {
		return err
	}

	if !needDoPeerInfo(srcPID, dstPID) {
		return nil
	}

	// update ClientErrorInfo/serviceErrorInfo and ClientBlackInfo
	if pieceStatus == config.PieceSUCCESS || pieceStatus == config.PieceSEMISUC {
		processPeerSucInfo(ps.clientErrorCount, ps.serviceErrorCount, srcPID, dstPID)
	}
	if pieceStatus == config.PieceFAILED {
		if err := pm.updateBlackInfo(srcPID, dstPID); err != nil {
			return err
		}
		processPeerFailInfo(ps, srcPID, dstPID)
	}
	return nil
}

func (pm *Manager) updateBlackInfo(srcPID, dstPID string) error {
	// update black List
	blackList, err := pm.clientBlackInfo.GetAsMap(srcPID)
	if err != nil && errorType.IsDataNotFound(err) {
		blackList = cutil.NewSyncMap()
		if err := pm.clientBlackInfo.Add(srcPID, blackList); err != nil {
			return err
		}
	}
	if _, err := blackList.Get(dstPID); err != nil &&
		errorType.IsDataNotFound(err) {
		if err := blackList.Add(dstPID, true); err != nil {
			return err
		}
	}
	return nil
}

// processPeerSucInfo sets the count of errors to 0
// when srcCID successfully downloads a piece from dstCID.
func processPeerSucInfo(clientErrorCount, serviceErrorCount *cutil.AtomicInt, srcPID, dstPID string) error {
	// update ClientErrorInfo
	if clientErrorCount != nil {
		clientErrorCount.Set(0)
	}

	// update ServiceErrorInfo
	if serviceErrorCount != nil {
		serviceErrorCount.Set(0)
	}

	return nil
}

// processPeerFailInfo adds one to the count of errors
// when srcCID failed to download a piece from dstCID.
// And add the dstCID to the blacklist of the srcCID.
func processPeerFailInfo(ps *peerState, srcCID, dstCID string) (err error) {

	// update clientErrorInfo
	if ps.clientErrorCount != nil {
		ps.clientErrorCount.Add(1)
	} else {
		ps.clientErrorCount = cutil.NewAtomicInt(1)
	}

	// update serviceErrorInfo
	if ps.serviceErrorCount != nil {
		ps.serviceErrorCount.Add(1)
	} else {
		ps.serviceErrorCount = cutil.NewAtomicInt(1)
	}

	return nil
}

// updateProducerLoad update the load of the clientID.
func updateProducerLoad(load *cutil.AtomicInt, taskID, peerID, pieceRange string) error {
	if load == nil {
		return nil
	}
	loadNew := load.Add(-1)
	if loadNew < 0 {
		logrus.Warnf("client load maybe illegal,taskID: %s,peerID: %s,pieceRange: %s,load: %d",
			taskID, peerID, pieceRange, loadNew)
		load.Add(1)
	}

	return nil
}

// needDoPeerInfo returns whether we should update the peer related info.
// It returns false when the PeerID is empty or represents a supernode.
func needDoPeerInfo(srcPID, dstPID string) bool {
	if cutil.IsEmptyStr(srcPID) || cutil.IsEmptyStr(dstPID) ||
		isSuperPID(srcPID) || isSuperPID(dstPID) {
		return false
	}
	return true
}

// TODO: implement it.
var isSuperPID = func(PeerID string) bool {
	return true
}

// isSuperCID returns whether the clientID represents supernode.
// TODO: implement it.
var isSuperCID = func(clientID string) bool {
	return strings.HasPrefix(clientID, config.SuperNodeCIdPrefix)
}

// generatePieceProgressKey returns a string as the key of PieceProgress.
func generatePieceProgressKey(taskID string, pieceNum int) (string, error) {
	if cutil.IsEmptyStr(taskID) || pieceNum < 0 {
		return "", errors.Wrapf(errorType.ErrInvalidValue,
			"failed to make piece progress key with taskID: %s and pieceNum: %d", taskID, pieceNum)
	}
	return fmt.Sprintf("%d@%s", pieceNum, taskID), nil
}
