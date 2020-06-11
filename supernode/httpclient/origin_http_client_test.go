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

package httpclient

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-check/check"

	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
)

func init() {
	check.Suite(&OriginHTTPClientTestSuite{})
}

func Test(t *testing.T) {
	check.TestingT(t)
}

type OriginHTTPClientTestSuite struct {
	client *OriginClient
}

func (s *OriginHTTPClientTestSuite) SetUpSuite(c *check.C) {
	s.client = NewOriginClient().(*OriginClient)
}

func (s *OriginHTTPClientTestSuite) TearDownSuite(c *check.C) {
}

func (s *OriginHTTPClientTestSuite) TestHTTPWithHeaders(c *check.C) {
	testString := "test bytes"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testString))
		if r.Method != "GET" {
			c.Errorf("Expected 'GET' request, got '%s'", r.Method)
		}
	}))
	defer ts.Close()

	httptest.NewRecorder()
	resp, err := s.client.HTTPWithHeaders(http.MethodGet, ts.URL, map[string]string{}, time.Second)
	c.Check(err, check.IsNil)
	defer resp.Body.Close()

	testBytes, err := ioutil.ReadAll(resp.Body)
	c.Check(err, check.IsNil)
	c.Check(string(testBytes), check.Equals, testString)
}

type testTransport struct {
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          http.NoBody,
		Status:        http.StatusText(http.StatusOK),
		StatusCode:    http.StatusOK,
		ContentLength: -1,
	}, nil
}

func (s *OriginHTTPClientTestSuite) TestRegisterTLSConfig(c *check.C) {
	protocol := "test"
	httputils.RegisterProtocol(protocol, &testTransport{})
	s.client.RegisterTLSConfig(protocol+"://test/test", true, nil)
	httpClientInterface, ok := s.client.clientMap.Load("test")
	c.Check(ok, check.Equals, true)
	httpClient, ok := httpClientInterface.(*http.Client)
	c.Check(ok, check.Equals, true)

	resp, err := httpClient.Get(protocol + "://test/test")
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()
	c.Assert(resp, check.NotNil)
	c.Assert(resp.ContentLength, check.Equals, int64(-1))
}

func (s *OriginHTTPClientTestSuite) TestCopyHeader(c *check.C) {
	dst := CopyHeader(nil, nil)
	c.Check(dst, check.NotNil)
	c.Check(len(dst), check.Equals, 0)

	src := map[string]string{"test": "1"}
	dst = CopyHeader(nil, src)
	c.Check(dst, check.DeepEquals, src)

	dst["test"] = "2"
	c.Check(src["test"], check.Equals, "1")
}
