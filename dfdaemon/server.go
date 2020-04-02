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

package dfdaemon

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/handler"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/proxy"
	dfgetConfig "github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/uploader"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Server represents the dfdaemon server.
type Server struct {
	server *http.Server
	proxy  *proxy.Proxy
}

// Option is the functional option for creating a server.
type Option func(s *Server) error

// WithTLSFromFile sets the TLS config for the server from the given key pair file.
func WithTLSFromFile(certFile, keyFile string) Option {
	return func(s *Server) error {
		if s.server.TLSConfig == nil {
			s.server.TLSConfig = &tls.Config{}
		}
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return errors.Wrap(err, "load key pair")
		}
		s.server.TLSConfig.Certificates = []tls.Certificate{cert}
		return nil
	}
}

// WithAddr sets the address the server listens on.
func WithAddr(addr string) Option {
	return func(s *Server) error {
		s.server.Addr = addr
		return nil
	}
}

// WithProxy sets the proxy.
func WithProxy(p *proxy.Proxy) Option {
	return func(s *Server) error {
		if p == nil {
			return errors.Errorf("nil proxy")
		}
		s.proxy = p
		return nil
	}
}

// New returns a new server instance.
func New(opts ...Option) (*Server, error) {
	p, _ := proxy.New()
	s := &Server{
		server: &http.Server{
			Addr: ":65001",
		},
		proxy: p,
	}
	// register dfdaemon build information
	version.NewBuildInfo("dfdaemon", prometheus.DefaultRegisterer)

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return s, err
		}
	}

	return s, nil
}

// NewFromConfig returns a new server instance from given configuration.
func NewFromConfig(cfg config.Properties) (*Server, error) {
	p, err := proxy.NewFromConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "create proxy")
	}

	opts := []Option{
		WithProxy(p),
		WithAddr(fmt.Sprintf(":%d", cfg.Port)),
	}

	if cfg.CertPem != "" && cfg.KeyPem != "" {
		opts = append(opts, WithTLSFromFile(cfg.CertPem, cfg.KeyPem))
	}

	return New(opts...)
}

func LaunchPeerServer(cfg config.Properties) error {
	peerServerConfig := dfgetConfig.NewConfig()
	peerServerConfig.RV.LocalIP = cfg.LocalIP
	peerServerConfig.RV.PeerPort = cfg.PeerPort
	peerServerConfig.RV.ServerAliveTime = 0
	port, err := uploader.LaunchPeerServer(peerServerConfig)
	if err != nil {
		return err
	}
	peerServerConfig.RV.PeerPort = port
	return nil
}

// Start runs dfdaemon's http server.
func (s *Server) Start() error {
	var err error
	_ = proxy.WithDirectHandler(handler.New())(s.proxy)
	s.server.Handler = s.proxy
	if s.server.TLSConfig != nil {
		logrus.Infof("start dfdaemon https server on %s", s.server.Addr)
		err = s.server.ListenAndServeTLS("", "")
	} else {
		logrus.Infof("start dfdaemon http server on %s", s.server.Addr)
		err = s.server.ListenAndServe()
	}
	return err
}

// Stop gracefully stops the dfdaemon http server.
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
