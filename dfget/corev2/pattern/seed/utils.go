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

package seed

import (
	"fmt"
	"strings"
)

func FlattenHeader(header map[string][]string) []string {
	var res []string
	for key, value := range header {
		// discard HTTP host header for backing to source successfully
		if strings.EqualFold(key, "host") {
			continue
		}
		if len(value) > 0 {
			for _, v := range value {
				res = append(res, fmt.Sprintf("%s:%s", key, v))
			}
		} else {
			res = append(res, fmt.Sprintf("%s:%s", key, ""))
		}
	}
	return res
}

func CopyHeader(src map[string][]string) map[string][]string {
	ret := make(map[string][]string)
	for k, v := range src {
		value := make([]string, len(v))
		copy(value, v)
		ret[k] = value
	}

	return ret
}

func FilterMatch(filter map[string]map[string]bool, firstKey string, secondKey string, match string) bool {
	v, ok := filter[firstKey]
	if !ok {
		return false
	}

	key := fmt.Sprintf("%s=%s", secondKey, match)
	dv, ok := v[key]
	if !ok {
		return false
	}

	return dv
}
