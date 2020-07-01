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

package pool

import (
	"bufio"
	"io"
	"sync"
)

var writerPool = &sync.Pool{}

// defaultSize 1MB
const defaultSize = 1 * 1024 * 1024

// AcquireWriter returns an empty Writer instance from writer pool.
func AcquireWriter(w io.Writer) *bufio.Writer {
	if writer := writerPool.Get(); writer != nil {
		writer := writer.(*bufio.Writer)
		writer.Reset(w)
		return writer
	}
	return bufio.NewWriterSize(w, defaultSize)
}

// ReleaseWriter returns buf acquired via AcquireWriter to writer pool.
// It will flush and reset the writer before putting to writer pool.
func ReleaseWriter(writer *bufio.Writer) {
	if writer != nil {
		_ = writer.Flush()
		writer.Reset(nil)
		writerPool.Put(writer)
	}
}
