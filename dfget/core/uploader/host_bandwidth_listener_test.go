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

package uploader

import (
	"github.com/go-check/check"
)

type HostBandwidthListenerTestSuite struct {
	stat1 netStat
	stat2 netStat
}

func init() {
	check.Suite(&HostBandwidthListenerTestSuite{})
}

func (s *HostBandwidthListenerTestSuite) SetUpSuite(*check.C) {
	const eth0 string = "eth0"
	s.stat1.Dev = make([]string, 1)
	s.stat1.Dev[0] = eth0
	s.stat2.Dev = make([]string, 1)
	s.stat2.Dev[0] = eth0

	s.stat1.Stat = make(map[string]*devStat)
	s.stat2.Stat = make(map[string]*devStat)

	devStat1 := devStat{Rx: uint64(1234567890), Tx: uint64(7812638716)}
	devStat2 := devStat{Rx: uint64(1334567890), Tx: uint64(7822638716)}
	s.stat1.Stat["eth0"] = &devStat1
	s.stat2.Stat["eth0"] = &devStat2
}

func (s *HostBandwidthListenerTestSuite) TearDownSuite(*check.C) {

}

func (s *HostBandwidthListenerTestSuite) TestGetLocalRate(c *check.C) {

}

func (s *HostBandwidthListenerTestSuite) TestGetInfo(c *check.C) {
	result := "eth0"
	stat := getInfo("eth0")
	num := uint64(0)
	c.Check(stat.Dev[0], check.Equals, result)
	c.Check(stat.Stat[stat.Dev[0]].Rx, check.FitsTypeOf, num)
	c.Check(stat.Stat[stat.Dev[0]].Tx, check.FitsTypeOf, num)
}

func (s *HostBandwidthListenerTestSuite) TestCalculateLocalRate(c *check.C) {
	rate := calculateLocalRate(&s.stat1, &s.stat2, 2, "eth0")

	c.Check(rate, check.Equals, int64(5000000))
}

func (s *HostBandwidthListenerTestSuite) TestGetHostNetInter(c *check.C) {
	c.Check(bandInfo.maxRate, check.Equals, int64(0))
	c.Check(bandInfo.inter, check.Equals, "")

	err := getHostNetInter()
	if err == nil {
		c.Check(bandInfo.maxRate, check.Not(check.Equals), int64(0))
		c.Check(bandInfo.inter, check.Not(check.Equals), "")
	}
}
