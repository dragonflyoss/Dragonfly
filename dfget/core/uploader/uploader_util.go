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
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	errType "github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/pkg/errors"
)

// uploader helper

// getTaskFile find the taskFile and return the File object.
func (ps *peerServer) getTaskFile(taskFileName string, params *uploadParam) (*os.File, error) {
	v, ok := ps.syncTaskMap.Load(taskFileName)
	if !ok {
		return nil, fmt.Errorf("failed to get taskPath: %s", taskFileName)
	}
	tc, ok := v.(*taskConfig)
	if !ok {
		return nil, fmt.Errorf("failed to assert: %s", taskFileName)
	}

	taskPath := helper.GetServiceFile(taskFileName, tc.dataDir)

	// Pre-check the length of the file as expected.
	fileInfo, err := os.Stat(taskPath)
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()
	if fileSize-params.start < params.pieceLen {
		return nil, errors.Wrapf(errType.ErrInsufficientFileLength, "file: %s", taskPath)
	}

	taskFile, err := os.Open(taskPath)
	if err != nil {
		return nil, fmt.Errorf("file:%s not found", taskPath)
	}
	return taskFile, nil
}

// parseRange validates the parameter range and parses it
func parseRange(rangeVal string) (*uploadParam, error) {
	if strings.Count(rangeVal, "=") != 1 {
		return nil, fmt.Errorf("invaild range: %s", rangeVal)
	}
	rangeStr := strings.Split(rangeVal, "=")[1]

	if strings.Count(rangeStr, "-") != 1 {
		return nil, fmt.Errorf("invaild range: %s", rangeStr)
	}
	rangeArr := strings.Split(rangeStr, "-")
	start, err := strconv.ParseInt(rangeArr[0], 10, 64)
	if err != nil {
		return nil, err
	}
	end, err := strconv.ParseInt(rangeArr[1], 10, 64)
	if err != nil {
		return nil, err
	}

	if end <= start {
		return nil, fmt.Errorf("The end of range: %d is less than or equal to the start: %d", end, start)
	}
	pieceLen := end - start + 1

	return &uploadParam{
		start:    start,
		pieceLen: pieceLen,
	}, nil
}

// transFile send the file to the remote.
func (ps *peerServer) transFile(f *os.File, w http.ResponseWriter, start, pieceLen int64) error {
	var total int64
	f.Seek(start, 0)

	remain := pieceLen
	bufSize := int64(256 * 1024)
	buf := make([]byte, bufSize)

	for remain > 0 {
		// read len(buf) of data
		num, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if num == 0 {
			if total == 0 {
				w.Write([]byte(""))
			}
			return nil
		}

		length := int64(num)
		if length > remain {
			length = remain
		}

		if ps.rateLimiter != nil {
			ps.rateLimiter.AcquireBlocking(int32(length))
		}

		w.Write(buf[:length])

		total += length
		remain = pieceLen - total

		if num < len(buf) {
			break
		}
	}
	return nil
}

// LaunchPeerServer helper

// FinishTask report a finished task to peer server.
func FinishTask(ip string, port int, taskFileName, cid, taskID, node string) error {
	url := fmt.Sprintf("http://%s:%d%sfinish?taskFileName=%s&cid=%s&taskId=%s&node=%s",
		ip, port, config.LocalHTTPPathClient,
		taskFileName, taskID, cid, node)
	code, _, err := util.Get(url, util.DefaultTimeout)
	if code == http.StatusOK {
		return nil
	}
	return err
}

// checkServer check if the server is available。
func checkServer(ip string, port int, dataDir, taskFileName string, totalLimit int,
	timeout time.Duration) (string, error) {
	// prepare the request body
	url := fmt.Sprintf("http://%s:%d%s%s", ip, port, config.LocalHTTPPathCheck, taskFileName)
	if timeout <= 0 {
		timeout = util.DefaultTimeout
	}
	headers := make(map[string]string)
	headers["dataDir"] = dataDir
	headers["totalLimit"] = strconv.Itoa(totalLimit)

	// send the request
	result, err := util.Do(url, headers, timeout)
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

func pingServer(ip string, port int) bool {
	url := fmt.Sprintf("http://%s:%d%s", ip, port, config.LocalHTTPPing)
	code, _, _ := util.Get(url, util.DefaultTimeout)
	return code == http.StatusOK
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
