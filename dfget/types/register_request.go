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

package types

// RegisterRequest contains all the parameters that need to be passed to the
// supernode when registering a downloading task.
type RegisterRequest struct {
	RawURL     string `json:"rawUrl"`
	TaskURL    string `json:"taskUrl"`
	Md5        string `json:"md5"`
	Identifier string `json:"identifier"`
	Version    string `json:"version"`
	Port       int    `json:"port"`
	Path       string `json:"path"`
	CallSystem string `json:"callSystem"`
	Cid        string `json:"cid"`
	IP         string `json:"ip"`
	HostName   string `json:"hostName"`
	Headers    string `json:"headers"`
	Dfdaemon   bool   `json:"dfdaemon"`
}
