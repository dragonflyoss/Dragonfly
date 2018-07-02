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
