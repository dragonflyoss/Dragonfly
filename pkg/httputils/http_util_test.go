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

package httputils

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/go-check/check"
	"github.com/valyala/fasthttp"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type HTTPUtilTestSuite struct {
	port int
	host string
	ln   net.Listener
}

func init() {
	rand.Seed(time.Now().Unix())
	check.Suite(&HTTPUtilTestSuite{})
}

func (s *HTTPUtilTestSuite) SetUpSuite(c *check.C) {
	s.port = rand.Intn(1000) + 63000
	s.host = fmt.Sprintf("127.0.0.1:%d", s.port)

	s.ln, _ = net.Listen("tcp", s.host)
	go fasthttp.Serve(s.ln, func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType(ApplicationJSONUtf8Value)
		ctx.SetStatusCode(fasthttp.StatusOK)
		req := &testJSONReq{}
		json.Unmarshal(ctx.Request.Body(), req)
		res := testJSONRes{
			Sum: req.A + req.B,
		}
		resByte, _ := json.Marshal(res)
		ctx.SetBody(resByte)
		time.Sleep(50 * time.Millisecond)
	})
}

func (s *HTTPUtilTestSuite) TearDownSuite(c *check.C) {
	s.ln.Close()
}

// ----------------------------------------------------------------------------
// unit tests

func (s *HTTPUtilTestSuite) TestPostJson(c *check.C) {
	code, body, e := PostJSON("http://"+s.host, req(1, 2), 60*time.Millisecond)
	checkOk(c, code, body, e, 3)

	_, _, e = PostJSON("http://"+s.host, req(1, 2), 50*time.Millisecond)
	c.Assert(e, check.NotNil)
	c.Assert(e.Error(), check.Equals, "timeout")

	code, body, e = PostJSON("http://"+s.host, req(2, 3), 0)
	checkOk(c, code, body, e, 5)

	code, body, e = PostJSON("http://"+s.host, nil, 0)
	checkOk(c, code, body, e, 0)
}

func (s *HTTPUtilTestSuite) TestGet(c *check.C) {
	code, body, e := Get("http://"+s.host, 0)
	checkOk(c, code, body, e, 0)

	_, _, e = Get("http://"+s.host, 50*time.Millisecond)
	c.Assert(e, check.NotNil)
	c.Assert(e.Error(), check.Equals, "timeout")
}

func (s *HTTPUtilTestSuite) TestHTTPStatusOk(c *check.C) {
	for i := fasthttp.StatusContinue; i <= fasthttp.StatusNetworkAuthenticationRequired; i++ {
		c.Assert(HTTPStatusOk(i), check.Equals, i == fasthttp.StatusOK)
	}
}

func (s *HTTPUtilTestSuite) TestHttpGet(c *check.C) {
	res, e := HTTPGetTimeout("http://"+s.host, nil, 0)
	c.Assert(e, check.IsNil)
	code := res.StatusCode
	body, e := ioutil.ReadAll(res.Body)
	c.Assert(e, check.IsNil)
	res.Body.Close()

	checkOk(c, code, body, e, 0)

	res, e = HTTPGetTimeout("http://"+s.host, nil, 60*time.Millisecond)
	c.Assert(e, check.IsNil)
	code = res.StatusCode
	body, e = ioutil.ReadAll(res.Body)
	c.Assert(e, check.IsNil)
	res.Body.Close()

	checkOk(c, code, body, e, 0)

	_, e = HTTPGetTimeout("http://"+s.host, nil, 20*time.Millisecond)
	c.Assert(e, check.NotNil)
	c.Assert(strings.Contains(e.Error(), context.DeadlineExceeded.Error()), check.Equals, true)
}

func (s *HTTPUtilTestSuite) TestParseQuery(c *check.C) {
	type req struct {
		A int    `request:"a"`
		B string `request:"b"`
		C int
	}
	r := req{1, "test", 3}
	x := ParseQuery(&r)
	c.Assert(x, check.Equals, "a=1&b=test")

	c.Assert(ParseQuery(nil), check.Equals, "")
}

func (s *HTTPUtilTestSuite) TestCheckConnect(c *check.C) {
	ip, e := CheckConnect("127.0.0.1", s.port, 0)
	c.Assert(e, check.IsNil)
	c.Assert(ip, check.Equals, "127.0.0.1")

	// Test IPv6
	_, e = CheckConnect("[::1]", s.port, 0)
	c.Assert(e, check.NotNil)
}

