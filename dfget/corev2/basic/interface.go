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

package basic

// Response defines the response.
type Response interface {
	Success() bool
	Data() interface{}
}

// RangeRequest defines the range request.
type RangeRequest interface {
	URL() string
	Offset() int64
	Size() int64
	Header() map[string]string

	// Extra gets the extra info.
	Extra() interface{}
}

// NotifyResult defines the result of notify.
type NotifyResult interface {
	Success() bool
	Error() error
	Data() interface{}
}

// Notify defines how to notify asynchronous call if finished and get the result.
type Notify interface {
	// Done returns a channel that's closed when work done.
	Done() <-chan struct{}

	// Result returns the NotifyResult and only valid after Done channel is closed.
	Result() NotifyResult
}
