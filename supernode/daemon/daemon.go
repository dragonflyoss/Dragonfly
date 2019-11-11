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

package daemon

import (
	"context"
	"os"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/plugins"
	"github.com/dragonflyoss/Dragonfly/supernode/server"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/client_golang/prometheus"
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
func New(cfg *config.Config, dfgetLogger *logrus.Logger) (*Daemon, error) {
	if err := plugins.Initialize(cfg); err != nil {
		return nil, err
	}

	s, err := server.New(cfg, dfgetLogger, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}

	return &Daemon{
		config: cfg,
		server: s,
	}, nil
}

// RegisterSuperNode registers the supernode as a peer.
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
