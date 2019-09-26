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
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"
)

var (
	defaultRateLimit    = 1000
	defaultPieceSize    = int64(4 * 1024 * 1024)
	defaultPieceSizeStr = fmt.Sprintf("%d", defaultPieceSize)
)

func pc(origin string) string {
	return pieceContent(defaultPieceSize, origin)
}

func pieceContent(pieceSize int64, origin string) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(int64(len(origin))|(pieceSize<<4)))
	buf := bytes.Buffer{}
	buf.Write(b)
	buf.Write([]byte(origin))
	buf.Write([]byte{config.PieceTailChar})
	return buf.String()
}

// newTestPeerServer inits the peer server for testing.
func newTestPeerServer(workHome string) (srv *peerServer) {
	cfg := helper.CreateConfig(nil, workHome)
	srv = newPeerServer(cfg, 0)
	srv.totalLimitRate = 1000
	srv.rateLimiter = ratelimiter.NewRateLimiter(int64(defaultRateLimit), 2)
	return srv
}

// initHelper creates a temporary file and store it in the syncTaskMap.
func initHelper(srv *peerServer, fileName, dataDir, content string) {
	helper.CreateTestFile(helper.GetServiceFile(fileName, dataDir), content)
	if srv != nil {
		srv.syncTaskMap.Store(fileName, &taskConfig{
			dataDir:   dataDir,
			rateLimit: defaultRateLimit,
		})
	}
}

func startTestServer(handler http.Handler) (ip string, port int, server *http.Server) {
	// run a server
	ip = "127.0.0.1"
	port = rand.Intn(1000) + 63000
	server = &http.Server{Addr: fmt.Sprintf("%s:%d", ip, port), Handler: handler}
	go server.ListenAndServe()
	return
}

func stopTestServer(server *http.Server) {
	if server == nil {
		return
	}

	c, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	server.Shutdown(c)
	cancel()
}

// ----------------------------------------------------------------------------
// handler helper

type HandlerHelper struct {
	method  string
	url     string
	body    io.Reader
	headers map[string]string
}

// ----------------------------------------------------------------------------
// upload header

var defaultUploadHeader = uploadHeader{
	rangeStr: fmt.Sprintf("bytes=0-%d", defaultPieceSize-1),
	num:      "0",
	size:     defaultPieceSizeStr,
}

type uploadHeader struct {
	rangeStr string
	num      string
	size     string
}

func (u uploadHeader) newRange(rangeStr string) uploadHeader {
	newU := u
	if !strings.HasPrefix(rangeStr, "bytes") {
		newU.rangeStr = "bytes=" + rangeStr
	} else {
		newU.rangeStr = rangeStr
	}
	return newU
}

// ----------------------------------------------------------------------------
// upload param

type uploadParamBuilder struct {
	up uploadParam
}

func (upb *uploadParamBuilder) build() *uploadParam {
	return &upb.up
}

func (upb *uploadParamBuilder) padSize(padSize int64) *uploadParamBuilder {
	upb.up.padSize = padSize
	return upb
}

func (upb *uploadParamBuilder) start(start int64) *uploadParamBuilder {
	upb.up.start = start
	return upb
}

func (upb *uploadParamBuilder) length(length int64) *uploadParamBuilder {
	upb.up.length = length
	return upb
}

func (upb *uploadParamBuilder) pieceSize(pieceSize int64) *uploadParamBuilder {
	upb.up.pieceSize = pieceSize
	return upb
}

func (upb *uploadParamBuilder) pieceNum(pieceNum int64) *uploadParamBuilder {
	upb.up.pieceNum = pieceNum
	return upb
}
