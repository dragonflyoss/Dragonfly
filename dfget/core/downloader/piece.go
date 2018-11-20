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
	"bytes"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
)

// Piece contains all information of a piece.
type Piece struct {
	TaskID    string
	SuperNode string
	DstCid    string
	Range     string
	Result    int
	Status    int
	PieceSize int32
	PieceNum  int
	Content   *bytes.Buffer
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
		Result:    config.ResultInvalid,
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
