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

package seed

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"

	api_types "github.com/dragonflyoss/Dragonfly/apis/types"
	dfCfg "github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/api"
	"github.com/dragonflyoss/Dragonfly/dfget/local/seed"
	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"

	"github.com/go-check/check"
	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
)

type mockPeerServer struct {
	server *http.Server
	port   int

	fs     *helper.MockFileServer
	ctx    context.Context
	cancel func()
	fsPort int
}

func newMockPeerServer() *mockPeerServer {
	ctx, cancel := context.WithCancel(context.Background())
	ps := &mockPeerServer{
		fs:     helper.NewMockFileServer(),
		ctx:    ctx,
		cancel: cancel,
	}

	ps.fsPort = ps.foundTcpListenPort()
	ps.port = ps.foundTcpListenPort()

	r := ps.initRouter()
	ps.server = &http.Server{
		Addr:    net.JoinHostPort("", fmt.Sprintf("%d", ps.port)),
		Handler: r,
	}

	return ps
}

func (ps *mockPeerServer) foundTcpListenPort() int {
	for i := 0; i < 1000; i++ {
		port := rand.Intn(1000+10) + 62000
		conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			conn.Close()
			continue
		}

		return port
	}

	panic("not found")
}

func (ps *mockPeerServer) start() error {
	if err := ps.fs.StartServer(ps.ctx, ps.fsPort); err != nil {
		return err
	}

	return ps.server.ListenAndServe()
}

func (ps *mockPeerServer) shutdown() error {
	if ps.cancel != nil {
		ps.cancel()
	}
	return ps.server.Shutdown(context.Background())
}

func (ps *mockPeerServer) registerFile(path string, size int64, repeatStr string) error {
	return ps.fs.RegisterFile(path, size, repeatStr)
}

func (ps *mockPeerServer) initRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc(config.PeerHTTPPathPrefix+"{commonFile:.*}", ps.uploadHandler).Methods("GET")
	return r
}

func (ps *mockPeerServer) uploadHandler(rw http.ResponseWriter, r *http.Request) {
	taskFileName := mux.Vars(r)["commonFile"]
	ps.proxyFile(rw, r, taskFileName)
}

func (ps *mockPeerServer) proxyFile(rw http.ResponseWriter, r *http.Request, path string) {
	proxyReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/%s", ps.fsPort, path), nil)
	if err != nil {
		sendResponse(rw, http.StatusInternalServerError, err.Error())
		return
	}

	hd := CopyHeader(r.Header)
	for k, vs := range hd {
		for _, v := range vs {
			proxyReq.Header.Add(k, v)
		}
	}

	res, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		sendResponse(rw, http.StatusInternalServerError, err.Error())
		return
	}

	defer res.Body.Close()

	for k, vs := range res.Header {
		for _, v := range vs {
			rw.Header().Add(k, v)
		}
	}

	rw.WriteHeader(res.StatusCode)
	io.Copy(rw, res.Body)
}

func sendResponse(rw http.ResponseWriter, code int, msg string) {
	rw.WriteHeader(code)
	rw.Write([]byte(msg))
}

type mockBufferWriterAt struct {
	buf *bytes.Buffer
}

func newMockBufferWriterAt() *mockBufferWriterAt {
	return &mockBufferWriterAt{
		buf: bytes.NewBuffer(nil),
	}
}

func (mb *mockBufferWriterAt) WriteAt(p []byte, off int64) (n int, err error) {
	if off != int64(mb.buf.Len()) {
		return 0, fmt.Errorf("failed to seek to %d", off)
	}

	return mb.buf.Write(p)
}

func (mb *mockBufferWriterAt) Bytes() []byte {
	return mb.buf.Bytes()
}

func (suite *seedSuite) readFromFileServer(path string, host string, off int64, size int64) ([]byte, error) {
	url := fmt.Sprintf("http://%s/%s", host, path)
	header := map[string]string{}

	if size > 0 {
		header["Range"] = fmt.Sprintf("bytes=%d-%d", off, off+size-1)
	}

	code, data, err := httputils.GetWithHeaders(url, header, 5*time.Second)
	if err != nil {
		return nil, err
	}

	if code >= 400 {
		return nil, fmt.Errorf("resp code %d", code)
	}

	return data, nil
}

func (suite *seedSuite) checkLocalDownloadDataFromFileServer(c *check.C, ld seed.Downloader, host string, path string, off int64, size int64) {
	buf := newMockBufferWriterAt()

	length, err := ld.DownloadToWriterAt(context.Background(), httputils.RangeStruct{StartIndex: off, EndIndex: off + size - 1}, 0, 0, buf, true)
	c.Check(err, check.IsNil)
	c.Check(size, check.Equals, length)

	expectData, err := suite.readFromFileServer(path, host, off, size)
	c.Check(err, check.IsNil)
	c.Check(string(buf.Bytes()), check.Equals, string(expectData))
}

