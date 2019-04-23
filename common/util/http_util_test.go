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

package util

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	errorType "github.com/dragonflyoss/Dragonfly/common/errors"

	"github.com/go-check/check"
	"github.com/valyala/fasthttp"
)

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
			errCheck: errorType.IsNilError,
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
			errCheck: errorType.IsNilError,
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
			errCheck: errorType.IsNilError,
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
			errCheck: errorType.IsNilError,
		},
		{
			rangeHTTPHeader: "bytes=0-66575",
			length:          65576,
			expected:        nil,
			errCheck:        errorType.IsRangeNotSatisfiable,
		},
		{
			rangeHTTPHeader: "bytes=0-65-575",
			length:          65576,
			expected:        nil,
			errCheck:        errorType.IsInvalidValue,
		},
		{
			rangeHTTPHeader: "bytes=0-hello",
			length:          65576,
			expected:        nil,
			errCheck:        errorType.IsInvalidValue,
		},
		{
			rangeHTTPHeader: "bytes=65575-0",
			length:          65576,
			expected:        nil,
			errCheck:        errorType.IsInvalidValue,
		},
		{
			rangeHTTPHeader: "bytes=-1-8",
			length:          65576,
			expected:        nil,
			errCheck:        errorType.IsInvalidValue,
		},
	}

	for _, v := range cases {
		result, err := GetRangeSE(v.rangeHTTPHeader, v.length)
		c.Check(v.errCheck(err), check.Equals, true)
		fmt.Println(v.rangeHTTPHeader)
		c.Check(result, check.DeepEquals, v.expected)
	}
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
