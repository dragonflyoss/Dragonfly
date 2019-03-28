package progress

import (
	errorType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// pieceState maintains the information about
// which peers the piece currently exists on.
type pieceState struct {
	pieceContainer *cutil.SyncMap
}

// newPieceState returns a new pieceState.
func newPieceState() *pieceState {
	return &pieceState{
		pieceContainer: cutil.NewSyncMap(),
	}
}

// add a peerID for the corresponding piece which means that
// there is a new peer node that owns this piece.
func (ps *pieceState) add(peerID string) error {
	if cutil.IsEmptyStr(peerID) {
		return errors.Wrap(errorType.ErrEmptyValue, "peerID")
	}

	ok, err := ps.pieceContainer.GetAsBool(peerID)
	if err == nil && ok {
		logrus.Warnf("peerID: %s is exist", peerID)
		return nil
	}

	if err != nil && !errorType.IsDataNotFound(err) {
		return err
	}

	return ps.pieceContainer.Add(peerID, true)
}

func (ps *pieceState) getAvailablePeers() []string {
	return ps.pieceContainer.ListKeyAsStringSlice()
}

func (ps *pieceState) delete(peerID string) error {
	return ps.pieceContainer.Remove(peerID)
}
