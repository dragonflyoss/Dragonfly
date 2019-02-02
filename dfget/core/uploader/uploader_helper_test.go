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

package uploader

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
)

var (
	defaultPieceSize    = int64(4 * 1024 * 1024)
	defaultPieceSizeStr = fmt.Sprintf("%d", defaultPieceSize)
)

func pc(origin string) string {
	return pieceContent(defaultPieceSize, origin)
}

func pieceContent(pieceSize int64, origin string) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(int64(len(origin))|(pieceSize<<4)))
	buf := bytes.Buffer{}
	buf.Write(b)
	buf.Write([]byte(origin))
	buf.Write([]byte{config.PieceTailChar})
	return buf.String()
}

// ----------------------------------------------------------------------------
// upload header

var defaultUploadHeader = uploadHeader{
	rangeStr: fmt.Sprintf("bytes=0-%d", defaultPieceSize-1),
	num:      "0",
	size:     defaultPieceSizeStr,
}

type uploadHeader struct {
	rangeStr string
	num      string
	size     string
}

func (u uploadHeader) newRange(rangeStr string) uploadHeader {
	newU := u
	if !strings.HasPrefix(rangeStr, "bytes") {
		newU.rangeStr = "bytes=" + rangeStr
	} else {
		newU.rangeStr = rangeStr
	}
	return newU
}

func (u uploadHeader) newNum(num int) uploadHeader {
	newU := u
	newU.num = fmt.Sprintf("%d", num)
	return newU
}

func (u uploadHeader) newSize(size int) uploadHeader {
	newU := u
	newU.size = fmt.Sprintf("%d", size)
	return newU
}
