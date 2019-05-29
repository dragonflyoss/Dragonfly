package cdn

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/sirupsen/logrus"
)

type protocolContent struct {
	taskID           string
	pieceNum         int
	pieceSize        int32
	pieceContentSize int32
	pieceContent     *bytes.Buffer
}

type superWriter struct {
	cdnStore    *store.Store
	cdnReporter *reporter
}

func newSuperWriter(cdnStore *store.Store, cdnReporter *reporter) *superWriter {
	return &superWriter{
		cdnStore:    cdnStore,
		cdnReporter: cdnReporter,
	}
}

// startWriter writes the stream data from the reader to the underlying storage.
func (cw *superWriter) startWriter(ctx context.Context, cfg *config.Config, reader io.Reader,
	task *types.TaskInfo, startPieceNum int, httpFileLength int64, pieceContSize int32) (int64, error) {
	// realFileLength is used to caculate the file Length dynamically
	realFileLength := int64(startPieceNum) * int64(pieceContSize)
	// the left size of data for a complete piece
	pieceContLeft := pieceContSize
	// the pieceNum currently processed
	curPieceNum := startPieceNum

	buf := make([]byte, pieceContSize)
	var bb = &bytes.Buffer{}

	// start writer pool
	routineCount := calculateRoutineCount(httpFileLength, task.PieceSize)
	var wg = &sync.WaitGroup{}
	jobCh := make(chan *protocolContent)
	cw.writerPool(ctx, wg, routineCount, jobCh)

	for {
		n, e := reader.Read(buf)
		if n > 0 {
			logrus.Debugf("success to read content with length: %d", n)
			realFileLength += int64(n)
			if int(pieceContLeft) <= n {
				bb.Write(buf[:pieceContLeft])
				pc := &protocolContent{
					taskID:           task.ID,
					pieceNum:         curPieceNum,
					pieceSize:        task.PieceSize,
					pieceContentSize: pieceContSize,
					pieceContent:     bb,
				}
				jobCh <- pc
				logrus.Debugf("send the protocolContent taskID: %s pieceNum: %d", task.ID, curPieceNum)

				realFileLength += config.PieceWrapSize
				curPieceNum++

				// write the data left to a new buffer
				// TODO: recycling bytes.Buffer
				bb = bytes.NewBuffer([]byte{})
				n -= int(pieceContLeft)
				if n > 0 {
					bb.Write(buf[pieceContLeft:])
				}
				pieceContLeft = pieceContSize
			} else {
				bb.Write(buf[:n])
			}
			pieceContLeft -= int32(n)
		}

		if e == io.EOF {
			if realFileLength == 0 || bb.Len() > 0 {
				jobCh <- &protocolContent{
					taskID:           task.ID,
					pieceNum:         curPieceNum,
					pieceSize:        task.PieceSize,
					pieceContentSize: int32(bb.Len()),
					pieceContent:     bb,
				}
				logrus.Debugf("send the protocolContent taskID: %s pieceNum: %d", task.ID, curPieceNum)

				realFileLength += config.PieceWrapSize
			}
			logrus.Infof("send all protocolContents with realFileLength(%d) and wait for superwriter", realFileLength)
			break
		}
		if e != nil {
			close(jobCh)
			return 0, e
		}
	}

	close(jobCh)
	wg.Wait()
	return realFileLength, nil
}
