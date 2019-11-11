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

package algorithm

// GCDSlice returns the greatest common divisor of a slice.
func GCDSlice(s []int) int {
	length := len(s)
	if length == 1 {
		return s[0]
	}

	return GCD(s[length-1], GCDSlice(s[:length-1]))
}

// GCD returns the greatest common divisor of x and y.
func GCD(x, y int) int {
	var z int
	for y != 0 {
		z = x % y
		x = y
		y = z
	}
	return x
}
