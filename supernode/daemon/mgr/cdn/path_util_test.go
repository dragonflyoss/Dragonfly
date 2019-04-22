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

package cdn

import (
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&CDNPathUtilTestSuite{})
}

type CDNPathUtilTestSuite struct {
}

func (s *CDNPathUtilTestSuite) TestGetDownloadPath(c *check.C) {
	taskID := "00c4e7b174af7ed61c414b36ef82810ac0c98142c03e5748c00e1d1113f3c882"

	c.Check(getDownloadPath(taskID), check.Equals,
		"/repo/download/00c/00c4e7b174af7ed61c414b36ef82810ac0c98142c03e5748c00e1d1113f3c882")

	c.Check(getUploadPath(taskID), check.Equals,
		"/repo/qtdown/00c/00c4e7b174af7ed61c414b36ef82810ac0c98142c03e5748c00e1d1113f3c882")

	c.Check(getMetaDataPath(taskID), check.Equals,
		"/repo/download/00c/00c4e7b174af7ed61c414b36ef82810ac0c98142c03e5748c00e1d1113f3c882.meta")

	c.Check(getMd5DataPath(taskID), check.Equals,
		"/repo/download/00c/00c4e7b174af7ed61c414b36ef82810ac0c98142c03e5748c00e1d1113f3c882.md5")

	c.Check(getHTTPPathStr(taskID), check.Equals,
		"qtdown/00c/00c4e7b174af7ed61c414b36ef82810ac0c98142c03e5748c00e1d1113f3c882")
}
