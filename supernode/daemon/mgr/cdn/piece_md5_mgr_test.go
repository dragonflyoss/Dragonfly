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
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/go-check/check"
)

type PieceMD5MgrTestSuite struct {
}

func init() {
	check.Suite(&PieceMD5MgrTestSuite{})
}

func (s *PieceMD5MgrTestSuite) TestPieceMD5(c *check.C) {
	mgr := newpieceMD5Mgr()
	taskID := "fooTaskID"
	pieceMd5s := map[int]string{
		0:  "foo-md5-0",
		1:  "foo-md5-1",
		5:  "foo-md5-5",
		10: "foo-md5-10",
	}

	for k, v := range pieceMd5s {
		err := mgr.setPieceMD5(taskID, k, v)
		c.Check(err, check.IsNil)
		pieceMD5, err := mgr.getPieceMD5(taskID, k)
		c.Check(err, check.IsNil)
		c.Check(pieceMD5, check.Equals, v)
	}

	_, err := mgr.getPieceMD5(taskID, 1000)
	c.Check(errortypes.IsDataNotFound(err), check.Equals, true)

	pieceMD5s, err := mgr.getPieceMD5sByTaskID(taskID)
	c.Check(err, check.IsNil)
	c.Check(pieceMD5s, check.DeepEquals, []string{
		"foo-md5-0",
		"foo-md5-1",
		"foo-md5-5",
		"foo-md5-10",
	})
}
