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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/util"

	"github.com/pkg/errors"
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

	// DefaultTimeout is the default timeout to check connect.
	DefaultTimeout = 500 * time.Millisecond
)

var (
	// DefaultBuiltInTransport is the transport for HTTPWithHeaders.
	DefaultBuiltInTransport *http.Transport

	// DefaultBuiltInHTTPClient is the http client for HTTPWithHeaders.
	DefaultBuiltInHTTPClient *http.Client
)

// DefaultHTTPClient is the default implementation of SimpleHTTPClient.
var DefaultHTTPClient SimpleHTTPClient = &defaultHTTPClient{}

// protocols stores custom protocols
// key: schema value: transport
var protocols = sync.Map{}

// validURLSchemas stores valid schemas
// when call RegisterProtocol, validURLSchemas will be also updated.
var validURLSchemas = "https?|HTTPS?"

// SimpleHTTPClient defines some http functions used frequently.
type SimpleHTTPClient interface {
	PostJSON(url string, body interface{}, timeout time.Duration) (code int, res []byte, e error)
	Get(url string, timeout time.Duration) (code int, res []byte, e error)
	PostJSONWithHeaders(url string, headers map[string]string, body interface{}, timeout time.Duration) (code int, resBody []byte, err error)
	GetWithHeaders(url string, headers map[string]string, timeout time.Duration) (code int, resBody []byte, err error)
}

func init() {
	http.DefaultClient.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	DefaultBuiltInTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	DefaultBuiltInHTTPClient = &http.Client{
		Transport: DefaultBuiltInTransport,
	}

	RegisterProtocolOnTransport(DefaultBuiltInTransport)
}

// ----------------------------------------------------------------------------
// defaultHTTPClient

type defaultHTTPClient struct {
}

var _ SimpleHTTPClient = &defaultHTTPClient{}

// PostJSON sends a POST request whose content-type is 'application/json;charset=utf-8'.
// When timeout <= 0, it will block until receiving response from server.
func (c *defaultHTTPClient) PostJSON(url string, body interface{}, timeout time.Duration) (
	code int, resBody []byte, err error) {
	return c.PostJSONWithHeaders(url, nil, body, timeout)
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

// PostJSONWithHeaders sends a POST request with headers whose content-type is 'application/json;charset=utf-8'.
// When timeout <= 0, it will block until receiving response from server.
func (c *defaultHTTPClient) PostJSONWithHeaders(url string, headers map[string]string, body interface{}, timeout time.Duration) (
	code int, resBody []byte, err error) {

	var jsonByte []byte

	if body != nil {
		jsonByte, err = json.Marshal(body)
		if err != nil {
			return fasthttp.StatusBadRequest, nil, err
		}
	}

	return do(url, headers, timeout, func(req *fasthttp.Request) error {
		req.SetBody(jsonByte)
		req.Header.SetMethod("POST")
		req.Header.SetContentType(ApplicationJSONUtf8Value)
		return nil
	})
}

// GetWithHeaders sends a GET request with headers to server.
// When timeout <= 0, it will block until receiving response from server.
func (c *defaultHTTPClient) GetWithHeaders(url string, headers map[string]string, timeout time.Duration) (
	code int, body []byte, e error) {
	return do(url, headers, timeout, nil)
}

// requestSetFunc a function that will set some values to the *req.
type requestSetFunc func(req *fasthttp.Request) error

func do(url string, headers map[string]string, timeout time.Duration, rsf requestSetFunc) (statusCode int, body []byte, err error) {
	// init request and response
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(url)
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	// set request
	if rsf != nil {
		err = rsf(req)
		if err != nil {
			return
		}
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// send request
	if timeout > 0 {
		err = fasthttp.DoTimeout(req, resp, timeout)
	} else {
		err = fasthttp.Do(req, resp)
	}
	if err != nil {
		return
	}

	statusCode = resp.StatusCode()
	data := resp.Body()
	body = make([]byte, len(data))
	copy(body, data)
	return
}

// ---------------------------------------------------------------------------
// util functions

// PostJSON sends a POST request whose content-type is 'application/json;charset=utf-8'.
func PostJSON(url string, body interface{}, timeout time.Duration) (int, []byte, error) {
	return DefaultHTTPClient.PostJSON(url, body, timeout)
}

// Get sends a GET request to server.
// When timeout <= 0, it will block until receiving response from server.
func Get(url string, timeout time.Duration) (int, []byte, error) {
	return DefaultHTTPClient.Get(url, timeout)
}

// PostJSONWithHeaders sends a POST request whose content-type is 'application/json;charset=utf-8'.
func PostJSONWithHeaders(url string, headers map[string]string, body interface{}, timeout time.Duration) (int, []byte, error) {
	return DefaultHTTPClient.PostJSONWithHeaders(url, headers, body, timeout)
}

// GetWithHeaders sends a GET request to server.
// When timeout <= 0, it will block until receiving response from server.
func GetWithHeaders(url string, headers map[string]string, timeout time.Duration) (code int, resBody []byte, err error) {
	return DefaultHTTPClient.GetWithHeaders(url, headers, timeout)
}

// Do performs the given http request and fills the given http response.
// When timeout <= 0, it will block until receiving response from server.
func Do(url string, headers map[string]string, timeout time.Duration) (string, error) {
	statusCode, body, err := do(url, headers, timeout, nil)
	if err != nil {
		return "", err
	}

	if statusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", statusCode)
	}

	result := string(body)

	return result, nil
}

// HTTPGet sends an HTTP GET request with headers.
func HTTPGet(url string, headers map[string]string) (*http.Response, error) {
	return HTTPWithHeaders("GET", url, headers, 0, nil)
}

// HTTPGetTimeout sends an HTTP GET request with timeout.
func HTTPGetTimeout(url string, headers map[string]string, timeout time.Duration) (*http.Response, error) {
	return HTTPWithHeaders("GET", url, headers, timeout, nil)
}

// HTTPGetWithTLS sends an HTTP GET request with TLS config.
func HTTPGetWithTLS(url string, headers map[string]string, timeout time.Duration, cacerts []string, insecure bool) (*http.Response, error) {
	roots := x509.NewCertPool()
	appendSuccess := false
	for _, certPath := range cacerts {
		certBytes, err := ioutil.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		appendSuccess = appendSuccess || roots.AppendCertsFromPEM(certBytes)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
	}
	if appendSuccess {
		tlsConfig.RootCAs = roots
	}
	return HTTPWithHeaders("GET", url, headers, timeout, tlsConfig)
}

// HTTPWithHeaders sends an HTTP request with headers and specified method.
func HTTPWithHeaders(method, url string, headers map[string]string, timeout time.Duration, tlsConfig *tls.Config) (*http.Response, error) {
	var (
		cancel func()
	)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	if timeout > 0 {
		timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		req = req.WithContext(timeoutCtx)
		cancel = cancelFunc
	}

	var c = DefaultBuiltInHTTPClient

	if tlsConfig != nil {
		// copy from http.DefaultTransport
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}

		RegisterProtocolOnTransport(transport)
		transport.TLSClientConfig = tlsConfig

		c = &http.Client{
			Transport: transport,
		}
	}

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if cancel == nil {
		return res, nil
	}

	// do cancel() when close the body.
	res.Body = newWithFuncReadCloser(res.Body, cancel)
	return res, nil
}

