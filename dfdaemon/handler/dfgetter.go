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

package handler

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/alibaba/Dragonfly/dfdaemon/constant"
	"github.com/alibaba/Dragonfly/dfdaemon/exception"
	. "github.com/alibaba/Dragonfly/dfdaemon/global"
)

type Downloader interface {
	//Download download url file to file name
	//return dst path and download error
	Download(url string, header map[string][]string, name string) (string, error)
}

type DFgetter struct {
	//output dir
	dstDir string
	//the urlfilter param of dfget
	urlFilter string
	//the totallimit and s param of dfget
	rateLimit string
	//the callsystem param of dfget
	callSystem string
	//the notbs param of dfget
	notbs bool

	once sync.Once
}

var getter = new(DFgetter)

func (dfgetter *DFgetter) Download(url string, header map[string][]string, name string) (string, error) {
	startTime := time.Now().Unix()
	cmdPath, args, dstPath := getter.parseCommand(url, header, name)
	cmd := exec.Command(cmdPath, args...)
	_, err := cmd.CombinedOutput()

	if cmd.ProcessState.Success() {
		log.Infof("dfget url:%s [SUCCESS] cost:%ds", url, time.Now().Unix()-startTime)
		return dstPath, nil
	} else {
		if value, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if value.ExitStatus() == constant.CODE_REQ_AUTH {
				return "", &exception.AuthError{}
			}
		}
		return "", errors.New(fmt.Sprintf("dfget fail(%s):%v", cmd.ProcessState.String(), err))
	}
}

func (dfgetter *DFgetter) parseCommand(url string, header map[string][]string, name string) (cmdPath string, args []string, dstPath string) {
	args = make([]string, 0, 32)
	args = append(append(args, "-u"), url)
	args = append(append(args, "-o"), getter.dstDir+name)
	if getter.notbs {
		args = append(args, "--notbs")
	}

	if strings.TrimSpace(getter.callSystem) != "" {
		args = append(append(args, "--callsystem"), strings.TrimSpace(getter.callSystem))
	}
	if strings.TrimSpace(getter.urlFilter) != "" {
		args = append(append(args, "-f"), strings.TrimSpace(getter.urlFilter))
	}
	if strings.TrimSpace(getter.rateLimit) != "" {
		args = append(append(args, "-s"), getter.rateLimit)
		args = append(append(args, "--totallimit"), getter.rateLimit)
	}

	if header != nil {
		for key, value := range header {
			if value != nil && len(value) > 0 {
				for _, oneV := range value {
					args = append(append(args, "--header"), fmt.Sprintf("%s:%s", key, oneV))
				}
			} else {
				args = append(append(args, "--header"), fmt.Sprintf("%s:%s", key, ""))
			}

		}
	}

	args = append(args, "--dfdaemon")

	dstPath = getter.dstDir + name
	cmdPath = G_CommandLine.DfPath

	return
}

func DownloadByGetter(url string, header map[string][]string, name string) (string, error) {
	log.Infof("start download url:%s to %s in repo", url, name)
	getter.once.Do(func() {
		getter.dstDir = G_CommandLine.DFRepo
		getter.callSystem = G_CommandLine.CallSystem
		getter.notbs = G_CommandLine.Notbs
		getter.rateLimit = G_CommandLine.RateLimit
		getter.urlFilter = G_CommandLine.Urlfilter
	})
	return getter.Download(url, header, name)
}
