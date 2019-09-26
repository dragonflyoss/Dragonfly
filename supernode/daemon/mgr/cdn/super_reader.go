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

package cdn

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"hash"
	"io"

	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type cdnCacheResult struct {
	pieceCount int
	fileLength int64
	pieceMd5s  []string
	fileMd5    hash.Hash
}

type superReader struct{}

func newSuperReader() *superReader {
	return &superReader{}
}

func (sr *superReader) readFile(ctx context.Context, reader io.Reader, calculatePieceMd5, calculateFileMd5 bool) (result *cdnCacheResult, err error) {
	result = &cdnCacheResult{}

	var pieceMd5 hash.Hash
	if calculatePieceMd5 {
		pieceMd5 = md5.New()
	}
	if calculateFileMd5 {
		result.fileMd5 = md5.New()
	}

	for {
		// read header and get piece content length
		ret, err := readHeader(reader, pieceMd5)
		if err != nil {
			if err == io.EOF {
				return result, nil
			}

			return result, errors.Wrapf(err, "failed to read header for count %d", result.pieceCount+1)
		}
		result.fileLength += config.PieceHeadSize
		pieceLen := getContentLengthByHeader(ret)
		logrus.Debugf("get piece length: %d with count: %d from header", pieceLen, result.pieceCount)

		// read content
		if err := readContent(reader, pieceLen, pieceMd5, result.fileMd5); err != nil {
			logrus.Errorf("failed to read content for count %d: %v", result.pieceCount, err)
			return result, err
		}
		result.fileLength += int64(pieceLen)

		// read tailer
		if err := readTailer(reader, pieceMd5); err != nil {
			return result, errors.Wrapf(err, "failed to read tailer for count %d", result.pieceCount)
		}
		result.fileLength++

		result.pieceCount++

		if calculatePieceMd5 {
			pieceSum := fileutils.GetMd5Sum(pieceMd5, nil)
			pieceLength := pieceLen + config.PieceWrapSize
			result.pieceMd5s = append(result.pieceMd5s, getPieceMd5Value(pieceSum, pieceLength))
			pieceMd5.Reset()
		}
	}
}

func readHeader(reader io.Reader, pieceMd5 hash.Hash) (uint32, error) {
	header := make([]byte, 4)

	n, err := reader.Read(header)
	if err != nil {
		return 0, err
	}
	if n != config.PieceHeadSize {
		return 0, fmt.Errorf("unexpected head size: %d", n)
	}

	if pieceMd5 != nil {
		pieceMd5.Write(header)
	}

	return binary.BigEndian.Uint32(header), nil
}

func readContent(reader io.Reader, pieceLen int32, pieceMd5 hash.Hash, fileMd5 hash.Hash) error {
	bufSize := int32(256 * 1024)
	if pieceLen < bufSize {
		bufSize = pieceLen
	}
	pieceContent := make([]byte, bufSize)
	var curContent int32

	for {
		if curContent+bufSize <= pieceLen {
			if err := binary.Read(reader, binary.BigEndian, pieceContent); err != nil {
				return err
			}
			curContent += bufSize

			// calculate the md5
			if !util.IsNil(pieceMd5) {
				pieceMd5.Write(pieceContent)
			}
			if !util.IsNil(fileMd5) {
				fileMd5.Write(pieceContent)
			}
		} else {
			readLen := pieceLen - curContent
			if err := binary.Read(reader, binary.BigEndian, pieceContent[:readLen]); err != nil {
				return err
			}
			curContent += readLen

			// calculate the md5
			if !util.IsNil(pieceMd5) {
				pieceMd5.Write(pieceContent[:readLen])
			}
			if !util.IsNil(fileMd5) {
				fileMd5.Write(pieceContent[:readLen])
			}
		}

		if curContent >= pieceLen {
			break
		}
	}

	return nil
}

func readTailer(reader io.Reader, pieceMd5 hash.Hash) error {
	tailer := make([]byte, 1)
	if err := binary.Read(reader, binary.BigEndian, tailer); err != nil {
		return err
	}
	if tailer[0] != config.PieceTailChar {
		return fmt.Errorf("unexpected tailer: %v", tailer[0])
	}

	if !util.IsNil(pieceMd5) {
		pieceMd5.Write(tailer)
	}
	return nil
}

func getMD5ByReadFile(reader io.Reader, pieceLen int32) (string, error) {
	if pieceLen <= 0 {
		return fileutils.GetMd5Sum(md5.New(), nil), nil
	}

	pieceMd5 := md5.New()
	if err := readContent(reader, pieceLen, pieceMd5, nil); err != nil {
		return "", err
	}

	return fileutils.GetMd5Sum(pieceMd5, nil), nil
}
