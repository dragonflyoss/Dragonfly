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
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
)

func TestUtil(t *testing.T) {
	suite.Run(t, new(TestUtilSuite))
}

type TestUtilSuite struct {
	suite.Suite
}

func (s *TestUtilSuite) TestParseJSONRequest() {
	cases := []struct {
		req       string
		validator ValidateFunc

		// expected
		target *testStruct
		err    string
	}{
		{"", nil, nil, "nil target"},
		{"", nil, newT(0), "empty request"},
		{"{x}", nil, newT(0), "invalid character"},
		{"{}", nil, newT(0), ""},
		{"{\"A\":1}", nil, newT(1), ""},
		{"{\"A\":1}", newT(1).validate, newT(1), "invalid"},
		{"{\"A\":2}", newT(2).validate, newT(2), ""},
	}

	for i, c := range cases {
		msg := fmt.Sprintf("case %d: %v", i, c)

		var obj *testStruct
		buf := bytes.NewBufferString(c.req)
		if c.target != nil {
			obj = &testStruct{}
		}
		e := ParseJSONRequest(buf, obj, c.validator)

		if c.err == "" {
			s.Nil(e, msg)
			s.NotNil(obj, msg)
			s.Equal(c.target, obj)
		} else {
			s.NotNil(e, msg)
			s.Contains(e.Error(), c.err, msg)
		}
	}

}

func (s *TestUtilSuite) TestEncodeResponse() {
	cases := []struct {
		code int
		data interface{}

		// expected
		err string
	}{
		{200, "", ""},
		{200, (*testStruct)(nil), ""},
		{200, 0, ""},
		{200, newT(1), ""},
		{400, newT(1), ""},
	}

	for i, c := range cases {
		msg := fmt.Sprintf("case %d: %v", i, c)
		w := httptest.NewRecorder()
		e := SendResponse(w, c.code, c.data)
		if c.err == "" {
			s.Nil(e, msg)
			s.Equal(c.code, w.Code, msg)
			if c.data == "" {
				s.Equal("\"\"", strings.TrimSpace(w.Body.String()), msg)
			} else {
				s.Equal(fmt.Sprintf("%v", c.data), strings.TrimSpace(w.Body.String()), msg)
			}
		} else {
			s.NotNil(e, msg)
		}
	}
}

func (s *TestUtilSuite) TestHandleErrorResponse() {
	cases := []struct {
		err error
		// expected
		code int
		out  string
	}{
		{
			errortypes.NewHTTPError(400, "user"),
			400, "{\"code\":400,\"message\":\"user\"}\n",
		},
		{
			fmt.Errorf("hello"),
			500, "{\"code\":500,\"message\":\"hello\"}\n",
		},
	}
	for i, c := range cases {
		msg := fmt.Sprintf("case %d: %v", i, c)
		w := httptest.NewRecorder()
		HandleErrorResponse(w, c.err)
		s.Equal(c.code, w.Code, msg)
		s.Equal(c.out, w.Body.String(), msg)
	}
}

func (s *TestUtilSuite) TestWrapHandler() {
	var tf HandlerFunc = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		switch req.Method {
		case "GET":
			return fmt.Errorf("test")
		case "POST":
			return errortypes.NewHTTPError(400, "test")
		}
		_ = SendResponse(rw, 200, "test")
		return nil
	}
	cases := []struct {
		method string

		// expected
		code int
		out  string
	}{
		{
			"GET",
			500, "{\"code\":500,\"message\":\"test\"}\n",
		},
		{
			"POST",
			400, "{\"code\":400,\"message\":\"test\"}\n",
		},
		{
			"PUT",
			200, "\"test\"\n",
		},
	}
	for i, c := range cases {
		msg := fmt.Sprintf("case %d: %v", i, c)
		h := WrapHandler(tf)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(c.method, "http:", nil)
		h(w, r)

		s.Equal(c.code, w.Code, msg)
		s.Equal(c.out, w.Body.String(), msg)
	}
}

// -----------------------------------------------------------------------------
// testing helpers

func newT(a int) *testStruct {
	return &testStruct{A: a}
}

type testStruct struct {
	A int
}

func (t *testStruct) validate(registry strfmt.Registry) error {
	if t.A <= 1 {
		return fmt.Errorf("invalid")
	}
	return nil
}

func (t *testStruct) String() string {
	if t == nil {
		return "null"
	}
	return fmt.Sprintf("{\"A\":%d}", t.A)
}
