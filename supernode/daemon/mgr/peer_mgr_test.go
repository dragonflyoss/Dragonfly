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

package mgr

import (
	"context"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/common/errors"

	"github.com/go-check/check"
)

func init() {
	check.Suite(&PeerMgrTestSuite{})
}

type PeerMgrTestSuite struct {
}

func (s *PeerMgrTestSuite) TestPeerMgr(c *check.C) {
	peerManager, _ := NewPeerManager()

	// register
	request := &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo",
		Port:     65001,
		Version:  "v0.3.0",
	}
	resp, err := peerManager.Register(context.Background(), request)
	c.Check(err, check.IsNil)

	// get
	id := resp.ID
	info, err := peerManager.Get(context.Background(), id)
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
	infoList, err := peerManager.List(context.Background(), nil)
	c.Check(err, check.IsNil)
	c.Check(infoList, check.DeepEquals, []*types.PeerInfo{expected})

	// deRegister
	err = peerManager.DeRegister(context.Background(), id)
	c.Check(err, check.IsNil)

	// get
	info, err = peerManager.Get(context.Background(), id)
	c.Check(errors.IsDataNotFound(err), check.Equals, true)
	c.Check(info, check.IsNil)
}

func (s *PeerMgrTestSuite) TestGet(c *check.C) {
	peerManager, _ := NewPeerManager()

	// register
	request := &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo",
		Port:     65001,
		Version:  "v0.3.0",
	}
	resp, err := peerManager.Register(context.Background(), request)
	c.Check(err, check.IsNil)

	// get with empty peerID
	info, err := peerManager.Get(context.Background(), "")
	c.Check(errors.IsEmptyValue(err), check.Equals, true)
	c.Check(info, check.IsNil)

	// get with not exist peerID
	info, err = peerManager.Get(context.Background(), "fooerror")
	c.Check(errors.IsDataNotFound(err), check.Equals, true)
	c.Check(info, check.IsNil)

	// get normally
	id := resp.ID
	info, err = peerManager.Get(context.Background(), id)
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

func (s *PeerMgrTestSuite) TestList(c *check.C) {
	peerManager, _ := NewPeerManager()
	// the first data
	request := &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo",
		Port:     65001,
		Version:  "v0.3.0",
	}
	resp, err := peerManager.Register(context.Background(), request)
	id := resp.ID
	info, err := peerManager.Get(context.Background(), id)

	// the second data
	request = &types.PeerCreateRequest{
		IP:       "192.168.10.11",
		HostName: "foo2",
		Port:     65001,
		Version:  "v0.3.0",
	}
	resp, err = peerManager.Register(context.Background(), request)
	id = resp.ID
	info2, err := peerManager.Get(context.Background(), id)

	// get all
	infoList, err := peerManager.List(context.Background(), nil)
	c.Check(err, check.IsNil)
	interfaceSlice := make([]interface{}, len(infoList))
	for k, v := range infoList {
		interfaceSlice[k] = v
	}
	c.Check(getPageValues(interfaceSlice, 0, 0, func(i, j int) bool {
		peeri := interfaceSlice[i].(*types.PeerInfo)
		peerj := interfaceSlice[j].(*types.PeerInfo)
		return time.Time(peeri.Created).Before(time.Time(peerj.Created))
	}), check.DeepEquals, []interface{}{info, info2})

	// get with pageNum=0 && pageSize=1 && sortDirect=asc
	infoList, err = peerManager.List(context.Background(), &PageFilter{
		pageNum:    0,
		pageSize:   1,
		sortDirect: "asc",
	})
	c.Check(err, check.IsNil)
	c.Check(infoList, check.DeepEquals, []*types.PeerInfo{info})

	// get with pageNum=0 && pageSize=1 && sortDirect=desc
	infoList, err = peerManager.List(context.Background(), &PageFilter{
		pageNum:    0,
		pageSize:   1,
		sortDirect: "desc",
	})
	c.Check(err, check.IsNil)
	c.Check(infoList, check.DeepEquals, []*types.PeerInfo{info2})

	// get with pageNum=1 && pageSize=1 && sortDirect=asc
	infoList, err = peerManager.List(context.Background(), &PageFilter{
		pageNum:    1,
		pageSize:   1,
		sortDirect: "asc",
	})
	c.Check(err, check.IsNil)
	c.Check(infoList, check.DeepEquals, []*types.PeerInfo{info2})
}
