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
	"context"
	"math/rand"
	"os"
	"time"

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/downloader"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/pool"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/sirupsen/logrus"
)

// PieceWriter will be used in p2p downloader
// we provide 2 implementations, one for downloading file, one for streaming
type PieceWriter interface {
	// PreRun initializes PieceWriter
	PreRun(ctx context.Context) error

	// Run starts to process piece data in background
	Run(ctx context.Context)

	// PostRun will run when finish a task
	PostRun(ctx context.Context) error

	// Wait will block util all piece are processed
	Wait()
}

// ClientWriter writes a file for uploading and a target file.
type ClientWriter struct {
	// clientQueue maintains a queue of tasks that need to be written to disk.
	// The downloader will put the piece into this queue after it downloaded a piece successfully.
	// And clientWriter will poll values from this queue constantly and write to disk.
	clientQueue queue.Queue

	// notifyQueue sends a notification when all operation about a piece have
	// been completed successfully.
	notifyQueue queue.Queue

	// finish indicates whether the task written is completed.
	finish chan struct{}

	// clientFilePath is the full path of the temp file.
	clientFilePath string
	// serviceFilePath is the full path of the temp service file which
	// always ends with ".service".
	serviceFilePath string
	// serviceFile holds a file object for the serviceFilePath.
	serviceFile *os.File

	syncQueue queue.Queue
	// pieceIndex records the number of pieces currently downloaded.
	pieceIndex int
	// result records whether the write operation was successful.
	result bool
	// acrossWrite indicates whether the target file location and temporary file location cross file systems.
	// If that, the value is true. And vice versa.
	// We judge this by trying to make a hard link.
	acrossWrite bool
	// p2pPattern records whether the pattern equals "p2p".
	p2pPattern bool

	// targetQueue maintains a queue of tasks that need to be written to target path.
	targetQueue queue.Queue
	// targetWriter holds an instance of targetWriter.
	targetWriter *TargetWriter

	// api holds an instance of SupernodeAPI to interact with supernode.
	api api.SupernodeAPI
	cfg *config.Config

	cdnSource apiTypes.CdnSource
}

// NewClientWriter creates and initialize a ClientWriter instance.
func NewClientWriter(clientFilePath, serviceFilePath string,
	clientQueue, notifyQueue queue.Queue,
	api api.SupernodeAPI, cfg *config.Config, cdnSource apiTypes.CdnSource) PieceWriter {
	clientWriter := &ClientWriter{
		clientQueue:     clientQueue,
		notifyQueue:     notifyQueue,
		clientFilePath:  clientFilePath,
		serviceFilePath: serviceFilePath,
		api:             api,
		cfg:             cfg,
		cdnSource:       cdnSource,
	}
	return clientWriter
}

func (cw *ClientWriter) PreRun(ctx context.Context) (err error) {
	cw.p2pPattern = helper.IsP2P(cw.cfg.Pattern)
	if cw.p2pPattern {
		if e := fileutils.Link(cw.cfg.RV.TempTarget, cw.clientFilePath); e != nil {
			logrus.Warn(e)
			cw.acrossWrite = true
		}

		cw.serviceFile, _ = fileutils.OpenFile(cw.serviceFilePath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)
		if err := fileutils.Link(cw.serviceFilePath, cw.clientFilePath); err != nil {
			return err
		}
	}

	cw.result = true
	cw.targetQueue = queue.NewQueue(0)
	cw.targetWriter, err = NewTargetWriter(cw.cfg.RV.TempTarget, cw.targetQueue, cw.cfg, cw.cdnSource)
	if err != nil {
		return
	}

	cw.syncQueue = startSyncWriter(nil)

	cw.finish = make(chan struct{})
	return
}

func (cw *ClientWriter) PostRun(ctx context.Context) (err error) {
	src := cw.clientFilePath
	if cw.acrossWrite || !helper.IsP2P(cw.cfg.Pattern) {
		src = cw.cfg.RV.TempTarget
	} else {
		if _, err := os.Stat(cw.clientFilePath); err != nil {
			logrus.Warnf("client file path:%s not found", cw.clientFilePath)
			if e := fileutils.Link(cw.serviceFilePath, cw.clientFilePath); e != nil {
				logrus.Warnln("hard link failed, instead of use copy")
				fileutils.CopyFile(cw.serviceFilePath, cw.clientFilePath)
			}
		}
	}
	if err = downloader.MoveFile(src, cw.cfg.RV.RealTarget, cw.cfg.Md5); err != nil {
		return
	}
	logrus.Infof("download successfully from dragonfly")
	return nil
}

