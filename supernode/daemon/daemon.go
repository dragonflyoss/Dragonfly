package daemon

import (
	"context"
	"os"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/plugins"
	"github.com/dragonflyoss/Dragonfly/supernode/server"

	"github.com/go-openapi/strfmt"
	"github.com/sirupsen/logrus"
)

// Daemon is a struct to identify main instance of supernode.
type Daemon struct {
	Name string

	config *config.Config

	// members of the Supernode cluster
	ClusterMember []string

	server *server.Server
}

// New creates a new Daemon.
func New(cfg *config.Config) (*Daemon, error) {
	if err := plugins.Initialize(cfg); err != nil {
		return nil, err
	}

	s, err := server.New(cfg)
	if err != nil {
		return nil, err
	}

	return &Daemon{
		config: cfg,
		server: s,
	}, nil
}

// RegisterSuperNode register the supernode as a peer.
func (d *Daemon) RegisterSuperNode() error {
	// construct the PeerCreateRequest for supernode.
	// TODO: add supernode version
	hostname, _ := os.Hostname()
	req := &types.PeerCreateRequest{
		IP:       strfmt.IPv4(d.config.AdvertiseIP),
		HostName: strfmt.Hostname(hostname),
		Port:     int32(d.config.DownloadPort),
	}

	resp, err := d.server.PeerMgr.Register(context.Background(), req)
	if err != nil {
		return err
	}

	d.config.SetSuperPID(resp.ID)
	return nil
}

// Run runs the daemon.
func (d *Daemon) Run() error {
	if err := d.server.Start(); err != nil {
		logrus.Errorf("failed to start HTTP server: %v", err)
		return err
	}
	return nil
}
