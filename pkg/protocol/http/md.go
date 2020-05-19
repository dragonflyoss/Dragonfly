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
	"net/http"

	"github.com/dragonflyoss/Dragonfly/pkg/protocol"
)

// NewHTTPMetaData generates an instance of protocol.Metadata.
func NewHTTPMetaData() protocol.Metadata {
	return &Headers{
		Header: make(http.Header),
	}
}

// Headers is an implementation of protocol.Metadata.
type Headers struct {
	http.Header
}

func (hd *Headers) Get(key string) (interface{}, error) {
	return hd.Header.Get(key), nil
}

func (hd *Headers) Set(key string, value interface{}) {
	hd.Header.Set(key, value.(string))
}

func (hd *Headers) Del(key string) {
	hd.Header.Del(key)
}

func (hd *Headers) All() interface{} {
	return hd.Header
}
