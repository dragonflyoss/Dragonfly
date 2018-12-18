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

	cfg "github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type dfgetSuit struct {
	suite.Suite
}

func (suit *dfgetSuit) Test_initFlagsNoArguments() {
	suit.Nil(cfg.Ctx.Node)
	suit.Equal(cfg.Ctx.LocalLimit, 0)
	suit.Equal(cfg.Ctx.TotalLimit, 0)
	suit.Equal(cfg.Ctx.Notbs, false)
	suit.Equal(cfg.Ctx.DFDaemon, false)
	suit.Equal(cfg.Ctx.Version, false)
	suit.Equal(cfg.Ctx.ShowBar, false)
	suit.Equal(cfg.Ctx.Console, false)
	suit.Equal(cfg.Ctx.Verbose, false)
	suit.Equal(cfg.Ctx.Help, false)
	suit.Equal(cfg.Ctx.URL, "")
}

func (suit *dfgetSuit) Test_initProperties() {
	cfg.Ctx.ConfigFiles = nil
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
		expected *cfg.Properties
	}{
		{configs: nil,
			expected: cfg.NewProperties()},
		{configs: []string{iniFile, yamlFile},
			expected: newProp(0, 0, 0, "1.1.1.1")},
		{configs: []string{yamlFile, iniFile},
			expected: newProp(1024000, 0, 0, "1.1.1.2")},
		{configs: []string{filepath.Join(dirName, "x"), yamlFile},
			expected: newProp(1024000, 0, 0, "1.1.1.2")},
	}

	for _, v := range cases {
		cfg.Reset()
		buf.Reset()
		cfg.Ctx.ClientLogger = logrus.StandardLogger()
		cfg.Ctx.ConfigFiles = v.configs
		localLimitStr := strconv.FormatInt(int64(v.expected.LocalLimit/1024), 10)
		totalLimitStr := strconv.FormatInt(int64(v.expected.TotalLimit/1024), 10)
		rootCmd.PersistentFlags().Parse([]string{
			"--locallimit", fmt.Sprintf("%sk", localLimitStr),
			"--totallimit", fmt.Sprintf("%sk", totalLimitStr)})
		initProperties()
		suit.EqualValues(cfg.Ctx.Node, v.expected.Nodes)
		suit.Equal(cfg.Ctx.LocalLimit, v.expected.LocalLimit)
		suit.Equal(cfg.Ctx.TotalLimit, v.expected.TotalLimit)
		suit.Equal(cfg.Ctx.ClientQueueSize, v.expected.ClientQueueSize)
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
	ctx := cfg.NewContext()
	end := ctx.StartTime.Add(100 * time.Millisecond)

	msg := resultMsg(ctx, end, nil)
	suit.Equal(msg, "download SUCCESS(0) cost:0.100s length:0 reason:0")

	ctx.BackSourceReason = cfg.BackSourceReasonRegisterFail
	msg = resultMsg(ctx, end, errors.New(1, "TestFail"))
	suit.Equal(msg, "download FAIL(1) cost:0.100s length:0 reason:1 error:"+
		`{"Code":1,"Msg":"TestFail"}`)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(dfgetSuit))
}

func newProp(local int, total int, size int, nodes ...string) *cfg.Properties {
	p := cfg.NewProperties()
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
