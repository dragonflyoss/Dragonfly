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

package api

import (
	"testing"
	"time"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type postJSONFunc = func(url string, body interface{}, timeout time.Duration) (int, []byte, error)
type getFunc = func(url string, timeout time.Duration) (int, []byte, error)

// mockHTTPClient fakes a customized implementation of util.SimpleHTTPClient.
type mockHTTPClient struct {
	postJSON postJSONFunc
	get      getFunc
}

func (m *mockHTTPClient) PostJSON(url string, body interface{}, timeout time.Duration) (
	int, []byte, error) {
	if m.postJSON != nil {
		return m.postJSON(url, body, timeout)
	}
	return 0, nil, nil
}

func (m *mockHTTPClient) Get(url string, timeout time.Duration) (int, []byte, error) {
	if m.get != nil {
		return m.get(url, timeout)
	}
	return 0, nil, nil
}

func (m *mockHTTPClient) reset() {
	m.postJSON = nil
	m.get = nil
}

func (m *mockHTTPClient) createPostJSONFunc(code int, res []byte, e error) postJSONFunc {
	return func(string, interface{}, time.Duration) (int, []byte, error) {
		return code, res, e
	}
}

func (m *mockHTTPClient) createGetFunc(code int, res []byte, e error) getFunc {
	return func(string, time.Duration) (int, []byte, error) {
		return code, res, e
	}
}
