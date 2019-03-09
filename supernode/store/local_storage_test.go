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

package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/plugins"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type LocalStorageSuite struct {
	workHome   string
	storeLocal *Store
}

func init() {
	check.Suite(&LocalStorageSuite{})
}

func (s *LocalStorageSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "supernode-storageDriver-StoreTestSuite-")
	pluginProps := map[config.PluginType][]*config.PluginProperties{
		config.StoragePlugin: {
			&config.PluginProperties{
				Name:    LocalStorageDriver,
				Enabled: true,
				Config:  "baseDir: " + path.Join(s.workHome, "download"),
			},
		},
	}
	cfg := &config.Config{
		Plugins: pluginProps,
	}
	plugins.Initialize(cfg)

	// init StorageManager
	sm, err := NewManager()
	c.Assert(err, check.IsNil)

	// init store with local storage
	s.storeLocal, err = sm.Get(LocalStorageDriver)
	c.Assert(err, check.IsNil)
}

func (s *LocalStorageSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
}

func (s *LocalStorageSuite) TestGetPutBytes(c *check.C) {
	var cases = []struct {
		raw      *Raw
		data     []byte
		expected string
	}{
		{
			raw: &Raw{
				key: "foo1",
			},
			data:     []byte("hello foo"),
			expected: "hello foo",
		},
		{
			raw: &Raw{
				key:    "foo2",
				offset: 0,
				length: 5,
			},
			data:     []byte("hello foo"),
			expected: "hello",
		},
		{
			raw: &Raw{
				key:    "foo3",
				offset: 2,
				length: -1,
			},
			data:     []byte("hello foo"),
			expected: "hello foo",
		},
	}

	for _, v := range cases {
		// put
		s.storeLocal.PutBytes(context.Background(), v.raw, v.data)

		// get
		result, err := s.storeLocal.GetBytes(context.Background(), v.raw)
		c.Assert(err, check.IsNil)
		c.Assert(string(result), check.Equals, v.expected)

		// stat
		s.checkStat(v.raw, c)

		// remove
		s.checkRemove(v.raw, c)
	}

}

func (s *LocalStorageSuite) TestGetPut(c *check.C) {
	var cases = []struct {
		raw      *Raw
		data     io.Reader
		expected string
	}{
		{
			raw: &Raw{
				key: "foo1.meta",
			},
			data:     strings.NewReader("hello meta file"),
			expected: "hello meta file",
		},
		{
			raw: &Raw{
				key:    "foo2.meta",
				offset: 2,
				length: 5,
			},
			data:     strings.NewReader("hello meta file"),
			expected: "hello",
		},
		{
			raw: &Raw{
				key:    "foo3.meta",
				offset: 2,
				length: -1,
			},
			data:     strings.NewReader("hello meta file"),
			expected: "hello meta file",
		},
	}

	for _, v := range cases {
		// put
		s.storeLocal.Put(context.Background(), v.raw, v.data)

		// get
		buf1 := new(bytes.Buffer)
		err := s.storeLocal.Get(context.Background(), v.raw, buf1)
		c.Assert(err, check.IsNil)
		c.Assert(buf1.String(), check.Equals, v.expected)

		// stat
		s.checkStat(v.raw, c)

		// remove
		s.checkRemove(v.raw, c)
	}

}

func (s *LocalStorageSuite) TestGetPrefix(c *check.C) {
	var cases = []struct {
		str      string
		expected string
	}{
		{"foo", "foo"},
		{"footest", "foo"},
		{"fo", "fo"},
	}

	for _, v := range cases {
		result := getPrefix(v.str)
		c.Check(result, check.Equals, v.expected)
	}
}

// helper function

func (s *LocalStorageSuite) checkStat(raw *Raw, c *check.C) {
	info, err := s.storeLocal.Stat(context.Background(), raw)
	c.Assert(err, check.IsNil)

	driver := s.storeLocal.driver.(*localStorage)
	pathTemp := path.Join(driver.BaseDir, getPrefix(raw.key), raw.key)
	f, _ := os.Stat(pathTemp)
	sys, _ := util.GetSys(f)

	c.Assert(info, check.DeepEquals, &StorageInfo{
		Path:       pathTemp,
		Size:       f.Size(),
		ModTime:    f.ModTime(),
		CreateTime: util.Ctime(sys),
	})
}

func (s *LocalStorageSuite) checkRemove(raw *Raw, c *check.C) {
	err := s.storeLocal.Remove(context.Background(), raw)
	c.Assert(err, check.IsNil)

	_, err = s.storeLocal.Stat(context.Background(), raw)
	c.Assert(err, check.DeepEquals, ErrNotFound)
}
