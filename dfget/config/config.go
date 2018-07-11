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
	"os"
	"os/user"
	"path"
	"time"

	"github.com/Sirupsen/logrus"
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
	Props = &Properties{
		Node:            []string{"127.0.0.1"},
		LocalLimit:      20 * 1024 * 1024,
		ClientQueueSize: 6,
	}
	Ctx = NewContext()
}

// ----------------------------------------------------------------------------

// Properties holds all configurable Properties.
type Properties struct {
	Node            []string
	LocalLimit      int
	TotalLimit      int
	ClientQueueSize int
}

// Load loads properties from config file.
func (p *Properties) Load(path string) error {
	return nil
}

// Context holds all the runtime context information.
type Context struct {
	URL             string   `json:"url"`
	Output          string   `json:"output"`
	LocalLimit      int      `json:"localLimit,omitempty"`
	TotalLimit      int      `json:"totalLimit,omitempty"`
	Timeout         int      `json:"timeout,omitempty"`
	Md5             string   `json:"md5,omitempty"`
	Identifier      string   `json:"identifier,omitempty"`
	CallSystem      string   `json:"callSystem,omitempty"`
	Pattern         string   `json:"pattern,omitempty"`
	Filter          []string `json:"filter,omitempty"`
	Header          []string `json:"header,omitempty"`
	Node            []string `json:"node,omitempty"`
	Notbs           bool     `json:"notbs,omitempty"`
	DFDaemon        bool     `json:"dfdaemon,omitempty"`
	Version         bool     `json:"version,omitempty"`
	ShowBar         bool     `json:"showBar,omitempty"`
	Console         bool     `json:"console,omitempty"`
	Verbose         bool     `json:"verbose,omitempty"`
	Help            bool     `json:"help,omitempty"`
	ClientQueueSize int      `json:"clientQueueSize,omitempty"`

	StartTime  time.Time `json:"startTime"`
	Sign       string    `json:"sign"`
	User       string    `json:"user"`
	WorkHome   string    `json:"workHome"`
	ConfigFile string    `json:"configFile"`

	ClientLogger *logrus.Logger `json:"-"`
	ServerLogger *logrus.Logger `json:"-"`
}

func (ctx *Context) String() string {
	js, _ := json.Marshal(ctx)
	return fmt.Sprintf("%s", js)
}

// NewContext creates and initialize a Context.
func NewContext() *Context {
	ctx := new(Context)
	ctx.StartTime = time.Now()
	ctx.Sign = fmt.Sprintf("%d-%.3f",
		os.Getpid(), float64(time.Now().UnixNano())/float64(time.Second))

	if currentUser, err := user.Current(); err == nil {
		ctx.User = currentUser.Username
		ctx.WorkHome = path.Join(currentUser.HomeDir, ".small-dragonfly")
	} else {
		panic(fmt.Errorf("get user error: %s", err))
	}
	ctx.ConfigFile = DefaultConfigFile
	return ctx
}
