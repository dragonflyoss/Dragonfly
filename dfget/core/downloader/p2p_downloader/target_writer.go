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
	"fmt"
	"os"

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/sirupsen/logrus"
)

// TargetWriter writes downloading file to disk.
type TargetWriter struct {
	// dst is the destination file path.
	dst string
	// dstFile holds the file object for dst path.
	dstFile *os.File
	// pieceQueue maintains a queue of tasks that need to be written to target path.
	pieceQueue queue.Queue
	// finish indicates whether the task written by `TargetWriter` is completed.
	finish chan struct{}
	// pieceIndex records the number of pieces currently downloaded.
	pieceIndex int
	// result records whether the write operation was successful.
	result bool

	syncQueue queue.Queue
	cfg       *config.Config

	cdnSource apiTypes.CdnSource
}

// NewTargetWriter creates and initialize a TargetWriter instance.
func NewTargetWriter(dst string, q queue.Queue, cfg *config.Config, cdnSource apiTypes.CdnSource) (*TargetWriter, error) {
	targetWriter := &TargetWriter{
		dst:        dst,
		pieceQueue: q,
		cfg:        cfg,
		cdnSource:  cdnSource,
	}
	if err := targetWriter.init(); err != nil {
		return nil, err
	}
	return targetWriter, nil
}

func (tw *TargetWriter) init() error {
	var err error
	tw.dstFile, err = fileutils.OpenFile(tw.dst, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0755)
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
func (tw *TargetWriter) Run(ctx context.Context) {
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
		if err := tw.write(piece, tw.cdnSource); err != nil {
			logrus.Errorf("write item:%s error:%v", piece, err)
			tw.cfg.BackSourceReason = config.BackSourceReasonWriteError
			tw.result = false
		}

		if tw.syncQueue != nil && tw.pieceIndex%4 == 0 {
			tw.syncQueue.Put(tw.dstFile.Fd())
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

func (tw *TargetWriter) write(piece *Piece, cdnSource apiTypes.CdnSource) error {
	tw.pieceIndex++
	return writePieceToFile(piece, tw.dstFile, cdnSource)
}