func (s *HTTPUtilTestSuite) TestGetRangeSE(c *check.C) {
	var cases = []struct {
		rangeHTTPHeader string
		length          int64
		expected        []*RangeStruct
		errCheck        func(error) bool
	}{
		{
			rangeHTTPHeader: "bytes=0-65575",
			length:          65576,
			expected: []*RangeStruct{
				{
					StartIndex: 0,
					EndIndex:   65575,
				},
			},
			errCheck: errortypes.IsNilError,
		},
		{
			rangeHTTPHeader: "bytes=2-2",
			length:          65576,
			expected: []*RangeStruct{
				{
					StartIndex: 2,
					EndIndex:   2,
				},
			},
			errCheck: errortypes.IsNilError,
		},
		{
			rangeHTTPHeader: "bytes=2-",
			length:          65576,
			expected: []*RangeStruct{
				{
					StartIndex: 2,
					EndIndex:   65575,
				},
			},
			errCheck: errortypes.IsNilError,
		},
		{
			rangeHTTPHeader: "bytes=-100",
			length:          65576,
			expected: []*RangeStruct{
				{
					StartIndex: 65476,
					EndIndex:   65575,
				},
			},
			errCheck: errortypes.IsNilError,
		},
		{
			rangeHTTPHeader: "bytes=0-66575",
			length:          65576,
			expected:        nil,
			errCheck:        errortypes.IsRangeNotSatisfiable,
		},
		{
			rangeHTTPHeader: "bytes=0-65-575",
			length:          65576,
			expected:        nil,
			errCheck:        errortypes.IsInvalidValue,
		},
		{
			rangeHTTPHeader: "bytes=0-hello",
			length:          65576,
			expected:        nil,
			errCheck:        errortypes.IsInvalidValue,
		},
		{
			rangeHTTPHeader: "bytes=65575-0",
			length:          65576,
			expected:        nil,
			errCheck:        errortypes.IsInvalidValue,
		},
		{
			rangeHTTPHeader: "bytes=-1-8",
			length:          65576,
			expected:        nil,
			errCheck:        errortypes.IsInvalidValue,
		},
	}

	for _, v := range cases {
		result, err := GetRangeSE(v.rangeHTTPHeader, v.length)
		c.Check(v.errCheck(err), check.Equals, true)
		fmt.Println(v.rangeHTTPHeader)
		c.Check(result, check.DeepEquals, v.expected)
	}
}

func (s *HTTPUtilTestSuite) TestConcurrencyPostJson(c *check.C) {
	wg := &sync.WaitGroup{}
	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func(x, y int) {
			defer wg.Done()
			code, body, e := PostJSON("http://"+s.host, req(x, y), 1*time.Second)
			time.Sleep(20 * time.Millisecond)
			checkOk(c, code, body, e, x+y)
		}(i, i)
	}

	wg.Wait()
}

func (s *HTTPUtilTestSuite) TestConstructRangeStr(c *check.C) {
	c.Check(ConstructRangeStr("200-1000"), check.DeepEquals, "bytes=200-1000")
}

// ----------------------------------------------------------------------------
// helper functions and structures

func checkOk(c *check.C, code int, body []byte, e error, sum int) {
	c.Assert(e, check.IsNil)
	c.Assert(code, check.Equals, fasthttp.StatusOK)

	var res = &testJSONRes{}
	e = json.Unmarshal(body, res)
	c.Check(e, check.IsNil)
	c.Check(res.Sum, check.Equals, sum)
}

func req(x int, y int) *testJSONReq {
	return &testJSONReq{x, y}
}

type testJSONReq struct {
	A int
	B int
}

type testJSONRes struct {
	Sum int
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

func (s *HTTPUtilTestSuite) TestRegisterProtocol(c *check.C) {
	protocol := "test"
	RegisterProtocol(protocol, &testTransport{})
	resp, err := HTTPWithHeaders(http.MethodGet,
		protocol+"://test/test",
		map[string]string{
			"test": "test",
		},
		time.Second,
		&tls.Config{},
	)
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()

	c.Assert(resp, check.NotNil)
	c.Assert(resp.ContentLength, check.Equals, int64(-1))
}
