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

	"github.com/dragonflyoss/Dragonfly/dfget/util"
	"github.com/sirupsen/logrus"
	"gopkg.in/gcfg.v1"
	"gopkg.in/warnings.v0"
	"gopkg.in/yaml.v2"
)

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
// Config

// Config holds all the runtime config information.
type Config struct {
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

func (cfg *Config) String() string {
	js, _ := json.Marshal(cfg)
	return string(js)
}

// NewConfig creates and initializes a Config.
func NewConfig() *Config {
	cfg := new(Config)
	cfg.StartTime = time.Now()
	cfg.Sign = fmt.Sprintf("%d-%.3f",
		os.Getpid(), float64(time.Now().UnixNano())/float64(time.Second))

	if currentUser, err := user.Current(); err == nil {
		cfg.User = currentUser.Username
		cfg.WorkHome = path.Join(currentUser.HomeDir, ".small-dragonfly")
		cfg.RV.MetaPath = path.Join(cfg.WorkHome, "meta", "host.meta")
		cfg.RV.SystemDataDir = path.Join(cfg.WorkHome, "data")
		cfg.RV.FileLength = -1
	} else {
		panic(fmt.Errorf("get user error: %s", err))
	}
	cfg.ConfigFiles = []string{DefaultYamlConfigFile, DefaultIniConfigFile}
	return cfg
}

// AssertConfig checks the config and panic if any error happens.
func AssertConfig(cfg *Config) {
	util.PanicIfNil(cfg, "runtime config is not initialized")
	util.PanicIfNil(cfg.ClientLogger, "client log is not initialized")
	if cfg.Pattern == "p2p" {
		util.PanicIfNil(cfg.ServerLogger, "server log is not initialized")
	}

	defer func() {
		if err := recover(); err != nil {
			cfg.ClientLogger.Panic(err)
		}
	}()

	util.PanicIfError(checkURL(cfg), "invalid url")
	util.PanicIfError(checkOutput(cfg), "invalid output")
}

func checkURL(cfg *Config) error {
	// shorter than the shortest case 'http://a.b'
	if len(cfg.URL) < 10 {
		return fmt.Errorf(cfg.URL)
	}
	reg := regexp.MustCompile(`(https?|HTTPS?)://([\w_]+:[\w_]+@)?([\w-]+\.)+[\w-]+(/[\w- ./?%&=]*)?`)
	if url := reg.FindString(cfg.URL); util.IsEmptyStr(url) {
		return fmt.Errorf(cfg.URL)
	}
	return nil
}

// This function must be called after checkURL
func checkOutput(cfg *Config) error {
	if util.IsEmptyStr(cfg.Output) {
		url := strings.TrimRight(cfg.URL, "/")
		idx := strings.LastIndexByte(url, '/')
		if idx < 0 {
			return fmt.Errorf("get output from url[%s] error", cfg.URL)
		}
		cfg.Output = url[idx+1:]
	}

	if !filepath.IsAbs(cfg.Output) {
		absPath, err := filepath.Abs(cfg.Output)
		if err != nil {
			return fmt.Errorf("get absolute path[%s] error: %v", cfg.Output, err)
		}
		cfg.Output = absPath
	}

	if f, err := os.Stat(cfg.Output); err == nil && f.IsDir() {
		return fmt.Errorf("path[%s] is directory but requires file path", cfg.Output)
	}

	// check permission
	for dir := cfg.Output; !util.IsEmptyStr(dir); dir = filepath.Dir(dir) {
		if err := syscall.Access(dir, syscall.O_RDWR); err == nil {
			break
		} else if os.IsPermission(err) {
			return fmt.Errorf("user[%s] path[%s] %v", cfg.User, cfg.Output, err)
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
