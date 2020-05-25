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

package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/protocol"
)

// Resource is an implementation of protocol.Resource for http protocol.
type Resource struct {
	url    string
	hd     *Headers
	client *Client
}

func (rs *Resource) Read(ctx context.Context, off int64, size int64) (rc io.ReadCloser, err error) {
	req, err := rs.newRequest(ctx, off, size)
	if err != nil {
		return nil, err
	}

	res, err := rs.doRequest(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil && res.Body != nil {
			res.Body.Close()
		}
	}()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("respnose code is not 200 or 206")
	}

	return res.Body, nil
}

func (rs *Resource) Length(ctx context.Context) (int64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := rs.newRequest(timeoutCtx, 0, 0)
	if err != nil {
		return 0, err
	}

	res, err := rs.doRequest(req)
	if err != nil {
		return 0, err
	}

	defer res.Body.Close()
	lenStr := res.Header.Get(config.StrContentLength)
	if lenStr == "" {
		return 0, fmt.Errorf("failed to get content length")
	}

	length, err := strconv.ParseInt(lenStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to prase %s to length: %sv", lenStr, err)
	}

	return length, nil
}

func (rs *Resource) Metadata(ctx context.Context) (protocol.Metadata, error) {
	return rs.hd, nil
}

func (rs *Resource) Expire(ctx context.Context) (bool, interface{}, error) {
	// need to implementation
	return false, nil, nil
}

func (rs *Resource) Call(ctx context.Context, request interface{}) (response interface{}, err error) {
	return nil, protocol.ErrNotImplementation
}

func (rs *Resource) Close() error {
	return nil
}

func (rs *Resource) newRequest(ctx context.Context, off, size int64) (*http.Request, error) {
	// off == 0 && size == 0 means all data.
	if (off < 0 || size < 0) || (off > 0 && size == 0) {
		return nil, fmt.Errorf("invalid argument")
	}

	req, err := http.NewRequest(http.MethodGet, rs.url, nil)
	if err != nil {
		return nil, err
	}

	if rs.hd != nil {
		for k, v := range rs.hd.Header {
			req.Header.Set(k, v[0])
		}
	}

	if size > 0 {
		req.Header.Set(config.StrRange, fmt.Sprintf("bytes=%d-%d", off, off+size-1))
	}

	req = req.WithContext(ctx)
	return req, nil
}

func (rs *Resource) doRequest(req *http.Request) (*http.Response, error) {
	return rs.client.client.Do(req)
}
