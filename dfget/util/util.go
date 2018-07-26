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
	"fmt"
	"math/rand"
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
