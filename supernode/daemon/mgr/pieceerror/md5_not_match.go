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

	"github.com/sirupsen/logrus"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/rangeutils"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

var _ Handler = &FileMd5NotMatchHandler{}

type FileMd5NotMatchHandler struct {
	gcManager  mgr.GCMgr
	cdnManager mgr.CDNMgr
}

func init() {
	Register(types.PieceErrorRequestErrorTypeFILEMD5NOTMATCH, NewFileMd5NotMatchHandler)
}

func NewFileMd5NotMatchHandler(gcManager mgr.GCMgr, cdnManager mgr.CDNMgr) (Handler, error) {
	return &FileMd5NotMatchHandler{
		gcManager:  gcManager,
		cdnManager: cdnManager,
	}, nil
}

func (fnmh *FileMd5NotMatchHandler) Handle(ctx context.Context, pieceErrorRequest *types.PieceErrorRequest) error {
	pieceNum := rangeutils.CalculatePieceNum(pieceErrorRequest.Range)

	// get piece MD5 by reading the meta file
	metaPieceMD5, err := fnmh.cdnManager.GetPieceMD5(ctx, pieceErrorRequest.TaskID, pieceNum, pieceErrorRequest.Range, "meta")
	if err != nil {
		logrus.Errorf("failed to get piece MD5 by read meta data taskID(%s) pieceRange(%s): %v",
			pieceErrorRequest.TaskID, pieceErrorRequest.Range, err)
	}

	// get piece Md5 by reading the source file on the local disk
	filePieceMD5, err := fnmh.cdnManager.GetPieceMD5(ctx, pieceErrorRequest.TaskID, pieceNum, pieceErrorRequest.Range, "file")
	if err != nil {
		logrus.Errorf("failed to get piece MD5 by read source file directly taskID(%s) pieceRange(%s): %v",
			pieceErrorRequest.TaskID, pieceErrorRequest.Range, err)
	}

	logrus.Debugf("success to get taskID(%s) pieceRange(%s) metaPieceMD5(%s) filePieceMD5(%s) expectedMD5(%s)",
		pieceErrorRequest.TaskID, pieceErrorRequest.Range, metaPieceMD5, filePieceMD5, pieceErrorRequest.ExpectedMd5)

	if filePieceMD5 != metaPieceMD5 &&
		filePieceMD5 != pieceErrorRequest.ExpectedMd5 {
		fnmh.gcManager.GCTask(ctx, pieceErrorRequest.TaskID, true)
	}

	return nil
}
