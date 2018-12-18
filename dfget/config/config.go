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

// Package config holds all Properties of dfget.
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	errType "github.com/dragonflyoss/Dragonfly/dfget/errors"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/gcfg.v1"
	"gopkg.in/warnings.v0"
	"gopkg.in/yaml.v2"
)

var (
	// Props is loaded from config file.
	Props *Properties

	// Ctx holds all the runtime Context information.
	Ctx *Context
)

func init() {
	Reset()
}

// Reset configuration and Context.
func Reset() {
	Props = NewProperties()
	Ctx = NewContext()
}

// ----------------------------------------------------------------------------
// Properties

// Properties holds all configurable Properties.
// Support INI(or conf) and YAML(since 0.2.0).
// Before 0.2.0, only support INI config and only have one property(node):
// 		[node]
// 		address=127.0.0.1,10.10.10.1
// Since 0.2.0, the INI config is just to be compatible with previous versions.
// The YAML config will have more properties:
// 		nodes:
// 		    - 127.0.0.1
// 		    - 10.10.10.1
// 		localLimit: 20971520
// 		totalLimit: 20971520
// 		clientQueueSize: 6
type Properties struct {
	Nodes           []string `yaml:"nodes"`
	LocalLimit      int      `yaml:"localLimit"`
	TotalLimit      int      `yaml:"totalLimit"`
	ClientQueueSize int      `yaml:"clientQueueSize"`
}

// NewProperties create a new properties with default values.
func NewProperties() *Properties {
	return &Properties{
		Nodes:           []string{DefaultNode},
		LocalLimit:      DefaultLocalLimit,
		ClientQueueSize: DefaultClientQueueSize,
	}
}

func (p *Properties) String() string {
	str, _ := json.Marshal(p)
	return string(str)
}

// Load loads properties from config file.
func (p *Properties) Load(path string) error {
	switch p.fileType(path) {
	case "ini":
		return p.loadFromIni(path)
	case "yaml":
		return p.loadFromYaml(path)
	}
	return fmt.Errorf("extension of %s is not in 'conf/ini/yaml/yml'", path)
}

// loadFromIni to be compatible with previous versions(before 0.2.0).
func (p *Properties) loadFromIni(path string) error {
	oldConfig := struct {
		Node struct {
			Address string
		}
	}{}

	if err := gcfg.ReadFileInto(&oldConfig, path); err != nil {
		// only fail on a fatal error occurred
		if e, ok := err.(warnings.List); !ok || e.Fatal != nil {
			return fmt.Errorf("read ini config from %s error: %v", path, err)
		}
	}
	p.Nodes = strings.Split(oldConfig.Node.Address, ",")
	return nil
}

func (p *Properties) loadFromYaml(path string) error {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read yaml config from %s error: %v", path, err)
	}
	err = yaml.Unmarshal(yamlFile, p)
	if err != nil {
		return fmt.Errorf("unmarshal yaml error:%v", err)
	}
	return nil
}

func (p *Properties) fileType(path string) string {
	ext := filepath.Ext(path)
	switch v := strings.ToLower(ext); v {
	case ".conf", ".ini":
		return "ini"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return v
	}
}

// ----------------------------------------------------------------------------
// Context

// Context holds all the runtime context information.
type Context struct {
	// URL download URL.
	URL string `json:"url"`

	// Output full output path.
	Output string `json:"output"`

	// LocalLimit rate limit about a single download task,format: 20M/m/K/k.
	LocalLimit int `json:"localLimit,omitempty"`

	// TotalLimit rate limit about the whole host,format: 20M/m/K/k.
	TotalLimit int `json:"totalLimit,omitempty"`

	// Timeout download timeout(second).
	Timeout int `json:"timeout,omitempty"`

	// Md5 expected file md5.
	Md5 string `json:"md5,omitempty"`

	// Identifier identify download task, it is available merely when md5 param not exist.
	Identifier string `json:"identifier,omitempty"`

	// CallSystem system name that executes dfget.
	CallSystem string `json:"callSystem,omitempty"`

	// Pattern download pattern, must be 'p2p' or 'cdn' or 'source',
	// default:`p2p`.
	Pattern string `json:"pattern,omitempty"`

	// Filter filter some query params of url, use char '&' to separate different params.
	// eg: -f 'key&sign' will filter 'key' and 'sign' query param.
	// in this way, different urls correspond one same download task that can use p2p mode.
	Filter []string `json:"filter,omitempty"`

	// Header of http request.
	// eg: --header='Accept: *' --header='Host: abc'.
	Header []string `json:"header,omitempty"`

	// Node specify supernodes.
	Node []string `json:"node,omitempty"`

	// Notbs indicates whether to not back source to download when p2p fails.
	Notbs bool `json:"notbs,omitempty"`

	// DFDaemon indicates whether the caller is from dfdaemon
	DFDaemon bool `json:"dfdaemon,omitempty"`

	// Version show version.
	Version bool `json:"version,omitempty"`

	// ShowBar show progress bar, it's conflict with `--console`.
	ShowBar bool `json:"showBar,omitempty"`

	// Console show log on console, it's conflict with `--showbar`.
	Console bool `json:"console,omitempty"`

	// Verbose indicates whether to be verbose.
	// If set true, log level will be 'debug'.
	Verbose bool `json:"verbose,omitempty"`

	// Help show help information.
	Help bool `json:"help,omitempty"`

	// Client queue size.
	// TODO: support setupFlags
	ClientQueueSize int `json:"clientQueueSize,omitempty"`

	// Start time.
	StartTime time.Time `json:"startTime"`

	// Sign the value is 'Pid + float64(time.Now().UnixNano())/float64(time.Second) format: "%d-%.3f"'.
	Sign string `json:"sign"`

	// Username of the system currently logged in.
	User string `json:"user"`

	// WorkHome work home path,
	// default: `$HOME/.small-dragonfly`.
	WorkHome string `json:"workHome"`

	// Config file paths,
	// default:["/etc/dragonfly.yaml","/etc/dragonfly.conf"].
	ConfigFiles []string `json:"configFile"`

	// RV stores the variables that are initialized and used at downloading task executing.
	RV RuntimeVariable `json:"-"`

	// The reason of backing to source.
	BackSourceReason int `json:"-"`

	// Client logger.
	ClientLogger *logrus.Logger `json:"-"`

	// Server logger, only created when Pattern equals 'p2p'.
	ServerLogger *logrus.Logger `json:"-"`
}

