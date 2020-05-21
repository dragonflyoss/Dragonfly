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

package config

import "github.com/dragonflyoss/Dragonfly/pkg/constants"

//type ReportRequest struct {
//	RawURL     string   `json:"rawUrl"`
//	TaskURL    string   `json:"taskUrl"`
//	Cid        string   `json:"cid"`
//	IP         string   `json:"ip"`
//	HostName   string   `json:"hostName"`
//	Port       int      `json:"port"`
//	Path       string   `json:"path"`
//	Version    string   `json:"version,omitempty"`
//	Md5        string   `json:"md5,omitempty"`
//	Identifier string   `json:"identifier,omitempty"`
//	CallSystem string   `json:"callSystem,omitempty"`
//	Headers    []string `json:"headers,omitempty"`
//	Dfdaemon   bool     `json:"dfdaemon,omitempty"`
//	Insecure   bool     `json:"insecure,omitempty"`
//	RootCAs    [][]byte `json:"rootCAs,omitempty"`
//	TaskID     string   `json:"taskId,omitempty"`
//	FileLength int64    `json:"fileLength,omitempty"`
//	AsSeed     bool     `json:"asSeed,omitempty"`
//}

type ReportResponse struct {
	AsSeed bool
	Code   int
}

func (resp *ReportResponse) Success() bool {
	return resp.Code == constants.Success
}

func (resp *ReportResponse) Data() interface{} {
	return resp
}
