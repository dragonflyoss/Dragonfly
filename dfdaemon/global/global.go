// Copyright 1999-2017 Alibaba Group.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package global

import (
	"regexp"
	"sync"

	log "github.com/sirupsen/logrus"
)

// CommandParam is a struct that stores all the command line parameters
type CommandParam struct {
	DfPath     string
	DFRepo     string
	RateLimit  string
	CallSystem string
	URLFilter  string
	Notbs      bool
	HostIP     string
	Registry   string //https://xxx.xx.x:port or http://xxx.xx.x:port
}

var (
	// HomeDir is the user home
	HomeDir string

	// UseHTTPS indicates whether to use HTTPS protocol
	UseHTTPS bool

	// CommandLine stores all the command line parameters
	CommandLine CommandParam

	// RegProto is the protocol(HTTP/HTTPS) of images registry
	RegProto string

	// RegDomain is the domain of images registry
	RegDomain string

	// DFPattern is the url patterns. Dfdaemon starts downloading by P2P if the downloading url matches DFPattern.
	DFPattern = make(map[string]*regexp.Regexp)

	rwMutex sync.RWMutex
)

// UpdateDFPattern is to update DFPattern from the giving string(CommandParam.DownRule).
func UpdateDFPattern(reg string) {
	if reg == "" {
		return
	}
	rwMutex.Lock()
	defer rwMutex.Unlock()
	if compiledReg, err := regexp.Compile(reg); err == nil {
		DFPattern[reg] = compiledReg
	} else {
		log.Warnf("pattern:%s is invalid", reg)
	}
}

// CopyDfPattern is to copy DFPattern's content.
func CopyDfPattern() []string {
	rwMutex.RLock()
	defer rwMutex.RUnlock()
	copiedPattern := make([]string, 0, len(DFPattern))
	for _, value := range DFPattern {
		copiedPattern = append(copiedPattern, value.String())
	}
	return copiedPattern
}

// MatchDfPattern returns true if location matches DFPattern, otherwise returns false.
func MatchDfPattern(location string) bool {
	useGetter := false
	rwMutex.RLock()
	defer rwMutex.RUnlock()
	for key, regex := range DFPattern {
		if regex.MatchString(location) {
			useGetter = true
			break
		}
		log.Debugf("location:%s not match reg:%s", location, key)
	}
	return useGetter
}
