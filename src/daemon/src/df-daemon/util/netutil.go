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
package util

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

//ParseInterfaceLimit parse speed of interface that it has prefix of eth
func NetLimit() string {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("parse default net limit error:%v", err)
		}
	}()
	if runtime.NumCPU() >= 24 {
		var ethtool string
		if path, err := exec.LookPath("ethtool"); err == nil {
			ethtool = path
		} else if _, err := os.Stat("/usr/sbin/ethtool"); err == nil || os.IsExist(err) {
			ethtool = "/usr/sbin/ethtool"
		}
		if ethtool == "" {
			log.Warn("ethtool not found")
		} else {
			var maxInterfaceLimit = uint64(0)
			if interfaces, err := net.Interfaces(); err == nil {
				compile := regexp.MustCompile("^[[:space:]]*([[:digit:]]+)[[:space:]]*Mb/s[[:space:]]*$")
				for _, dev := range interfaces {
					if strings.HasPrefix(dev.Name, "eth") {
						cmd := exec.Command(ethtool, dev.Name)
						if stdoutPipe, err := cmd.StdoutPipe(); err == nil {
							if err := cmd.Start(); err != nil {
								log.Warnf("ethtool %s error:%v", dev.Name, err)
							} else {
								scanner := bufio.NewScanner(stdoutPipe)

								for scanner.Scan() {
									fields := strings.Split(strings.TrimSpace(scanner.Text()), ":")
									if len(fields) == 2 {
										if strings.ToLower(strings.TrimSpace(fields[0])) == "speed" {
											speed := compile.FindStringSubmatch(fields[1])
											if tmpLimit, err := strconv.ParseUint(speed[1], 0, 32); err == nil {
												tmpLimit = tmpLimit * 8 / 10
												if tmpLimit > maxInterfaceLimit {
													maxInterfaceLimit = tmpLimit
												}
											}

										}
									}

								}
								cmd.Wait()
							}

						}
					}
				}
			}
			if maxInterfaceLimit > 0 {
				return strconv.FormatUint(maxInterfaceLimit/8, 10) + "M"
			}
		}
	}
	return "20M"

}
