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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (transFunc transportFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return transFunc(req)
}

func newMockClient(handler func(*http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{
		Transport: transportFunc(handler),
	}
}

func errorMockResponse(statusCode int, message string) func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		header := http.Header{}
		header.Set("Content-Type", "application/json")

		body, err := json.Marshal(&types.Error{
			Message: message,
		})
		if err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode: statusCode,
			Body:       ioutil.NopCloser(bytes.NewReader(body)),
			Header:     header,
		}, nil
	}
}
