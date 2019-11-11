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

// Package util provides some utility tools for other components.
// Such as net-transporting, file-operating, rate-limiter.
package util

import (
	"encoding/json"
	"reflect"
	"strconv"
)

// Max returns the larger of x or y.
func Max(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

// Min returns the smaller of x or y.
func Min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// IsNil returns whether the value  is nil.
func IsNil(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}

	return false
}

// IsTrue returns whether the value is true.
func IsTrue(value bool) bool {
	return value
}

// IsPositive returns whether the value is a positive number.
func IsPositive(value int64) bool {
	return value > 0
}

// IsNatural returns whether the value>=0.
func IsNatural(value string) bool {
	if v, err := strconv.Atoi(value); err == nil {
		return v >= 0
	}
	return false
}

// IsNumeric returns whether the value is a numeric.
// If the bitSize of value below 0 or above 64 an error is returned.
func IsNumeric(value string) bool {
	if _, err := strconv.Atoi(value); err != nil {
		return false
	}
	return true
}

// JSONString returns json string of the v.
func JSONString(v interface{}) string {
	if str, e := json.Marshal(v); e == nil {
		return string(str)
	}
	return ""
}
