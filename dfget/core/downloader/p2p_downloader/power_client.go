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

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/limitreader"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/pool"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"

	"github.com/sirupsen/logrus"
)

const (
	// downloadPieceTimeout specifies the timeout for piece downloading.
	// If the actual execution time exceeds this threshold, a warning will be thrown.
	downloadPieceTimeout = 2.0 * time.Second
)

// PowerClient downloads file from dragonfly.
type PowerClient struct {
	// taskID is a string which represents a unique task.
	taskID string
	// headers is the extra HTTP headers when downloading a piece.
	headers []string
	// node indicates the IP address of the currently registered supernode.
	node string
	// pieceTask is the data when successfully pulling piece task
	// and the task is continuing.
	pieceTask *types.PullPieceTaskResponseContinueData

	cfg *config.Config
	// queue maintains a queue of tasks that to be downloaded.
	// When the download fails, the piece is requeued.
	queue queue.Queue
	// clientQueue maintains a queue of tasks that need to be written to disk.
	// A piece will be putted into this queue after it be downloaded successfully.
	clientQueue queue.Queue

	// rateLimiter limits the download speed.
	rateLimiter *ratelimiter.RateLimiter

	// total indicates the total length of the downloaded piece.
	total int64
	// readCost records how long it took to download the piece.
	readCost time.Duration

	// downloadAPI holds an instance of DownloadAPI.
	downloadAPI api.DownloadAPI

	clientError *types.ClientErrorRequest

	cdnSource  apiTypes.CdnSource
	fileLength int64
}

// Run starts run the task.
func (pc *PowerClient) Run() error {
	startTime := time.Now()

	content, err := pc.downloadPiece()

	timeDuring := time.Since(startTime).Seconds()
	logrus.Debugf("client range:%s cost:%.3f from peer:%s:%d, readCost:%.3f, length:%d",
		pc.pieceTask.Range, timeDuring, pc.pieceTask.PeerIP, pc.pieceTask.PeerPort,
		pc.readCost.Seconds(), pc.total)

	if err != nil {
		logrus.Errorf("failed to read piece cont(%s) from dst:%s:%d, wait 20 ms: %v",
			pc.pieceTask.Range, pc.pieceTask.PeerIP, pc.pieceTask.PeerPort, err)
		time.AfterFunc(time.Millisecond*20, func() {
			pc.queue.Put(pc.failPiece())
		})
		return err
	}

	piece := pc.successPiece(content)
	pc.clientQueue.Put(piece)
	pc.queue.Put(piece)
	return nil
}

// ClientError returns the client error if occurred
func (pc *PowerClient) ClientError() *types.ClientErrorRequest {
	return pc.clientError
}

func (pc *PowerClient) downloadPiece() (content *pool.Buffer, e error) {
	dstIP := pc.pieceTask.PeerIP
	peerPort := pc.pieceTask.PeerPort

	// check that the target download peer is available
	if dstIP != "" && dstIP != pc.node {
		if _, e = httputils.CheckConnect(dstIP, peerPort, -1); e != nil {
			return nil, e
		}
	}

	// send download request
	startTime := time.Now()
	timeout := netutils.CalculateTimeout(int64(pc.pieceTask.PieceSize), pc.cfg.MinRate, config.DefaultMinRate, 10*time.Second)
	resp, err := pc.downloadAPI.Download(dstIP, peerPort, pc.createDownloadRequest(), timeout)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("success to get resp timeSince(%v)", time.Since(startTime))
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return nil, errortypes.ErrRangeNotSatisfiable
	}
	if !pc.is2xxStatus(resp.StatusCode) {
		if resp.StatusCode == http.StatusNotFound {
			pc.initFileNotExistError()
		}
		return nil, errortypes.New(resp.StatusCode, pc.readBody(resp.Body))
	}

	pieceMetaArr := strings.Split(pc.pieceTask.PieceMd5, ":")
	pieceMD5 := pieceMetaArr[0]

	// start to read data from resp
	// use limitReader to limit the download speed
	limitReader := limitreader.NewLimitReaderWithLimiter(pc.rateLimiter, resp.Body, pieceMD5 != "")
	content = pool.AcquireBufferSize(int(pc.pieceTask.PieceSize))
	defer func() {
		// if an error happened, the content cannot be released outside.
		if e != nil {
			pool.ReleaseBuffer(content)
			content = nil
		}
	}()
	if pc.total, e = content.ReadFrom(limitReader); e != nil {
		return nil, e
	}
	pc.readCost = time.Since(startTime)

	// Verify md5 code
	if pieceMD5 != "" {
		if realMd5 := limitReader.Md5(); realMd5 != pieceMD5 {
			pc.initFileMd5NotMatchError(dstIP, realMd5, pieceMD5)
			return nil, fmt.Errorf("piece range:%s md5 not match, expected:%s real:%s",
				pc.pieceTask.Range, pieceMD5, realMd5)
		}
	}

	if timeDuring := time.Since(startTime); timeDuring > downloadPieceTimeout {
		logrus.Warnf("client range:%s cost:%.3f from peer:%s, readCost:%.3f, length:%d",
			pc.pieceTask.Range, timeDuring.Seconds(), dstIP, pc.readCost.Seconds(), pc.total)
	}
	return content, nil
}

