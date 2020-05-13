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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type dfgetSuit struct {
	suite.Suite
}

func (suit *dfgetSuit) Test_initFlagsNoArguments() {
	initProperties()
	suit.Equal(cfg.Nodes, []string(nil))
	suit.Equal(cfg.LocalLimit, 20*rate.MB)
	suit.Equal(cfg.TotalLimit, rate.Rate(0))
	suit.Equal(cfg.Notbs, false)
	suit.Equal(cfg.DFDaemon, false)
	suit.Equal(cfg.Console, false)
	suit.Equal(cfg.Verbose, false)
	suit.Equal(cfg.URL, "")
}

func (suit *dfgetSuit) Test_initProperties() {
	cfg.ConfigFiles = nil
	dirName, _ := ioutil.TempDir("/tmp", "dfget-TestInitProperties-")
	defer os.RemoveAll(dirName)

	iniFile := filepath.Join(dirName, "dragonfly.ini")
	yamlFile := filepath.Join(dirName, "dragonfly.yaml")
	iniContent := []byte("[node]\naddress=1.1.1.1")
	yamlContent := []byte("nodes:\n  - 1.1.1.2\nlocalLimit: 1000K\ntotalLimit: 1000k")
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
			expected: newProp(0, 0, 0, "1.1.1.1:8002")},
		{configs: []string{yamlFile, iniFile},
			expected: newProp(int(rate.KB*1000), int(rate.KB*1000), 0, "1.1.1.2:8002")},
		{configs: []string{filepath.Join(dirName, "x"), yamlFile},
			expected: newProp(int(rate.KB*1000), int(rate.KB*1000), 0, "1.1.1.2:8002")},
	}

	for _, v := range cases {
		cfg = config.NewConfig()
		buf.Reset()
		cfg.ConfigFiles = v.configs
		rootCmd.Flags().Parse([]string{
			"--locallimit", v.expected.LocalLimit.String(),
			"--totallimit", v.expected.TotalLimit.String()})
		initProperties()
		suit.EqualValues(cfg.Nodes, config.NodeWeightSlice2StringSlice(v.expected.Supernodes))
		suit.Equal(cfg.LocalLimit, v.expected.LocalLimit)
		suit.Equal(cfg.TotalLimit, v.expected.TotalLimit)
		suit.Equal(cfg.ClientQueueSize, v.expected.ClientQueueSize)
	}
}

func (suit *dfgetSuit) Test_HeaderFlags() {
	originCfg := *cfg
	defer func() {
		cfg = &originCfg
	}()
	flagSet := rootCmd.Flags()
	flagSet.Parse([]string{"--header", "Host: abc", "--header", "Date:Mon, 30 Dec 2019"})

	suit.Equal(cfg.Header, []string{"Host: abc", "Date:Mon, 30 Dec 2019"})
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
	suit.Equal(msg, "download SUCCESS cost:0.100s length:-1 reason:0")

	msg = resultMsg(cfg, end, errortypes.New(1, "TestFail"))
	suit.Equal(msg, "download FAIL(1) cost:0.100s length:-1 reason:0 error:"+
		`{"Code":1,"Msg":"TestFail"}`)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(dfgetSuit))
}

func newProp(local int, total int, size int, nodes ...string) *config.Properties {
	p := config.NewProperties()
	if nodes != nil {
		p.Supernodes, _ = config.ParseNodesSlice(nodes)
	}
	if local != 0 {
		p.LocalLimit = rate.Rate(local)
	}
	if total != 0 {
		p.TotalLimit = rate.Rate(total)
	}
	if size != 0 {
		p.ClientQueueSize = size
	}
	return p
}
