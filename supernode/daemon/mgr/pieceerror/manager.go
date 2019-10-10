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

package pieceerror

import (
	"context"
	"fmt"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ mgr.PieceErrorMgr = &Manager{}

const (
	ErrHandlerChanSize = 100
	HandleErrorPool    = 4
	GCHandlingInterval = 1 * time.Second
	GCHandlingDelay    = 3 * time.Second
)

// handlerStore stores all registered handler.
var handlerStore = syncmap.NewSyncMap()

type handlerInitFunc func(gcManager mgr.GCMgr, cdnManager mgr.CDNMgr) (handler Handler, err error)

func Register(errType string, initer handlerInitFunc) {
	handlerStore.Add(errType, initer)
}

// TODO: make it pluggable
type Handler interface {
	Handle(ctx context.Context, pieceErrorRequest *types.PieceErrorRequest) error
}

type Manager struct {
	cfg      *config.Config
	handlers map[string]Handler

	gcManager  mgr.GCMgr
	cdnManager mgr.CDNMgr

	// error handler
	pieceErrChan       chan *types.PieceErrorRequest
	errorHandlingStore *syncmap.SyncMap
	handledStore       *syncmap.SyncMap
}

func NewManager(cfg *config.Config, gcManager mgr.GCMgr, cdnManager mgr.CDNMgr) (*Manager, error) {
	return &Manager{
		cfg:                cfg,
		handlers:           make(map[string]Handler),
		gcManager:          gcManager,
		cdnManager:         cdnManager,
		pieceErrChan:       make(chan *types.PieceErrorRequest, ErrHandlerChanSize),
		errorHandlingStore: syncmap.NewSyncMap(),
		handledStore:       syncmap.NewSyncMap(),
	}, nil
}

// HandlePieceError the peer should report the error with related info when
// it failed to download a piece from supernode.
// And the supernode should handle the piece Error and do some repair operations.
func (em *Manager) HandlePieceError(ctx context.Context, pieceErrorRequest *types.PieceErrorRequest) error {
	// ignore the error that isn't caused by downloading from supernode
	if !em.cfg.IsSuperPID(pieceErrorRequest.DstPid) {
		return nil
	}

	// if the error is handling, we should ignore it.
	_, err := em.errorHandlingStore.Get(pieceErrorRequest.TaskID)
	if err == nil {
		return nil
	}
	if !errortypes.IsDataNotFound(err) {
		logrus.Errorf("failed to get taskID(%s) from errorHandlingStore: %v", pieceErrorRequest.TaskID, err)
		return err
	}

	select {
	case em.pieceErrChan <- pieceErrorRequest:
		em.errorHandlingStore.Add(pieceErrorRequest.TaskID, true)
		return nil
	default:
		logrus.Warnf("drop piece error request: %+v", pieceErrorRequest)
		return fmt.Errorf("%d piece errors are being processed already", ErrHandlerChanSize)
	}
}

// StartHandleError starts a goroutine to handle the piece error.
func (em *Manager) StartHandleError(ctx context.Context) {
	em.initHandlers()

	go func() {
		em.startHandleErrorPool(ctx)
	}()

	go func() {
		ticker := time.NewTicker(GCHandlingInterval)
		for range ticker.C {
			em.deleteHandling(ctx)
		}
	}()
}

func (em *Manager) initHandlers() {
	rangeFunc := func(key, value interface{}) bool {
		initFunc, ok := value.(handlerInitFunc)
		if !ok {
			return true
		}

		errType, ok := key.(string)
		if !ok {
			return true
		}

		handler, err := initFunc(em.gcManager, em.cdnManager)
		if err != nil {
			logrus.Errorf("failed to init handler type %s: %v", errType, err)
			return true
		}

		em.handlers[errType] = handler
		return true
	}

	handlerStore.Range(rangeFunc)
}

func (em *Manager) deleteHandling(ctx context.Context) {
	rangeFunc := func(key, value interface{}) bool {
		handledTime, ok := value.(time.Time)
		if !ok {
			return true
		}
		if time.Since(handledTime) < GCHandlingDelay {
			return true
		}

		if taskID, ok := key.(string); ok {
			em.errorHandlingStore.Delete(taskID)
			em.handledStore.Delete(taskID)
		}
		return true
	}

	em.handledStore.Range(rangeFunc)
}
func (em *Manager) startHandleErrorPool(ctx context.Context) {
	for i := 0; i < HandleErrorPool; i++ {
		go func() {
			for per := range em.pieceErrChan {
				if err := em.handleError(ctx, per); err != nil {
					logrus.Errorf("failed to handle error %+v:%v", per, err)
				}
			}
		}()
	}
}

func (em *Manager) handleError(ctx context.Context, pieceError *types.PieceErrorRequest) error {
	// add taskID to handledStore regardless of the result of handler
	defer func() {
		em.handledStore.Add(pieceError.TaskID, time.Now())
	}()

	handler, err := em.getHandler(ctx, pieceError.ErrorType)
	if err != nil {
		return errors.Wrapf(err, "failed to get handler")
	}

	return handler.Handle(ctx, pieceError)
}

func (em *Manager) getHandler(ctx context.Context, errType string) (Handler, error) {
	if v, ok := em.handlers[errType]; ok {
		return v, nil
	}

	return nil, fmt.Errorf("unregistered error handler")
}
