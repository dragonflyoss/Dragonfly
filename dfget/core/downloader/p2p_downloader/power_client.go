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
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/sirupsen/logrus"
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

	total    int64
	readCost time.Duration
}

// Run starts run the task.
func (pc *PowerClient) Run() error {
	startTime := time.Now()

	content, err := pc.downloadPiece()

	timeDuring := time.Since(startTime).Seconds()
	pc.log().Debugf("client range:%s cost:%.3f from peer:%s:%d, readCost:%.3f, length:%d",
		pc.pieceTask.Range, timeDuring, pc.pieceTask.PeerIP, pc.pieceTask.PeerPort,
		pc.readCost.Seconds(), pc.total)

	if err != nil {
		pc.log().Errorf("read piece cont error:%v from dst:%s:%d",
			err, pc.pieceTask.PeerIP, pc.pieceTask.PeerPort)
		pc.queue.Put(pc.failPiece())
		return err
	}

	piece := pc.successPiece(content)
	pc.clientQueue.Put(piece)
	pc.queue.Put(piece)
	return nil
}

func (pc *PowerClient) downloadPiece() (content *bytes.Buffer, e error) {
	pieceMetaArr := strings.Split(pc.pieceTask.PieceMd5, ":")
	pieceMD5 := pieceMetaArr[0]
	dstIP := pc.pieceTask.PeerIP
	peerPort := pc.pieceTask.PeerPort

	// check that the target download peer is available
	if dstIP != pc.node {
		if _, e = util.CheckConnect(dstIP, peerPort, -1); e != nil {
			return nil, e
		}
	}

	// send download request
	startTime := time.Now()
	resp, err := downloadAPI.Download(dstIP, peerPort, pc.createDownloadRequest())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return nil, errors.ErrRangeNotSatisfiable
	}
	if !pc.is2xxStatus(resp.StatusCode) {
		return nil, errors.New(resp.StatusCode, pc.readBody(resp.Body))
	}

	// start to read data from resp
	// use limitReader to limit the download speed
	limitReader := util.NewLimitReaderWithLimiter(pc.rateLimiter, resp.Body, pieceMD5 != "")
	content = &bytes.Buffer{}
	if pc.total, e = content.ReadFrom(limitReader); err != nil {
		return nil, err
	}
	pc.readCost = time.Now().Sub(startTime)

	// Verify md5 code
	if realMd5 := limitReader.Md5(); realMd5 != pieceMD5 {
		return nil, fmt.Errorf("piece range:%s md5 not match, expected:%s real:%s",
			pc.pieceTask.Range, pieceMD5, realMd5)
	}

	if timeDuring := time.Since(startTime).Seconds(); timeDuring > 2.0 {
		pc.log().Warnf("client range:%s cost:%.3f from peer:%s, readCost:%.3f, length:%d",
			pc.pieceTask.Range, timeDuring, dstIP, pc.readCost.Seconds(), pc.total)
	}
	return content, nil
}

func (pc *PowerClient) createDownloadRequest() *api.DownloadRequest {
	return &api.DownloadRequest{
		Path:       pc.pieceTask.Path,
		PieceRange: pc.pieceTask.Range,
		PieceNum:   pc.pieceTask.PieceNum,
		PieceSize:  pc.pieceTask.PieceSize,
	}
}

func (pc *PowerClient) successPiece(content *bytes.Buffer) *Piece {
	piece := NewPieceContent(pc.taskID, pc.node, pc.pieceTask.Cid, pc.pieceTask.Range,
		config.ResultSemiSuc, config.TaskStatusRunning, content)
	piece.PieceSize = pc.pieceTask.PieceSize
	piece.PieceNum = pc.pieceTask.PieceNum
	return piece
}

func (pc *PowerClient) failPiece() *Piece {
	return NewPiece(pc.taskID, pc.node, pc.pieceTask.Cid, pc.pieceTask.Range,
		config.ResultFail, config.TaskStatusRunning)
}

func (pc *PowerClient) is2xxStatus(code int) bool {
	return code >= 200 && code < 300
}

func (pc *PowerClient) readBody(body io.ReadCloser) string {
	buf := &bytes.Buffer{}
	if _, e := buf.ReadFrom(body); e != nil {
		return ""
	}
	return buf.String()
}

func (pc *PowerClient) log() *logrus.Logger {
	return pc.cfg.ClientLogger
}
