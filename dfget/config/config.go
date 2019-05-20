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
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	errType "github.com/dragonflyoss/Dragonfly/common/errors"
	cutil "github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfget/util"

	"github.com/pkg/errors"
	"gopkg.in/gcfg.v1"
	"gopkg.in/warnings.v0"
)

// ----------------------------------------------------------------------------
// Properties

// Properties holds all configurable Properties.
// Support INI(or conf) and YAML(since 0.3.0).
// Before 0.3.0, only support INI config and only have one property(node):
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
	// Nodes specify supernodes.
	Nodes []string `yaml:"nodes"`

	// LocalLimit rate limit about a single download task,format: 20M/m/K/k.
	LocalLimit int `yaml:"localLimit"`

	// TotalLimit rate limit about the whole host,format: 20M/m/K/k.
	TotalLimit int `yaml:"totalLimit"`

	// ClientQueueSize is the size of client queue
	// which controls the number of pieces that can be processed simultaneously.
	// It is only useful when the Pattern equals "source".
	// The default value is 6.
	ClientQueueSize int `yaml:"clientQueueSize"`
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
		return cutil.LoadYaml(path, p)
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

	// ClientQueueSize is the size of client queue
	// which controls the number of pieces that can be processed simultaneously.
	// It is only useful when the pattern not equals "source".
	// The default value is 6.
	ClientQueueSize int `json:"clientQueueSize,omitempty"`

	// Start time.
	StartTime time.Time `json:"startTime"`

	// Sign the value is 'Pid + float64(time.Now().UnixNano())/float64(time.Second) format: "%d-%.3f"'.
	// It is unique for downloading task, and is used for debugging.
	Sign string `json:"sign"`

	// Username of the system currently logged in.
	User string `json:"user"`

	// WorkHome work home path,
	// default: `$HOME/.small-dragonfly`.
	WorkHome string `json:"workHome"`

	// Config file paths,
	// default:["/etc/dragonfly/dfget.yml","/etc/dragonfly.conf"].
	//
	// NOTE: It is recommended to use `/etc/dragonfly/dfget.yml` as default,
	// and the `/etc/dragonfly.conf` is just to ensure compatibility with previous versions.
	ConfigFiles []string `json:"configFile"`

	// RV stores the variables that are initialized and used at downloading task executing.
	RV RuntimeVariable `json:"-"`

	// The reason of backing to source.
	BackSourceReason int `json:"-"`
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

	// TODO: Use parameters instead of currentUser.HomeDir.
	currentUser, err := user.Current()
	if err != nil {
		util.Printer.Println(fmt.Sprintf("get user error: %s", err))
		os.Exit(CodeGetUserError)
	}

	cfg.User = currentUser.Username
	cfg.WorkHome = path.Join(currentUser.HomeDir, ".small-dragonfly")
	cfg.RV.MetaPath = path.Join(cfg.WorkHome, "meta", "host.meta")
	cfg.RV.SystemDataDir = path.Join(cfg.WorkHome, "data")
	cfg.RV.FileLength = -1
	cfg.ConfigFiles = []string{DefaultYamlConfigFile, DefaultIniConfigFile}
	return cfg
}

// AssertConfig checks the config and return errors.
func AssertConfig(cfg *Config) (err error) {
	if cutil.IsNil(cfg) {
		return errors.Wrap(errType.ErrNotInitialized, "runtime config")
	}

	if !cutil.IsValidURL(cfg.URL) {
		return errors.Wrapf(errType.ErrInvalidValue, "url: %v", err)
	}

	if err := checkOutput(cfg); err != nil {
		return errors.Wrapf(errType.ErrInvalidValue, "output: %v", err)
	}
	return nil
}

// This function must be called after checkURL
func checkOutput(cfg *Config) error {
	if cutil.IsEmptyStr(cfg.Output) {
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
	for dir := cfg.Output; !cutil.IsEmptyStr(dir); dir = filepath.Dir(dir) {
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
	// MetaPath specify the path of meta file which store the meta info of the peer that should be persisted.
	// Only server port information is stored currently.
	MetaPath string

	// SystemDataDir specify a default directory to store temporary files.
	SystemDataDir string

	// DataDir specify a directory to store temporary files.
	// For now, the value of `DataDir` always equals `SystemDataDir`,
	// and there is no difference between them.
	// TODO: If there is insufficient disk space, we should set it to the `TargetDir`.
	DataDir string

	// RealTarget specify the full target path whose value is equal to the `Output`.
	RealTarget string

	// TargetDir is the directory of the RealTarget path.
	TargetDir string

	// TempTarget is a temp file path that try to determine
	// whether the `TargetDir` and the `DataDir` belong to the same disk by making a hard link.
	TempTarget string

	// Cid means the client ID which is a string composed of `localIP + "-" + sign` which represents a peer node.
	// NOTE: Multiple dfget processes on the same peer have different CIDs.
	Cid string

	// TaskURL is generated from rawURL which may contains some queries or parameter.
	// Dfget will filter some volatile queries such as timestamps via --filter parameter of dfget.
	TaskURL string

	// TaskFileName is a string composed of `the last element of RealTarget path + "-" + sign`.
	TaskFileName string

	// LocalIP is the native IP which can connect supernode successfully.
	LocalIP string

	// PeerPort is the TCP port on which the file upload service listens as a peer node.
	PeerPort int

	// FileLength the length of the file to download.
	FileLength int64

	// DataExpireTime specify the caching duration for which
	// cached files keep no accessed by any process.
	// After this period, the cached files will be deleted.
	DataExpireTime time.Duration

	// ServerAliveTime specify the alive duration for which
	// uploader keeps no accessing by any uploading requests.
	// After this period, the uploader will automically exit.
	ServerAliveTime time.Duration
}

func (rv *RuntimeVariable) String() string {
	js, _ := json.Marshal(rv)
	return string(js)
}
