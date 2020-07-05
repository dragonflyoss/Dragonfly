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

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/pkg/limitreader"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/sirupsen/logrus"
)

// ClientWriter writes a file for uploading and a target file.
type ClientStreamWriter struct {
	// clientQueue maintains a queue of tasks that need to be written to disk.
	// The downloader will put the piece into this queue after it downloaded a piece successfully.
	// And clientWriter will poll values from this queue constantly and write to disk.
	clientQueue queue.Queue

	// notifyQueue sends a notification when all operation about a piece have
	// been completed successfully.
	notifyQueue queue.Queue

	// finish indicates whether the task written is completed.
	finish chan struct{}

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

	// limitReader supports limit rate and calculates md5
	limitReader *limitreader.LimitReader

	cache map[int]*Piece

	// api holds an instance of SupernodeAPI to interact with supernode.
	api api.SupernodeAPI
	cfg *config.Config

	cdnSource apiTypes.CdnSource
}

// NewClientStreamWriter creates and initialize a ClientStreamWriter instance.
func NewClientStreamWriter(clientQueue, notifyQueue queue.Queue, api api.SupernodeAPI, cfg *config.Config) *ClientStreamWriter {
	pr, pw := io.Pipe()
	limitReader := limitreader.NewLimitReader(pr, int64(cfg.LocalLimit), cfg.Md5 != "")
	clientWriter := &ClientStreamWriter{
		clientQueue: clientQueue,
		notifyQueue: notifyQueue,
		pipeReader:  pr,
		pipeWriter:  pw,
		limitReader: limitReader,
		api:         api,
		cfg:         cfg,
		cache:       make(map[int]*Piece),
	}
	return clientWriter
}

func (csw *ClientStreamWriter) PreRun(ctx context.Context) (err error) {
	csw.p2pPattern = helper.IsP2P(csw.cfg.Pattern)
	csw.result = true
	csw.finish = make(chan struct{})
	return
}

func (csw *ClientStreamWriter) PostRun(ctx context.Context) (err error) {
	return nil
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
		go sendSuccessPiece(csw.api, csw.cfg.RV.Cid, piece, time.Since(startTime), csw.notifyQueue)
	}
	return err
}

func (csw *ClientStreamWriter) writePieceToPipe(p *Piece) error {
	for {
		// must write piece by order
		// when received PieceNum is greater then pieceIndex, cache it
		if p.PieceNum != csw.pieceIndex {
			if p.PieceNum < csw.pieceIndex {
				logrus.Warnf("piece number should be greater than %d, received piece number: %d",
					csw.pieceIndex, p.PieceNum)
				break
			}
			csw.cache[p.PieceNum] = p
			break
		}

		_, err := p.WriteTo(csw.pipeWriter, csw.cdnSource == apiTypes.CdnSourceSource)
		if err != nil {
			return err
		}

		csw.pieceIndex++
		// next piece may be already in cache, check it
		next, ok := csw.cache[csw.pieceIndex]
		if ok {
			p = next
			delete(csw.cache, csw.pieceIndex)
			continue
		}
		break
	}

	return nil
}

func (csw *ClientStreamWriter) Read(p []byte) (n int, err error) {
	n, err = csw.limitReader.Read(p)
	// all data received, calculate md5
	if err == io.EOF && csw.cfg.Md5 != "" {
		realMd5 := csw.limitReader.Md5()
		if realMd5 != csw.cfg.Md5 {
			return n, fmt.Errorf("md5 not match, expected: %s real: %s", csw.cfg.Md5, realMd5)
		}
	}
	return n, err
}
