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

package stringutils

import "unicode"

// SubString returns the subString of {str} which begins at {start} and end at {end - 1}.
func SubString(str string, start, end int) string {
	runes := []rune(str)
	length := len(runes)
	if start < 0 || start >= length ||
		end <= 0 || end > length ||
		start > end {
		return ""
	}

	return string(runes[start:end])
}

// IsEmptyStr returns whether the string s is empty.
func IsEmptyStr(s string) bool {
	for _, v := range s {
		if !unicode.IsSpace(v) {
			return false
		}
	}
	return true
}
