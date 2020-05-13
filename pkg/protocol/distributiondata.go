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

package protocol

import (
	"context"
	"io"
)

// DataEncoder defines how to encode/decode data.
type DataEncoder interface {
	// Encode data.
	Encode(io.Reader) (io.Reader, error)

	// Decode data.
	Decode(io.Reader) (io.Reader, error)
}

// DataType defines the type of DistributionData.
type DataType interface {
	// String return the type string.
	String() string

	// Encoder return the encoder of the type.
	Encoder() DataEncoder
}

// DistributionData defines the protocol of distribute data which is exchanged in peers.
type DistributionData interface {
	// Type gets the data type.
	Type() DataType

	// Size gets the size of data.
	Size() int64

	// Metadata gets the metadata.
	Metadata() interface{}

	// Content gets the content of data.
	Content(ctx context.Context) (io.Reader, error)
}

type eofDataType struct{}

func (ty eofDataType) String() string {
	return "EOF"
}

func (ty *eofDataType) Encoder() DataEncoder {
	return nil
}

// EOFDistributionData represents the eof of file.
type eofDistributionData struct{}

func (eof *eofDistributionData) Size() int64 {
	return 0
}

func (eof *eofDistributionData) Metadata() interface{} {
	return nil
}

func (eof *eofDistributionData) Content(ctx context.Context) (io.Reader, error) {
	return nil, io.EOF
}

func (eof *eofDistributionData) Type() DataType {
	return &eofDataType{}
}

func NewEoFDistributionData() DistributionData {
	return &eofDistributionData{}
}
