package gc

import (
	"context"

	"github.com/pkg/errors"
)

func (gcm *Manager) gcDfgetTasksWithTaskID(ctx context.Context, taskID string, cids []string) []error {
	var errSlice []error
	for _, cid := range cids {
		if err := gcm.progressMgr.DeleteCID(ctx, cid); err != nil {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to delete dfgetTask(%s) progress info", cid))
		}
		if err := gcm.dfgetTaskMgr.Delete(ctx, cid, taskID); err != nil {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to delete dfgetTask(%s) info", cid))
		}
	}

	if len(errSlice) != 0 {
		return nil
	}

	return errSlice
}

func (gcm *Manager) gcDfgetTasks(ctx context.Context, keys map[string]string, cids []string) []error {
	var errSlice []error
	for _, cid := range cids {
		if err := gcm.progressMgr.DeleteCID(ctx, cid); err != nil {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to delete dfgetTask(%s) progress info", cid))
		}
		if err := gcm.dfgetTaskMgr.Delete(ctx, cid, keys[cid]); err != nil {
			errSlice = append(errSlice, errors.Wrapf(err, "failed to delete dfgetTask(%s) info", cid))
		}
	}

	if len(errSlice) != 0 {
		return nil
	}

	return errSlice
}
