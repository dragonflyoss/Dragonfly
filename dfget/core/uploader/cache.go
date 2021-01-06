/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Module cache implements the cache for stream mode.
// dfget will initialize the cache window during register phase;
// client stream writer in downloader will pass the successfully downloaded pieces to cache;
// uploader will visit the cache to fetch the pieces to share.
package uploader

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"

	"github.com/dragonflyoss/Dragonfly/dfget/types"

	"github.com/dragonflyoss/Dragonfly/dfget/core/api"

	"github.com/pkg/errors"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/sirupsen/logrus"
)

type registerStreamTaskRequest struct {
	taskID     string
	windowSize int
	pieceSize  int
	node       string
	cid        string
}

type cacheManager interface {
	// Register registers the cache for the task. The error would be returned if the cache has already existed.
	register(req *registerStreamTaskRequest) error

	// Store stores the piece to the cache of the given taskID. The cache will be updated.
	store(taskID string, content []byte) error

	// Load loads the piece content from the cache to uploader to share.
	load(taskID string, up *uploadParam) ([]byte, int64, error)
}

type fifoCacheManager struct {
	// pieceContent stores the content of the piece without the pad, in the unit of taskID.
	// key:taskID, value:map[pieceNum] -> []byte
	pieceContent *syncmap.SyncMap

	// window stores the current state of the sliding window for the task.
	// key:taskID, value:window
	windowSet *syncmap.SyncMap

	// ------------------- for update piece status ------------------
	// fileLength stores the length of the successfully downloaded content,
	// which is then used to calculate the piece range.
	// key:taskID, value:int64
	fileLength *syncmap.SyncMap

	// pieceRange stores the range of the piece content.
	// key:taskID, value:string
	pieceRange *syncmap.SyncMap

	// taskID2CID stores the mapping from taskID to CID.
	// key:taskID, value:string
	taskID2CID *syncmap.SyncMap

	// node records the identification of supernode
	node string

	// supernode API
	supernodeAPI api.SupernodeAPI
}

type window struct {
	// una means the oldest unacknowledged number of piece
	una int
	// wnd means the size of the window in piece number
	wnd int
	// start means the start piece num of the sliding window
	start int

	// mutex lock
	mu sync.Mutex
}

var _ cacheManager = &fifoCacheManager{}

func newFIFOCacheManager() *fifoCacheManager {
	return &fifoCacheManager{
		pieceContent: syncmap.NewSyncMap(),
		windowSet:    syncmap.NewSyncMap(),
		fileLength:   syncmap.NewSyncMap(),
		pieceRange:   syncmap.NewSyncMap(),
		taskID2CID:   syncmap.NewSyncMap(),
		supernodeAPI: api.NewSupernodeAPI(),
	}
}

// the parameters have been checked in caller, so we don't check them again
func (cm *fifoCacheManager) register(req *registerStreamTaskRequest) error {
	// if the task has already existed
	if _, ok := cm.windowSet.Load(req.taskID); ok {
		logrus.Warnf("the task with taskID (%s) has already been running", req.taskID)
		return errortypes.ErrInvalidValue
	}

	// otherwise, try to insert the new window
	cm.windowSet.Store(req.taskID, &window{
		wnd: req.windowSize,
		mu:  sync.Mutex{},
	})

	// update the supernode related fields
	cm.taskID2CID.Store(req.taskID, req.cid)
	cm.fileLength.Store(req.taskID, int64(0))
	cm.node = req.node

	return nil
}

func (cm *fifoCacheManager) store(taskID string, content []byte) error {
	w, err := getAsWindowState(cm.windowSet, taskID)
	if err != nil {
		return err
	}

	// add the lock
	w.mu.Lock()

	// if the window is full, pop the first piece out
	if w.una-w.start == w.wnd {
		key := generatePieceKey(taskID, w.start)

		// send the request to supernode to notify the deletion of cache piece
		err = cm.sendDeleteCacheRequest(taskID, key)
		if err != nil {
			logrus.Debugf("cache pieces sync with supernode failed, taskID: %s, pieceNum: %d", taskID, w.start)
			return err
		}

		// delete the piece locally
		cm.pieceContent.Delete(key)
		w.start++
	}

	key := generatePieceKey(taskID, w.una)
	cm.pieceContent.Store(key, content)

	// update piece range and file length
	originFileLength, err := cm.fileLength.GetAsInt64(taskID)
	if err != nil {
		return err
	}
	currentFileLength := originFileLength + int64(len(content))
	cm.pieceRange.Store(key, generatePieceRange(originFileLength, currentFileLength))
	cm.fileLength.Store(taskID, currentFileLength)

	w.una++

	// release the lock
	w.mu.Unlock()
	return nil
}

func (cm *fifoCacheManager) load(taskID string, up *uploadParam) (content []byte, size int64, err error) {
	// fetch the content from cache
	key := generatePieceKey(taskID, int(up.pieceNum))
	content, err = getAsPieceContent(cm.pieceContent, key)
	if err != nil {
		return
	}

	return content, int64(cap(content)), err
}

func (cm *fifoCacheManager) sendDeleteCacheRequest(taskID, key string) (err error) {
	pieceRange, err := cm.pieceRange.GetAsString(key)
	if err != nil {
		return err
	}

	cid, err := cm.taskID2CID.GetAsString(taskID)
	if err != nil {
		return err
	}

	req := &types.ReportPieceRequest{
		TaskID:     taskID,
		Cid:        cid,
		PieceRange: pieceRange,
	}

	_, err = cm.supernodeAPI.DeleteStreamCache(cm.node, req)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------
// helper functions

func getAsWindowState(m *syncmap.SyncMap, key string) (*window, error) {
	v, ok := m.Load(key)
	if !ok {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "key: %s not found", key)
	}

	if value, ok := v.(*window); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "key %s: %v", key, v)
}

func getAsPieceContent(m *syncmap.SyncMap, key string) ([]byte, error) {
	v, ok := m.Load(key)
	if !ok {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "key: %s not found", key)
	}

	if value, ok := v.([]byte); ok {
		return value, nil
	}
	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "key %s: %v", key, v)
}

func generatePieceKey(taskID string, pieceNum int) string {
	return taskID + ":" + strconv.Itoa(pieceNum)
}

func generatePieceRange(start, end int64) string {
	return fmt.Sprintf("%d-%d", start, end)
}
