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

package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// RespError defines the response error.
type RespError struct {
	code int
	msg  string
}

// Error implements the error interface.
func (e RespError) Error() string {
	return e.msg
}

// Code returns the response code.
func (e RespError) Code() int {
	return e.code
}

// Response wraps the http.Response and other states.
type Response struct {
	StatusCode int
	Status     string
	Body       io.ReadCloser
}

func (client *APIClient) get(ctx context.Context, path string, query url.Values, headers map[string][]string) (*Response, error) {
	return client.sendRequest(ctx, "GET", path, query, nil, headers)
}

func (client *APIClient) post(ctx context.Context, path string, query url.Values, obj interface{}, headers map[string][]string) (*Response, error) {
	body, err := objectToJSONStream(obj)
	if err != nil {
		return nil, err
	}

	return client.sendRequest(ctx, "POST", path, query, body, headers)
}

func (client *APIClient) put(ctx context.Context, path string, query url.Values, obj interface{}, headers map[string][]string) (*Response, error) {
	body, err := objectToJSONStream(obj)
	if err != nil {
		return nil, err
	}

	return client.sendRequest(ctx, "PUT", path, query, body, headers)
}

func (client *APIClient) postRawData(ctx context.Context, path string, query url.Values, data io.Reader, headers map[string][]string) (*Response, error) {
	return client.sendRequest(ctx, "POST", path, query, data, headers)
}

func (client *APIClient) delete(ctx context.Context, path string, query url.Values, headers map[string][]string) (*Response, error) {
	return client.sendRequest(ctx, "DELETE", path, query, nil, headers)
}

func (client *APIClient) hijack(ctx context.Context, path string, query url.Values, obj interface{}, header map[string][]string) (net.Conn, *bufio.Reader, error) {
	body, err := objectToJSONStream(obj)
	if err != nil {
		return nil, nil, err
	}

	req, err := client.newRequest("POST", path, query, body, header)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "tcp")

	req.Host = client.addr
	conn, err := net.DialTimeout(client.proto, client.addr, defaultTimeout)
	if err != nil {
		return nil, nil, err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			return nil, nil, err
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			return nil, nil, err
		}
	}

	//lint:ignore SA1019 we do not migrate this to 'net/http.Client' as it does not implement Hijack now.
	clientconn := httputil.NewClientConn(conn, nil)
	defer clientconn.Close()

	if _, err := clientconn.Do(req); err != nil {
		return nil, nil, err
	}

	rwc, br := clientconn.Hijack()

	return rwc, br, nil
}

func (client *APIClient) newRequest(method, path string, query url.Values, body io.Reader, header map[string][]string) (*http.Request, error) {
	fullPath := client.baseURL + client.GetAPIPath(path, query)
	req, err := http.NewRequest(method, fullPath, body)
	if err != nil {
		return nil, err
	}

	for k, v := range header {
		req.Header[k] = v
	}

	return req, err
}

func (client *APIClient) sendRequest(ctx context.Context, method, path string, query url.Values, body io.Reader, headers map[string][]string) (*Response, error) {
	req, err := client.newRequest(method, path, query, body, headers)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	resp, err := cancellableDo(ctx, client.HTTPCli, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, RespError{code: resp.StatusCode, msg: string(data)}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       resp.Body,
	}, nil
}

func cancellableDo(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	type contextResp struct {
		response *http.Response
		err      error
	}

	ctxResp := make(chan contextResp, 1)
	go func() {
		resp, err := client.Do(req)
		ctxResp <- contextResp{
			response: resp,
			err:      err,
		}
	}()

	select {
	case <-ctx.Done():
		<-ctxResp
		return nil, ctx.Err()

	case resp := <-ctxResp:
		return resp.response, resp.err
	}
}

func objectToJSONStream(obj interface{}) (io.Reader, error) {
	if obj != nil {
		b, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(b), nil
	}

	return nil, nil
}
