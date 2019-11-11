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
	"testing"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&StoreTestSuite{})
}

type StoreTestSuite struct {
}

type SortStruct struct {
	intField    int
	stringField string
}

func (s *StoreTestSuite) TestStore(c *check.C) {
	store := NewStore()

	cases := []SortStruct{
		{1, "c"},
		{3, "a"},
		{2, "b"},
	}

	for _, v := range cases {
		err := store.Put(v.stringField, v)
		c.Check(err, check.IsNil)

		value, err := store.Get(v.stringField)
		c.Check(err, check.IsNil)
		ss, ok := value.(SortStruct)
		c.Check(ok, check.Equals, true)
		c.Check(ss, check.DeepEquals, v)
	}

	result := store.List()
	pageResult := GetPageValues(result, 0, 0, func(i, j int) bool {
		tempA := result[i].(SortStruct)
		tempB := result[j].(SortStruct)
		return tempA.intField < tempB.intField
	})
	c.Check(pageResult, check.DeepEquals, []interface{}{
		SortStruct{1, "c"},
		SortStruct{2, "b"},
		SortStruct{3, "a"},
	})

	pageResult = GetPageValues(result, 0, 0, func(i, j int) bool {
		tempA := result[i].(SortStruct)
		tempB := result[j].(SortStruct)
		return tempA.stringField < tempB.stringField
	})
	c.Check(pageResult, check.DeepEquals, []interface{}{
		SortStruct{3, "a"},
		SortStruct{2, "b"},
		SortStruct{1, "c"},
	})

	pageResult = GetPageValues(result, 0, 2, func(i, j int) bool {
		tempA := result[i].(SortStruct)
		tempB := result[j].(SortStruct)
		return tempA.intField < tempB.intField
	})
	c.Check(pageResult, check.DeepEquals, []interface{}{
		SortStruct{1, "c"},
		SortStruct{2, "b"},
	})

	pageResult = GetPageValues(result, 1, 2, func(i, j int) bool {
		tempA := result[i].(SortStruct)
		tempB := result[j].(SortStruct)
		return tempA.intField < tempB.intField
	})
	c.Check(pageResult, check.DeepEquals, []interface{}{
		SortStruct{3, "a"},
	})

	for _, v := range cases {
		err := store.Delete(v.stringField)
		c.Check(err, check.IsNil)
		_, err = store.Get(v.stringField)
		c.Check(errortypes.IsDataNotFound(err), check.Equals, true)
	}
}