// HTTPStatusOk reports whether the http response code is 200.
func HTTPStatusOk(code int) bool {
	return fasthttp.StatusOK == code
}

// ParseQuery only parses the fields with tag 'request' of the query to parameters.
// query must be a pointer to a struct.
func ParseQuery(query interface{}) string {
	if util.IsNil(query) {
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
	t := time.Duration(timeout) * time.Millisecond
	if timeout <= 0 {
		t = DefaultTimeout
	}

	var conn net.Conn
	addr := fmt.Sprintf("%s:%d", ip, port)
	// Just temporarily limit users can only use IP addr for IPv4 format.
	// In the near future, if we want to support IPv6, we can revise logic as below.
	if conn, e = net.DialTimeout("tcp4", addr, t); e == nil {
		localIP = conn.LocalAddr().String()
		conn.Close()
		if idx := strings.LastIndexByte(localIP, ':'); idx >= 0 {
			localIP = localIP[:idx]
		}
	}
	return
}

// ConstructRangeStr wraps the rangeStr as a HTTP Range header value.
func ConstructRangeStr(rangeStr string) string {
	return fmt.Sprintf("bytes=%s", rangeStr)
}

// RangeStruct contains the start and end of a http header range.
type RangeStruct struct {
	StartIndex int64
	EndIndex   int64
}

// GetRangeSE parses the start and the end from range HTTP header and returns them.
func GetRangeSE(rangeHTTPHeader string, length int64) ([]*RangeStruct, error) {
	var rangeStr = rangeHTTPHeader

	// when rangeHTTPHeader looks like "bytes=0-1023", and then gets "0-1023".
	if strings.ContainsAny(rangeHTTPHeader, "=") {
		rangeSlice := strings.Split(rangeHTTPHeader, "=")
		if len(rangeSlice) != 2 {
			return nil, errors.Wrapf(errortypes.ErrInvalidValue, "invalid range: %s, should be like bytes=0-1023", rangeStr)
		}
		rangeStr = rangeSlice[1]
	}

	var result []*RangeStruct

	rangeArr := strings.Split(rangeStr, ",")
	rangeCount := len(rangeArr)
	if rangeCount == 0 {
		result = append(result, &RangeStruct{
			StartIndex: 0,
			EndIndex:   length - 1,
		})
		return result, nil
	}

	for i := 0; i < rangeCount; i++ {
		if strings.Count(rangeArr[i], "-") != 1 {
			return nil, errors.Wrapf(errortypes.ErrInvalidValue, "invalid range: %s, should be like 0-1023", rangeArr[i])
		}

		// -{length}
		if strings.HasPrefix(rangeArr[i], "-") {
			rangeStruct, err := handlePrefixRange(rangeArr[i], length)
			if err != nil {
				return nil, err
			}
			result = append(result, rangeStruct)
			continue
		}

		// {startIndex}-
		if strings.HasSuffix(rangeArr[i], "-") {
			rangeStruct, err := handleSuffixRange(rangeArr[i], length)
			if err != nil {
				return nil, err
			}
			result = append(result, rangeStruct)
			continue
		}

		rangeStruct, err := handlePairRange(rangeArr[i], length)
		if err != nil {
			return nil, err
		}
		result = append(result, rangeStruct)
	}
	return result, nil
}

func handlePrefixRange(rangeStr string, length int64) (*RangeStruct, error) {
	downLength, err := strconv.ParseInt(strings.TrimPrefix(rangeStr, "-"), 10, 64)
	if err != nil || downLength < 0 {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "failed to parse range: %s to int: %v", rangeStr, err)
	}

	if downLength > length {
		return nil, errors.Wrapf(errortypes.ErrRangeNotSatisfiable, "range: %s", rangeStr)
	}

	return &RangeStruct{
		StartIndex: length - downLength,
		EndIndex:   length - 1,
	}, nil
}

