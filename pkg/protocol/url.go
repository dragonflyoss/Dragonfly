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

// Metadata defines how to operate the metadata.
type Metadata interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{})
	Del(key string)
	All() interface{}
}

// Resource defines the way how to get some information from remote resource.
// An instance should bind a resource.
// Developers can implement their own Resource which could support different protocol.
type Resource interface {
	// Read gets range data from the binding resource.
	Read(ctx context.Context, off int64, size int64) (io.ReadCloser, error)

	// Length gets the length of binding resource.
	Length(ctx context.Context) (int64, error)

	// Metadata gets the metadata of binding resource.
	Metadata(ctx context.Context) (Metadata, error)

	// Expire gets if the binding resource is expired.
	Expire(ctx context.Context) (bool, interface{}, error)

	// Call allows user defined request.
	Call(ctx context.Context, request interface{}) (response interface{}, err error)

	// Close the resource.
	io.Closer
}

// Client defines how to get resource.
type Client interface {
	// GetResource gets resource by argument.
	GetResource(url string, md Metadata) Resource
}

// ClientBuilder defines how to create an instance of Client.
type ClientBuilder interface {
	// NewProtocolClient creates an instance of Client.
	NewProtocolClient(clientOpt interface{}) (Client, error)
}

// ClientRegister defines how to register pair <protocol, ClientBuilder>.
type ClientRegister interface {
	// ClientRegister registers pair <protocol, ClientBuilder>.
	RegisterProtocol(protocol string, builder ClientBuilder)

	// GetClientBuilder gets the ClientBuilder by protocol.
	GetClientBuilder(protocol string) (ClientBuilder, error)
}
