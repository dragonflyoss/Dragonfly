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

package downloader

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/ratelimiter"

	"github.com/go-check/check"
)

var (
	port         = 65005
	downloadMock = func() (*http.Response, error) { return nil, nil }
)

type PowerClientTestSuite struct {
	powerClient *PowerClient
	ln          net.Listener
}

func init() {
	check.Suite(&PowerClientTestSuite{})
}

func (s *PowerClientTestSuite) SetUpSuite(c *check.C) {
	s.reset()
	s.upServer(port)
}

func (s *PowerClientTestSuite) TearDownTest(c *check.C) {
	s.ln.Close()
}

func (s *PowerClientTestSuite) TestDownloadPiece(c *check.C) {
	// dstIP != pc.node && CheckConnect Fail
	s.powerClient.pieceTask.PeerIP = "127.0.0.2"
	content, err := s.powerClient.downloadPiece()
	c.Check(content, check.IsNil)
	c.Check(err, check.NotNil)
	s.reset()

	// dstIP != pc.node && CheckConnect Success && Download Fail
	s.powerClient.node = "127.0.0.2"
	downloadMock = func() (*http.Response, error) {
		return nil, fmt.Errorf("error")
	}
	content, err = s.powerClient.downloadPiece()
	c.Check(content, check.IsNil)
	c.Check(err, check.DeepEquals, fmt.Errorf("error"))
	s.reset()

	// dstIP == pc.node && Download Success && StatusCode == 416
	body1 := ioutil.NopCloser(bytes.NewReader([]byte("RangeNotSatisfiable")))
	defer func() {
		body1.Close()
	}()
	downloadMock = func() (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusRequestedRangeNotSatisfiable,
			Body:       body1,
		}, nil
	}
	content, err = s.powerClient.downloadPiece()
	c.Check(content, check.IsNil)
	c.Check(err, check.DeepEquals, errortypes.ErrRangeNotSatisfiable)

	// dstIP == pc.node && Download Success && StatusCode == 416
	body2 := ioutil.NopCloser(bytes.NewReader([]byte("test")))
	defer func() {
		body2.Close()
	}()
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       body2,
	}
	downloadMock = func() (*http.Response, error) {
		return resp, nil
	}
	content, err = s.powerClient.downloadPiece()
	c.Check(content, check.IsNil)
	c.Check(err, check.NotNil)

	// dstIP == pc.node && Download Success && StatusCode == 200 && md5 not match
	s.powerClient.pieceTask.PieceMd5 = "foo"
	body3 := ioutil.NopCloser(bytes.NewReader([]byte("hello")))
	defer func() {
		body3.Close()
	}()
	resp = &http.Response{
		StatusCode: http.StatusOK,
		Body:       body3,
	}
	downloadMock = func() (*http.Response, error) {
		return resp, nil
	}
	content, err = s.powerClient.downloadPiece()
	// fmt.Println(err.Error())
	c.Check(content, check.IsNil)
	c.Check(err, check.NotNil)
	s.reset()

	// dstIP == pc.node && Download Success && StatusCode == 200 && md5 match
	s.powerClient.pieceTask.PieceMd5 = "5d41402abc4b2a76b9719d911017c592"
	body4 := ioutil.NopCloser(bytes.NewReader([]byte("hello")))
	resp = &http.Response{
		StatusCode: http.StatusOK,
		Body:       body4,
	}
	downloadMock = func() (*http.Response, error) {
		return resp, nil
	}
	content, err = s.powerClient.downloadPiece()
	c.Check(content, check.NotNil)
	c.Check(content.String(), check.Equals, "hello")
	c.Check(err, check.IsNil)
}

func (s *PowerClientTestSuite) TestReadBody(c *check.C) {
	powerClient := &PowerClient{}
	var cases = []struct {
		body     io.ReadCloser
		expected string
	}{
		{body: ioutil.NopCloser(bytes.NewReader([]byte("hello"))), expected: "hello"},
		{body: ioutil.NopCloser(bytes.NewReader([]byte(""))), expected: ""},
	}

	for _, v := range cases {
		defer func() {
			v.body.Close()
		}()
		result := powerClient.readBody(v.body)
		c.Assert(result, check.Equals, v.expected)
	}
}

func (s *PowerClientTestSuite) reset() {
	s.powerClient = &PowerClient{
		cfg:         &config.Config{RV: config.RuntimeVariable{Cid: ""}},
		node:        "127.0.0.1",
		rateLimiter: ratelimiter.NewRateLimiter(int64(5), 2),
		downloadAPI: NewMockDownloadAPI(),
		pieceTask: &types.PullPieceTaskResponseContinueData{
			PieceMd5: "",
			PeerIP:   "127.0.0.1",
			PeerPort: port,
		},
	}
}

// upServer up a local server.
func (s *PowerClientTestSuite) upServer(port int) {
	host := fmt.Sprintf("127.0.0.1:%d", port)
	s.ln, _ = net.Listen("tcp", host)
	go http.Serve(s.ln, nil)
}

// downloadMockAPI is a mock implementation of interface DownloadAPI.
type downloadMockAPI struct {
}

// NewMockDownloadAPI returns a new mock DownloadAPI.
func NewMockDownloadAPI() api.DownloadAPI {
	return &downloadMockAPI{}
}

func (d *downloadMockAPI) Download(ip string, port int, req *api.DownloadRequest, timeout time.Duration) (*http.Response, error) {
	return downloadMock()
}