func (suite *seedSuite) TestPeerDownloader(c *check.C) {
	ps := newMockPeerServer()
	go ps.start()

	ps.registerFile("fileA", 500*1024, "abcde01234")
	ps.registerFile("fileB", 10*1024*1024, "11111abcde")

	files := []string{"fileA", "fileB"}
	fileLens := []int64{10 * 1024 * 1024, 40 * 1024 * 1024}

	port := ps.port
	host := "0.0.0.0"

	fsHost := fmt.Sprintf("127.0.0.1:%d", ps.fsPort)

	mss := &mockSupernodeSet{
		supernodes: map[string]*mockSupernode{},
		enableMap:  map[string]struct{}{},
	}
	supernodes := []string{"1.1.1.1:8002"}
	for _, s := range supernodes {
		mss.addSupernode(s)
	}

	nodes := []dfCfg.DFGetCommonConfig{
		{
			Cid:  "local",
			IP:   "127.0.0.1",
			Port: 40901,
		},
		{
			Cid:  "remote1",
			IP:   host,
			Port: port,
		},
	}

	tasks := []*api_types.TaskInfo{
		{
			ID:         "task1",
			TaskURL:    "http://task1",
			FileLength: fileLens[0],
			AsSeed:     true,
		},
		{
			ID:         "task2",
			TaskURL:    "http://task2",
			FileLength: fileLens[1],
			AsSeed:     true,
		},
	}

	sp := mss.getSupernode(supernodes[0])
	c.Assert(sp, check.NotNil)
	sp.addNode(nodes[0].Cid, nodes[0].IP, nodes[0].Port)
	sp.addNode(nodes[1].Cid, nodes[1].IP, nodes[1].Port)

	sp.addTask(tasks[0], nodes[0].Cid, files[0])
	sp.addTask(tasks[0], nodes[1].Cid, files[0])

	sp.addTask(tasks[1], nodes[0].Cid, files[1])
	sp.addTask(tasks[1], nodes[1].Cid, files[1])

	localCfg := &Config{
		DFGetCommonConfig: nodes[0],
	}

	sAPI := newMockSupernodeAPI(mss)
	manager := newSupernodeManager(context.Background(), localCfg, supernodes, sAPI,
		intervalOpt{heartBeatInterval: 3 * time.Second, fetchNetworkInterval: 3 * time.Second})

	waitCh := make(chan struct{})

	manager.AddRequest(tasks[0].TaskURL)
	manager.AddRequest(tasks[1].TaskURL)
	manager.ActiveFetchP2PNetwork(activeFetchSt{url: tasks[0].TaskURL, waitCh: waitCh})
	timeout := false

	timer := time.NewTimer(time.Second * 2)
	defer timer.Stop()
	select {
	case <-timer.C:
		timeout = true
		break
	case <-waitCh:
		break
	}

	c.Assert(timeout, check.Equals, false)

	localPeer := &api_types.PeerInfo{
		ID:   nodes[0].Cid,
		IP:   strfmt.IPv4(nodes[0].IP),
		Port: int32(nodes[0].Port),
	}

	df := newDownloaderFactory(manager, localPeer, api.NewDownloadAPI())

	// download tasks[0] from remote1
	opt1 := seed.DownloaderFactoryCreateOpt{
		URL:         tasks[0].TaskURL,
		RateLimiter: ratelimiter.NewRateLimiter(0, 0),
	}

	d1 := df.Create(opt1)
	c.Assert(d1, check.NotNil)

	suite.checkLocalDownloadDataFromFileServer(c, d1, fsHost, files[0], 0, 500*1024)
	suite.checkLocalDownloadDataFromFileServer(c, d1, fsHost, files[0], 0, 100*1024)
	for i := 0; i < 5; i++ {
		suite.checkLocalDownloadDataFromFileServer(c, d1, fsHost, files[0], int64(i*100*1024), 100*1024)
	}

	// download tasks[1] from remote1
	opt2 := seed.DownloaderFactoryCreateOpt{
		URL:         tasks[1].TaskURL,
		RateLimiter: ratelimiter.NewRateLimiter(0, 0),
	}

	d2 := df.Create(opt2)
	c.Assert(d2, check.NotNil)
	suite.checkLocalDownloadDataFromFileServer(c, d2, fsHost, files[1], 0, 10*1024*1024)
	suite.checkLocalDownloadDataFromFileServer(c, d2, fsHost, files[1], 0, 200*1024)
	for i := 0; i < 100; i++ {
		suite.checkLocalDownloadDataFromFileServer(c, d2, fsHost, files[1], int64(i*100*1024), 100*1024)
	}
}
