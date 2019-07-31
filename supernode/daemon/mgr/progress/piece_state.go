package progress

import (
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// pieceState maintains the information about
// which peers the piece currently exists on.
type pieceState struct {
	pieceContainer *syncmap.SyncMap
}

// newPieceState returns a new pieceState.
func newPieceState() *pieceState {
	return &pieceState{
		pieceContainer: syncmap.NewSyncMap(),
	}
}

// add a peerID for the corresponding piece which means that
// there is a new peer node that owns this piece.
func (ps *pieceState) add(peerID string) error {
	if stringutils.IsEmptyStr(peerID) {
		return errors.Wrap(errortypes.ErrEmptyValue, "peerID")
	}

	ok, err := ps.pieceContainer.GetAsBool(peerID)
	if err == nil && ok {
		logrus.Warnf("peerID: %s is exist", peerID)
		return nil
	}

	if err != nil && !errortypes.IsDataNotFound(err) {
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
