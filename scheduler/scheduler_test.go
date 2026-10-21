/*
 *     Copyright 2020 The Dragonfly Authors
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

package scheduler

import (
	"errors"
	"net"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestGetSystemSomaxconn(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		err  error
		want int
	}{
		{name: "valid", data: []byte("8192\n"), want: 8192},
		{name: "invalid", data: []byte("not-a-number"), want: 4096},
		{name: "non-positive", data: []byte("0"), want: 4096},
		{name: "read error", err: errors.New("read failed"), want: 4096},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, getSystemSomaxconnFrom(func(string) ([]byte, error) {
				return tc.data, tc.err
			}))
		})
	}

	assert.Positive(t, getSystemSomaxconn())
}

func TestListenWithCustomBacklog(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("custom listener uses Linux socket syscalls")
	}

	listener, err := listenWithCustomBacklog("tcp4", "127.0.0.1:0")
	assert.NoError(t, err)
	if !assert.NotNil(t, listener) {
		return
	}
	defer func() {
		_ = listener.Close()
	}()

	address, ok := listener.Addr().(*net.TCPAddr)
	if assert.True(t, ok) {
		assert.NotZero(t, address.Port)
	}
}

func TestListenWithCustomBacklogErrors(t *testing.T) {
	tests := []struct {
		name    string
		network string
		addr    string
		step    string
	}{
		{name: "invalid address", network: "tcp", addr: "invalid-address", step: "resolve"},
		{name: "socket", network: "tcp4", addr: "127.0.0.1:0", step: "socket"},
		{name: "reuse address", network: "tcp4", addr: "127.0.0.1:0", step: "setsockopt"},
		{name: "bind", network: "tcp4", addr: "127.0.0.1:0", step: "bind"},
		{name: "listen", network: "tcp4", addr: "127.0.0.1:0", step: "listen"},
		{name: "file listener", network: "tcp4", addr: "127.0.0.1:0", step: "fileListener"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ops := defaultListenerOps()
			calls := 0
			fail := func() error {
				calls++
				return errors.New(tc.step + " failed")
			}
			if tc.step == "socket" {
				ops.socket = func(int, int, int) (int, error) { return -1, fail() }
			}
			if tc.step == "setsockopt" {
				ops.setsockoptInt = func(int, int, int, int) error { return fail() }
			}
			if tc.step == "bind" {
				ops.bind = func(int, unix.Sockaddr) error { return fail() }
			}
			if tc.step == "listen" {
				ops.listen = func(int, int) error { return fail() }
			}
			if tc.step == "fileListener" {
				ops.fileListener = func(*os.File) (net.Listener, error) { return nil, fail() }
			}

			_, err := listenWithCustomBacklogWithOps(tc.network, tc.addr, ops)
			if tc.step == "resolve" {
				assert.Error(t, err)
				return
			}
			assert.Error(t, err)
			assert.Equal(t, 1, calls)
		})
	}
}