func handleSuffixRange(rangeStr string, length int64) (*RangeStruct, error) {
	startIndex, err := strconv.ParseInt(strings.TrimSuffix(rangeStr, "-"), 10, 64)
	if err != nil || startIndex < 0 {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "failed to parse range: %s to int: %v", rangeStr, err)
	}

	if startIndex > length {
		return nil, errors.Wrapf(errortypes.ErrRangeNotSatisfiable, "range: %s", rangeStr)
	}

	return &RangeStruct{
		StartIndex: startIndex,
		EndIndex:   length - 1,
	}, nil
}

func handlePairRange(rangeStr string, length int64) (*RangeStruct, error) {
	rangePair := strings.Split(rangeStr, "-")

	startIndex, err := strconv.ParseInt(rangePair[0], 10, 64)
	if err != nil || startIndex < 0 {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "failed to parse range: %s to int: %v", rangeStr, err)
	}
	if startIndex > length {
		return nil, errors.Wrapf(errortypes.ErrRangeNotSatisfiable, "range: %s", rangeStr)
	}

	endIndex, err := strconv.ParseInt(rangePair[1], 10, 64)
	if err != nil || endIndex < 0 {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "failed to parse range: %s to int: %v", rangeStr, err)
	}
	if endIndex > length {
		return nil, errors.Wrapf(errortypes.ErrRangeNotSatisfiable, "range: %s", rangeStr)
	}

	if endIndex < startIndex {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "range: %s, the start is larger the end", rangeStr)
	}

	return &RangeStruct{
		StartIndex: startIndex,
		EndIndex:   endIndex,
	}, nil
}

// RegisterProtocol registers custom protocols in global variable "protocols" which will be used in dfget and supernode
// Example:
//   protocols := "helloworld"
//   newTransport := funcNewTransport
//   httputils.RegisterProtocol(protocols, newTransport)
// RegisterProtocol must be called before initialise dfget or supernode instances.
func RegisterProtocol(scheme string, rt http.RoundTripper) {
	validURLSchemas += "|" + scheme
	protocols.Store(scheme, rt)
}

// RegisterProtocolOnTransport registers all new protocols in "protocols" for a special Transport
// this function will be used in supernode and dfwget
func RegisterProtocolOnTransport(tr *http.Transport) {
	protocols.Range(
		func(key, value interface{}) bool {
			tr.RegisterProtocol(key.(string), value.(http.RoundTripper))
			return true
		})
}

func GetValidURLSchemas() string {
	return validURLSchemas
}

func newWithFuncReadCloser(rc io.ReadCloser, f func()) io.ReadCloser {
	return &withFuncReadCloser{
		f:          f,
		ReadCloser: rc,
	}
}

type withFuncReadCloser struct {
	f func()
	io.ReadCloser
}

func (wrc *withFuncReadCloser) Close() error {
	if wrc.f != nil {
		wrc.f()
	}
	return wrc.ReadCloser.Close()
}
