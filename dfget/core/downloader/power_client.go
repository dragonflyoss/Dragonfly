/*
 * Copyright 1999-2018 Alibaba Group.
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
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
)

// ----------------------------------------------------------------------------
// PowerClient

// PowerClient downloads file from dragonfly.
type PowerClient struct {
	taskID      string
	node        string
	pieceTask   *types.PullPieceTaskResponseContinueData
	rateLimiter *util.RateLimiter
	ctx         *config.Context
}

// ----------------------------------------------------------------------------
// ClientWriter

// NewClientWriter creates and initialize a ClientWriter instance.
func NewClientWriter(taskFileName string, cid string, clientQueue util.Queue) (*ClientWriter, error) {
	clientWriter := &ClientWriter{
		taskFileName: taskFileName,
		cid:          cid,
		clintQueue:   clientQueue,
	}
	if err := clientWriter.init(); err != nil {
		return nil, err
	}
	return clientWriter, nil
}

// ClientWriter writes a file for uploading and a target file.
type ClientWriter struct {
	taskFileName string
	cid          string
	clintQueue   util.Queue
	finish       chan struct{}

	serviceFile *os.File

	syncQueue   util.Queue
	pieceIndex  int
	result      bool
	acrossWrite bool
	total       int

	targetFinish chan struct{}
	targetQueue  util.Queue
	targetWriter *TargetWriter

	ctx *config.Context
}

func (cw *ClientWriter) init() (err error) {
	clientFilePath := helper.GetTaskFile(cw.taskFileName, cw.ctx.RV.DataDir)
	if e := util.Link(cw.ctx.RV.TempTarget, clientFilePath); e != nil {
		cw.ctx.ClientLogger.Warn(e)
		cw.acrossWrite = true
	}

	serviceFilePath := helper.GetServiceFile(cw.taskFileName, cw.ctx.RV.DataDir)
	cw.serviceFile, _ = util.OpenFile(serviceFilePath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)

	util.Link(serviceFilePath, clientFilePath)

	cw.targetQueue = util.NewQueue(0)
	cw.targetWriter, err = NewTargetWriter(cw.ctx.RV.TempTarget, cw.targetQueue, cw.ctx)
	if err != nil {
		return
	}

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
			cw.serviceFile.Truncate(0)
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
			cw.ctx.ClientLogger.Errorf("write item:%s error:%v", piece, err)
			cw.ctx.BackSourceReason = config.BackSourceReasonWriteError
			cw.result = false
		}
	}
	cw.serviceFile.Close()
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
	start := int64(piece.PieceNum) * (int64(piece.PieceSize) - 5)

	cw.pieceIndex++
	cw.serviceFile.Seek(start, 0)
	buf := bufio.NewWriterSize(cw.serviceFile, 4*1024*1024)
	_, err := io.Copy(buf, piece.RawContent())
	buf.Flush()

	return err
}

// ----------------------------------------------------------------------------
// TargetWriter

// NewTargetWriter creates and initialize a TargetWriter instance.
func NewTargetWriter(dst string, queue util.Queue, ctx *config.Context) (*TargetWriter, error) {
	targetWriter := &TargetWriter{
		dst:        dst,
		pieceQueue: queue,
		ctx:        ctx,
	}
	if err := targetWriter.init(); err != nil {
		return nil, err
	}
	return targetWriter, nil
}

// TargetWriter writes downloading file to disk.
type TargetWriter struct {
	dst        string
	dstFile    *os.File
	pieceQueue util.Queue
	finish     chan struct{}
	pieceIndex int
	result     bool
	syncQueue  util.Queue
	ctx        *config.Context
}

func (tw *TargetWriter) init() error {
	var err error
	tw.dstFile, err = util.OpenFile(tw.dst, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("open target file:%s error:%v", tw.dst, err)
	}

	tw.finish = make(chan struct{})
	tw.pieceIndex = 0
	tw.result = true
	tw.syncQueue = startSyncWriter(nil)
	return nil
}

// Run starts writing downloading file.
func (tw *TargetWriter) Run() {
	for {
		item := tw.pieceQueue.Poll()
		state, ok := item.(string)
		if ok && state == last {
			tw.dstFile.Sync()
			break
		}
		if !tw.result {
			continue
		}
		if ok && state == reset {
			tw.dstFile.Truncate(0)
			continue
		}

		piece, ok := item.(*Piece)
		if !ok {
			continue
		}
		if err := tw.write(piece); err != nil {
			tw.ctx.ClientLogger.Errorf("write item:%s error:%v", piece, err)
			tw.ctx.BackSourceReason = config.BackSourceReasonWriteError
			tw.result = false
		}
	}
	tw.dstFile.Close()
	close(tw.finish)
}

// Wait the Run is finished.
func (tw *TargetWriter) Wait() {
	if tw.finish != nil {
		<-tw.finish
	}
}

func (tw *TargetWriter) write(piece *Piece) error {
	start := int64(piece.PieceNum) * (int64(piece.PieceSize) - 5)

	tw.pieceIndex++
	tw.dstFile.Seek(start, 0)
	buf := bufio.NewWriterSize(tw.dstFile, 4*1024*1024)
	_, err := io.Copy(buf, piece.RawContent())
	buf.Flush()

	if tw.syncQueue != nil && tw.pieceIndex%4 == 0 {
		tw.syncQueue.Put(tw.dstFile.Fd())
	}

	return err
}

func startSyncWriter(queue util.Queue) util.Queue {
	return nil
}
