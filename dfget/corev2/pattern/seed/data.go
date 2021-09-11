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
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/dragonflyoss/Dragonfly/pkg/protocol"
)

const (
	SeedDataType = "SeedData"
)

var (
	defaultSeedDataType = &seedDataType{}
)

type seedDataType struct{}

func (dt seedDataType) String() string {
	return SeedDataType
}

func (dt seedDataType) Encoder() protocol.DataEncoder {
	return nil
}

// if set direct, content will only could be read once; else will be cached in buffer.
func NewSeedData(rc io.ReadCloser, size int64, direct bool) (protocol.DistributionData, error) {
	data := &seedData{
		rc:     rc,
		size:   size,
		direct: direct,
	}

	// cache buffer
	if !direct {
		defer rc.Close()
		buf := &bytes.Buffer{}
		n, _ := io.Copy(buf, rc)
		if n != size {
			return nil, fmt.Errorf("the count of data not expected, expect %d, but got %d", size, n)
		}

		data.buf = buf
	}

	return data, nil
}

type seedData struct {
	rc     io.ReadCloser
	size   int64
	direct bool
	buf    *bytes.Buffer

	sync.Mutex
	readCount int
}

func (sd *seedData) Type() protocol.DataType {
	return defaultSeedDataType
}

// Size gets the size of data.
func (sd *seedData) Size() int64 {
	return sd.size
}

// Metadata gets the metadata.
func (sd *seedData) Metadata() interface{} {
	return nil
}

// Content gets the content of data.
func (sd *seedData) Content(ctx context.Context) (io.Reader, error) {
	sd.Lock()
	defer sd.Unlock()

	if sd.direct && sd.readCount > 0 {
		return nil, fmt.Errorf("not allow to read again")
	}

	if sd.direct {
		sd.readCount++
		return sd.rc, nil
	}

	return bytes.NewReader(sd.buf.Bytes()), nil
}
