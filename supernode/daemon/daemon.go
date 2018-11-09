package daemon

import (
	"github.com/alibaba/Dragonfly/supernode/config"
	"github.com/alibaba/Dragonfly/supernode/server"
	"github.com/sirupsen/logrus"
)

// Daemon is a struct to identify main instance of supernode.
type Daemon struct {
	// SupernodeID is the ID of supernode, which is the same as Client ID of dfget.
	SupernodeID string

	Name string

	config *config.Config

	// members of the Supernode cluster
	ClusterMember []string

	server *server.Server
}

// New creates a new Daemon.
func New(cfg *config.Config) (*Daemon, error) {
	s, err := server.New(cfg)
	if err != nil {
		return nil, err
	}

	return &Daemon{
		config: cfg,
		server: s,
	}, nil
}

// Run runs the daemon.
func (d *Daemon) Run() error {
	if err := d.server.Start(); err != nil {
		logrus.Errorf("failed to start HTTP server: %v", err)
		return err
	}
	return nil
}
