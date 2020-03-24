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
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/rangeutils"
	"github.com/dragonflyoss/Dragonfly/version"
)

// DownloadRequest wraps the request which is sent to peer
// for downloading one piece.
type DownloadRequest struct {
	Path       string
	PieceRange string
	PieceNum   int
	PieceSize  int32
	Headers    map[string]string
}

// DownloadAPI defines the download method between dfget and peer server.
type DownloadAPI interface {
	// Download downloads a piece and returns an HTTP response.
	Download(ip string, port int, req *DownloadRequest, timeout time.Duration) (*http.Response, error)
}

// downloadAPI is an implementation of interface DownloadAPI.
type downloadAPI struct {
}

var _ DownloadAPI = &downloadAPI{}

// NewDownloadAPI returns a new DownloadAPI.
func NewDownloadAPI() DownloadAPI {
	return &downloadAPI{}
}

func (d *downloadAPI) Download(ip string, port int, req *DownloadRequest, timeout time.Duration) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("nil dwonload request")
	}
	headers := make(map[string]string)
	headers[config.StrPieceNum] = strconv.Itoa(req.PieceNum)
	headers[config.StrPieceSize] = fmt.Sprint(req.PieceSize)
	headers[config.StrUserAgent] = "dfget/" + version.DFGetVersion
	if req.Headers != nil {
		for k, v := range req.Headers {
			headers[k] = v
		}
	}

	var (
		url      string
		rangeStr string
	)
	if isFromSource(req) {
		rangeStr = getRealRange(req.PieceRange, headers[config.StrRange])
		url = req.Path
	} else {
		rangeStr = req.PieceRange
		url = fmt.Sprintf("http://%s:%d%s", ip, port, req.Path)
	}
	headers[config.StrRange] = httputils.ConstructRangeStr(rangeStr)

	return httputils.HTTPGetTimeout(url, headers, timeout)
}

func isFromSource(req *DownloadRequest) bool {
	return strings.Contains(req.Path, "://")
}

// getRealRange
// pieceRange: "start-end"
// rangeHeaderValue: "bytes=sourceStart-sourceEnd"
// return: "realStart-realEnd"
func getRealRange(pieceRange string, rangeHeaderValue string) string {
	if rangeHeaderValue == "" {
		return pieceRange
	}
	rangeEle := strings.Split(rangeHeaderValue, "=")
	if len(rangeEle) != 2 {
		return pieceRange
	}

	lower, upper, err := rangeutils.ParsePieceIndex(rangeEle[1])
	if err != nil {
		return pieceRange
	}
	start, end, err := rangeutils.ParsePieceIndex(pieceRange)
	if err != nil {
		return pieceRange
	}

	realStart := start + lower
	realEnd := end + lower
	if realEnd > upper {
		realEnd = upper
	}
	return fmt.Sprintf("%d-%d", realStart, realEnd)
}
