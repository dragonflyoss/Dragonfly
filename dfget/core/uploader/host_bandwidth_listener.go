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

package uploader

import (
	"runtime"
	"time"

	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

var bandInfo bandStat

// the rate is calculate as 90 percent of usable bandwidth
func getHostDynamicRate(maxRate int64) (int64, error) {
	currentRate, err := getLocalRate(bandInfo.inter, 10)
	if err != nil {
		return 0, err
	}

	//dynamicRate is the 90% of the usable bandwidth
	dynamicRate := (maxRate - currentRate) * 9 / 10

	if dynamicRate < 0 {
		return 0, fmt.Errorf("dynamic rate caculate smaller than 0, some error happened")
	}

	return dynamicRate, nil
}

type netStat struct {
	Dev  []string
	Stat map[string]*devStat
}

type devStat struct {
	Name string
	Rx   uint64
	Tx   uint64
}

type bandStat struct {
	inter   string
	maxRate int64
}

// determine the main net interface
func getHostNetInter() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("dynamic rate mode only supports for linux")
	}
	stat1 := getInfo("*")
	time.Sleep(time.Duration(5) * time.Second)
	stat2 := getInfo("*")

	//calculate rate to find out the main net interface
	for _, key := range stat1.Dev {
		if _, ok := stat2.Stat[key]; ok {
			rate := calculateLocalRate(&stat1, &stat2, 5, key)
			if rate > bandInfo.maxRate {
				bandInfo.maxRate = rate
				bandInfo.inter = key
			}
		}
	}
	if bandInfo.inter == "" {
		return fmt.Errorf("failed to detect net interface")
	}
	return nil
}

func readLines(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}
	return ret, nil
}

// get network information by read "/proc/net/dev" in Linux
func getInfo(inter string) (ret netStat) {

	lines, _ := readLines("/proc/net/dev")

	ret.Dev = make([]string, 0)
	ret.Stat = make(map[string]*devStat)

	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.Fields(strings.TrimSpace(fields[1]))

		if inter != "*" && inter != key {
			continue
		}

		c := new(devStat)
		c.Name = key
		r, err := strconv.ParseInt(value[0], 10, 64)
		if err != nil {
			logrus.Errorf("parse interface %s Rx rate %s to type int64 fail, error: %v", key, value[0], err)
			break
		}
		c.Rx = uint64(r)

		t, err := strconv.ParseInt(value[8], 10, 64)
		if err != nil {
			logrus.Errorf("parse interface %s Tx rate %s to type int64 fail, error: %v", key, value[0], err)
			break
		}
		c.Tx = uint64(t)

		ret.Dev = append(ret.Dev, key)
		ret.Stat[key] = c
	}

	return
}

// get host network interface information
func getLocalRate(inter string, t int) (int64, error) {
	stat1 := getInfo(inter)
	if _, ok := stat1.Stat[inter]; !ok {
		return 0, fmt.Errorf("failed to get network interface: %s", inter)
	}

	time.Sleep(time.Duration(t) * time.Second)

	stat2 := getInfo(inter)
	if _, ok := stat1.Stat[inter]; !ok {
		return 0, fmt.Errorf("failed to get network interface: %s", inter)
	}

	return calculateLocalRate(&stat1, &stat2, t, inter), nil
}

// calculate total rate which is addition of receive rate and transfer rate
func calculateLocalRate(stat1 *netStat, stat2 *netStat, t int, inter string) int64 {

	//calculate only tx because we only have to control the
	return int64(stat2.Stat[inter].Tx-stat1.Stat[inter].Tx) / int64(t)
}
