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

package seed

import (
	"bytes"
	"io"
)

// CopyBufferToWriterAt copy data from rd and write to io.WriterAt.
func CopyBufferToWriterAt(off int64, writerAt io.WriterAt, rd io.Reader) (n int64, err error) {
	buffer := bytes.NewBuffer(nil)
	bufSize := int64(256 * 1024)

	for {
		_, err := io.CopyN(buffer, rd, bufSize)
		if err != nil && err != io.EOF {
			return 0, err
		}

		wcount, werr := writerAt.WriteAt(buffer.Bytes(), off)
		n += int64(wcount)
		if werr != nil {
			return n, err
		}

		if err == io.EOF {
			return n, io.EOF
		}

		buffer.Reset()
		off += int64(wcount)
	}
}
