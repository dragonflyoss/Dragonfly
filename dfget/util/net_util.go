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
	"strconv"
	"strings"
)

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
