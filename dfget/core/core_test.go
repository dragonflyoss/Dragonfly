/*
 * Copyright 1999-2018 Alibaba Group.
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

package core

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/alibaba/Dragonfly/dfget/config"
	"github.com/go-check/check"
	"github.com/sirupsen/logrus"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type CoreTestSuite struct {
	workHome string
}

func init() {
	check.Suite(&CoreTestSuite{})
}

func (s *CoreTestSuite) SetUpSuite(c *check.C) {
	s.workHome, _ = ioutil.TempDir("/tmp", "dfget-CoreTestSuite-")
}

func (s *CoreTestSuite) TearDownSuite(c *check.C) {
	if s.workHome != "" {
		if err := os.RemoveAll(s.workHome); err != nil {
			fmt.Printf("remove path:%s error", s.workHome)
		}
	}
}

func (s *CoreTestSuite) TestPrepare(c *check.C) {
	buf := &bytes.Buffer{}
	ctx := s.createContext(buf)

	err := prepare(ctx)
	fmt.Printf("%s\nerror:%v", buf.String(), err)
}

func (s *CoreTestSuite) createContext(writer io.Writer) *config.Context {
	if writer == nil {
		writer = &bytes.Buffer{}
	}
	ctx := config.NewContext()
	ctx.WorkHome = s.workHome
	ctx.MetaPath = path.Join(ctx.WorkHome, "meta", "host.meta")
	ctx.SystemDataDir = path.Join(ctx.WorkHome, "data")

	logrus.StandardLogger().Out = writer
	ctx.ClientLogger = logrus.StandardLogger()
	ctx.ServerLogger = logrus.StandardLogger()
	return ctx
}
