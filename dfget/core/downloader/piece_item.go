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
)

// PieceItem contains all information of a piece.
type PieceItem struct {
	DstCid        string
	SrcCid        string
	Range         string
	TaskID        string
	SuperNode     string
	Result        int
	Status        int
	PieceSize     int32
	PieceNum      int
	PieceContents bytes.Buffer
}

// RawContents return raw contents.
func (p *PieceItem) RawContents() *bytes.Buffer {
	contents := p.PieceContents.Bytes()
	length := len(contents)
	if length >= 5 {
		return bytes.NewBuffer(contents[4 : length-1])
	}
	return nil
}
