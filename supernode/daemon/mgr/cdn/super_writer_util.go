package cdn

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"hash"
	"sync"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/sirupsen/logrus"
)

func calculateRoutineCount(httpFileLength int64, pieceSize int32) int {
	routineSize := config.CDNWriterRoutineLimit
	if httpFileLength < 0 || pieceSize <= 0 {
		return routineSize
	}

	if httpFileLength == 0 {
		return 1
	}

	pieceContSize := pieceSize - config.PieceWrapSize
	tmpSize := (int)((httpFileLength + int64(pieceContSize-1)) / int64(pieceContSize))
	if tmpSize == 0 {
		tmpSize = 1
	}
	if tmpSize < routineSize {
		routineSize = tmpSize
	}
	return routineSize
}

func (cw *superWriter) writerPool(ctx context.Context, wg *sync.WaitGroup, n int, jobCh chan *protocolContent) {
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			var pieceMd5 = md5.New()
			for job := range jobCh {
				if err := cw.writeToFile(ctx, job.pieceContent, job.taskID, job.pieceNum, job.pieceContentSize, job.pieceSize, pieceMd5); err != nil {
					logrus.Errorf("failed to write taskID %s pieceNum %d file: %v", job.taskID, job.pieceNum, err)
					// NOTE: should we redo the job?
					continue
				}

				// report piece status
				pieceMd5Value := fmt.Sprintf("%x", pieceMd5.Sum(nil))
				if cw.cdnReporter != nil {
					if err := cw.cdnReporter.reportPieceStatus(ctx, job.taskID, job.pieceNum, pieceMd5Value, config.PieceSUCCESS); err != nil {
						// NOTE: should we do this job again?
						logrus.Errorf("failed to report piece status taskID %s pieceNum %d pieceMD5 %s: %v", job.taskID, job.pieceNum, pieceMd5Value, err)
						continue
					}
				}
			}
			wg.Done()
		}(i)
	}
}

// writeToFile wraps the piece content with piece header and tailer,
// and then writes to the storage.
func (cw *superWriter) writeToFile(ctx context.Context, bytesBuffer *bytes.Buffer, taskID string, pieceNum int, pieceContSize, PieceSize int32, pieceMd5 hash.Hash) error {
	bufferLength := bytesBuffer.Len()
	if bufferLength < 0 {
		return nil
	}

	var resultBuf = &bytes.Buffer{}

	// write piece header
	var header = make([]byte, 4)
	binary.BigEndian.PutUint32(header, getPieceHeader(pieceContSize, PieceSize))
	resultBuf.Write(header)

	// write piece content
	var pieceContent []byte
	if bufferLength > 0 {
		pieceContent = make([]byte, bufferLength)
		if _, err := bytesBuffer.Read(pieceContent); err != nil {
			return err
		}
		bytesBuffer.Reset()
		binary.Write(resultBuf, binary.BigEndian, pieceContent)
	}

	// write piece tailer
	tailer := []byte{config.PieceTailChar}
	binary.Write(resultBuf, binary.BigEndian, tailer)

	if pieceMd5 != nil {
		pieceMd5.Write(header)
		if len(pieceContent) > 0 {
			pieceMd5.Write(pieceContent)
		}
		pieceMd5.Write(tailer)
	}
	// write to the storage
	return cw.cdnStore.Put(ctx, &store.Raw{
		Bucket: config.DownloadHome,
		Key:    getDownloadKey(taskID),
		Offset: int64(pieceNum) * int64(PieceSize),
	}, resultBuf)
}