// Run starts writing downloading file.
func (cw *ClientWriter) Run(ctx context.Context) {
	go cw.targetWriter.Run(ctx)

	for {
		item := cw.clientQueue.Poll()
		state, ok := item.(string)
		if ok && state == last {
			if !cw.acrossWrite {
				cw.serviceFile.Sync()
			}
			break
		}
		if ok && state == reset {
			if cw.serviceFile != nil {
				cw.serviceFile.Truncate(0)
			}
			if cw.acrossWrite {
				cw.targetQueue.Put(state)
			}
			continue
		}
		if !cw.result {
			continue
		}

		piece, ok := item.(*Piece)
		if !ok {
			continue
		}
		if err := cw.write(piece); err != nil {
			logrus.Errorf("write item:%s error:%v", piece, err)
			cw.cfg.BackSourceReason = config.BackSourceReasonWriteError
			cw.result = false
		}
	}
	if cw.serviceFile != nil {
		cw.serviceFile.Close()
	}
	cw.targetQueue.Put(last)
	cw.targetWriter.Wait()
	close(cw.finish)
}

// Wait for Run whether is finished.
func (cw *ClientWriter) Wait() {
	if cw.finish != nil {
		<-cw.finish
	}
}

func (cw *ClientWriter) write(piece *Piece) error {
	startTime := time.Now()
	if !cw.p2pPattern {
		cw.targetQueue.Put(piece)
		go sendSuccessPiece(cw.api, cw.cfg.RV.Cid, piece, time.Since(startTime), cw.notifyQueue)
		return nil
	}

	if cw.acrossWrite {
		piece.IncWriter()
		cw.targetQueue.Put(piece)
	}

	cw.pieceIndex++
	err := writePieceToFile(piece, cw.serviceFile, cw.cdnSource)
	if err == nil {
		go sendSuccessPiece(cw.api, cw.cfg.RV.Cid, piece, time.Since(startTime), cw.notifyQueue)
	}
	return err
}

func writePieceToFile(piece *Piece, file *os.File, cdnSource apiTypes.CdnSource) error {
	var pieceHeader = 5
	// the piece is not wrapped with source cdn type
	noWrapper := (cdnSource == apiTypes.CdnSourceSource)
	if noWrapper {
		pieceHeader = 0
	}

	start := int64(piece.PieceNum) * (int64(piece.PieceSize) - int64(pieceHeader))
	if _, err := file.Seek(start, 0); err != nil {
		return err
	}

	writer := pool.AcquireWriter(file)
	_, err := piece.WriteTo(writer, noWrapper)
	pool.ReleaseWriter(writer)
	writer = nil
	return err
}

func startSyncWriter(q queue.Queue) queue.Queue {
	return nil
}

func sendSuccessPiece(api api.SupernodeAPI, cid string, piece *Piece, cost time.Duration, notifyQueue queue.Queue) {
	reportPieceRequest := &types.ReportPieceRequest{
		TaskID:     piece.TaskID,
		Cid:        cid,
		DstCid:     piece.DstCid,
		PieceRange: piece.Range,
	}

	var retry = 0
	var maxRetryTime = 3
	for {
		if retry >= maxRetryTime {
			logrus.Errorf("failed to report piece to supernode with request(%+v) even after retrying max retry time", reportPieceRequest)
			break
		}

		_, err := api.ReportPiece(piece.SuperNode, reportPieceRequest)
		if err == nil {
			if notifyQueue != nil {
				notifyQueue.Put("success")
			}
			if retry > 0 {
				logrus.Warnf("success to report piece with request(%+v) after retrying (%d) times", reportPieceRequest, retry)
			}
			break
		}

		sleepTime := time.Duration(rand.Intn(500)+50) * time.Millisecond
		logrus.Warnf("failed to report piece to supernode with request(%+v) for (%d) times and will retry after sleep %.3fs", reportPieceRequest, retry, sleepTime.Seconds())
		time.Sleep(sleepTime)
		retry++
	}

	if cost.Seconds() > 2.0 {
		logrus.Infof(
			"async writer and report suc from dst:%s... cost:%.3f for range:%s",
			piece.DstCid[:25], cost.Seconds(), piece.Range)
	}
}
