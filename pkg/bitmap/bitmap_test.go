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

package bitmap

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type BitMapSuite struct {
	tmpDir string
}

func init() {
	check.Suite(&BitMapSuite{})
}

func (suite *BitMapSuite) SetUpSuite(c *check.C) {
	suite.tmpDir = "./testdata"
	err := os.MkdirAll(suite.tmpDir, 0774)
	c.Assert(err, check.IsNil)
}

func (suite *BitMapSuite) TearDownSuite(c *check.C) {
	if suite.tmpDir != "" {
		os.RemoveAll(suite.tmpDir)
	}
}

func (suite *BitMapSuite) TestBitMap(c *check.C) {
	// bits are in [0, 100 * 64 - 1]
	bm, err := NewBitMap(100, false)
	c.Assert(err, check.IsNil)
	err = bm.Set(0, 100, true)
	c.Assert(err, check.IsNil)

	var i uint32
	fmt.Printf("bp1, bm[0]: %x, bm[1]: %x\n", bm.bm[0], bm.bm[1])
	for i = 0; i <= 100; i++ {
		rs, err := bm.Get(uint32(i), uint32(i), true)
		c.Assert(err, check.IsNil)
		c.Assert(len(rs), check.Equals, 1)
		c.Assert(rs[0].StartIndex, check.Equals, i)
		c.Assert(rs[0].EndIndex, check.Equals, i)
	}

	var start, end uint32

	// random 10000 to set [start, end]
	for i = 0; i <= 1000; i++ {
		n1 := uint32(rand.Int31n(101))
		n2 := uint32(rand.Int31n(101))
		if n1 < n2 {
			start = n1
			end = n2
		} else {
			start = n2
			end = n1
		}

		rs, err := bm.Get(uint32(start), uint32(end), true)
		c.Assert(err, check.IsNil)
		c.Assert(len(rs), check.Equals, 1)
		c.Assert(rs[0].StartIndex, check.Equals, start)
		c.Assert(rs[0].EndIndex, check.Equals, end)
	}

	err = bm.Set(200, 300, true)
	c.Assert(err, check.IsNil)
	rs, err := bm.Get(0, 250, true)
	c.Assert(err, check.IsNil)
	c.Assert(len(rs), check.Equals, 2)
	c.Assert(rs[0].StartIndex, check.Equals, uint32(0))
	c.Assert(rs[0].EndIndex, check.Equals, uint32(100))
	c.Assert(rs[1].StartIndex, check.Equals, uint32(200))
	c.Assert(rs[1].EndIndex, check.Equals, uint32(250))

	rs, err = bm.Get(0, 250, false)
	c.Assert(err, check.IsNil)
	c.Assert(len(rs), check.Equals, 1)
	c.Assert(rs[0].StartIndex, check.Equals, uint32(101))
	c.Assert(rs[0].EndIndex, check.Equals, uint32(199))

	err = bm.Set(100, 200, false)
	c.Assert(err, check.IsNil)
	rs, err = bm.Get(0, 250, false)
	c.Assert(err, check.IsNil)
	c.Assert(len(rs), check.Equals, 1)
	c.Assert(rs[0].StartIndex, check.Equals, uint32(100))
	c.Assert(rs[0].EndIndex, check.Equals, uint32(200))

	err = bm.Set(300, 303, true)
	c.Assert(err, check.IsNil)
	fmt.Printf("index: %d, bp1, bm[4]: %x, bm[5]: %x\n", -1, bm.bm[4], bm.bm[5])
	rs, err = bm.Get(200, 400, true)
	c.Assert(err, check.IsNil)
	c.Assert(len(rs), check.Equals, 1)
	c.Assert(rs[0].StartIndex, check.Equals, uint32(201))
	c.Assert(rs[0].EndIndex, check.Equals, uint32(303))

	err = bm.Set(305, 308, true)
	c.Assert(err, check.IsNil)
	fmt.Printf("index: %d, bp1, bm[4]: %x, bm[5]: %x\n", -2, bm.bm[4], bm.bm[5])
	rs, err = bm.Get(200, 400, true)
	c.Assert(err, check.IsNil)
	c.Assert(len(rs), check.Equals, 2)
	c.Assert(rs[0].StartIndex, check.Equals, uint32(201))
	c.Assert(rs[0].EndIndex, check.Equals, uint32(303))
	c.Assert(rs[1].StartIndex, check.Equals, uint32(305))
	c.Assert(rs[1].EndIndex, check.Equals, uint32(308))

	err = bm.Set(310, 313, true)
	c.Assert(err, check.IsNil)
	fmt.Printf("index: %d, bp1, bm[4]: %x, bm[5]: %x\n", -3, bm.bm[4], bm.bm[5])
	rs, err = bm.Get(200, 400, true)
	c.Assert(err, check.IsNil)
	c.Assert(len(rs), check.Equals, 3)
	c.Assert(rs[0].StartIndex, check.Equals, uint32(201))
	c.Assert(rs[0].EndIndex, check.Equals, uint32(303))
	c.Assert(rs[1].StartIndex, check.Equals, uint32(305))
	c.Assert(rs[1].EndIndex, check.Equals, uint32(308))
	c.Assert(rs[2].StartIndex, check.Equals, uint32(310))
	c.Assert(rs[2].EndIndex, check.Equals, uint32(313))

	err = bm.Set(315, 318, true)
	c.Assert(err, check.IsNil)
	fmt.Printf("index: %d, bp1, bm[4]: %x, bm[5]: %x\n", -4, bm.bm[4], bm.bm[5])
	rs, err = bm.Get(200, 400, true)
	c.Assert(err, check.IsNil)
	c.Assert(len(rs), check.Equals, 4)
	c.Assert(rs[0].StartIndex, check.Equals, uint32(201))
	c.Assert(rs[0].EndIndex, check.Equals, uint32(303))
	c.Assert(rs[1].StartIndex, check.Equals, uint32(305))
	c.Assert(rs[1].EndIndex, check.Equals, uint32(308))
	c.Assert(rs[2].StartIndex, check.Equals, uint32(310))
	c.Assert(rs[2].EndIndex, check.Equals, uint32(313))
	c.Assert(rs[3].StartIndex, check.Equals, uint32(315))
	c.Assert(rs[3].EndIndex, check.Equals, uint32(318))

	for i := 0; i < 50; i++ {
		err = bm.Set(uint32(300+i*5), uint32(300+i*5+3), true)
		c.Assert(err, check.IsNil)
	}

	rs, err = bm.Get(0, 1000, true)
	c.Assert(err, check.IsNil)
	c.Check(len(rs), check.Equals, 51)
	c.Check(rs[0].StartIndex, check.Equals, uint32(0))
	c.Check(rs[0].EndIndex, check.Equals, uint32(99))
	c.Check(rs[1].StartIndex, check.Equals, uint32(201))
	c.Check(rs[1].EndIndex, check.Equals, uint32(303))

	for i := 0; i < 49; i++ {
		c.Check(rs[i+2].StartIndex, check.Equals, uint32(305+i*5))
		c.Check(rs[i+2].EndIndex, check.Equals, uint32(308+i*5))
	}
}

