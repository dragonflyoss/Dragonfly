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

package types

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/constants"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type TypesSuite struct{}

func init() {
	check.Suite(&TypesSuite{})
}

func (suite *TypesSuite) SetUpTest(c *check.C) {
	rand.Seed(time.Now().UnixNano())
}

// ----------------------------------------------------------------------------
// Testing BaseResponse

func (suite *TypesSuite) TestNewBaseResponse(c *check.C) {
	code := rand.Intn(100)
	msg := strconv.Itoa(rand.Int())
	res := NewBaseResponse(code, msg)
	c.Assert(res.Code, check.Equals, code)
	c.Assert(res.Msg, check.Equals, msg)
}

func (suite *TypesSuite) TestBaseResponse_IsSuccess(c *check.C) {
	var cases = []struct {
		code     int
		expected bool
	}{
		// [1]
		{1, true},
		// [2, n)
		{rand.Intn(10000) + 2, false},
		// (-n, 0]
		{-rand.Intn(10000), false},
	}

	var res *BaseResponse
	for _, cc := range cases {
		res = NewBaseResponse(cc.code, "")
		c.Assert(res.IsSuccess(), check.Equals, cc.expected)
	}
}

// ----------------------------------------------------------------------------
// Testing PullPieceTaskResponse

func (suite *TypesSuite) TestPullPieceTaskResponse_FinishData(c *check.C) {
	res := &PullPieceTaskResponse{BaseResponse: &BaseResponse{}}

	c.Assert(res.FinishData(), check.IsNil)

	res.Code = constants.CodePeerFinish
	c.Assert(res.FinishData(), check.IsNil)

	res.Data = []byte("x")
	c.Assert(res.FinishData(), check.IsNil)

	res.Data = []byte("{\"fileLength\":1}")
	c.Assert(res.FinishData(), check.NotNil)
	c.Assert(res.FinishData().FileLength, check.Equals, int64(1))
	c.Assert(strings.Index(res.FinishData().String(), "\"fileLength\":1") > 0,
		check.Equals, true)
}

func (suite *TypesSuite) TestPullPieceTaskResponse_ContinueData(c *check.C) {
	res := &PullPieceTaskResponse{BaseResponse: &BaseResponse{}}

	c.Assert(res.ContinueData(), check.IsNil)

	res.Code = constants.CodePeerContinue
	c.Assert(res.ContinueData(), check.IsNil)

	res.Data = []byte("x")
	c.Assert(res.ContinueData(), check.IsNil)

	res.Data = []byte("[{\"pieceNum\":1}]")
	c.Assert(res.ContinueData(), check.NotNil)
	c.Assert(len(res.ContinueData()), check.Equals, 1)
	c.Assert(res.ContinueData()[0].PieceNum, check.Equals, 1)
	c.Assert(strings.Index(res.ContinueData()[0].String(), "\"pieceNum\":1") > 0,
		check.Equals, true)
}
