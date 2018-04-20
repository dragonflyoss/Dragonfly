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

	log "github.com/Sirupsen/logrus"
)

type CommandParam struct {
	DfPath     string
	DFRepo     string
	RateLimit  string
	CallSystem string
	Urlfilter  string
	Notbs      bool

	Version  bool
	Verbose  bool
	Help     bool
	HostIp   string
	Port     uint
	Registry string //https://xxx.xx.x:port or http://xxx.xx.x:port
	DownRule string

	CertFile string
	KeyFile  string
}

var (
	//user home
	G_HomeDir string

	//df-daemon home
	G_DfHome string

	G_UseHttps bool

	G_CommandLine CommandParam

	G_RegProto string

	G_RegDomain string

	G_DFPattern = make(map[string]*regexp.Regexp)

	rwMutex sync.RWMutex
)

func UpdateDFPattern(reg string) {
	if reg == "" {
		return
	}
	rwMutex.Lock()
	defer rwMutex.Unlock()
	if compiledReg, err := regexp.Compile(reg); err == nil {
		G_DFPattern[reg] = compiledReg
	} else {
		log.Warnf("pattern:%s is invalid", reg)
	}
}

func CopyDfPattern() []string {
	rwMutex.RLock()
	defer rwMutex.RUnlock()
	copiedPattern := make([]string, 0, len(G_DFPattern))
	for _, value := range G_DFPattern {
		copiedPattern = append(copiedPattern, value.String())
	}
	return copiedPattern
}

func MatchDfPattern(location string) bool {
	useGetter := false
	rwMutex.RLock()
	defer rwMutex.RUnlock()
	for key, regex := range G_DFPattern {
		if regex.MatchString(location) {
			useGetter = true
			break
		}
		log.Debugf("location:%s not match reg:%s", location, key)
	}
	return useGetter
}
