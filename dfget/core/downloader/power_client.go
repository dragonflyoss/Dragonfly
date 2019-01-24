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
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
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
	cfg         *config.Config
	queue       util.Queue
	clientQueue util.Queue
}

// Run starts run the task.
func (pc *PowerClient) Run() (err error) {
	pieceMetaArr := strings.Split(pc.pieceTask.PieceMd5, ":")
	pieceMD5 := pieceMetaArr[0]
	dstIP := pc.pieceTask.PeerIP
	peerPort := pc.pieceTask.PeerPort

	defer func() {
		if err != nil {
			pc.cfg.ClientLogger.Errorf("read piece cont error:%s from dst:%s", err, dstIP)
			// TODO handle dst_ip == self.node
		}
	}()

	_, err = util.CheckConnect(dstIP, peerPort, -1)
	if dstIP == pc.node || err == nil {
		url := fmt.Sprintf("http://%s:%d%s", dstIP, peerPort, pc.pieceTask.Path)
		startTime := time.Now().Unix()

		headers := make(map[string]string)
		headers["Range"] = pc.pieceTask.Range
		headers["pieceNum"] = strconv.Itoa(pc.pieceTask.PieceNum)
		headers["pieceSize"] = strconv.Itoa(pc.pieceTask.PieceSize)
		resp, err := httpGetWithHeaders(url, headers)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		buf := make([]byte, 0, 256*1024)
		pieceCont := bytes.NewBuffer(buf)
		reader := NewLimitReader(resp.Body, pc.cfg.LocalLimit, pieceMD5 != "")
		total, err := pieceCont.ReadFrom(reader)
		pc.cfg.ClientLogger.Infof("get pieceCont total: %d", total)
		if err != nil {
			return err
		}
		// TODO handle read timeout

		readFinish := time.Now().Unix()
		realMd5 := reader.Md5()
		if realMd5 != pieceMD5 {
			pc.cfg.ClientLogger.Errorf("piece range:%s error,realMd5:%s,expectedMd5:%s,dstIp:%s,total:%d", pc.pieceTask.Range, realMd5, pieceMD5, dstIP, total)
			return fmt.Errorf("md5 not match, expected:%s real:%s", pieceMD5, realMd5)
		}
		piece := NewPieceContent(pc.taskID, pc.node, pc.pieceTask.Cid, pc.pieceTask.Range, config.ResultSemiSuc, config.TaskStatusRunning, pieceCont)
		// NOTE should unify the type
		piece.PieceSize = int32(pc.pieceTask.PieceSize)
		piece.PieceNum = pc.pieceTask.PieceNum
		pc.clientQueue.Put(piece)
		pc.queue.Put(piece)

		endTime := time.Now().Unix()
		timeDuring := endTime - startTime
		if timeDuring > 2.0 {
			pc.cfg.ClientLogger.Warnf("client range:%s cost:%.3f from peer:%s,its readCost:%.3f,cont length:%d", pc.pieceTask.Range, timeDuring, dstIP, readFinish-startTime, total)
		}
		return nil
	}

	piece := NewPiece(pc.taskID, pc.node, pc.pieceTask.Cid, pc.pieceTask.Range, config.ResultFail, config.TaskStatusRunning)
	pc.queue.Put(piece)
	return nil
}

// ----------------------------------------------------------------------------
// ClientWriter

// NewClientWriter creates and initialize a ClientWriter instance.
func NewClientWriter(taskFileName, cid, clientFilePath, serviceFilePath string, clientQueue util.Queue, Cfg *config.Config) (*ClientWriter, error) {
	clientWriter := &ClientWriter{
		taskFileName:    taskFileName,
		cid:             cid,
		clintQueue:      clientQueue,
		Cfg:             Cfg,
		clientFilePath:  clientFilePath,
		serviceFilePath: serviceFilePath,
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

	clientFilePath  string
	serviceFilePath string
	serviceFile     *os.File

	syncQueue   util.Queue
	pieceIndex  int
	result      bool
	acrossWrite bool
	total       int

	targetFinish chan struct{}
	targetQueue  util.Queue
	targetWriter *TargetWriter

	Cfg *config.Config
}

func (cw *ClientWriter) init() (err error) {
	if e := util.Link(cw.Cfg.RV.TempTarget, cw.clientFilePath); e != nil {
		cw.Cfg.ClientLogger.Warn(e)
		cw.acrossWrite = true
	}

	cw.serviceFile, _ = util.OpenFile(cw.serviceFilePath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)

	util.Link(cw.serviceFilePath, cw.clientFilePath)

	cw.result = true
	cw.targetQueue = util.NewQueue(0)
	cw.targetWriter, err = NewTargetWriter(cw.Cfg.RV.TempTarget, cw.targetQueue, cw.Cfg)
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
			cw.Cfg.ClientLogger.Errorf("write item:%s error:%v", piece, err)
			cw.Cfg.BackSourceReason = config.BackSourceReasonWriteError
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
	if cw.acrossWrite {
		cw.targetQueue.Put(piece)
	}

	return err
}

// ----------------------------------------------------------------------------
// TargetWriter

// NewTargetWriter creates and initialize a TargetWriter instance.
func NewTargetWriter(dst string, queue util.Queue, Cfg *config.Config) (*TargetWriter, error) {
	targetWriter := &TargetWriter{
		dst:        dst,
		pieceQueue: queue,
		Cfg:        Cfg,
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
	Cfg        *config.Config
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
			tw.Cfg.ClientLogger.Errorf("write item:%s error:%v", piece, err)
			tw.Cfg.BackSourceReason = config.BackSourceReasonWriteError
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
