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

package peer

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	dutil "github.com/dragonflyoss/Dragonfly/supernode/daemon/util"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/go-check/check"
	"github.com/prometheus/client_golang/prometheus"
	prom_testutil "github.com/prometheus/client_golang/prometheus/testutil"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

func init() {
	check.Suite(&PeerMgrTestSuite{})
}

type PeerMgrTestSuite struct {
}

func (s *PeerMgrTestSuite) TestPeerMgr(c *check.C) {
	manager, _ := NewManager(prometheus.NewRegistry())
	peers := manager.metrics.peers
	// register
	request := &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo",
		Port:     65001,
		Version:  version.DFGetVersion,
	}
	resp, err := manager.Register(context.Background(), request)
	c.Check(err, check.IsNil)

	c.Assert(1, check.Equals,
		int(prom_testutil.ToFloat64(peers.WithLabelValues("192.168.10.11"))))
	// get
	id := resp.ID
	info, err := manager.Get(context.Background(), id)
	c.Check(err, check.IsNil)
	expected := &types.PeerInfo{
		ID:       id,
		IP:       request.IP,
		HostName: request.HostName,
		Port:     request.Port,
		Version:  request.Version,
		Created:  info.Created,
	}
	c.Check(info, check.DeepEquals, expected)

	// list
	infoList, err := manager.List(context.Background(), nil)
	c.Check(err, check.IsNil)
	c.Check(infoList, check.DeepEquals, []*types.PeerInfo{expected})

	// deRegister
	err = manager.DeRegister(context.Background(), id)
	c.Check(err, check.IsNil)

	c.Assert(0, check.Equals,
		int(prom_testutil.ToFloat64(peers.WithLabelValues("192.168.10.11"))))

	// get
	info, err = manager.Get(context.Background(), id)
	c.Check(errortypes.IsDataNotFound(err), check.Equals, true)
	c.Check(info, check.IsNil)
}

func (s *PeerMgrTestSuite) TestGet(c *check.C) {
	manager, _ := NewManager(prometheus.NewRegistry())

	// register
	request := &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo",
		Port:     65001,
		Version:  version.DFGetVersion,
	}
	resp, err := manager.Register(context.Background(), request)
	c.Check(err, check.IsNil)

	// get with empty peerID
	info, err := manager.Get(context.Background(), "")
	c.Check(errortypes.IsEmptyValue(err), check.Equals, true)
	c.Check(info, check.IsNil)

	// get with not exist peerID
	info, err = manager.Get(context.Background(), "fooerror")
	c.Check(errortypes.IsDataNotFound(err), check.Equals, true)
	c.Check(info, check.IsNil)

	// get normally
	id := resp.ID
	info, err = manager.Get(context.Background(), id)
	c.Check(err, check.IsNil)
	expected := &types.PeerInfo{
		ID:       id,
		IP:       request.IP,
		HostName: request.HostName,
		Port:     request.Port,
		Version:  request.Version,
		Created:  info.Created,
	}
	c.Check(info, check.DeepEquals, expected)
}

func (s *PeerMgrTestSuite) TestGetAllPeerIDs(c *check.C) {
	manager, _ := NewManager(prometheus.NewRegistry())

	// the first data
	request := &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo",
		Port:     65001,
		Version:  version.DFGetVersion,
	}
	resp, err := manager.Register(context.Background(), request)
	c.Check(err, check.IsNil)
	id := resp.ID

	// the second data
	request = &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "bar",
		Port:     65001,
		Version:  version.DFGetVersion,
	}
	resp, err = manager.Register(context.Background(), request)
	c.Check(err, check.IsNil)
	id2 := resp.ID

	// get all peer ids
	ids := manager.GetAllPeerIDs(context.Background())
	sort.Strings(ids)
	c.Check(ids, check.DeepEquals, []string{id2, id})
}

func (s *PeerMgrTestSuite) TestList(c *check.C) {
	manager, _ := NewManager(prometheus.NewRegistry())
	// the first data
	request := &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo",
		Port:     65001,
		Version:  version.DFGetVersion,
	}
	resp, err := manager.Register(context.Background(), request)
	c.Check(err, check.IsNil)
	id := resp.ID
	info, err := manager.Get(context.Background(), id)
	c.Check(err, check.IsNil)

	// the second data
	request = &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo2",
		Port:     65001,
		Version:  version.DFGetVersion,
	}
	resp, err = manager.Register(context.Background(), request)
	c.Check(err, check.IsNil)
	id = resp.ID
	info2, err := manager.Get(context.Background(), id)
	c.Check(err, check.IsNil)

	// get all
	infoList, err := manager.List(context.Background(), nil)
	c.Check(err, check.IsNil)
	interfaceSlice := make([]interface{}, len(infoList))
	for k, v := range infoList {
		interfaceSlice[k] = v
	}
	c.Check(dutil.GetPageValues(interfaceSlice, 0, 0, func(i, j int) bool {
		peeri := interfaceSlice[i].(*types.PeerInfo)
		peerj := interfaceSlice[j].(*types.PeerInfo)
		return time.Time(peeri.Created).Before(time.Time(peerj.Created))
	}), check.DeepEquals, []interface{}{info, info2})

	// get with pageNum=0 && pageSize=1 && sortDirect=asc
	infoList, err = manager.List(context.Background(), &dutil.PageFilter{
		PageNum:    0,
		PageSize:   1,
		SortDirect: "asc",
	})
	c.Check(err, check.IsNil)
	c.Check(infoList, check.DeepEquals, []*types.PeerInfo{info})

	// get with pageNum=0 && pageSize=1 && sortDirect=desc
	infoList, err = manager.List(context.Background(), &dutil.PageFilter{
		PageNum:    0,
		PageSize:   1,
		SortDirect: "desc",
	})
	c.Check(err, check.IsNil)
	c.Check(infoList, check.DeepEquals, []*types.PeerInfo{info2})

	// get with pageNum=1 && pageSize=1 && sortDirect=asc
	infoList, err = manager.List(context.Background(), &dutil.PageFilter{
		PageNum:    1,
		PageSize:   1,
		SortDirect: "asc",
	})
	c.Check(err, check.IsNil)
	c.Check(infoList, check.DeepEquals, []*types.PeerInfo{info2})
}
