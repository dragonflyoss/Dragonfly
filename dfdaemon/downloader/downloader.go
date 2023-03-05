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

package downloader

import (
	"context"
	"fmt"
	"io"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
)

// Interface specifies on how an plugin can download a file.
type Interface interface {
	// DownloadContext downloads the resource as specified in url, and it accepts
	// a context parameter so that it can handle timeouts correctly.
	DownloadContext(ctx context.Context, url string, header map[string][]string, name string) (string, error)
}

type Stream interface {
	// DownloadContext downloads the resource as specified in url, and it accepts
	// a context parameter so that it can handle timeouts correctly.
	DownloadStreamContext(ctx context.Context, url string, header map[string][]string, name string) (io.Reader, error)
}

// Factory is a function that returns a new downloader.
type Factory func() Interface
type StreamFactory func() Stream

type StreamFactoryBuilder func(patternConfig config.DownloadConfig, c config.Properties) Stream

var (
	registerFactory = map[string]StreamFactoryBuilder{}
)

func Register(pattern string, builder StreamFactoryBuilder) {
	registerFactory[pattern] = builder
}

func NewStreamFactory(pattern string, patternConfig config.DownloadConfig, c config.Properties) StreamFactory {
	builder, ok := registerFactory[pattern]
	if !ok {
		panic(fmt.Sprintf("pattern %s not registered", pattern))
	}

	return func() Stream {
		return builder(patternConfig, c)
	}
}
