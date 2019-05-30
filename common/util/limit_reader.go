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

package util

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"
)

// NewLimitReader create LimitReader
// src: reader
// rate: bytes/second
func NewLimitReader(src io.Reader, rate int, calculateMd5 bool) *LimitReader {
	return NewLimitReaderWithLimiter(newRateLimiterWithDefaultWindow(rate), src, calculateMd5)
}

// NewLimitReaderWithLimiter create LimitReader with a rateLimiter.
// src: reader
// rate: bytes/second
func NewLimitReaderWithLimiter(rateLimiter *RateLimiter, src io.Reader, calculateMd5 bool) *LimitReader {
	var md5sum hash.Hash
	if calculateMd5 {
		md5sum = md5.New()
	}
	return &LimitReader{
		Src:     src,
		Limiter: rateLimiter,
		md5sum:  md5sum,
	}
}

// NewLimitReaderWithMD5Sum create LimitReader with a md5 sum.
// src: reader
// rate: bytes/second
func NewLimitReaderWithMD5Sum(src io.Reader, rate int, md5sum hash.Hash) *LimitReader {
	return NewLimitReaderWithLimiterAndMD5Sum(src, newRateLimiterWithDefaultWindow(rate), md5sum)
}

// NewLimitReaderWithLimiterAndMD5Sum create LimitReader with rateLimiter and md5 sum.
// src: reader
// rate: bytes/second
func NewLimitReaderWithLimiterAndMD5Sum(src io.Reader, rateLimiter *RateLimiter, md5sum hash.Hash) *LimitReader {
	return &LimitReader{
		Src:     src,
		Limiter: rateLimiter,
		md5sum:  md5sum,
	}
}

func newRateLimiterWithDefaultWindow(rate int) *RateLimiter {
	return NewRateLimiter(TransRate(rate), 2)
}

// LimitReader read stream with RateLimiter.
type LimitReader struct {
	Src     io.Reader
	Limiter *RateLimiter
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
