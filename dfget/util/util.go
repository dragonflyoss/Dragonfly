/*
 * Copyright 1999-2018 Alibaba Group.
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
	"fmt"
	"os"
	"reflect"
)

var (
	// Printer is global StdPrinter.
	Printer = &StdPrinter{Out: os.Stdout}
)

// Max returns the larger of x or y.
func Max(x, y int32) int32 {
	if x > y {
		return x
	}
	return y
}

// Min returns the smaller of x or y.
func Min(x, y int32) int32 {
	if x < y {
		return x
	}
	return y
}

// IsEmptyStr returns whether the string x is empty.
func IsEmptyStr(s string) bool {
	return s == ""
}

// IsNil returns whether the value  is nil.
func IsNil(value interface{}) (result bool) {
	if value == nil {
		result = true
	} else {
		switch v := reflect.ValueOf(value); v.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			return v.IsNil()
		}
	}
	return
}

// PanicIfNil panic if the obj is nil.
func PanicIfNil(obj interface{}, msg string) {
	if IsNil(obj) {
		panic(fmt.Errorf(msg))
	}
}

// PanicIfError panic if an error happened.
func PanicIfError(err error, msg string) {
	if err != nil {
		panic(fmt.Errorf("%s: %v", msg, err))
	}
}

// JSONString returns json string of the v.
func JSONString(v interface{}) string {
	if str, e := json.Marshal(v); e == nil {
		return string(str)
	}
	return ""
}
