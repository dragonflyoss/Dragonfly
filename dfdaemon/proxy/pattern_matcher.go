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

package proxy

import (
	"net/http"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/downloader"
)

const (
	StrPattern = "dragonfly-pattern"
)

type patternBuildWrapper struct {
	streamFactory downloader.StreamFactory
}

// PatternMatcher matches the request to select pattern to run proxy.
type PatternMatcher struct {
	defaultPattern string
	patternMap     map[string]*patternBuildWrapper
	matchFunc      func(*http.Request) string
	pv             config.Properties
}

func NewPatternMatcher(pv config.Properties, matchFunc func(*http.Request) string) *PatternMatcher {
	matcher := &PatternMatcher{
		pv:             pv,
		defaultPattern: pv.DefaultDownloadMode,
	}

	patternMap := make(map[string]*patternBuildWrapper)
	for _, conf := range pv.DownloadConf {
		pattern := conf.Name
		streamFactory := downloader.NewStreamFactory(pattern, conf, pv)
		patternMap[pattern] = &patternBuildWrapper{
			streamFactory: streamFactory,
		}
	}

	if matchFunc == nil {
		matchFunc = matcher.matchRequest
	}

	matcher.patternMap = patternMap
	matcher.matchFunc = matchFunc

	return matcher
}

// Match matches the input request to select pattern to fetch function of  DownloadStreamContext to proxy.
func (matcher *PatternMatcher) Match(req *http.Request) downloader.Stream {
	pattern := matcher.matchFunc(req)
	if pattern == "" {
		pattern = matcher.defaultPattern
	}

	wrapper, exist := matcher.patternMap[pattern]
	if !exist {
		// if pattern not found, choose defaultPattern.
		wrapper = matcher.patternMap[matcher.defaultPattern]
	}

	// build an instance of downloader.Stream.
	return wrapper.streamFactory()
}

func (matcher *PatternMatcher) matchRequest(req *http.Request) string {
	return req.Header.Get(StrPattern)
}
