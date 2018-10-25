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

package downloader

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"

	"github.com/alibaba/Dragonfly/dfget/util"
)

// NewLimitReader create LimitReader
// src: reader
// rate: bytes/second
func NewLimitReader(src io.Reader, rate int, calculateMd5 bool) *LimitReader {
	var md5sum hash.Hash
	if calculateMd5 {
		md5sum = md5.New()
	}
	if rate <= 0 {
		rate = 10 * 1024 * 1024
	}
	rate = (rate/1000 + 1) * 1000
	return &LimitReader{
		Src:     src,
		Limiter: util.NewRateLimiter(int32(rate), 2),
		md5sum:  md5sum,
	}
}

// LimitReader read stream with RateLimiter.
type LimitReader struct {
	Src     io.Reader
	Limiter *util.RateLimiter
	md5sum  hash.Hash
}

func (lr *LimitReader) Read(p []byte) (n int, err error) {
	n, e := lr.Src.Read(p)
	if e != nil && e != io.EOF {
		return n, e
	}
	if n > 0 {
		if lr.md5sum != nil {
			lr.md5sum.Write(p[:n])
		}
		lr.Limiter.AcquireBlocking(int32(n))
	}
	return n, e
}

// Md5 calculate the md5 of all contents read
func (lr *LimitReader) Md5() string {
	if lr.md5sum != nil {
		return fmt.Sprintf("%x", lr.md5sum.Sum(nil))
	}
	return ""
}
