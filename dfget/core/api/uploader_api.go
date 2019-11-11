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

package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
)

// UploaderAPI defines the communication methods between dfget and uploader.
type UploaderAPI interface {
	// ParseRate sends a request to uploader to calculate the rateLimit dynamically
	// for the speed limit of the whole host machine.
	ParseRate(ip string, port int, req *ParseRateRequest) (string, error)

	// CheckServer checks the peer server on port whether is available.
	CheckServer(ip string, port int, req *CheckServerRequest) (string, error)

	// FinishTask report a finished task to peer server.
	FinishTask(ip string, port int, req *FinishTaskRequest) error

	// PingServer send a request to determine whether the server has started.
	PingServer(ip string, port int) bool
}

// uploaderAPI is an implementation of interface UploaderAPI.
type uploaderAPI struct {
	timeout time.Duration
}

var _ UploaderAPI = &uploaderAPI{}

// NewUploaderAPI returns a new UploaderAPI.
func NewUploaderAPI(timeout time.Duration) UploaderAPI {
	return &uploaderAPI{
		timeout: timeout,
	}
}

func (u *uploaderAPI) ParseRate(ip string, port int, req *ParseRateRequest) (string, error) {
	headers := make(map[string]string)
	headers[config.StrRateLimit] = strconv.Itoa(req.RateLimit)

	url := fmt.Sprintf("http://%s:%d%s%s", ip, port, config.LocalHTTPPathRate, req.TaskFileName)
	return httputils.Do(url, headers, u.timeout)
}

func (u *uploaderAPI) CheckServer(ip string, port int, req *CheckServerRequest) (string, error) {
	headers := make(map[string]string)
	headers[config.StrDataDir] = req.DataDir
	headers[config.StrTotalLimit] = strconv.Itoa(req.TotalLimit)

	url := fmt.Sprintf("http://%s:%d%s%s", ip, port, config.LocalHTTPPathCheck, req.TaskFileName)
	return httputils.Do(url, headers, u.timeout)
}

func (u *uploaderAPI) FinishTask(ip string, port int, req *FinishTaskRequest) error {
	url := fmt.Sprintf("http://%s:%d%sfinish?"+
		config.StrTaskFileName+"=%s&"+
		config.StrTaskID+"=%s&"+
		config.StrClientID+"=%s&"+
		config.StrSuperNode+"=%s",
		ip, port, config.LocalHTTPPathClient,
		req.TaskFileName, req.TaskID, req.ClientID, req.Node)

	code, body, err := httputils.Get(url, u.timeout)
	if code == http.StatusOK {
		return nil
	}
	if err == nil {
		return fmt.Errorf("%d:%s", code, body)
	}
	return err
}

func (u *uploaderAPI) PingServer(ip string, port int) bool {
	url := fmt.Sprintf("http://%s:%d%s", ip, port, config.LocalHTTPPing)
	code, _, _ := httputils.Get(url, u.timeout)
	return code == http.StatusOK
}
