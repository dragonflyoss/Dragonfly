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

package dflog

import (
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

func TestLog(t *testing.T) {
	suite.Run(t, &logTestSuite{})
}

type logTestSuite struct {
	suite.Suite
}

func (ts *logTestSuite) TestDebug() {
	r := ts.Require()

	l := logrus.New()
	r.Nil(Init(l, WithDebug(true)))
	r.Equal(logrus.DebugLevel, l.Level)
}

func (ts *logTestSuite) TestLogFile() {
	r := ts.Require()

	f, err := ioutil.TempFile("", "")
	r.Nil(err)
	defer os.RemoveAll(f.Name())

	l := logrus.New()
	r.Nil(Init(l, WithLogFile(f.Name(), -1, -1)))
	lumberjack := getLumberjack(l)
	r.NotNil(lumberjack)
	r.Equal(f.Name(), lumberjack.Filename)
}

func (ts *logTestSuite) TestMaxSizeMB() {
	r := ts.Require()

	f, err := ioutil.TempFile("", "")
	r.Nil(err)
	defer os.RemoveAll(f.Name())

	l := logrus.New()
	r.Nil(Init(l, WithLogFile(f.Name(), 10, 0), WithMaxSizeMB(20)))
	lumberjack := getLumberjack(l)
	r.NotNil(lumberjack)
	r.Equal(20, lumberjack.MaxSize)
}

func (ts *logTestSuite) TestConsole() {
	r := ts.Require()

	l := logrus.New()
	r.Nil(Init(l, WithConsole()))
	for _, level := range logrus.AllLevels {
		r.Equal(1, len(l.Hooks[level]))
	}
}

func (ts *logTestSuite) TestFormatter() {
	r := ts.Require()

	l := logrus.New()

	sign := uuid.New()
	r.Nil(Init(l, WithSign(sign)))

	buf := bytes.NewBuffer(nil)
	l.SetOutput(buf)

	content := uuid.New()
	l.Infoln(content)

	format := regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}) (\S+) sign:(\S+) : (.+)`)
	match := format.FindAllStringSubmatch(buf.String(), -1)
	r.Len(match, 1)
	r.Len(match[0], 5)
	r.Equal("INFO", match[0][2])
	r.Equal(sign, match[0][3])
	r.Equal(content, match[0][4])
}
