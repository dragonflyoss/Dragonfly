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

	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/version"
)

// DownloadRequest wraps the request which is sent to peer
// for downloading one piece.
type DownloadRequest struct {
	Path       string
	PieceRange string
	PieceNum   int
	PieceSize  int32
}

// DownloadAPI defines the download method between dfget and peer server.
type DownloadAPI interface {
	// Download downloads a piece and returns an HTTP response.
	Download(ip string, port int, req *DownloadRequest) (*http.Response, error)
}

// downloadAPI is an implementation of interface DownloadAPI.
type downloadAPI struct {
}

var _ DownloadAPI = &downloadAPI{}

// NewDownloadAPI returns a new DownloadAPI.
func NewDownloadAPI() DownloadAPI {
	return &downloadAPI{}
}

func (d *downloadAPI) Download(ip string, port int, req *DownloadRequest) (*http.Response, error) {
	headers := make(map[string]string)
	headers[config.StrRange] = config.StrBytes + "=" + req.PieceRange
	headers[config.StrPieceNum] = strconv.Itoa(req.PieceNum)
	headers[config.StrPieceSize] = fmt.Sprint(req.PieceSize)
	headers[config.StrUserAgent] = "dfget/" + version.DFGetVersion

	url := fmt.Sprintf("http://%s:%d%s", ip, port, req.Path)
	return util.HTTPGetWithHeaders(url, headers)
}
