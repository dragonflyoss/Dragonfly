package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// PeerMgr as an interface defines all operations against Peer.
// A Peer represents a web server that provides file downloads for others.
type PeerMgr interface {
	// Register a peer with specified peerInfo.
	// Supernode will generate a unique peerID for every Peer with PeerInfo provided.
	Register(ctx context.Context, peerInfo *types.PeerInfo) (peerID string, err error)

	// DeRegister offline a peer service and
	// NOTE: update the info related for scheduler.
	DeRegister(ctx context.Context, peerID string) error

	// Get the peer Info with specified peerID.
	Get(ctx context.Context, peerID string) (*types.PeerInfo, error)

	// List return a list of peers info with filter.
	List(ctx context.Context, filter map[string]string) (peerList []*types.PeerInfo, err error)

	// Update the status of specified peer.
	//
	// Supernode will update the status of peer in the following situations:
	// 1) When an exception occurs to the peer server.
	// 2) When peer sends a request take the server offline.
	//
	// NOTE: update the info related for scheduler.
	Update(ctx context.Context, peerID string, peerInfo *types.PeerInfo) error
}
