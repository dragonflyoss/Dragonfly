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

package syncmap

import (
	"sort"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/atomiccount"

	"github.com/go-check/check"
	"github.com/willf/bitset"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

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

func (suite *SyncMapUtilSuite) TestRemove(c *check.C) {
	mmap := NewSyncMap()
	mmap.Add("aaa", true)
	mmap.Add("bbb", true)
	mmap.Add("ccc", true)

	mmap.Remove("ccc")

	result := mmap.ListKeyAsStringSlice()
	sort.Strings(result)
	c.Check(result, check.DeepEquals, []string{"aaa", "bbb"})
}

func (suite *SyncMapUtilSuite) TestGetAsInt(c *check.C) {
	mmap := NewSyncMap()
	mmap.Add("aaa", 111)

	result, _ := mmap.GetAsInt("aaa")
	c.Check(result, check.DeepEquals, 111)
}

func (suite *SyncMapUtilSuite) TestGetAsString(c *check.C) {
	mmap := NewSyncMap()
	mmap.Add("aaa", "value")

	result, _ := mmap.GetAsString("aaa")
	c.Check(result, check.DeepEquals, "value")
}

func (suite *SyncMapUtilSuite) TestGetAsBool(c *check.C) {
	mmap := NewSyncMap()
	mmap.Add("aaa", true)

	result, _ := mmap.GetAsBool("aaa")
	c.Check(result, check.DeepEquals, true)
}

func (suite *SyncMapUtilSuite) TestGetAsMap(c *check.C) {
	expected := NewSyncMap()
	expected.Add("expected", true)

	mmap := NewSyncMap()
	mmap.Add("aaa", expected)

	result, _ := mmap.GetAsMap("aaa")
	c.Check(result, check.DeepEquals, expected)
}

func (suite *SyncMapUtilSuite) TestGetAsBitset(c *check.C) {
	expected := bitset.New(111)
	mmap := NewSyncMap()
	mmap.Add("aaa", expected)

	result, _ := mmap.GetAsBitset("aaa")
	c.Check(result, check.DeepEquals, expected)
}

func (suite *SyncMapUtilSuite) TestGetAsInt64(c *check.C) {
	expected := int64(111)
	mmap := NewSyncMap()
	mmap.Add("aaa", expected)

	result, _ := mmap.GetAsInt64("aaa")
	c.Check(result, check.DeepEquals, expected)
}

func (suite *SyncMapUtilSuite) TestGetAsTime(c *check.C) {
	expected := time.Now()
	mmap := NewSyncMap()
	mmap.Add("aaa", expected)

	result, _ := mmap.GetAsTime("aaa")
	c.Check(result, check.DeepEquals, expected)
}

func (suite *SyncMapUtilSuite) TestGetAsAtomicInt(c *check.C) {
	expected := atomiccount.NewAtomicInt(10)
	mmap := NewSyncMap()
	mmap.Add("aaa", expected)

	result, _ := mmap.GetAsAtomicInt("aaa")
	c.Check(result, check.DeepEquals, expected)

	result, err := mmap.GetAsAtomicInt("nonexist")
	c.Check(err, check.NotNil)
	c.Check(result, check.IsNil)
}
