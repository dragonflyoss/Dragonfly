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
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/pool"
)

// Piece contains all information of a piece.
type Piece struct {
	// TaskID a string which represents a unique task.
	TaskID string `json:"taskID"`

	// SuperNode indicates the IP address of the currently registered supernode.
	SuperNode string `json:"superNode"`

	// DstCid means the clientID of the target peer node for downloading the piece.
	DstCid string `json:"dstCid"`

	// Range indicates the range of specific piece in the task, example "0-45565".
	Range string `json:"range"`

	// Result of the piece downloaded.
	Result int `json:"result"`

	// Status of the downloading task.
	Status int `json:"status"`

	// PieceSize the length of the piece.
	PieceSize int32 `json:"pieceSize"`

	// PieceNum represents the position of the piece in the pieces list by cutting files.
	PieceNum int `json:"pieceNum"`

	// Content uses a buffer to temporarily store the piece content.
	Content *pool.Buffer `json:"-"`

	// length the length of the content.
	length int64

	// writerNum record the writer number which will write this piece.
	writerNum int32
}

// WriteTo writes piece raw data in content buffer to w.
// If the piece has wrapper, the piece content will remove the head and tail before writing.
func (p *Piece) WriteTo(w io.Writer, noWrapper bool) (n int64, err error) {
	defer p.TryResetContent()

	content := p.RawContent(noWrapper)
	if content != nil {
		return content.WriteTo(w)
	}
	return 0, fmt.Errorf("piece content length less than 5 bytes")
}

// IncWriter increase a writer for the piece.
func (p *Piece) IncWriter() {
	atomic.AddInt32(&p.writerNum, 1)
}

// RawContent returns raw contents,
// If the piece has wrapper, and the piece content will remove the head and tail.
func (p *Piece) RawContent(noWrapper bool) *bytes.Buffer {
	contents := p.Content.Bytes()
	length := len(contents)

	if noWrapper {
		return bytes.NewBuffer(contents[:])
	}
	if length >= 5 {
		return bytes.NewBuffer(contents[4 : length-1])
	}
	return nil
}

// ContentLength returns the content length.
func (p *Piece) ContentLength() int64 {
	if p.length <= 0 && p.Content != nil {
		p.length = int64(p.Content.Len())
	}
	return p.length
}

func (p *Piece) String() string {
	if b, e := json.Marshal(p); e == nil {
		return string(b)
	}
	return ""
}

// ResetContent reset contents and returns it back to buffer pool.
func (p *Piece) TryResetContent() {
	if atomic.AddInt32(&p.writerNum, -1) > 0 {
		return
	}

	if p.Content == nil {
		return
	}
	if p.length == 0 {
		p.length = int64(p.Content.Len())
	}
	pool.ReleaseBuffer(p.Content)
	p.Content = nil
}

// NewPiece creates a Piece.
func NewPiece(taskID, node, dstCid, pieceRange string, result, status int, cdnSource apiTypes.CdnSource) *Piece {
	return &Piece{
		TaskID:    taskID,
		SuperNode: node,
		DstCid:    dstCid,
		Range:     pieceRange,
		Result:    result,
		Status:    status,
		Content:   nil,
		writerNum: 1,
	}
}

// NewPieceSimple creates a Piece with default value.
func NewPieceSimple(taskID string, node string, status int, cdnSource apiTypes.CdnSource) *Piece {
	return &Piece{
		TaskID:    taskID,
		SuperNode: node,
		Status:    status,
		Result:    constants.ResultInvalid,
		Content:   nil,
		writerNum: 1,
	}
}

// NewPieceContent creates a Piece with specified content.
func NewPieceContent(taskID, node, dstCid, pieceRange string,
	result, status int, contents *pool.Buffer, cdnSource apiTypes.CdnSource) *Piece {
	return &Piece{
		TaskID:    taskID,
		SuperNode: node,
		DstCid:    dstCid,
		Range:     pieceRange,
		Result:    result,
		Status:    status,
		Content:   contents,
		length:    int64(contents.Len()),
		writerNum: 1,
	}
}
