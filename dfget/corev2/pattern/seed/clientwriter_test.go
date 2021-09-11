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

package seed

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/protocol"

	"github.com/go-check/check"
)

func newSeedData(data []byte, direct bool) (protocol.DistributionData, error) {
	return NewSeedData(ioutil.NopCloser(bytes.NewReader(data)), int64(len(data)), direct)
}

func generateBytes(str string, repeat int) []byte {
	result := make([]byte, len(str)*repeat)
	count := len(str)
	b := []byte(str)
	for i := 0; i < repeat; i++ {
		copy(result[i*count:], b)
	}

	return result
}

type nopWriteCloser struct {
	io.Writer
	closed bool
}

func (nw *nopWriteCloser) Close() error {
	nw.closed = true
	return nil
}

func (suite *seedSuite) TestClientWriter(c *check.C) {
	bw := bytes.NewBuffer(nil)
	nwc := &nopWriteCloser{Writer: bw}
	cw := NewClientWriter()

	data := generateBytes("abcde", 5)
	sData, err := newSeedData(data, true)
	c.Assert(err, check.IsNil)

	notify, err := cw.Run(context.Background(), nwc)
	c.Assert(err, check.IsNil)

	err = cw.PutData(sData)
	c.Assert(err, check.IsNil)

	err = cw.PutData(protocol.NewEoFDistributionData())
	c.Assert(err, check.IsNil)

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	finish := false

	select {
	case <-timer.C:
		c.Fatalf("expect not timeout")
	case <-notify.Done():
		finish = true
	}

	c.Assert(notify.Result().Success(), check.Equals, true)
	err = notify.Result().Error()
	c.Assert(err, check.IsNil)
	c.Assert(nwc.closed, check.Equals, true)
	c.Assert(finish, check.Equals, true)
	c.Assert(bw.String(), check.Equals, string(data))
}
