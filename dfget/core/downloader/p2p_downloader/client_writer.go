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
	"bufio"
	"io"
	"os"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/sirupsen/logrus"
)

// ClientWriter writes a file for uploading and a target file.
type ClientWriter struct {
	taskFileName string
	cid          string
	clintQueue   util.Queue
	finish       chan struct{}

	clientFilePath  string
	serviceFilePath string
	serviceFile     *os.File

	syncQueue   util.Queue
	pieceIndex  int
	result      bool
	acrossWrite bool
	p2pPattern  bool
	total       int

	targetFinish chan struct{}
	targetQueue  util.Queue
	targetWriter *TargetWriter

	api api.SupernodeAPI
	cfg *config.Config
}

// NewClientWriter creates and initialize a ClientWriter instance.
func NewClientWriter(taskFileName, cid, clientFilePath, serviceFilePath string,
	clientQueue util.Queue, api api.SupernodeAPI, cfg *config.Config) (*ClientWriter, error) {
	clientWriter := &ClientWriter{
		taskFileName:    taskFileName,
		cid:             cid,
		clintQueue:      clientQueue,
		clientFilePath:  clientFilePath,
		serviceFilePath: serviceFilePath,
		api:             api,
		cfg:             cfg,
	}
	if err := clientWriter.init(); err != nil {
		return nil, err
	}
	return clientWriter, nil
}

func (cw *ClientWriter) init() (err error) {
	cw.p2pPattern = helper.IsP2P(cw.cfg.Pattern)
	if cw.p2pPattern {
		if e := util.Link(cw.cfg.RV.TempTarget, cw.clientFilePath); e != nil {
			logrus.Warn(e)
			cw.acrossWrite = true
		}

		cw.serviceFile, _ = util.OpenFile(cw.serviceFilePath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)
		util.Link(cw.serviceFilePath, cw.clientFilePath)
	}

	cw.result = true
	cw.targetQueue = util.NewQueue(0)
	cw.targetWriter, err = NewTargetWriter(cw.cfg.RV.TempTarget, cw.targetQueue, cw.cfg)
	if err != nil {
		return
	}
	go cw.targetWriter.Run()

	cw.syncQueue = startSyncWriter(nil)

	cw.finish = make(chan struct{})
	return
}

// Run starts writing downloading file.
func (cw *ClientWriter) Run() {
	for {
		item := cw.clintQueue.Poll()
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
		if err := cw.write(piece, time.Now()); err != nil {
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

func (cw *ClientWriter) write(piece *Piece, startTime time.Time) error {
	if !cw.p2pPattern {
		cw.targetQueue.Put(piece)
		return nil
	}

	if cw.acrossWrite {
		cw.targetQueue.Put(piece)
	}

	start := int64(piece.PieceNum) * (int64(piece.PieceSize) - 5)

	cw.pieceIndex++
	cw.serviceFile.Seek(start, 0)
	buf := bufio.NewWriterSize(cw.serviceFile, 4*1024*1024)
	_, err := io.Copy(buf, piece.RawContent())
	buf.Flush()

	go cw.sendSuccessPiece(piece, time.Since(startTime))
	return err
}

func startSyncWriter(queue util.Queue) util.Queue {
	return nil
}

func (cw *ClientWriter) sendSuccessPiece(piece *Piece, cost time.Duration) {
	cw.api.ReportPiece(piece.SuperNode, &types.ReportPieceRequest{
		TaskID:     piece.TaskID,
		Cid:        cw.cfg.RV.Cid,
		DstCid:     piece.DstCid,
		PieceRange: piece.Range,
	})
	if cost.Seconds() > 2.0 {
		logrus.Infof(
			"async writer and report suc from dst:%s... cost:%.3f for range:%s",
			piece.DstCid[:25], cost.Seconds(), piece.Range)
	}
}
