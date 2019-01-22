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
	tempFileCOntent = "hello File"
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
	tc := &taskConfig{dataDir: s.dataDir}
	s.servicePath = path.Join(s.dataDir, taskFileName+".service")

	createTestFile(s.servicePath)
	syncTaskMap.Store(taskFileName, tc)

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

func (s *UploadUtilTestSuite) TearDownSuite(c *check.C) {
	s.ln.Close()

	if s.dataDir != "" {
		if err := os.RemoveAll(s.dataDir); err != nil {
			fmt.Printf("remove path:%s error", s.dataDir)
		}
	}
}

// TestGetTaskFile
func (s *UploadUtilTestSuite) TestGetTaskFile(c *check.C) {
	f, err := getTaskFile(taskFileName)
	defer f.Close()

	// check get file correctly
	c.Assert(err, check.IsNil)
	c.Assert(f, check.NotNil)

	// check read file correctly
	result, err := ioutil.ReadAll(f)
	c.Assert(err, check.IsNil)
	c.Assert(string(result), check.Equals, tempFileCOntent)
}

func (s *UploadUtilTestSuite) TestTransFile(c *check.C) {
	f, _ := getTaskFile(taskFileName)
	defer f.Close()

	// normal test when start offset equals zero
	rr := httptest.NewRecorder()
	err := transFile(f, rr, 0, 10)
	c.Check(err, check.IsNil)
	c.Check(rr.Body.String(), check.Equals, tempFileCOntent)

	// normal test when start offset not equals zero
	rr = httptest.NewRecorder()
	err = transFile(f, rr, 1, 5)
	c.Check(err, check.IsNil)
	c.Check(rr.Body.String(), check.Equals, tempFileCOntent[1:6])

	// readLen more than file data test
	rr = httptest.NewRecorder()
	err = transFile(f, rr, 0, 20)
	c.Check(err, check.IsNil)
	c.Check(rr.Body.String(), check.Equals, tempFileCOntent)

	// readLen less than file data test
	rr = httptest.NewRecorder()
	err = transFile(f, rr, 0, 5)
	c.Check(err, check.IsNil)
	c.Check(rr.Body.String(), check.Equals, tempFileCOntent[:5])

}

func (s *UploadUtilTestSuite) TestCheckServer(c *check.C) {
	// normal test
	result, err := checkServer(s.ip, s.port, s.dataDir, taskFileName, 10*time.Millisecond)
	c.Check(err, check.IsNil)
	c.Check(result, check.Equals, taskFileName)

	// timeout equals zero test
	result, err = checkServer(s.ip, s.port, s.dataDir, taskFileName, 0)
	c.Check(err, check.IsNil)
	c.Check(result, check.Equals, taskFileName)

	// error url test
	result, err = checkServer(s.ip+"1", s.port, s.dataDir, taskFileName, 0)
	c.Check(err, check.NotNil)
	c.Check(result, check.Equals, "")
}

func (s *UploadUtilTestSuite) TestParseRange(c *check.C) {
	type args struct {
		rangeStr string
	}
	tests := []struct {
		name    string
		args    args
		want    *uploadParam
		wantErr bool
	}{
		{
			name: "normalTest",
			args: args{
				rangeStr: "0-65575",
			},
			want: &uploadParam{
				start:    0,
				pieceLen: 65576,
				readLen:  65576,
			},
			wantErr: false,
		},
		{
			name: "MultDashesTest",
			args: args{
				rangeStr: "0-65-575",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NotIntTest",
			args: args{
				rangeStr: "0-hello",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "EndLessStartTest",
			args: args{
				rangeStr: "65575-0",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := parseRange(tt.args.rangeStr)
		if tt.wantErr {
			c.Check(err, check.NotNil)
		} else {
			c.Check(err, check.IsNil)
		}
		c.Check(got, check.DeepEquals, tt.want)
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

// createTestFile create a temp file and write a string.
func createTestFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(tempFileCOntent)
	return nil
}
