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
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/exception"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/global"
)

// Downloader is an interface to download file
type Downloader interface {
	// Download download url file to file name
	// return dst path and download error
	Download(url string, header map[string][]string, name string) (string, error)
}

// DFGetter implements Downloader to download file by dragonfly
type DFGetter struct {
	// output dir
	dstDir string
	// the urlfilter param of dfget
	urlFilter string
	// the totallimit and locallimit(s) param of dfget
	rateLimit string
	// the callsystem param of dfget
	callSystem string
	// the notbs param of dfget
	notbs bool

	once sync.Once
}

// NewDFGetter returns the default DFGetter.
func NewDFGetter() *DFGetter {
	return &DFGetter{}
}

// Download is the method of DFGetter to download by dragonfly.
func (dfGetter *DFGetter) Download(url string, header map[string][]string, name string) (string, error) {
	startTime := time.Now().Unix()
	cmdPath, args, dstPath := dfGetter.parseCommand(url, header, name)
	cmd := exec.Command(cmdPath, args...)
	_, err := cmd.CombinedOutput()

	if cmd.ProcessState.Success() {
		log.Infof("dfget url:%s [SUCCESS] cost:%ds", url, time.Now().Unix()-startTime)
		return dstPath, nil
	}
	if value, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		if value.ExitStatus() == constant.CodeReqAuth {
			return "", &exception.AuthError{}
		}
	}
	return "", fmt.Errorf("dfget fail(%s):%v", cmd.ProcessState.String(), err)
}

func (dfGetter *DFGetter) parseCommand(url string, header map[string][]string, name string) (
	cmdPath string, args []string, dstPath string) {
	args = make([]string, 0, 32)
	args = append(append(args, "-u"), url)
	args = append(append(args, "-o"), dfGetter.dstDir+name)
	if dfGetter.notbs {
		args = append(args, "--notbs")
	}

	if strings.TrimSpace(dfGetter.callSystem) != "" {
		args = append(append(args, "--callsystem"), strings.TrimSpace(dfGetter.callSystem))
	}
	if strings.TrimSpace(dfGetter.urlFilter) != "" {
		args = append(append(args, "-f"), strings.TrimSpace(dfGetter.urlFilter))
	}
	if strings.TrimSpace(dfGetter.rateLimit) != "" {
		args = append(append(args, "-s"), dfGetter.rateLimit)
		args = append(append(args, "--totallimit"), dfGetter.rateLimit)
	}

	if header != nil {
		for key, value := range header {
			// discard HTTP host header for backing to source successfully
			if strings.EqualFold(key, "host") {
				continue
			}
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

	dstPath = dfGetter.dstDir + name
	cmdPath = global.CommandLine.DfPath

	return
}
