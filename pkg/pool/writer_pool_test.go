// WriterPool is no-op under race detector, so all these tests do not work.
// +build !race

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
	"bytes"
	"io/ioutil"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	// Limit to 1 processor to make sure that the goroutine doesn't migrate
	// to another P between AcquireWriter and ReleaseWriter calls.
	prev := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(prev)

	tmp := writerPool
	writerPool = &sync.Pool{}

	buf := &bytes.Buffer{}
	w1 := AcquireWriter(buf)
	w1.WriteString("test")
	w1.Flush()
	require.Equal(t, "test", buf.String())

	// get the old writer from pool
	ReleaseWriter(w1)
	w2 := AcquireWriter(buf)
	require.True(t, w1 == w2)

	// get a new writer from pool
	w3 := AcquireWriter(buf)
	require.True(t, w1 != w3)

	ReleaseWriter(w2)
	ReleaseWriter(w3)

	writerPool = tmp
	tmp = nil
}

func BenchmarkAcquireWriter(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := AcquireWriter(ioutil.Discard)
		w.WriteString("test")
		ReleaseWriter(w)
	}
}
