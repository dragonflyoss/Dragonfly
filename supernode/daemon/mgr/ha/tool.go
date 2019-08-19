package ha

import (
	"context"

	"github.com/go-openapi/strfmt"
)

// Tool is an interface that use etcd/zookeeper/yourImplement tools to manager supernode cluster.
type Tool interface {
	// WatchStandbySupernodesChange watches other supernodes,supernode will be notified if other superodes changes.
	WatchSupernodesChange(ctx context.Context, key string) error

	// Close closes the tool.
	Close(ctx context.Context) error

	// SendSupernodesInfo send supernode info to other supernode.
	SendSupernodesInfo(ctx context.Context, key, ip, pID string, listenPort, downloadPort, rpcPort int, hostName strfmt.Hostname, timeout int64) error
}
