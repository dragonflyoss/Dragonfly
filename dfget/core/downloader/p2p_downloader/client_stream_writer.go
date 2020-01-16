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
	"io"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/sirupsen/logrus"
)

// ClientWriter writes a file for uploading and a target file.
type ClientStreamWriter struct {
	// clientQueue maintains a queue of tasks that need to be written to disk.
	// The downloader will put the piece into this queue after it downloaded a piece successfully.
	// And clientWriter will poll values from this queue constantly and write to disk.
	clientQueue queue.Queue
	// finish indicates whether the task written is completed.
	finish chan struct{}

	syncQueue queue.Queue
	// pieceIndex records the number of pieces currently downloaded.
	pieceIndex int
	// result records whether the write operation was successful.
	result bool

	// p2pPattern records whether the pattern equals "p2p".
	p2pPattern bool

	// pipeWriter is the writer half of a pipe, all piece data will be wrote into pipeWriter
	pipeWriter *io.PipeWriter

	// pipeReader is the read half of a pipe
	pipeReader *io.PipeReader

	cache map[int]*Piece

	// api holds an instance of SupernodeAPI to interact with supernode.
	api api.SupernodeAPI
	cfg *config.Config
}

// NewClientStreamWriter creates and initialize a ClientStreamWriter instance.
func NewClientStreamWriter(clientQueue queue.Queue, api api.SupernodeAPI, cfg *config.Config) (*ClientStreamWriter, error) {
	pr, pw := io.Pipe()
	clientWriter := &ClientStreamWriter{
		clientQueue: clientQueue,
		pipeReader:  pr,
		pipeWriter:  pw,
		api:         api,
		cfg:         cfg,
	}
	if err := clientWriter.init(); err != nil {
		return nil, err
	}
	return clientWriter, nil
}

func (csw *ClientStreamWriter) init() (err error) {
	csw.p2pPattern = helper.IsP2P(csw.cfg.Pattern)
	csw.result = true
	csw.finish = make(chan struct{})
	return
}

// Run starts writing pipe.
func (csw *ClientStreamWriter) Run(ctx context.Context) {
	for {
		item := csw.clientQueue.Poll()
		state, ok := item.(string)
		if ok && state == last {
			break
		}
		if ok && state == reset {
			// stream could not reset, just return error
			//
			csw.pipeWriter.CloseWithError(fmt.Errorf("stream writer not support reset"))
			continue
		}
		if !csw.result {
			continue
		}

		piece, ok := item.(*Piece)
		if !ok {
			continue
		}
		if err := csw.write(piece); err != nil {
			logrus.Errorf("write item:%s error:%v", piece, err)
			csw.cfg.BackSourceReason = config.BackSourceReasonWriteError
			csw.result = false
		}
	}

	csw.pipeWriter.Close()
	close(csw.finish)
}

// Wait for Run whether is finished.
func (csw *ClientStreamWriter) Wait() {
	if csw.finish != nil {
		<-csw.finish
	}
}

func (csw *ClientStreamWriter) write(piece *Piece) error {
	startTime := time.Now()
	// TODO csw.p2pPattern

	err := csw.writePieceToPipe(piece)
	if err == nil {
		go csw.sendSuccessPiece(piece, time.Since(startTime))
	}
	return err
}

func (csw *ClientStreamWriter) writePieceToPipe(p *Piece) error {
	for {
		// must write piece by order
		// when received PieceNum is great then pieceIndex, cache it
		if p.PieceNum != csw.pieceIndex {
			if p.PieceNum < csw.pieceIndex {
				return fmt.Errorf("piece number should great than %d", csw.pieceIndex)
			}
			csw.cache[p.PieceNum] = p
			break
		}

		_, err := io.Copy(csw.pipeWriter, p.RawContent())
		if err != nil {
			return err
		}

		csw.pieceIndex++
		// next piece may be already in cache, check it
		next, ok := csw.cache[csw.pieceIndex]
		if ok {
			p = next
			continue
		}
		break
	}

	return nil
}

func (csw *ClientStreamWriter) sendSuccessPiece(piece *Piece, cost time.Duration) {
	csw.api.ReportPiece(piece.SuperNode, &types.ReportPieceRequest{
		TaskID:     piece.TaskID,
		Cid:        csw.cfg.RV.Cid,
		DstCid:     piece.DstCid,
		PieceRange: piece.Range,
	})
	if cost.Seconds() > 2.0 {
		logrus.Infof(
			"async writer and report suc from dst:%s... cost:%.3f for range:%s",
			piece.DstCid[:25], cost.Seconds(), piece.Range)
	}
}

func (csw *ClientStreamWriter) Read(p []byte) (n int, err error) {
	return csw.pipeReader.Read(p)
}
