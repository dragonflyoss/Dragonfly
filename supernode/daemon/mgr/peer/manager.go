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
	"fmt"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/metricsutils"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	dutil "github.com/dragonflyoss/Dragonfly/supernode/daemon/util"
	"github.com/dragonflyoss/Dragonfly/supernode/util"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var _ mgr.PeerMgr = &Manager{}

type metrics struct {
	peers *prometheus.GaugeVec
}

func newMetrics(register prometheus.Registerer) *metrics {
	return &metrics{
		peers: metricsutils.NewGauge(config.SubsystemSupernode, "peers",
			"Current status of peers", []string{"peer"}, register),
	}
}

// Manager is an implement of the interface of PeerMgr.
type Manager struct {
	peerStore *dutil.Store
	metrics   *metrics
}

// NewManager returns a new Manager Object.
func NewManager(register prometheus.Registerer) (*Manager, error) {
	return &Manager{
		peerStore: dutil.NewStore(),
		metrics:   newMetrics(register),
	}, nil
}

// Register a peer and generate a unique ID as returned.
func (pm *Manager) Register(ctx context.Context, peerCreateRequest *types.PeerCreateRequest) (peerCreateResponse *types.PeerCreateResponse, err error) {
	if peerCreateRequest == nil {
		return nil, errors.Wrap(errortypes.ErrEmptyValue, "peer create request")
	}

	ipString := peerCreateRequest.IP.String()
	if !netutils.IsValidIP(ipString) {
		return nil, errors.Wrapf(errortypes.ErrInvalidValue, "peer IP: %s", ipString)
	}

	id := generatePeerID(peerCreateRequest)
	peerInfo := &types.PeerInfo{
		ID:       id,
		IP:       peerCreateRequest.IP,
		HostName: peerCreateRequest.HostName,
		Port:     peerCreateRequest.Port,
		Version:  peerCreateRequest.Version,
		Created:  strfmt.DateTime(time.Now()),
	}
	pm.peerStore.Put(id, peerInfo)
	pm.metrics.peers.WithLabelValues(peerInfo.IP.String()).Inc()

	return &types.PeerCreateResponse{
		ID: id,
	}, nil
}

// DeRegister is a peer from p2p network.
func (pm *Manager) DeRegister(ctx context.Context, peerID string) error {
	peerInfo, err := pm.getPeerInfo(peerID)
	if err != nil {
		return err
	}

	pm.peerStore.Delete(peerID)
	// NOTE: DeRegister will be called asynchronously.
	pm.metrics.peers.WithLabelValues(peerInfo.IP.String()).Dec()
	return nil
}

// Get returns the peerInfo of the specified peerID.
func (pm *Manager) Get(ctx context.Context, peerID string) (*types.PeerInfo, error) {
	util.GetLock(peerID, true)
	defer util.ReleaseLock(peerID, true)

	return pm.getPeerInfo(peerID)
}

// GetAllPeerIDs returns all peerIDs.
func (pm *Manager) GetAllPeerIDs(ctx context.Context) (peerIDs []string) {
	return pm.peerStore.ListKeyAsStringSlice()
}

// List returns all filtered peerInfo by filter.
func (pm *Manager) List(ctx context.Context, filter *dutil.PageFilter) (
	peerList []*types.PeerInfo, err error) {

	// when filter is nil, return all values.
	if filter == nil {
		listResult := pm.peerStore.List()
		peerList, err = pm.assertPeerInfoSlice(listResult)
		if err != nil {
			return nil, err
		}
		return
	}

	// validate the filter
	err = dutil.ValidateFilter(filter, nil)
	if err != nil {
		return nil, err
	}

	listResult := pm.peerStore.List()

	// For PeerInfo, there is no need to sort by field;
	// and the order of insertion is used by default.
	less := getLessFunc(listResult, dutil.IsDESC(filter.SortDirect))
	peerList, err = pm.assertPeerInfoSlice(dutil.GetPageValues(listResult, filter.PageNum, filter.PageSize, less))

	return
}

// getPeerInfo gets peer info with specified peerID and
// returns the underlying PeerInfo value.
func (pm *Manager) getPeerInfo(peerID string) (*types.PeerInfo, error) {
	// return error if peerID is empty
	if stringutils.IsEmptyStr(peerID) {
		return nil, errors.Wrap(errortypes.ErrEmptyValue, "peerID")
	}

	// get value form store
	v, err := pm.peerStore.Get(peerID)
	if err != nil {
		return nil, err
	}

	// type assertion
	if info, ok := v.(*types.PeerInfo); ok {
		return info, nil
	}
	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "peerID %s: %v", peerID, v)
}

func (pm *Manager) assertPeerInfoSlice(s []interface{}) ([]*types.PeerInfo, error) {
	peerInfos := make([]*types.PeerInfo, 0)
	for _, v := range s {
		// type assertion
		info, ok := v.(*types.PeerInfo)
		if !ok {
			return nil, errors.Wrapf(errortypes.ErrConvertFailed, "value  %v", v)
		}
		peerInfos = append(peerInfos, info)
	}
	return peerInfos, nil
}

func getLessFunc(listResult []interface{}, desc bool) (less func(i, j int) bool) {
	lessTemp := func(i, j int) bool {
		peeri, ok := listResult[i].(*types.PeerInfo)
		if !ok {
			return false
		}
		peerj := listResult[j].(*types.PeerInfo)
		if !ok {
			return false
		}
		return time.Time(peeri.Created).Before(time.Time(peerj.Created))
	}
	if desc {
		less = func(i, j int) bool {
			return lessTemp(j, i)
		}
	}

	if less == nil {
		less = lessTemp
	}
	return
}

// generatePeerID generates an ID with hostname and ip.
// Use timestamp to ensure the uniqueness.
func generatePeerID(peerInfo *types.PeerCreateRequest) string {
	return fmt.Sprintf("%s-%s-%d", peerInfo.HostName.String(), peerInfo.IP.String(), time.Now().UnixNano())
}
