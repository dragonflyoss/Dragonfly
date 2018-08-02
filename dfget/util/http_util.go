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
	"encoding/json"
	"time"

	"github.com/valyala/fasthttp"
)

/* http content types */
const (
	ApplicationJSONUtf8Value = "application/json;charset=utf-8"
)

// PostJSON send a POST request whose content-type is 'application/json;charset=utf-8'.
func PostJSON(url string, body interface{}, timeout time.Duration) (
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
