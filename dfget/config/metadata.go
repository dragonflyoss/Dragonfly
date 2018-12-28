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

import (
	"encoding/json"
	"io/ioutil"
)

// NewMetaData create a MetaData instance.
func NewMetaData(metaPath string) *MetaData {
	return &MetaData{
		metaPath: metaPath,
	}
}

// MetaData stores meta information that should be persisted.
type MetaData struct {
	ServicePort int `json:"servicePort"`

	metaPath string `json:"-"`
}

// Persist writes meta information into storage.
func (md *MetaData) Persist() error {
	if content, err := json.Marshal(md); err == nil {
		return ioutil.WriteFile(md.metaPath, content, 0755)
	} else {
		return err
	}
}

// Load loads meta information from storage.
func (md *MetaData) Load() error {
	if content, err := ioutil.ReadFile(md.metaPath); err == nil {
		return json.Unmarshal(content, md)
	} else {
		return err
	}
}
