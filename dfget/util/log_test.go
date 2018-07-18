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

package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"

	"github.com/go-check/check"
	"github.com/sirupsen/logrus"
)

func (suite *DFGetUtilSuite) TestCreateLogger(c *check.C) {
	logger, tmpFile, r, err := tempFileAndLogger("debug", "x")
	defer cleanTempFile(tmpFile, err)

	var checkLogs = func(level logrus.Level, msg string) {
		line, _, _ := r.ReadLine()
		tmpStr := strings.Split(strings.Trim(string(line), "\n"), " ")
		c.Assert(len(tmpStr) >= 6, check.Equals, true)
		c.Assert(tmpStr[2], check.Equals, strings.ToUpper(level.String()))
		c.Assert(tmpStr[3], check.Equals, "sign:x")
		c.Assert(strings.Join(tmpStr[5:], " "), check.Equals, msg)
	}

	var testPanic = func() {
		msg := "panic test"
		defer func() {
			if e := recover(); e != nil {
				s := e.(*logrus.Entry)
				c.Assert(s.Level, check.Equals, logrus.PanicLevel)
				c.Assert(s.Message, check.Equals, msg)
			}
		}()
		logger.Panic(msg)
	}

	logger.Debug("test")
	checkLogs(logrus.DebugLevel, "test")
	logger.Info([]int{1, 2, 3})
	checkLogs(logrus.InfoLevel, "[1 2 3]")
	logger.Warn("test")
	checkLogs(logrus.WarnLevel, "test")

	testPanic()
}

func (suite *DFGetUtilSuite) TestCreateLogger_differentLevel(c *check.C) {
	var (
		tmpPath     = "/tmp"
		tmpFileName = fmt.Sprintf("dfget_test_%d.log", rand.Int63())
		logger      *logrus.Logger
	)
	defer os.Remove(path.Join(tmpPath, tmpFileName))

	var testLevel = func(level string, expected logrus.Level) {
		logger = CreateLogger(tmpPath, tmpFileName, level, "")
		c.Assert(logger.Level, check.Equals, expected)
	}

	testLevel("", logrus.InfoLevel)
	testLevel("info", logrus.InfoLevel)
	testLevel("debug", logrus.DebugLevel)
	testLevel("Debug", logrus.DebugLevel)
	testLevel("warn", logrus.WarnLevel)
	testLevel("error", logrus.ErrorLevel)
	testLevel("panic", logrus.PanicLevel)
	testLevel("fatal", logrus.FatalLevel)
}

func (suite *DFGetUtilSuite) TestAddConsoleLog(c *check.C) {
	logger, tmpFile, r, err := tempFileAndLogger("debug", "")
	defer cleanTempFile(tmpFile, err)

	// remove timestamp from logs to avoid testing failure
	logger.Formatter = &logrus.TextFormatter{DisableTimestamp: true}
	AddConsoleLog(logger)
	testOut := &bytes.Buffer{}
	for _, v := range logger.Hooks {
		v[0].(*ConsoleHook).logger.Out = testOut
	}

	var testCase = func(log func()) {
		testOut.Reset()
		log()
		line, _, _ := r.ReadLine()
		c.Assert(string(line), check.Equals, strings.TrimRight(testOut.String(), "\n"))
	}

	testCase(func() { logger.Info("test") })
	testCase(func() { logger.Infoln("test") })
	testCase(func() { logger.Infof("test:%s", []string{"1", "2"}) })
	testCase(func() { logger.Debug("test") })
	testCase(func() { logger.Warn("test") })
	testCase(func() { logger.Error("test") })

	testCase(func() {
		defer func() { recover() }()
		logger.Panic("test")
	})
}

func tempFileAndLogger(level string, sign string) (*logrus.Logger, *os.File, *bufio.Reader, error) {
	tmpPath := "/tmp"
	tmpFile, err := ioutil.TempFile(tmpPath, "dfget_test")
	tmpFileName := strings.TrimLeft(tmpFile.Name(), "/tmp/")
	r := bufio.NewReader(tmpFile)
	logger := CreateLogger(tmpPath, tmpFileName, level, sign)
	return logger, tmpFile, r, err
}

func cleanTempFile(file *os.File, err error) {
	if err == nil {
		file.Close()
		os.Remove(file.Name())
	}
}
