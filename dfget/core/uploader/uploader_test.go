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

package uploader

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&UploaderTestSuite{})
}

type UploaderTestSuite struct {
}

func (s *UploaderTestSuite) TestIsRunning(c *check.C) {
	p2p = nil
	c.Assert(isRunning(), check.Equals, false)

	p2p = &peerServer{finished: make(chan struct{})}
	c.Assert(isRunning(), check.Equals, true)
	close(p2p.finished)
	c.Assert(isRunning(), check.Equals, false)
	p2p = nil
}

func (s *UploaderTestSuite) TestWaitForShutdown(c *check.C) {
	res := make(chan error)

	p2p = nil
	e := waitForStartup(res)
	c.Assert(e, check.NotNil)

	p2p = &peerServer{finished: make(chan struct{})}
	e = waitForStartup(res)
	c.Assert(e, check.NotNil)

	time.AfterFunc(50*time.Millisecond, func() { res <- nil })
	e = waitForStartup(res)
	c.Assert(e, check.IsNil)

	time.AfterFunc(50*time.Millisecond, func() { res <- fmt.Errorf("test") })
	e = waitForStartup(res)
	c.Assert(e, check.NotNil)
	c.Assert(e.Error(), check.Equals, "test")
}
