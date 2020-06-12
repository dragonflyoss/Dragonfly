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
	"time"
)

type requestState struct {
	// the request url
	url string
	// the time when the url firstly requested
	firstTime time.Time
	// the recent time when the url requested
	recentTime time.Time
}

func newRequestState(url string) *requestState {
	return &requestState{
		url:        url,
		firstTime:  time.Now(),
		recentTime: time.Now(),
	}
}

func (rs *requestState) copy() *requestState {
	return &requestState{
		url:        rs.url,
		firstTime:  rs.firstTime,
		recentTime: rs.recentTime,
	}
}

func (rs *requestState) updateRecentTime() {
	rs.recentTime = time.Now()
}
