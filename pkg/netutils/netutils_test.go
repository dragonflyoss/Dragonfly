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

package netutils

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"

	"github.com/go-check/check"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type NetUtilSuite struct{}

func init() {
	check.Suite(&NetUtilSuite{})
}

func (suite *NetUtilSuite) TestExtractHost(c *check.C) {
	host := ExtractHost("1:0")
	c.Assert(host, check.Equals, "1")
}

func (suite *NetUtilSuite) TestGetIPAndPortFromNode(c *check.C) {
	var cases = []struct {
		node         string
		defaultPort  int
		expectedIP   string
		expectedPort int
	}{
		{"127.0.0.1", 8002, "127.0.0.1", 8002},
		{"127.0.0.1:8001", 8002, "127.0.0.1", 8001},
		{"127.0.0.1:abcd", 8002, "127.0.0.1", 8002},
	}

	for _, v := range cases {
		ip, port := GetIPAndPortFromNode(v.node, v.defaultPort)
		c.Check(ip, check.Equals, v.expectedIP)
		c.Check(port, check.Equals, v.expectedPort)
	}
}

func (suite *NetUtilSuite) TestNetLimit(c *check.C) {
	speed := NetLimit()
	if runtime.NumCPU() < 24 {
		c.Assert(*speed, check.Equals, 20*rate.MB)
	}
}

func (suite *NetUtilSuite) TestFilterURLParam(c *check.C) {
	var cases = []struct {
		url      string
		filter   []string
		expected string
	}{
		{
			url:      "http://www.a.b.com",
			filter:   nil,
			expected: "http://www.a.b.com",
		},
		{
			url:      "http://www.a.b.com?key1=value1",
			filter:   nil,
			expected: "http://www.a.b.com?key1=value1",
		},
		{
			url:      "http://www.a.b.com?key1=value1",
			filter:   []string{"key1"},
			expected: "http://www.a.b.com",
		},
		{
			url:      "http://www.a.b.com?key1=value1",
			filter:   []string{"key2"},
			expected: "http://www.a.b.com?key1=value1",
		},
		{
			url:      "http://www.a.b.com?key1=value1&key2=value2",
			filter:   []string{"key2"},
			expected: "http://www.a.b.com?key1=value1",
		},
		{
			url:      "http://www.a.b.com?key1=value1&key2=value2&key3=value3",
			filter:   []string{"key2"},
			expected: "http://www.a.b.com?key1=value1&key3=value3",
		},
		{
			url:      "http://www.a.b.com?key1=value1&key2=value2&key3=value3",
			filter:   []string{"key2", "key3"},
			expected: "http://www.a.b.com?key1=value1",
		},
		{
			url:      "http://www.a.b.com?key1=value1&key2=value2&key3=value3",
			filter:   []string{"key2 ", "key3"},
			expected: "http://www.a.b.com?key1=value1&key2=value2",
		},
		{
			url:      "http://www.a.b.com?key1=value1&key2=value2&key1=value3",
			filter:   []string{"key1"},
			expected: "http://www.a.b.com?key2=value2",
		},
	}
	for _, v := range cases {
		result := FilterURLParam(v.url, v.filter)
		c.Assert(result, check.Equals, v.expected)
	}
}

