package mgr

import (
	"context"
	"fmt"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/common/util"
	errorType "github.com/dragonflyoss/Dragonfly/supernode/errors"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
)

// PeerMgr as an interface defines all operations against Peer.
// A Peer represents a web server that provides file downloads for others.
type PeerMgr interface {
	// Register a peer with specified peerInfo.
	// Supernode will generate a unique peerID for every Peer with PeerInfo provided.
	Register(ctx context.Context, peerCreateRequest *types.PeerCreateRequest) (peerCreateResponse *types.PeerCreateResponse, err error)

	// DeRegister offline a peer service and
	// NOTE: update the info related for scheduler.
	DeRegister(ctx context.Context, peerID string) error

	// Get the peer Info with specified peerID.
	Get(ctx context.Context, peerID string) (*types.PeerInfo, error)

	// List return a list of peers info with filter.
	List(ctx context.Context, filter *PageFilter) (peerList []*types.PeerInfo, err error)
}

// PeerManager is an implement of the interface of PeerMgr.
type PeerManager struct {
	peerStore *Store
}

// NewPeerManager return a new PeerManager Object.
func NewPeerManager() (*PeerManager, error) {
	return &PeerManager{
		peerStore: NewStore(),
	}, nil
}

// Register a peer and genreate a unique ID as returned.
func (pm *PeerManager) Register(ctx context.Context, peerCreateRequest *types.PeerCreateRequest) (peerCreateResponse *types.PeerCreateResponse, err error) {
	if peerCreateRequest == nil {
		return nil, errors.Wrap(errorType.ErrEmptyValue, "peer create request")
	}

	ipString := peerCreateRequest.IP.String()
	if !util.IsValidIP(ipString) {
		return nil, errors.Wrapf(errorType.ErrInvalidValue, "peer IP: %s", ipString)
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

	return &types.PeerCreateResponse{
		ID: id,
	}, nil
}

// DeRegister a peer from p2p network.
func (pm *PeerManager) DeRegister(ctx context.Context, peerID string) error {
	if _, err := pm.getPeerInfo(peerID); err != nil {
		return err
	}

	pm.peerStore.Delete(peerID)
	return nil
}

// Get returns the peerInfo of the specified peerID.
func (pm *PeerManager) Get(ctx context.Context, peerID string) (*types.PeerInfo, error) {
	return pm.getPeerInfo(peerID)
}

// List returns all filtered peerInfo by filter.
func (pm *PeerManager) List(ctx context.Context, filter *PageFilter) (
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
	err = validateFilter(filter, nil)
	if err != nil {
		return nil, err
	}

	listResult := pm.peerStore.List()

	// For PeerInfo, there is no need to sort by field;
	// and the order of insertion is used by default.
	less := getLessFunc(listResult, isDESC(filter.sortDirect))
	peerList, err = pm.assertPeerInfoSlice(getPageValues(listResult, filter.pageNum, filter.pageSize, less))

	return
}

// getPeerInfo gets peer info with specified peerID and
// returns the underlying PeerInfo value.
func (pm *PeerManager) getPeerInfo(peerID string) (*types.PeerInfo, error) {
	// return error if peerID is empty
	if util.IsEmptyStr(peerID) {
		return nil, errors.Wrap(errorType.ErrEmptyValue, "peerID")
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
	return nil, errors.Wrapf(errorType.ErrConvertFailed, "peerID %s: %v", peerID, v)
}

func (pm *PeerManager) assertPeerInfoSlice(s []interface{}) ([]*types.PeerInfo, error) {
	peerInfos := make([]*types.PeerInfo, 0)
	for _, v := range s {
		// type assertion
		info, ok := v.(*types.PeerInfo)
		if !ok {
			return nil, errors.Wrapf(errorType.ErrConvertFailed, "value  %v", v)
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
