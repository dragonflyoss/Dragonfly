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
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/version"
)

// FinishTask reports a finished task to peer server.
func FinishTask(ip string, port int, taskFileName, cid, taskID, node string) error {
	req := &api.FinishTaskRequest{
		TaskFileName: taskFileName,
		TaskID:       taskID,
		ClientID:     cid,
		Node:         node,
	}

	return uploaderAPI.FinishTask(ip, port, req)
}

// checkServer checks if the server is available.
func checkServer(ip string, port int, dataDir, taskFileName string, totalLimit int) (string, error) {

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

func updateServicePortInMeta(metaPath string, port int) {
	meta := config.NewMetaData(metaPath)
	meta.Load()
	if meta.ServicePort != port {
		meta.ServicePort = port
		meta.Persist()
	}
}
