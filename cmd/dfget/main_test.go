package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alibaba/Dragonfly/cmd/dfget/options"
	cfg "github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/version"

	"github.com/go-check/check"
	"github.com/sirupsen/logrus"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type MainSuite struct{}

func init() {
	check.Suite(&MainSuite{})
}

func (suite *MainSuite) TestInitProperties(c *check.C) {
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

		initProperties()
		c.Assert(cfg.Ctx.Node, check.DeepEquals, v.expected.Nodes)
		c.Assert(cfg.Ctx.LocalLimit, check.Equals, v.expected.LocalLimit)
		c.Assert(cfg.Ctx.TotalLimit, check.Equals, v.expected.TotalLimit)
		c.Assert(cfg.Ctx.ClientQueueSize, check.Equals, v.expected.ClientQueueSize)
	}
}

func (suite *MainSuite) TestInitialize(c *check.C) {
	const workHomeEnv = "DFGET_TEST_WORK_HOME"
	if hasTestEnv() {
		args := testArgs()
		cfg.Ctx.WorkHome = os.Getenv(workHomeEnv)
		os.Args = args
		options.CliOut = os.Stdout
		initialize()
		return
	}

	tmpDir, _ := ioutil.TempDir("/tmp", "dfget-cli-test-")
	defer os.RemoveAll(tmpDir)

	var cases = []struct {
		args   []string
		exMsg  string
		exCode int
	}{
		{args: []string{}, exMsg: "please use", exCode: 0},
		{args: []string{"-h"}, exMsg: "Usage", exCode: 0},
		{args: []string{"-v"}, exMsg: version.DFGetVersion, exCode: 0},
		{args: []string{"-u"}, exMsg: "flag needs an argument", exCode: 2},
		{args: []string{"-u", "http://www.taobao.com"}, exMsg: "", exCode: 0},
	}

	for _, v := range cases {
		cmd := helperCommand("TestInitialize", v.args...)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", workHomeEnv, tmpDir))
		bs, err := cmd.Output()
		c.Assert(strings.Contains(string(bs), v.exMsg), check.Equals, true,
			check.Commentf("args:%v out:%s", v.args, bs))
		if e, ok := err.(*exec.ExitError); ok {
			c.Assert(e.Success(), check.Equals, v.exCode == 0, check.Commentf(
				"args:%v err:%v", v.args, err))
		}
	}
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

func helperCommand(name string, args ...string) (cmd *exec.Cmd) {
	cs := []string{"-check.f", fmt.Sprintf("^%s$", name), "--"}
	cs = append(cs, args...)
	cmd = exec.Command(os.Args[0], cs...)
	cmd.Env = testEnv()
	return cmd
}

func testArgs() []string {
	args := []string{os.Args[0]}
	if len(os.Args) > 4 {
		args = append(args, os.Args[4:]...)
	}
	return args
}

func testEnv() []string {
	return []string{"DFGET_TEST_HELPER_PROCESS=1"}
}

func hasTestEnv() bool {
	return os.Getenv("DFGET_TEST_HELPER_PROCESS") == "1"
}
