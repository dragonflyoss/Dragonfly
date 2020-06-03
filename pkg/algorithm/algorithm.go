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

import (
	"math/rand"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ContainsString returns whether the value is in arr.
func ContainsString(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

// Shuffle pseudo-randomizes the order of elements.
// n is the number of elements.
// swap swaps the elements with indexes i and j.
// copy from rand.Shuffle of go1.10.
func Shuffle(n int, swap func(int, int)) {
	if n < 2 {
		return
	}
	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j := int(rand.Int63n(int64(i + 1)))
		swap(i, j)
	}
	for ; i > 0; i-- {
		j := int(int31n(int32(i + 1)))
		swap(i, j)
	}
}

func int31n(n int32) int32 {
	v := rand.Uint32()
	prod := uint64(v) * uint64(n)
	low := uint32(prod)
	if low < uint32(n) {
		thresh := uint32(-n) % uint32(n)
		for low < thresh {
			v = rand.Uint32()
			prod = uint64(v) * uint64(n)
			low = uint32(prod)
		}
	}
	return int32(prod >> 32)
}

// GCDSlice returns the greatest common divisor of a slice.
// It returns 1 when s is empty because that any number divided by 1 is still
// itself.
func GCDSlice(s []int) int {
	length := len(s)
	if length == 0 {
		return 1
	}
	if length == 1 {
		return s[0]
	}
	commonDivisor := s[0]
	for i := 1; i < length; i++ {
		if commonDivisor == 1 {
			return commonDivisor
		}
		commonDivisor = GCD(commonDivisor, s[i])
	}
	return commonDivisor
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

// DedupStringArr removes duplicate string in array.
func DedupStringArr(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}

	out := make([]string, len(input))
	copy(out, input)
	sort.Strings(out)

	idx := 0
	for i := 1; i < len(input); i++ {
		if out[idx] != out[i] {
			idx++
			out[idx] = out[i]
		}
	}

	return out[:idx+1]
}
