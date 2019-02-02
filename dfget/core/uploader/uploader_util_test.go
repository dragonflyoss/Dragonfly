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
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/go-check/check"
	"github.com/gorilla/mux"
)

var (
	taskFileName    = "testFile"
	tempFileContent = "hello File"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type UploadUtilTestSuite struct {
	dataDir     string
	servicePath string
	host        string
	ip          string
	port        int
	ln          net.Listener
}

func init() {
	check.Suite(&UploadUtilTestSuite{})
}

func (s *UploadUtilTestSuite) SetUpSuite(c *check.C) {
	s.dataDir, _ = ioutil.TempDir("/tmp", "dfget-UploadUtilTestSuite-")
	newTestPeerServer(s.dataDir)
	initHelper(taskFileName, s.dataDir, tempFileContent)

	s.startTestServer()
}

func (s *UploadUtilTestSuite) TearDownSuite(c *check.C) {
	s.ln.Close()

	if s.dataDir != "" {
		if err := os.RemoveAll(s.dataDir); err != nil {
			fmt.Printf("remove path:%s error", s.dataDir)
		}
	}
}

func (s *UploadUtilTestSuite) TestGetTaskFile(c *check.C) {
	// normal test
	f, _, err := p2p.getTaskFile(taskFileName)
	defer f.Close()
	// check get file correctly
	c.Assert(err, check.IsNil)
	c.Assert(f, check.NotNil)
	// check read file correctly
	result, err := ioutil.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert(string(result), check.Equals, tempFileContent)
}

func (s *UploadUtilTestSuite) TestUploadPiece(c *check.C) {
	f, size, _ := p2p.getTaskFile(taskFileName)
	defer f.Close()

	var up = func(start, end int64, pad bool) *uploadParam {
		up := &uploadParam{
			start:     start,
			end:       end,
			length:    end - start + 1,
			pieceNum:  0,
			pieceSize: defaultPieceSize,
		}
		amendRange(size, pad, up)
		return up
	}

	var cases = []struct {
		start    int64
		end      int64
		pad      bool
		expected string
	}{
		// normal test when start offset equals zero
		{0, 10, false, tempFileContent},
		// normal test when start offset not equals zero
		{1, 5, false, tempFileContent[1:6]},
		// range length more than file data test
		{0, 20, false, tempFileContent},
		// range length less than file data test
		{0, 5, false, tempFileContent[:6]},
		// with piece meta data
		{0, 5, true, tempFileContent[:1]},
		{0, 4, true, ""},
		{0, 15, true, tempFileContent},
	}

	for _, v := range cases {
		rr := httptest.NewRecorder()
		p := up(v.start, v.end, v.pad)
		err := p2p.uploadPiece(f, rr, p)
		c.Check(err, check.IsNil)
		cmt := check.Commentf("content:'%s' start:%d end:%d pad:%v",
			tempFileContent, v.start, v.end, v.pad)
		if v.pad {
			c.Check(rr.Body.String(), check.DeepEquals,
				pieceContent(p.pieceSize, v.expected), cmt)
		} else {
			c.Check(rr.Body.String(), check.Equals, v.expected, cmt)
		}
	}
}

func (s *UploadUtilTestSuite) TestAmendRange(c *check.C) {

}

func (s *UploadUtilTestSuite) TestCheckServer(c *check.C) {
	// normal test
	result, err := checkServer(s.ip, s.port, s.dataDir, taskFileName, 0, 10*time.Millisecond)
	c.Check(err, check.IsNil)
	c.Check(result, check.Equals, taskFileName)

	// timeout equals zero test
	result, err = checkServer(s.ip, s.port, s.dataDir, taskFileName, 0, 0)
	c.Check(err, check.IsNil)
	c.Check(result, check.Equals, taskFileName)

	// error url test
	result, err = checkServer(s.ip+"1", s.port, s.dataDir, taskFileName, 0, 0)
	c.Check(err, check.NotNil)
	c.Check(result, check.Equals, "")
}

func (s *UploadUtilTestSuite) TestParseParams(c *check.C) {
	uh := defaultUploadHeader

	tests := []struct {
		name    string
		header  uploadHeader
		want    *uploadParam
		wantErr bool
	}{
		{
			name:   "normalTest",
			header: uh.newRange("0-65575"),
			want: &uploadParam{
				start:     0,
				end:       65575,
				length:    65576,
				pieceSize: defaultPieceSize,
			},
			wantErr: false,
		},
		{
			name:    "MultiDashesTest",
			header:  uh.newRange("0-65-575"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "NotIntTest",
			header:  uh.newRange("0-hello"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "EndLessStartTest",
			header:  uh.newRange("65575-0"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "NegativeStartTest",
			header:  uh.newRange("-1-8"),
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		h := tt.header
		got, err := parseParams(h.rangeStr, h.num, h.size)
		if tt.wantErr {
			c.Check(err, check.NotNil)
		} else {
			c.Check(err, check.IsNil)
		}
		c.Check(got, check.DeepEquals, tt.want,
			check.Commentf("%s:%v", tt.name, tt.header))
	}
}

func (s *UploadUtilTestSuite) TestGetPort(c *check.C) {
	servicePort := 8080
	metaPath := path.Join(s.dataDir, "meta")
	meta := config.NewMetaData(metaPath)
	meta.ServicePort = servicePort
	err := meta.Persist()
	c.Check(err, check.IsNil)

	port := getPortFromMeta(metaPath)
	c.Check(port, check.Equals, servicePort)
}

func (s *UploadUtilTestSuite) startTestServer() {
	// run a server
	s.ip = "127.0.0.1"
	s.port = rand.Intn(1000) + 63000
	s.host = fmt.Sprintf("%s:%d", s.ip, s.port)
	s.ln, _ = net.Listen("tcp", s.host)
	checkHandler := func(w http.ResponseWriter, r *http.Request) {
		fileName := mux.Vars(r)["taskFileName"]
		fmt.Fprintf(w, "%s@%s", fileName, version.DFGetVersion)
	}
	r := mux.NewRouter()
	r.HandleFunc(config.LocalHTTPPathCheck+"{taskFileName:.*}", checkHandler).Methods("GET")
	go http.Serve(s.ln, r)
}
