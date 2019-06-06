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

	log "github.com/sirupsen/logrus"
)

const (
	separator = "&"
)

var defaultRateLimit = "20M"

// NetLimit parse speed of interface that it has prefix of eth
func NetLimit() string {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("parse default net limit error:%v", err)
		}
	}()
	if runtime.NumCPU() < 24 {
		return defaultRateLimit
	}

	var ethtool string
	if path, err := exec.LookPath("ethtool"); err == nil {
		ethtool = path
	} else if _, err := os.Stat("/usr/sbin/ethtool"); err == nil || os.IsExist(err) {
		ethtool = "/usr/sbin/ethtool"
	}
	if ethtool == "" {
		log.Warn("ethtool not found")
		return defaultRateLimit
	}

	var maxInterfaceLimit = uint64(0)
	interfaces, err := net.Interfaces()
	if err != nil {
		return defaultRateLimit
	}
	compile := regexp.MustCompile("^[[:space:]]*([[:digit:]]+)[[:space:]]*Mb/s[[:space:]]*$")

	for _, dev := range interfaces {
		if !strings.HasPrefix(dev.Name, "eth") {
			continue
		}
		cmd := exec.Command(ethtool, dev.Name)
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			continue
		}

		if err := cmd.Start(); err != nil {
			log.Warnf("ethtool %s error:%v", dev.Name, err)
			continue
		}
		scanner := bufio.NewScanner(stdoutPipe)

		for scanner.Scan() {
			fields := strings.Split(strings.TrimSpace(scanner.Text()), ":")
			if len(fields) != 2 {
				continue
			}
			if strings.ToLower(strings.TrimSpace(fields[0])) != "speed" {
				continue
			}
			speed := compile.FindStringSubmatch(fields[1])
			if tmpLimit, err := strconv.ParseUint(speed[1], 0, 32); err == nil {
				tmpLimit = tmpLimit * 8 / 10
				if tmpLimit > maxInterfaceLimit {
					maxInterfaceLimit = tmpLimit
				}
			}

		}
		cmd.Wait()
	}

	if maxInterfaceLimit > 0 {
		return strconv.FormatUint(maxInterfaceLimit/8, 10) + "M"
	}

	return defaultRateLimit
}

// ExtractHost extracts host ip from the giving string.
func ExtractHost(hostAndPort string) string {
	fields := strings.Split(strings.TrimSpace(hostAndPort), ":")
	return fields[0]
}

// GetIPAndPortFromNode return ip and port by parsing the node value.
// It will return defaultPort as the value of port
// when the node is a string without port or with an illegal port.
func GetIPAndPortFromNode(node string, defaultPort int) (string, int) {
	if IsEmptyStr(node) {
		return "", defaultPort
	}

	nodeFields := strings.Split(node, ":")
	switch len(nodeFields) {
	case 1:
		return nodeFields[0], defaultPort
	case 2:
		port, err := strconv.Atoi(nodeFields[1])
		if err != nil {
			return nodeFields[0], defaultPort
		}
		return nodeFields[0], port
	default:
		return "", defaultPort
	}
}

// FilterURLParam filters request queries in URL.
// Eg:
// If you pass parameters as follows:
//     url: http://a.b.com/locate?key1=value1&key2=value2&key3=value3
//     filter: key2
// and then you will get the following value as the return:
//     http://a.b.com/locate?key1=value1&key3=value3
func FilterURLParam(url string, filters []string) string {
	rawUrls := strings.SplitN(url, "?", 2)
	if len(filters) <= 0 || len(rawUrls) != 2 || strings.TrimSpace(rawUrls[1]) == "" {
		return url
	}
	filtersMap := slice2Map(filters)

	var params []string
	for _, param := range strings.Split(rawUrls[1], separator) {
		kv := strings.SplitN(param, "=", 2)
		if !(len(kv) >= 1 && isExist(filtersMap, kv[0])) {
			params = append(params, param)
		}
	}
	if len(params) > 0 {
		return rawUrls[0] + "?" + strings.Join(params, separator)
	}
	return rawUrls[0]
}

// ConvertHeaders converts headers from array type to map type for http request.
func ConvertHeaders(headers []string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	hm := make(map[string]string)
	for _, header := range headers {
		kv := strings.SplitN(header, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		if v == "" {
			continue
		}
		if _, in := hm[k]; in {
			hm[k] = hm[k] + "," + v
		} else {
			hm[k] = v
		}
	}
	return hm
}

// IsValidURL returns whether the string url is a valid HTTP URL.
func IsValidURL(url string) bool {
	// shorter than the shortest case 'http://a.b'
	if len(url) < 10 {
		return false
	}
	reg := regexp.MustCompile(`(https?|HTTPS?)://([\w_]+:[\w_]+@)?([\w-]+\.)+[\w-]+(/[\w- ./?%&=]*)?`)
	if result := reg.FindString(url); IsEmptyStr(result) {
		return false
	}
	return true
}

// IsValidIP returns whether the string ip is a valid IP Address.
func IsValidIP(ip string) bool {
	if strings.TrimSpace(ip) == "" {
		return false
	}

	// str is a regex which matches a digital
	// greater than or equal to 0 and less than or equal to 255
	str := "(?:25[0-5]|2[0-4]\\d|[01]?\\d?\\d)"
	result, err := regexp.MatchString("^(?:"+str+"\\.){3}"+str+"$", ip)
	if err != nil {
		return false
	}

	return result
}

// GetAllIPs returns all non-loopback addresses.
func GetAllIPs() (ipList []string, err error) {
	// get all system's unicast interface addresses.
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	// filter all loopback addresses.
	for _, v := range addrs {
		if ipNet, ok := v.(*net.IPNet); ok {
			if !ipNet.IP.IsLoopback() {
				ipList = append(ipList, ipNet.IP.String())
			}
		}
	}
	return
}

// slice2Map translate a slice to a map with
// the value in slice as the key and true as the value.
func slice2Map(value []string) map[string]bool {
	mmap := make(map[string]bool)
	for _, v := range value {
		mmap[v] = true
	}
	return mmap
}

// isExist returns whether the map contains the key.
func isExist(mmap map[string]bool, key string) bool {
	if _, ok := mmap[key]; ok {
		return true
	}
	return false
}
