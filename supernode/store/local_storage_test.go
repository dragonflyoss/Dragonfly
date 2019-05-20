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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
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
				Config:  "baseDir: " + path.Join(s.workHome, "repo"),
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
		putRaw      *Raw
		getRaw      *Raw
		data        []byte
		getErrCheck func(error) bool
		expected    string
	}{
		{
			putRaw: &Raw{
				Key: "foo1",
			},
			getRaw: &Raw{
				Key: "foo1",
			},
			data:        []byte("hello foo"),
			getErrCheck: IsNilError,
			expected:    "hello foo",
		},
		{
			putRaw: &Raw{
				Key: "foo2",
			},
			getRaw: &Raw{
				Key:    "foo2",
				Offset: 0,
				Length: 5,
			},
			getErrCheck: IsNilError,
			data:        []byte("hello foo"),
			expected:    "hello",
		},
		{
			putRaw: &Raw{
				Key: "foo3",
			},
			getRaw: &Raw{
				Key:    "foo3",
				Offset: 0,
				Length: -1,
			},
			getErrCheck: IsInvalidValue,
			data:        []byte("hello foo"),
			expected:    "",
		},
		{
			putRaw: &Raw{
				Bucket: "download",
				Key:    "foo0/foo.txt",
			},
			getRaw: &Raw{
				Bucket: "download",
				Key:    "foo0/foo.txt",
			},
			data:        []byte("hello foo"),
			getErrCheck: IsNilError,
			expected:    "hello foo",
		},
	}

	for _, v := range cases {
		// put
		err := s.storeLocal.PutBytes(context.Background(), v.putRaw, v.data)
		c.Assert(err, check.IsNil)

		// get
		result, err := s.storeLocal.GetBytes(context.Background(), v.getRaw)
		c.Assert(v.getErrCheck(err), check.Equals, true)
		if err == nil {
			c.Assert(string(result), check.Equals, v.expected)
		}

		// stat
		s.checkStat(v.putRaw, c)

		// remove
		s.checkRemove(v.putRaw, c)
	}

}

func (s *LocalStorageSuite) TestGetPut(c *check.C) {
	var cases = []struct {
		putRaw      *Raw
		getRaw      *Raw
		data        io.Reader
		getErrCheck func(error) bool
		expected    string
	}{
		{
			putRaw: &Raw{
				Key: "foo1.meta",
			},
			getRaw: &Raw{
				Key: "foo1.meta",
			},
			data:        strings.NewReader("hello meta file"),
			getErrCheck: IsNilError,
			expected:    "hello meta file",
		},
		{
			putRaw: &Raw{
				Key: "foo2.meta",
			},
			getRaw: &Raw{
				Key:    "foo2.meta",
				Offset: 2,
				Length: 5,
			},
			data:        strings.NewReader("hello meta file"),
			getErrCheck: IsNilError,
			expected:    "llo m",
		},
		{
			putRaw: &Raw{
				Key: "foo3.meta",
			},
			getRaw: &Raw{
				Key:    "foo3.meta",
				Offset: 2,
				Length: -1,
			},
			getErrCheck: IsInvalidValue,
			data:        strings.NewReader("hello meta file"),
			expected:    "llo meta file",
		},
		{
			putRaw: &Raw{
				Key: "foo4.meta",
			},
			getRaw: &Raw{
				Key:    "foo4.meta",
				Offset: 30,
				Length: 5,
			},
			getErrCheck: IsRangeNotSatisfiable,
			data:        strings.NewReader("hello meta file"),
			expected:    "",
		},
	}

	for _, v := range cases {
		// put
		s.storeLocal.Put(context.Background(), v.putRaw, v.data)

		// get
		r, err := s.storeLocal.Get(context.Background(), v.getRaw)
		c.Check(v.getErrCheck(err), check.Equals, true)
		if err == nil {
			result, err := ioutil.ReadAll(r)
			c.Assert(err, check.IsNil)
			c.Assert(string(result[:]), check.Equals, v.expected)
		}

		// stat
		s.checkStat(v.putRaw, c)

		// remove
		s.checkRemove(v.putRaw, c)
	}

}

func (s *LocalStorageSuite) TestPutParallel(c *check.C) {
	var key = "fooPutParallel"
	var routineCount = 4
	var testStr = "hello"
	var testStrLength = len(testStr)

	var wg sync.WaitGroup
	for k := 0; k < routineCount; k++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.storeLocal.Put(context.TODO(), &Raw{
				Key:    key,
				Offset: int64(i) * int64(testStrLength),
			}, strings.NewReader(testStr))
		}(k)
	}
	wg.Wait()

	info, err := s.storeLocal.Stat(context.TODO(), &Raw{Key: key})
	c.Check(err, check.IsNil)
	c.Check(info.Size, check.Equals, int64(routineCount)*int64(testStrLength))
}

func (s *LocalStorageSuite) BenchmarkPutParallel(c *check.C) {
	var wg sync.WaitGroup
	for k := 0; k < c.N; k++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.storeLocal.Put(context.Background(), &Raw{
				Key:    "foo.bech",
				Offset: int64(i) * 5,
			}, strings.NewReader("hello"))
		}(k)
	}
	wg.Wait()
}

func (s *LocalStorageSuite) BenchmarkPutSerial(c *check.C) {
	for k := 0; k < c.N; k++ {
		s.storeLocal.Put(context.Background(), &Raw{
			Key:    "foo1.bech",
			Offset: int64(k) * 5,
		}, strings.NewReader("hello"))
	}
}

// helper function

func (s *LocalStorageSuite) checkStat(raw *Raw, c *check.C) {
	info, err := s.storeLocal.Stat(context.Background(), raw)
	c.Assert(IsNilError(err), check.Equals, true)

	driver := s.storeLocal.driver.(*localStorage)
	pathTemp := path.Join(driver.BaseDir, raw.Bucket, raw.Key)
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
	c.Assert(IsNilError(err), check.Equals, true)

	_, err = s.storeLocal.Stat(context.Background(), raw)
	c.Assert(IsKeyNotFound(err), check.Equals, true)
}
