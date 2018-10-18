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
	"fmt"
	"hash"
	"io"

	"github.com/alibaba/Dragonfly/dfget/util"
)

// NewLimitReader create LimitReader
// src: reader
// rate: bytes/second
func NewLimitReader(src io.Reader, rate int, md5sum hash.Hash) *LimitReader {
	if rate <= 0 {
		rate = 10 * 1024 * 1024
	}
	rate = (rate/1000 + 1) * 1000
	return &LimitReader{
		Src:     src,
		Limiter: util.NewRateLimiter(int32(rate), 2),
		Md5sum:  md5sum,
	}
}

// LimitReader read stream with RateLimiter.
type LimitReader struct {
	Src     io.Reader
	Limiter *util.RateLimiter
	Md5sum  hash.Hash
}

func (lr *LimitReader) Read(p []byte) (n int, err error) {
	n, e := lr.Src.Read(p)
	if e != nil {
		return n, e
	}
	if lr.Md5sum != nil {
		lr.Md5sum.Write(p[:n])
	}
	lr.Limiter.AcquireBlocking(int32(n))
	return n, e
}

// Md5 calculate the md5 of all contents read
func (lr *LimitReader) Md5() string {
	if lr.Md5sum != nil {
		return fmt.Sprintf("%x", lr.Md5sum.Sum(nil))
	}
	return ""
}
