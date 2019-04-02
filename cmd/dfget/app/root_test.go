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

package app

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/common/errors"
	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfget/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type dfgetSuit struct {
	suite.Suite
}

func (suit *dfgetSuit) Test_initFlagsNoArguments() {
	suit.Nil(cfg.Node)
	suit.Equal(cfg.LocalLimit, 0)
	suit.Equal(cfg.TotalLimit, 0)
	suit.Equal(cfg.Notbs, false)
	suit.Equal(cfg.DFDaemon, false)
	suit.Equal(cfg.Version, false)
	suit.Equal(cfg.ShowBar, false)
	suit.Equal(cfg.Console, false)
	suit.Equal(cfg.Verbose, false)
	suit.Equal(cfg.Help, false)
	suit.Equal(cfg.URL, "")
}

func (suit *dfgetSuit) Test_initProperties() {
	cfg.ConfigFiles = nil
	dirName, _ := ioutil.TempDir("/tmp", "dfget-TestInitProperties-")
	defer os.RemoveAll(dirName)

	iniFile := filepath.Join(dirName, "dragonfly.ini")
	yamlFile := filepath.Join(dirName, "dragonfly.yaml")
	iniContent := []byte("[node]\naddress=1.1.1.1")
	yamlContent := []byte("nodes:\n  - 1.1.1.2\nlocalLimit: 1024000")
	ioutil.WriteFile(iniFile, iniContent, os.ModePerm)
	ioutil.WriteFile(yamlFile, yamlContent, os.ModePerm)

	var buf = &bytes.Buffer{}
	logrus.StandardLogger().Out = buf

	var cases = []struct {
		configs  []string
		expected *config.Properties
	}{
		{configs: nil,
			expected: config.NewProperties()},
		{configs: []string{iniFile, yamlFile},
			expected: newProp(0, 0, 0, "1.1.1.1")},
		{configs: []string{yamlFile, iniFile},
			expected: newProp(1024000, 0, 0, "1.1.1.2")},
		{configs: []string{filepath.Join(dirName, "x"), yamlFile},
			expected: newProp(1024000, 0, 0, "1.1.1.2")},
	}

	for _, v := range cases {
		cfg = config.NewConfig()
		buf.Reset()
		cfg.ConfigFiles = v.configs
		localLimitStr := strconv.FormatInt(int64(v.expected.LocalLimit/1024), 10)
		totalLimitStr := strconv.FormatInt(int64(v.expected.TotalLimit/1024), 10)
		rootCmd.Flags().Parse([]string{
			"--locallimit", fmt.Sprintf("%sk", localLimitStr),
			"--totallimit", fmt.Sprintf("%sk", totalLimitStr)})
		initProperties()
		suit.EqualValues(cfg.Node, v.expected.Nodes)
		suit.Equal(cfg.LocalLimit, v.expected.LocalLimit)
		suit.Equal(cfg.TotalLimit, v.expected.TotalLimit)
		suit.Equal(cfg.ClientQueueSize, v.expected.ClientQueueSize)
	}
}

func (suit *dfgetSuit) Test_transLimit() {
	var cases = map[string]struct {
		i   int
		err string
	}{
		"20M":   {20971520, ""},
		"20m":   {20971520, ""},
		"10k":   {10240, ""},
		"10K":   {10240, ""},
		"10x":   {0, "invalid unit 'x' of '10x', 'KkMm' are supported"},
		"10.0x": {0, "invalid syntax"},
		"ab":    {0, "invalid syntax"},
		"abM":   {0, "invalid syntax"},
	}

	for k, v := range cases {
		i, e := transLimit(k)
		suit.Equal(i, v.i)
		if util.IsEmptyStr(v.err) {
			suit.Nil(e)
		} else {
			suit.NotNil(e)
			suit.True(strings.Contains(e.Error(), v.err), true)
		}
	}
}

func (suit *dfgetSuit) Test_transFilter() {
	var cases = []string{
		"a&b&c",
		"a",
		"",
		"abc",
	}

	var expectedResult = [][]string{
		0: {"a", "b", "c"},
		1: {"a"},
		2: nil,
		3: {"abc"},
	}

	testResult := make([][]string, len(cases))
	for i, v := range cases {
		filters := transFilter(v)
		testResult[i] = filters
	}

	suit.EqualValues(expectedResult, testResult)
}

func (suit *dfgetSuit) TestResultMsg() {
	cfg := config.NewConfig()
	end := cfg.StartTime.Add(100 * time.Millisecond)

	msg := resultMsg(cfg, end, nil)
	suit.Equal(msg, "download SUCCESS(0) cost:0.100s length:-1 reason:0")

	msg = resultMsg(cfg, end, errors.New(1, "TestFail"))
	suit.Equal(msg, "download FAIL(1) cost:0.100s length:-1 reason:0 error:"+
		`{"Code":1,"Msg":"TestFail"}`)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(dfgetSuit))
}

func newProp(local int, total int, size int, nodes ...string) *config.Properties {
	p := config.NewProperties()
	if nodes != nil {
		p.Nodes = nodes
	}
	if local != 0 {
		p.LocalLimit = local
	}
	if total != 0 {
		p.TotalLimit = total
	}
	if size != 0 {
		p.ClientQueueSize = size
	}
	return p
}
