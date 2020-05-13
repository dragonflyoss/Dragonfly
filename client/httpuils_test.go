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

package client

import (
	"errors"
	"testing"
)

func TestParseHost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected error
	}{
		{
			name:     "tcp host",
			host:     "tcp://github.com",
			expected: nil,
		},
		{
			name:     "unix host",
			host:     "unix://github.com",
			expected: nil,
		},
		{
			name:     "http host",
			host:     "http://github.com",
			expected: nil,
		},
		{
			name:     "https host",
			host:     "https://github.com",
			expected: nil,
		},
		{
			name:     "not support url scheme",
			host:     "wss://github.com",
			expected: errors.New("not support url scheme wss"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseHost(tt.host)
			if (nil == err) != (nil == tt.expected) {
				t.Errorf("expected: %v, got: %v\n", tt.expected, err)
			}
		})
	}
}
