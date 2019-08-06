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

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"

	"github.com/sirupsen/logrus"
)

var _ Handler = &FileNotExistHandler{}

type FileNotExistHandler struct {
	gcManager  mgr.GCMgr
	cdnManager mgr.CDNMgr
}

func init() {
	Register(types.PieceErrorRequestErrorTypeFILENOTEXIST, NewFileNotExistHandler)
}

func NewFileNotExistHandler(gcManager mgr.GCMgr, cdnManager mgr.CDNMgr) (Handler, error) {
	return &FileNotExistHandler{
		gcManager:  gcManager,
		cdnManager: cdnManager,
	}, nil
}

func (feh *FileNotExistHandler) Handle(ctx context.Context, pieceErrorRequest *types.PieceErrorRequest) error {
	if feh.cdnManager.CheckFile(ctx, pieceErrorRequest.TaskID) {
		return nil
	}

	logrus.Warnf("taskID(%s) file data doesn't exist, start to gc this task", pieceErrorRequest.TaskID)

	feh.gcManager.GCTask(ctx, pieceErrorRequest.TaskID, true)
	return nil
}