func (ctx *Context) String() string {
	js, _ := json.Marshal(ctx)
	return string(js)
}

// NewContext creates and initialize a Context.
func NewContext() *Context {
	ctx := new(Context)
	ctx.StartTime = time.Now()
	ctx.Sign = fmt.Sprintf("%d-%.3f",
		os.Getpid(), float64(time.Now().UnixNano())/float64(time.Second))

	// NOTE: Maybe change it to not depend on homedir of currentUser.
	currentUser, err := user.Current()
	if err != nil {
		util.Printer.Println(fmt.Sprintf("get user error: %s", err))
		os.Exit(CodeGetUserError)
	}

	ctx.User = currentUser.Username
	ctx.WorkHome = path.Join(currentUser.HomeDir, ".small-dragonfly")
	ctx.RV.MetaPath = path.Join(ctx.WorkHome, "meta", "host.meta")
	ctx.RV.SystemDataDir = path.Join(ctx.WorkHome, "data")
	ctx.ConfigFiles = []string{DefaultYamlConfigFile, DefaultIniConfigFile}
	return ctx
}

// AssertContext checks the ctx.
func AssertContext(ctx *Context) (err error) {

	if util.IsNil(ctx) {
		return errors.Wrap(errType.ErrNotInitialized, "runtime context")
	}
	if util.IsNil(ctx.ClientLogger) {
		return errors.Wrap(errType.ErrNotInitialized, "client log")
	}
	if util.IsNil(ctx.ServerLogger) {
		return errors.Wrap(errType.ErrNotInitialized, "server log")
	}

	defer func() {
		if err != nil {
			ctx.ClientLogger.Errorf("assert context error: %v", err)
		}
	}()

	if err := checkURL(ctx); err != nil {
		return errors.Wrap(errType.ErrInvalidValue, "url")
	}

	if err := checkOutput(ctx); err != nil {
		return errors.Wrap(errType.ErrInvalidValue, "output")
	}
	return nil
}

func checkURL(ctx *Context) error {
	// shorter than the shortest case 'http://a.b'
	if len(ctx.URL) < 10 {
		return fmt.Errorf(ctx.URL)
	}
	reg := regexp.MustCompile(`(https?|HTTPS?)://([\w_]+:[\w_]+@)?([\w-]+\.)+[\w-]+(/[\w- ./?%&=]*)?`)
	if url := reg.FindString(ctx.URL); util.IsEmptyStr(url) {
		return fmt.Errorf(ctx.URL)
	}
	return nil
}

// This function must be called after checkURL
func checkOutput(ctx *Context) error {
	if util.IsEmptyStr(ctx.Output) {
		url := strings.TrimRight(ctx.URL, "/")
		idx := strings.LastIndexByte(url, '/')
		if idx < 0 {
			return fmt.Errorf("get output from url[%s] error", ctx.URL)
		}
		ctx.Output = url[idx+1:]
	}

	if !filepath.IsAbs(ctx.Output) {
		absPath, err := filepath.Abs(ctx.Output)
		if err != nil {
			return fmt.Errorf("get absolute path[%s] error: %v", ctx.Output, err)
		}
		ctx.Output = absPath
	}

	if f, err := os.Stat(ctx.Output); err == nil && f.IsDir() {
		return fmt.Errorf("path[%s] is directory but requires file path", ctx.Output)
	}

	// check permission
	for dir := ctx.Output; !util.IsEmptyStr(dir); dir = filepath.Dir(dir) {
		if err := syscall.Access(dir, syscall.O_RDWR); err == nil {
			break
		} else if os.IsPermission(err) {
			return fmt.Errorf("user[%s] path[%s] %v", ctx.User, ctx.Output, err)
		}
	}
	return nil
}

// RuntimeVariable stores the variables that are initialized and used
// at downloading task executing.
type RuntimeVariable struct {
	MetaPath      string
	SystemDataDir string
	DataDir       string
	RealTarget    string
	TargetDir     string
	TempTarget    string
	Cid           string
	TaskURL       string
	TaskFileName  string
	LocalIP       string
	PeerPort      int
	FileLength    int64
}

func (rv *RuntimeVariable) String() string {
	js, _ := json.Marshal(rv)
	return string(js)
}
