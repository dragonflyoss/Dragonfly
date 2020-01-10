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
	"net/http"
	"net/url"

	"github.com/dragonflyoss/Dragonfly/pkg/util"

	"github.com/go-check/check"
)

func init() {
	check.Suite(&FilterTestSuite{})
}

type FilterTestSuite struct {
}

func (f *FilterTestSuite) TestParseFilter(c *check.C) {
	var cases = []struct {
		name     string
		req      *http.Request
		excepted *PageFilter
		errNil   bool
	}{
		{
			name: "normal test",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "pageNum=1&pageSize=10&sortDirect=ASC&sortKey=aaa",
				},
			},
			excepted: &PageFilter{
				PageNum:    1,
				PageSize:   10,
				SortDirect: "ASC",
				SortKey:    []string{"aaa"},
			},
			errNil: true,
		},
		{
			name: "err pageNum test",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "pageNum=-1&pageSize=10&sortDirect=ASC",
				},
			},
			excepted: nil,
			errNil:   false,
		},
		{
			name: "err pageSize test",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "pageNum=1&pageSize=-1&sortDirect=ASC",
				},
			},
			excepted: nil,
			errNil:   false,
		},
		{
			name: "err sortDirect test",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "pageNum=1&pageSize=10&sortDirect=ACS",
				},
			},
			excepted: nil,
			errNil:   false,
		},
		{
			name: "err sortKey test",
			req: &http.Request{
				URL: &url.URL{
					RawQuery: "pageNum=1&pageSize=10&sortDirect=ACS&sortKey=ccc",
				},
			},
			excepted: nil,
			errNil:   false,
		},
	}

	for _, v := range cases {
		result, err := ParseFilter(v.req, map[string]bool{
			"aaa": true,
			"bbb": true,
			"ccc": false,
		})
		c.Check(result, check.DeepEquals, v.excepted)
		c.Check(util.IsNil(err), check.Equals, v.errNil)
	}
}

func (f *FilterTestSuite) TestStoi(c *check.C) {
	var cases = []struct {
		str      string
		excepted int
		errNil   bool
	}{
		{"", 0, true},
		{"0", 0, true},
		{"25", 25, true},
		{"aaa", -1, false},
	}

	for _, v := range cases {
		result, err := stoi(v.str)
		c.Check(result, check.Equals, v.excepted)
		c.Check(util.IsNil(err), check.Equals, v.errNil)
	}
}

func (f *FilterTestSuite) TestIsDESC(c *check.C) {
	var cases = []struct {
		str      string
		excepted bool
	}{
		{"desc", true},
		{"dEsc", true},
		{"DESC", true},
		{"dsce", false},
	}

	for _, v := range cases {
		result := IsDESC(v.str)
		c.Check(result, check.Equals, v.excepted)
	}
}