func (suite *BitMapSuite) TestRestoreBitMap(c *check.C) {
	bm, err := NewBitMap(100, false)
	c.Assert(err, check.IsNil)
	err = bm.Set(0, 10, true)
	c.Assert(err, check.IsNil)
	err = bm.Set(100, 200, true)
	c.Assert(err, check.IsNil)
	res, err := bm.Get(0, 100*64-1, true)
	fmt.Println(res)
	c.Assert(err, check.IsNil)
	c.Assert(len(res), check.Equals, 2)
	c.Assert(res[0].StartIndex, check.Equals, uint32(0))
	c.Assert(res[0].EndIndex, check.Equals, uint32(10))
	c.Assert(res[1].StartIndex, check.Equals, uint32(100))
	c.Assert(res[1].EndIndex, check.Equals, uint32(200))

	bmPath := filepath.Join(suite.tmpDir, "TestRestoreBitMap.bits")
	data := bm.Encode()

	err = ioutil.WriteFile(bmPath, data, 0644)
	c.Assert(err, check.IsNil)

	readData, err := ioutil.ReadFile(bmPath)
	c.Assert(err, check.IsNil)
	bm1, err := RestoreBitMap(readData)
	c.Assert(err, check.IsNil)
	c.Assert(bm1.maxBitIndex, check.Equals, uint32(100*64-1))
	res, err = bm1.Get(0, 100*64-1, true)
	c.Assert(err, check.IsNil)
	c.Assert(len(res), check.Equals, 2)
	c.Assert(res[0].StartIndex, check.Equals, uint32(0))
	c.Assert(res[0].EndIndex, check.Equals, uint32(10))
	c.Assert(res[1].StartIndex, check.Equals, uint32(100))
	c.Assert(res[1].EndIndex, check.Equals, uint32(200))
}

func (suite *BitMapSuite) TestInvalidBitMap(c *check.C) {
	_, err := NewBitMap(sizeOf64BitsLimit+1, false)
	c.Assert(err, check.NotNil)

	bm1, err := NewBitMap(100000, true)
	c.Assert(err, check.IsNil)

	err = bm1.Set(0, bm1.maxBitIndex, true)
	c.Assert(err, check.IsNil)

	err = bm1.Set(1, 1, true)
	c.Assert(err, check.IsNil)

	err = bm1.Set(1000, 0, true)
	c.Assert(err, check.NotNil)

	err = bm1.Set(1000, bm1.maxBitIndex+1, true)
	c.Assert(err, check.NotNil)

	_, err = bm1.Get(0, bm1.maxBitIndex, true)
	c.Assert(err, check.IsNil)

	_, err = bm1.Get(1, 1, true)
	c.Assert(err, check.IsNil)

	_, err = bm1.Get(1000, 0, true)
	c.Assert(err, check.NotNil)

	_, err = bm1.Get(1000, bm1.maxBitIndex+1, true)
	c.Assert(err, check.NotNil)

	// test restore
	data := make([]byte, sizeOf64BitsLimit<<3)
	_, err = RestoreBitMap(data)
	c.Assert(err, check.IsNil)

	data1 := make([]byte, (sizeOf64BitsLimit<<3)+1)
	_, err = RestoreBitMap(data1)
	c.Assert(err, check.NotNil)

}
