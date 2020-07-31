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
	"context"
	"io"
	"sync"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/regist"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

type RegularDownloadTimeoutTask struct {
}

type StreamDownloadTimeoutTask struct {
	Config       *config.Config
	SupernodeAPI api.SupernodeAPI
	UploaderAPI  api.UploaderAPI
	Result       *regist.RegisterResult
}

var _ DownloadTimeoutTask = &RegularDownloadTimeoutTask{}
var _ DownloadTimeoutTask = &StreamDownloadTimeoutTask{}

type protocolContent struct {
	taskID           string
	pieceNum         int
	pieceSize        int32
	pieceContentSize int32
	pieceContent     *bytes.Buffer
}

func (sdt *StreamDownloadTimeoutTask) startWriter(ctx context.Context, reader io.Reader) error {
	// parameter check
	if reader == nil {
		return errors.Wrap(errortypes.ErrEmptyValue, "empty stream reader")
	}

	// get piece content size which does not include the piece header and tail
	pieceContSize := sdt.Result.PieceSize - config.PieceMetaSize
	// the left size of data for a complete piece
	pieceContLeft := pieceContSize
	// pieceNum is used to track the successfully processed pieces
	var pieceNum int64
	// realFileLength is used to calculate the file length dynamically
	var realFileLength int64
	// realHTTPFileLength is used to calculate the http file length dynamically
	var realHTTPFileLength int64

	buf := make([]byte, pieceContSize)
	var bb = &bytes.Buffer{}

	// start writer pool
	routineCount := calculateRoutineCount(sdt.Result.FileLength, sdt.Result.PieceSize)
	var wg = &sync.WaitGroup{}
	jobCh := make(chan *protocolContent)
	sdt.writePool(ctx, wg, routineCount, jobCh)

	for {
		n, e := reader.Read(buf)
		if n > 0 {
			logrus.Debugf("success to read content with length: %d", n)
			realFileLength += int64(n)
			realHTTPFileLength += int64(n)
			if int(pieceContLeft) <= n {
				bb.Write(buf[:pieceContLeft])
				pc := &protocolContent{
					taskID:           sdt.Result.TaskID,
					pieceNum:         int(pieceNum),
					pieceSize:        sdt.Result.PieceSize,
					pieceContentSize: pieceContSize,
					pieceContent:     bb,
				}
				jobCh <- pc
				logrus.Debugf("send the protocolContent taskID: %s pieceNum: %d", sdt.Result.TaskID, pieceNum)

				realFileLength += config.PieceMetaSize
				pieceNum++

				// write the data left to a new buffer
				// TODO: recycling bytes.Buffer
				bb = bytes.NewBuffer([]byte{})
				n -= int(pieceContLeft)
				if n > 0 {
					bb.Write(buf[pieceContLeft : int(pieceContLeft)+n])
				}
				pieceContLeft = pieceContSize
			} else {
				bb.Write(buf[:n])
			}
			pieceContLeft -= int32(n)
		}

		if e == io.EOF {
			if realFileLength == 0 || bb.Len() > 0 {
				jobCh <- &protocolContent{
					taskID:           sdt.Result.TaskID,
					pieceNum:         int(pieceNum),
					pieceSize:        sdt.Result.PieceSize,
					pieceContentSize: int32(bb.Len()),
					pieceContent:     bb,
				}
				logrus.Debugf("send the protocolContent taskID: %s pieceNum: %d", sdt.Result.TaskID, pieceNum)

				realFileLength += config.PieceMetaSize
			}
			logrus.Infof("sent all protocolContents with realFileLength(%d) and wait for delivering to uploader", int(realFileLength))
			break
		}
		if e != nil {
			close(jobCh)
			return e
		}
	}

	close(jobCh)
	wg.Wait()
	return nil
}

func (sdt *StreamDownloadTimeoutTask) writePool(ctx context.Context, wg *sync.WaitGroup,
	n int, jobCh chan *protocolContent) {
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			for job := range jobCh {
				// construct the request
				req := &api.UploadStreamPieceRequest{
					TaskID:    job.taskID,
					PieceNum:  job.pieceNum,
					PieceSize: job.pieceContentSize,
					Content:   job.pieceContent,
				}

				// send the upload request
				if err := sdt.UploaderAPI.DeliverPieceToUploader(sdt.Config.RV.LocalIP, sdt.Config.RV.PeerPort, req); err != nil {
					logrus.Errorf("failed to deliver the %d-th piece to uploader with taskID: %s: %v",
						job.pieceNum, job.taskID, err)
					// TODO: should there be recovery work?
					continue
				}
			}
			wg.Done()
		}(i)
	}
}

func calculateRoutineCount(httpFileLength int64, pieceSize int32) int {
	routineSize := config.StreamWriterRoutineLimit
	if httpFileLength < 0 || pieceSize <= 0 {
		return routineSize
	}

	if httpFileLength == 0 {
		return 1
	}

	pieceContSize := pieceSize - config.PieceMetaSize
	tmpSize := (int)((httpFileLength + int64(pieceContSize-1)) / int64(pieceContSize))
	if tmpSize == 0 {
		tmpSize = 1
	}
	if tmpSize < routineSize {
		routineSize = tmpSize
	}
	return routineSize
}
