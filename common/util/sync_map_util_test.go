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

package util

import (
	"sort"

	"github.com/go-check/check"
)

type SyncMapUtilSuite struct{}

func init() {
	check.Suite(&SyncMapUtilSuite{})
}

func (suite *SyncMapUtilSuite) TestListKeyAsStringSlice(c *check.C) {
	mmap := NewSyncMap()
	mmap.Add("aaa", true)
	mmap.Add("bbb", true)
	mmap.Add("111", true)

	result := mmap.ListKeyAsStringSlice()
	sort.Strings(result)
	c.Check(result, check.DeepEquals, []string{"111", "aaa", "bbb"})
}

func (suite *SyncMapUtilSuite) TestListKeyAsIntSlice(c *check.C) {
	mmap := NewSyncMap()
	mmap.Add("aaa", true)
	mmap.Add("1", true)
	mmap.Add("2", true)

	result := mmap.ListKeyAsIntSlice()
	sort.Ints(result)
	c.Check(result, check.DeepEquals, []int{1, 2})
}
