package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/util"
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
	List(ctx context.Context, filter *util.PageFilter) (peerList []*types.PeerInfo, err error)
}
