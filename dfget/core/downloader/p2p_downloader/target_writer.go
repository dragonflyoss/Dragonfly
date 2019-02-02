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
	"fmt"
	"io"
	"os"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
)

// TargetWriter writes downloading file to disk.
type TargetWriter struct {
	dst        string
	dstFile    *os.File
	pieceQueue util.Queue
	finish     chan struct{}
	pieceIndex int
	result     bool
	syncQueue  util.Queue
	cfg        *config.Config
}

// NewTargetWriter creates and initialize a TargetWriter instance.
func NewTargetWriter(dst string, queue util.Queue, cfg *config.Config) (*TargetWriter, error) {
	targetWriter := &TargetWriter{
		dst:        dst,
		pieceQueue: queue,
		cfg:        cfg,
	}
	if err := targetWriter.init(); err != nil {
		return nil, err
	}
	return targetWriter, nil
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
			tw.cfg.ClientLogger.Errorf("write item:%s error:%v", piece, err)
			tw.cfg.BackSourceReason = config.BackSourceReasonWriteError
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
