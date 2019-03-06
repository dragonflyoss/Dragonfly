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

	"github.com/dragonflyoss/Dragonfly/common/constants"
	"github.com/dragonflyoss/Dragonfly/common/util"
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
	Content *bytes.Buffer `json:"-"`
}

// RawContent return raw contents.
func (p *Piece) RawContent() *bytes.Buffer {
	contents := p.Content.Bytes()
	length := len(contents)
	if length >= 5 {
		return bytes.NewBuffer(contents[4 : length-1])
	}
	return nil
}

func (p *Piece) String() string {
	if b, e := json.Marshal(p); e == nil {
		return string(b)
	}
	return ""
}

// NewPiece creates a Piece.
func NewPiece(taskID, node, dstCid, pieceRange string, result, status int) *Piece {
	return &Piece{
		TaskID:    taskID,
		SuperNode: node,
		DstCid:    dstCid,
		Range:     pieceRange,
		Result:    result,
		Status:    status,
		Content:   &bytes.Buffer{},
	}
}

// NewPieceSimple creates a Piece with default value.
func NewPieceSimple(taskID string, node string, status int) *Piece {
	return &Piece{
		TaskID:    taskID,
		SuperNode: node,
		Status:    status,
		Result:    constants.ResultInvalid,
		Content:   &bytes.Buffer{},
	}
}

// NewPieceContent creates a Piece with specified content.
func NewPieceContent(taskID, node, dstCid, pieceRange string,
	result, status int, contents *bytes.Buffer) *Piece {
	if util.IsNil(contents) {
		contents = &bytes.Buffer{}
	}
	return &Piece{
		TaskID:    taskID,
		SuperNode: node,
		DstCid:    dstCid,
		Range:     pieceRange,
		Result:    result,
		Status:    status,
		Content:   contents,
	}
}
