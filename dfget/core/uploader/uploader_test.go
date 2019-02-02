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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/helper"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/go-check/check"
)

var (
	taskFile         = "taskfile"
	emptyFile        = "emptyfile"
	fileContent      = helper.CreateRandomString(2000)
	defaultRateLimit = 1000
)

type HandlerHelper struct {
	method  string
	url     string
	body    io.Reader
	headers map[string]string
}

type UploaderTestSuite struct {
	workHome string
}

func init() {
	check.Suite(&UploaderTestSuite{})
}

func (u *UploaderTestSuite) SetUpSuite(c *check.C) {
	u.workHome, _ = ioutil.TempDir("/tmp", "dfget-UploadTestSuite-")
	newTestPeerServer(u.workHome)

	initHelper(taskFile, u.workHome, fileContent)
	initHelper(emptyFile, u.workHome, "")
}

func (u *UploaderTestSuite) TearDownSuite(c *check.C) {
	if u.workHome != "" {
		if err := os.RemoveAll(u.workHome); err != nil {
			fmt.Printf("remove path:%s error", u.workHome)
		}
	}
	p2p = nil
}

func (u *UploaderTestSuite) TestUploadHandler(c *check.C) {
	headers := make(map[string]string)
	headers[config.StrPieceSize] = defaultPieceSizeStr
	headers[config.StrPieceNum] = "0"

	// normal test
	headers["range"] = "bytes=0-1999"
	if rr, err := u.testHandlerHelper(&HandlerHelper{
		method:  http.MethodGet,
		url:     config.PeerHTTPPathPrefix + taskFile,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusPartialContent)
		c.Check(rr.Body.String(), check.Equals, pc(fileContent[:1995]))

		// TODO: limit read check
	}

	// RangeNotSatisfiable
	headers["range"] = "bytes=0-1"
	if rr, err := u.testHandlerHelper(&HandlerHelper{
		method:  http.MethodGet,
		url:     config.PeerHTTPPathPrefix + emptyFile,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusRequestedRangeNotSatisfiable)
	}

	// not found test
	if rr, err := u.testHandlerHelper(&HandlerHelper{
		method:  http.MethodGet,
		url:     config.PeerHTTPPathPrefix + "foo",
		body:    nil,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusInternalServerError)
	}

	// bad request test
	headers["range"] = "bytes=0-x"
	if rr, err := u.testHandlerHelper(&HandlerHelper{
		method:  http.MethodGet,
		url:     config.PeerHTTPPathPrefix + taskFile,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusBadRequest)
	}
}

func (u *UploaderTestSuite) TestParseRateHandler(c *check.C) {
	headers := make(map[string]string)

	// normal test
	testRateLimit := 1000
	total := testRateLimit + defaultRateLimit
	headers["rateLimit"] = strconv.Itoa(testRateLimit)
	if rr, err := u.testHandlerHelper(&HandlerHelper{
		method:  http.MethodGet,
		url:     config.LocalHTTPPathRate + taskFile,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusOK)
		limit := (testRateLimit*p2p.totalLimitRate + total - 1) / total
		c.Check(rr.Body.String(), check.Equals, strconv.Itoa(limit))
	}

	// totalLimitRate zero test
	p2p.totalLimitRate = 0
	if rr, err := u.testHandlerHelper(&HandlerHelper{
		method:  http.MethodGet,
		url:     config.LocalHTTPPathRate + taskFile,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusOK)
		c.Check(rr.Body.String(), check.Equals, strconv.Itoa(testRateLimit))
	}
	p2p.totalLimitRate = 1000

	// wrong rateLimit test
	headers["rateLimit"] = "foo"
	if rr, err := u.testHandlerHelper(&HandlerHelper{
		method:  http.MethodGet,
		url:     config.LocalHTTPPathRate + taskFile,
		headers: headers,
	}); err == nil {
		c.Check(rr.Code, check.Equals, http.StatusBadRequest)
	}
}

func (u *UploaderTestSuite) testHandlerHelper(hh *HandlerHelper) (*httptest.ResponseRecorder, error) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest(hh.method, hh.url, hh.body)
	if err != nil {
		return nil, err
	}

	// Set request headers.
	for k, v := range hh.headers {
		req.Header.Set(k, v)
	}

	// We create a ResponseRecorder
	// (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Init a router.
	r := p2p.initRouter()
	r.ServeHTTP(rr, req)

	return rr, nil
}

// newTestPeerServer init the peer server for testing.
func newTestPeerServer(workHome string) {
	buf := &bytes.Buffer{}
	cfg := helper.CreateConfig(buf, workHome)
	p2p = newPeerServer(cfg, 0)
	p2p.totalLimitRate = 1000
	p2p.rateLimiter = util.NewRateLimiter(int32(defaultRateLimit), 2)
}

// initHelper create a temporary file and store it in the syncTaskMap.
func initHelper(fileName, workHome, content string) {
	helper.CreateTestFile(helper.GetServiceFile(fileName, workHome), content)
	p2p.syncTaskMap.Store(fileName, &taskConfig{
		dataDir:   workHome,
		rateLimit: defaultRateLimit,
	})
}
