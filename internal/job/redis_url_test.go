/*
 *     Copyright 2026 The Dragonfly Authors
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

package job

import (
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/dragonflyoss/machinery/v1"
)

func TestRedisURL(t *testing.T) {
	for _, tc := range []struct {
		username string
		password string
		db       int
	}{
		{"worker +:@", "test @:/%+", 11},
		{"", "test +:@/%", 12},
		{"", "", 0},
	} {
		addrs := []string{"sentinel-1:26379", "sentinel-2:26379", "sentinel-3:26379"}
		parsed, err := url.Parse(redisURL(addrs, tc.username, tc.password, tc.db))
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Host != strings.Join(addrs, ",") || parsed.Path != "/"+strconv.Itoa(tc.db) {
			t.Fatal("Redis addresses or database changed")
		}
		if tc.username == "" && tc.password == "" {
			if parsed.User != nil {
				t.Fatal("unexpected authentication in URL")
			}
			continue
		}
		password, _ := parsed.User.Password()
		if parsed.User.Username() != tc.username || password != tc.password {
			t.Fatal("URL userinfo did not round trip")
		}
	}
}

func TestRedisURLStandaloneRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name     string
		username string
		password string
	}{
		{"ACL credentials", "worker +:@", "test @:/%+"},
		{"password only", "", "test +:@/%"},
		{"no authentication", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const address = "127.0.0.1:6379"
			const database = 1
			host, username, password, db, err := machinery.ParseRedisURL(redisURL([]string{address}, tc.username, tc.password, database))
			if err != nil {
				t.Fatal(err)
			}
			if host != address || db != database {
				t.Fatal("Redis address or database changed")
			}
			if username != tc.username || password != tc.password {
				t.Fatal("Redis credentials did not round trip through the standalone parser")
			}
		})
	}
}
