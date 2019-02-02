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
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/sirupsen/logrus"
)

// uploader helper

// getTaskFile find the taskFile and return the File object.
func (ps *peerServer) getTaskFile(taskFileName string) (*os.File, int64, error) {
	errSize := int64(-1)

	v, ok := ps.syncTaskMap.Load(taskFileName)
	if !ok {
		return nil, errSize, fmt.Errorf("failed to get taskPath: %s", taskFileName)
	}
	tc, ok := v.(*taskConfig)
	if !ok {
		return nil, errSize, fmt.Errorf("failed to assert: %s", taskFileName)
	}

	taskPath := helper.GetServiceFile(taskFileName, tc.dataDir)

	fileInfo, err := os.Stat(taskPath)
	if err != nil {
		return nil, errSize, err
	}

	taskFile, err := os.Open(taskPath)
	if err != nil {
		return nil, errSize, err
	}
	return taskFile, fileInfo.Size(), nil
}

func amendRange(size int64, needPad bool, up *uploadParam) error {
	if needPad {
		up.padSize = config.PieceMetaSize
		// we must send an whole piece with both piece head and tail
		if up.length < up.padSize {
			return errors.ErrRangeNotSatisfiable
		}
		up.start -= up.pieceNum * up.padSize
		up.end = up.start + (up.length - up.padSize) - 1
	}

	if up.start >= size && !needPad {
		return errors.ErrRangeNotSatisfiable
	}

	if up.end >= size {
		up.end = size - 1
		up.length = size - up.start + up.padSize
		if size == 0 {
			up.length = up.padSize
		}
	}

	return nil
}

// parseParams validates the parameter range and parses it
func parseParams(rangeVal, pieceNumStr, pieceSizeStr string) (*uploadParam, error) {
	var (
		err error
		up  = &uploadParam{}
	)

	if up.pieceNum, err = strconv.ParseInt(pieceNumStr, 10, 64); err != nil {
		return nil, err
	}

	if up.pieceSize, err = strconv.ParseInt(pieceSizeStr, 10, 64); err != nil {
		return nil, err
	}

	if strings.Count(rangeVal, "=") != 1 {
		return nil, fmt.Errorf("invaild range: %s", rangeVal)
	}
	rangeStr := strings.Split(rangeVal, "=")[1]

	if strings.Count(rangeStr, "-") != 1 {
		return nil, fmt.Errorf("invaild range: %s", rangeStr)
	}
	rangeArr := strings.Split(rangeStr, "-")
	if up.start, err = strconv.ParseInt(rangeArr[0], 10, 64); err != nil {
		return nil, err
	}
	if up.end, err = strconv.ParseInt(rangeArr[1], 10, 64); err != nil {
		return nil, err
	}

	if up.end <= up.start {
		return nil, fmt.Errorf("invalid range: %s", rangeStr)
	}
	up.length = up.end - up.start + 1
	return up, nil
}

// uploadPiece send a piece of the file to the remote peer.
func (ps *peerServer) uploadPiece(f *os.File, w http.ResponseWriter, up *uploadParam) error {
	w.Header().Set(config.StrContentLength, strconv.FormatInt(up.length, 10))
	sendHeader(w, http.StatusPartialContent)

	f.Seek(up.start, 0)

	remain := up.length - up.padSize
	buf := make([]byte, 256*1024)

	if up.padSize > 0 {
		binary.BigEndian.PutUint32(buf, uint32((remain)|(up.pieceSize)<<4))
		w.Write(buf[:config.PieceHeadSize])
		defer w.Write([]byte{config.PieceTailChar})
	}

	for remain > 0 {
		// read len(buf) of data
		num, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if num == 0 {
			logrus.Warnf("empty range:%s-%s of file:%s",
				up.start, up.end, f.Name())
			break
		}

		length := int64(num)
		if length > remain {
			length = remain
		}

		if ps.rateLimiter != nil {
			ps.rateLimiter.AcquireBlocking(int32(length))
		}

		w.Write(buf[:length])
		remain -= length

		if num < len(buf) {
			break
		}
	}

	return nil
}

// LaunchPeerServer helper

// FinishTask report a finished task to peer server.
func FinishTask(ip string, port int, taskFileName, cid, taskID, node string) error {
	req := &api.FinishTaskRequest{
		TaskFileName: taskFileName,
		TaskID:       taskID,
		ClientID:     cid,
		Node:         node,
	}

	return uploaderAPI.FinishTask(ip, port, req)
}

// checkServer check if the server is available。
func checkServer(ip string, port int, dataDir, taskFileName string, totalLimit int,
	timeout time.Duration) (string, error) {

	// prepare the request body
	req := &api.CheckServerRequest{
		TaskFileName: taskFileName,
		TotalLimit:   totalLimit,
		DataDir:      dataDir,
	}

	// send the request
	result, err := uploaderAPI.CheckServer(ip, port, req)
	if err != nil {
		return "", err
	}

	// parse resp result
	resultSuffix := "@" + version.DFGetVersion
	if strings.HasSuffix(result, resultSuffix) {
		return result[:len(result)-len(resultSuffix)], nil
	}
	return "", nil
}

func generatePort(inc int) int {
	lowerLimit := config.ServerPortLowerLimit
	upperLimit := config.ServerPortUpperLimit
	return int(time.Now().Unix()/300)%(upperLimit-lowerLimit) + lowerLimit + inc
}

func getPortFromMeta(metaPath string) int {
	meta := config.NewMetaData(metaPath)
	if err := meta.Load(); err != nil {
		return 0
	}
	return meta.ServicePort
}
