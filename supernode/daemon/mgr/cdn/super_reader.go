package cdn

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"hash"
	"io"

	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"

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
		// read header and get piece content legth
		ret, err := readHeader(ctx, reader, pieceMd5)
		if err != nil {
			if err == io.EOF {
				return result, nil
			}

			logrus.Errorf("failed to read header for count %d: %v", result.pieceCount+1, err)
			return result, err
		}
		result.fileLength += config.PieceHeadSize
		pieceLen := getContentLengthByHeader(ret)
		logrus.Infof("get piece length: %d with count: %d from header", pieceLen, result.pieceCount)

		// read content
		if err := readContent(ctx, reader, pieceLen, pieceMd5, result.fileMd5); err != nil {
			logrus.Errorf("failed to read content for count %d: %v", result.pieceCount, err)
			return result, err
		}
		result.fileLength += int64(pieceLen)

		// read tailer
		if err := readTailer(ctx, reader, pieceMd5); err != nil {
			logrus.Errorf("failed to read tailer for count %d: %v", result.pieceCount, err)
			return result, err
		}
		result.fileLength++

		result.pieceCount++

		if calculatePieceMd5 {
			result.pieceMd5s = append(result.pieceMd5s, fmt.Sprintf("%x", pieceMd5.Sum(nil)))
			pieceMd5.Reset()
		}
	}
}

func readHeader(ctx context.Context, reader io.Reader, pieceMd5 hash.Hash) (uint32, error) {
	header := make([]byte, 4)

	n, err := reader.Read(header)
	if err != nil {
		return 0, err
	}
	if n != config.PieceHeadSize {
		return 0, fmt.Errorf("unexected head size: %d", n)
	}

	if !cutil.IsNil(pieceMd5) {
		pieceMd5.Write(header)
	}

	return binary.BigEndian.Uint32(header), nil
}

func readContent(ctx context.Context, reader io.Reader, pieceLen int32, pieceMd5 hash.Hash, fileMd5 hash.Hash) error {
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
			if !cutil.IsNil(pieceMd5) {
				pieceMd5.Write(pieceContent)
			}
			if !cutil.IsNil(fileMd5) {
				fileMd5.Write(pieceContent)
			}
		} else {
			readLen := pieceLen - curContent
			if err := binary.Read(reader, binary.BigEndian, pieceContent[:readLen]); err != nil {
				return err
			}
			curContent += readLen

			// calculate the md5
			if !cutil.IsNil(pieceMd5) {
				pieceMd5.Write(pieceContent[:readLen])
			}
			if !cutil.IsNil(fileMd5) {
				fileMd5.Write(pieceContent[:readLen])
			}
		}

		if curContent >= pieceLen {
			break
		}
	}

	return nil
}

func readTailer(ctx context.Context, reader io.Reader, pieceMd5 hash.Hash) error {
	tailer := make([]byte, 1)
	if err := binary.Read(reader, binary.BigEndian, tailer); err != nil {
		return err
	}
	if tailer[0] != config.PieceTailChar {
		return fmt.Errorf("unexpected tailer: %v", tailer[0])
	}

	if !cutil.IsNil(pieceMd5) {
		pieceMd5.Write(tailer)
	}
	return nil
}
