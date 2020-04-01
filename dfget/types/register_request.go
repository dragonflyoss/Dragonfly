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

package types

import (
	"encoding/json"
)

// RegisterRequest contains all the parameters that need to be passed to the
// supernode when registering a downloading task.
type RegisterRequest struct {
	SupernodeIP string   `json:"superNodeIp"`
	RawURL      string   `json:"rawUrl"`
	TaskURL     string   `json:"taskUrl"`
	Cid         string   `json:"cid"`
	IP          string   `json:"ip"`
	HostName    string   `json:"hostName"`
	Port        int      `json:"port"`
	Path        string   `json:"path"`
	Version     string   `json:"version,omitempty"`
	Md5         string   `json:"md5,omitempty"`
	Identifier  string   `json:"identifier,omitempty"`
	CallSystem  string   `json:"callSystem,omitempty"`
	Headers     []string `json:"headers,omitempty"`
	Dfdaemon    bool     `json:"dfdaemon,omitempty"`
	Insecure    bool     `json:"insecure,omitempty"`
	RootCAs     [][]byte `json:"rootCAs,omitempty"`
	TaskID      string   `json:"taskId,omitempty"`
	FileLength  int64    `json:"fileLength,omitempty"`
	AsSeed      bool     `json:"asSeed,omitempty"`
}

func (r *RegisterRequest) String() string {
	if b, e := json.Marshal(r); e == nil {
		return string(b)
	}
	return ""
}
