/*
 * Copyright 1999-2018 Alibaba Group.
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
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

/* http content types */
const (
	ApplicationJSONUtf8Value = "application/json;charset=utf-8"
)

const (
	// RequestTag is the tag name for parsing structure to query parameters.
	// see function ParseQuery.
	RequestTag = "request"
)

// DefaultHTTPClient is the default implementation of SimpleHTTPClient.
var DefaultHTTPClient SimpleHTTPClient = &defaultHTTPClient{}

// SimpleHTTPClient defines some http functions used frequently.
type SimpleHTTPClient interface {
	PostJSON(url string, body interface{}, timeout time.Duration) (code int, res []byte, e error)
	Get(url string, timeout time.Duration) (code int, res []byte, e error)
}

// ----------------------------------------------------------------------------
// defaultHTTPClient

type defaultHTTPClient struct {
}

// PostJSON send a POST request whose content-type is 'application/json;charset=utf-8'.
// When timeout <= 0, it will block until receiving response from server.
func (c *defaultHTTPClient) PostJSON(url string, body interface{}, timeout time.Duration) (
	code int, resBody []byte, err error) {

	var jsonByte []byte

	if body != nil {
		jsonByte, err = json.Marshal(body)
		if err != nil {
			return fasthttp.StatusBadRequest, nil, err
		}
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	req.SetBody(jsonByte)
	req.Header.SetMethod("POST")
	req.Header.SetContentType(ApplicationJSONUtf8Value)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if timeout > 0 {
		err = fasthttp.DoTimeout(req, resp, timeout)
	} else {
		err = fasthttp.Do(req, resp)
	}
	return resp.StatusCode(), resp.Body(), err
}

// Get sends a GET request to server.
// When timeout <= 0, it will block until receiving response from server.
func (c *defaultHTTPClient) Get(url string, timeout time.Duration) (
	code int, body []byte, e error) {
	if timeout > 0 {
		return fasthttp.GetTimeout(nil, url, timeout)
	}
	return fasthttp.Get(nil, url)
}

// ---------------------------------------------------------------------------
// util functions

// PostJSON send a POST request whose content-type is 'application/json;charset=utf-8'.
func PostJSON(url string, body interface{}, timeout time.Duration) (int, []byte, error) {
	return DefaultHTTPClient.PostJSON(url, body, timeout)
}

// Get sends a GET request to server.
// When timeout <= 0, it will block until receiving response from server.
func Get(url string, timeout time.Duration) (int, []byte, error) {
	return DefaultHTTPClient.Get(url, timeout)
}

// HTTPStatusOk reports whether the http response code is 200.
func HTTPStatusOk(code int) bool {
	return fasthttp.StatusOK == code
}

// ParseQuery only parses the fields with tag 'request' of the query to parameters.
// query must be a pointer to a struct.
func ParseQuery(query interface{}) string {
	if IsNil(query) {
		return ""
	}

	b := bytes.Buffer{}
	wrote := false
	t := reflect.TypeOf(query).Elem()
	v := reflect.ValueOf(query).Elem()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get(RequestTag)
		if tag != "" {
			if wrote {
				b.WriteByte('&')
			}
			b.WriteString(tag)
			b.WriteByte('=')
			b.WriteString(fmt.Sprintf("%v", v.Field(i)))
			wrote = true
		}
	}
	return b.String()
}

// CheckConnect checks the network connectivity between local and remote.
// param timeout: its unit is milliseconds, reset to 500 ms if <= 0
// returns localIP
func CheckConnect(ip string, port int, timeout int) (localIP string, e error) {
	if timeout <= 0 {
		timeout = 500
	}

	var conn net.Conn
	t := time.Duration(timeout) * time.Millisecond
	addr := fmt.Sprintf("%s:%d", ip, port)
	if conn, e = net.DialTimeout("tcp", addr, t); e == nil {
		localIP = conn.LocalAddr().String()
		conn.Close()
		if idx := strings.LastIndexByte(localIP, ':'); idx >= 0 {
			localIP = localIP[:idx]
		}
	}
	return
}