func (pc *PowerClient) createDownloadRequest() *api.DownloadRequest {
	pieceRange := pc.pieceTask.Range
	headers := netutils.ConvertHeaders(pc.headers)
	if pc.cdnSource == apiTypes.CdnSourceSource {
		if pc.fileLength > 0 {
			pieceRange = wipeOutOfRange(pc.pieceTask.Range, pc.fileLength)
		}
		if headers == nil {
			headers = make(map[string]string)
		}
		headers[config.StrCDNSource] = string(apiTypes.CdnSourceSource)
	}

	return &api.DownloadRequest{
		Path:       pc.pieceTask.Path,
		PieceRange: pieceRange,
		PieceNum:   pc.pieceTask.PieceNum,
		PieceSize:  pc.pieceTask.PieceSize,
		Headers:    headers,
	}
}

func (pc *PowerClient) successPiece(content *pool.Buffer) *Piece {
	piece := NewPieceContent(pc.taskID, pc.node, pc.pieceTask.Cid, pc.pieceTask.Range,
		constants.ResultSemiSuc, constants.TaskStatusRunning, content, pc.cdnSource)
	piece.PieceSize = pc.pieceTask.PieceSize
	piece.PieceNum = pc.pieceTask.PieceNum
	return piece
}

func (pc *PowerClient) failPiece() *Piece {
	return NewPiece(pc.taskID, pc.node, pc.pieceTask.Cid, pc.pieceTask.Range,
		constants.ResultFail, constants.TaskStatusRunning, pc.cdnSource)
}

func (pc *PowerClient) initFileNotExistError() {
	pc.clientError = &types.ClientErrorRequest{
		ErrorType: constants.ClientErrorFileNotExist,
		SrcCid:    pc.cfg.RV.Cid,
		DstCid:    pc.pieceTask.Cid,
		TaskID:    pc.taskID,
	}
}

func (pc *PowerClient) initFileMd5NotMatchError(dstIP, realMd5, expectedMd5 string) {
	pc.clientError = &types.ClientErrorRequest{
		ErrorType:   constants.ClientErrorFileMd5NotMatch,
		SrcCid:      pc.cfg.RV.Cid,
		DstCid:      pc.pieceTask.Cid,
		DstIP:       dstIP,
		TaskID:      pc.taskID,
		Range:       pc.pieceTask.Range,
		RealMd5:     realMd5,
		ExpectedMd5: expectedMd5,
	}
}

func (pc *PowerClient) is2xxStatus(code int) bool {
	return code >= 200 && code < 300
}

func (pc *PowerClient) readBody(body io.ReadCloser) string {
	buf := &bytes.Buffer{}
	if _, e := buf.ReadFrom(body); e != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}