func (suite *NetUtilSuite) TestIsValidURL(c *check.C) {
	httputils.RegisterProtocol("test", nil)
	httputils.RegisterProtocol("TEST", nil)
	var cases = map[string]bool{
		"":                      false,
		"abcdefg":               true,
		"////a//":               false,
		"a////a//":              true,
		"a/b":                   true,
		"a.com////a//":          true,
		"a:b@a.com":             true,
		"a:b@127.0.0.1":         true,
		"a:b@127.0.0.1?a=b":     true,
		"a:b@127.0.0.1?a=b&c=d": true,
		"127.0.0.1":             true,
		"127.0.0.1?a=b":         true,
		"127.0.0.1:":            true,
		"127.0.0.1:8080":        true,
		"127.0.0.1:8080/我":      true,
		"127.0.0.1:8080/我?x=1":  true,
		"a.b":                   true,
		"www.taobao.com":        true,
		"github.com/dragonflyoss/Dragonfly/issues?" +
			"q=is%3Aissue+is%3Aclosed": true,
		// FIXME because x://y/z is valid, below urls is valid
		//"http:/www.a.b.com":     false,
		//"https://github.com/dragonflyoss/Dragonfly/issues?" +
		//	"q=is%3Aissue+is%3Aclosed": false,
	}

	for k, v := range cases {
		for _, scheme := range []string{"http", "https", "HTTP", "HTTPS", "test", "TEST"} {
			url := fmt.Sprintf("%s://%s", scheme, k)
			result := IsValidURL(url)
			c.Logf("%v %s", result, url)
			c.Assert(result, check.Equals, v)
		}
	}
}

func (suite *NetUtilSuite) TestIsValidIP(c *check.C) {
	var cases = []struct {
		ip       string
		expected bool
	}{
		{
			ip:       "192.168.1.1",
			expected: true,
		},
		{
			ip:       "0.0.0.0",
			expected: true,
		},
		{
			ip:       "255.255.255.255",
			expected: true,
		},
		{
			ip:       "256.255.255.255",
			expected: false,
		},
		{
			ip:       "aaa.255.255.255",
			expected: false,
		},
	}
	for _, v := range cases {
		result := IsValidIP(v.ip)
		c.Assert(result, check.Equals, v.expected)
	}
}

func (suite *NetUtilSuite) TestConvertHeaders(c *check.C) {
	cases := []struct {
		h []string
		e map[string]string
	}{
		{
			h: []string{"a:1", "a:2", "b:", "b", "c:3"},
			e: map[string]string{"a": "1,2", "c": "3"},
		},
		{
			h: []string{},
			e: nil,
		},
	}
	for _, v := range cases {
		headers := ConvertHeaders(v.h)
		c.Assert(headers, check.DeepEquals, v.e)
	}
}

func (suite *NetUtilSuite) TestConvertTimeStringToInt(c *check.C) {
	timeStr := "Fri, 15 Jun 2018 14:40:41 GMT"
	result, err := ConvertTimeStringToInt(timeStr)
	c.Check(err, check.IsNil)
	c.Check(result, check.Equals, int64(1529073641000))
}

func (suite *NetUtilSuite) TestConvertTimeIntToString(c *check.C) {
	timestamp := int64(1529073641000)
	result, err := ConvertTimeIntToString(timestamp)
	c.Check(err, check.IsNil)
	c.Check(result, check.Equals, "Fri, 15 Jun 2018 14:40:41 GMT")
}

func (suite *NetUtilSuite) TestCalculateTimeout(c *check.C) {
	var cases = []struct {
		fileLength     int64
		minRate        rate.Rate
		defaultMinRate rate.Rate
		reservedTime   time.Duration
		expectedResult time.Duration
	}{
		{
			fileLength:     1000,
			minRate:        20,
			defaultMinRate: 50,
			reservedTime:   0,
			expectedResult: time.Duration(50) * time.Second,
		},
		{
			fileLength:     1000,
			minRate:        50,
			defaultMinRate: 20,
			reservedTime:   100 * time.Millisecond,
			expectedResult: time.Duration(20)*time.Second + 100*time.Millisecond,
		},
		{
			fileLength:     1000,
			minRate:        0,
			defaultMinRate: 50,
			reservedTime:   0,
			expectedResult: time.Duration(20) * time.Second,
		},
		{
			fileLength:     1000,
			minRate:        0,
			defaultMinRate: 0,
			reservedTime:   0,
			expectedResult: 0,
		},
	}

	for _, ca := range cases {
		result := CalculateTimeout(ca.fileLength, ca.minRate, ca.defaultMinRate, ca.reservedTime)
		c.Assert(result, check.DeepEquals, ca.expectedResult)
	}
}
