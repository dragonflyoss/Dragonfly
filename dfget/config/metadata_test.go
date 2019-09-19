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
	"io/ioutil"
	"os"

	"github.com/go-check/check"
)

func (suite *ConfigSuite) TestMetaData(c *check.C) {
	tmp, _ := ioutil.TempDir("/tmp", "dfget-TestMetaData-")
	defer os.RemoveAll(tmp)

	var cases = []struct {
		path string
		port int
		e    *MetaData
	}{
		{path: tmp + "/1", port: 1, e: &MetaData{ServicePort: 1}},
		{path: tmp, port: 1, e: nil},
	}

	for _, v := range cases {
		meta := NewMetaData(v.path)
		meta.ServicePort = v.port
		err := meta.Persist()
		if v.e != nil {
			c.Assert(err, check.IsNil)
			err := meta.Load()
			c.Assert(err, check.IsNil)
			v.e.MetaPath = v.path
			c.Assert(v.e, check.DeepEquals, meta)
		} else {
			c.Assert(err, check.NotNil)
		}
	}

}
