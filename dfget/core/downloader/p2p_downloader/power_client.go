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

package downloader

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
)

// PowerClient downloads file from dragonfly.
type PowerClient struct {
	taskID      string
	node        string
	pieceTask   *types.PullPieceTaskResponseContinueData
	cfg         *config.Config
	queue       util.Queue
	clientQueue util.Queue

	rateLimiter *util.RateLimiter
}

// Run starts run the task.
func (pc *PowerClient) Run() (err error) {
	pieceMetaArr := strings.Split(pc.pieceTask.PieceMd5, ":")
	pieceMD5 := pieceMetaArr[0]
	dstIP := pc.pieceTask.PeerIP
	peerPort := pc.pieceTask.PeerPort

	defer func() {
		if err != nil {
			pc.cfg.ClientLogger.Errorf("failed to read piece cont from dst:%s:%d, error:%s",
				dstIP, peerPort, err)
		}
	}()

	// check that the target download peer is available
	if dstIP != pc.node {
		if _, err = util.CheckConnect(dstIP, peerPort, -1); err != nil {
			piece := NewPiece(pc.taskID, pc.node, pc.pieceTask.Cid, pc.pieceTask.Range, config.ResultFail, config.TaskStatusRunning)
			pc.queue.Put(piece)
			return nil
		}
	}

	// send download request
	req := &api.DownloadRequest{
		Path:       pc.pieceTask.Path,
		PieceRange: pc.pieceTask.Range,
		PieceNum:   pc.pieceTask.PieceNum,
		PieceSize:  pc.pieceTask.PieceSize,
	}

	startTime := time.Now()
	resp, err := downloadAPI.Download(dstIP, peerPort, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// start to read data from resp
	// use limitReader to limit the download speed
	limitReader := util.NewLimitReaderWithLimiter(pc.rateLimiter, resp.Body, pieceMD5 != "")

	buf := make([]byte, 0, 256*1024)
	pieceCont := bytes.NewBuffer(buf)

	total, err := pieceCont.ReadFrom(limitReader)
	if err != nil {
		return err
	}

	readFinish := time.Now()

	// Verify md5 code
	realMd5 := limitReader.Md5()
	if realMd5 != pieceMD5 {
		return fmt.Errorf("piece range:%s md5 not match, expected:%s real:%s", pc.pieceTask.Range, pieceMD5, realMd5)
	}

	piece := NewPieceContent(pc.taskID, pc.node, pc.pieceTask.Cid, pc.pieceTask.Range, config.ResultSemiSuc, config.TaskStatusRunning, pieceCont)
	piece.PieceSize = pc.pieceTask.PieceSize
	piece.PieceNum = pc.pieceTask.PieceNum
	pc.clientQueue.Put(piece)
	pc.queue.Put(piece)

	timeDuring := time.Since(startTime).Seconds()
	if timeDuring > 2.0 {
		pc.cfg.ClientLogger.Warnf("client range:%s cost:%.3f from peer:%s,its readCost:%.3f,cont length:%d", pc.pieceTask.Range, timeDuring, dstIP, readFinish.Sub(startTime).Seconds(), total)
	}
	return nil
}
