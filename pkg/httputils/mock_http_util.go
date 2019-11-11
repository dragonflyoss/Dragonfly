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
	"time"
)

type postJSONFunc = func(url string, body interface{},
	timeout time.Duration) (int, []byte, error)
type getFunc = func(url string, timeout time.Duration) (int, []byte, error)
type postJSONWithHeadersFunc = func(url string, headers map[string]string, body interface{},
	timeout time.Duration) (int, []byte, error)
type getWithHeadersFunc = func(url string, headers map[string]string,
	timeout time.Duration) (int, []byte, error)

// MockHTTPClient fakes a customized implementation of util.SimpleHTTPClient.
type MockHTTPClient struct {
	PostJSONFunc            postJSONFunc
	GetFunc                 getFunc
	PostJSONWithHeadersFunc postJSONWithHeadersFunc
	GetWithHeadersFunc      getWithHeadersFunc
}

// NewMockHTTPClient returns a new MockHTTPClient instance.
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{}
}

// PostJSON mocks base method.
func (m *MockHTTPClient) PostJSON(url string, body interface{}, timeout time.Duration) (
	int, []byte, error) {
	if m.PostJSONFunc != nil {
		return m.PostJSONFunc(url, body, timeout)
	}
	return 0, nil, nil
}

// Get mocks base method.
func (m *MockHTTPClient) Get(url string, timeout time.Duration) (int, []byte, error) {
	if m.GetFunc != nil {
		return m.GetFunc(url, timeout)
	}
	return 0, nil, nil
}

// PostJSONWithHeaders mocks base method.
func (m *MockHTTPClient) PostJSONWithHeaders(url string, headers map[string]string, body interface{}, timeout time.Duration) (
	int, []byte, error) {
	if m.PostJSONWithHeadersFunc != nil {
		return m.PostJSONWithHeadersFunc(url, headers, body, timeout)
	}
	return 0, nil, nil
}

// GetWithHeaders mocks base method.
func (m *MockHTTPClient) GetWithHeaders(url string, headers map[string]string, timeout time.Duration) (
	int, []byte, error) {
	if m.GetWithHeadersFunc != nil {
		return m.GetWithHeadersFunc(url, headers, timeout)
	}
	return 0, nil, nil
}

// Reset the MockHTTPClient.
func (m *MockHTTPClient) Reset() {
	m.PostJSONFunc = nil
	m.GetFunc = nil

	m.PostJSONWithHeadersFunc = nil
	m.GetWithHeadersFunc = nil
}

// CreatePostJSONFunc returns a mock postJSONFunc func
// which will always return the specific results.
func (m *MockHTTPClient) CreatePostJSONFunc(code int, res []byte, e error) postJSONFunc {
	return func(string, interface{}, time.Duration) (int, []byte, error) {
		return code, res, e
	}
}

// CreateGetFunc returns a mock getFunc func
// which will always return the specific results.
func (m *MockHTTPClient) CreateGetFunc(code int, res []byte, e error) getFunc {
	return func(string, time.Duration) (int, []byte, error) {
		return code, res, e
	}
}

// CreatePostJSONWithHeadersFunc returns a mock postJSONWithHeadersFunc func
// which will always return the specific results.
func (m *MockHTTPClient) CreatePostJSONWithHeadersFunc(code int, res []byte, e error) postJSONWithHeadersFunc {
	return func(string, map[string]string, interface{}, time.Duration) (int, []byte, error) {
		return code, res, e
	}
}

// CreateGetWithHeadersFunc returns a mock getWithHeadersFunc func
// which will always return the specific results.
func (m *MockHTTPClient) CreateGetWithHeadersFunc(code int, res []byte, e error) getWithHeadersFunc {
	return func(string, map[string]string, time.Duration) (int, []byte, error) {
		return code, res, e
	}
}
