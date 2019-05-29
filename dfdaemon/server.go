package dfdaemon

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/dragonflyoss/Dragonfly/cmd/dfdaemon/app/options"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/handler"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/proxy"
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

// Server represents the dfdaemon server
type Server struct {
	server *http.Server
	proxy  *proxy.Proxy
}

// Option is the functional option for creating a server
type Option func(s *Server) error

// WithTLSFromFile sets the tls config for the server from the given key pair file
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

// WithAddr sets the address the server listens on
func WithAddr(addr string) Option {
	return func(s *Server) error {
		s.server.Addr = addr
		return nil
	}
}

// WithProxy sets the proxy
func WithProxy(p *proxy.Proxy) Option {
	return func(s *Server) error {
		if p == nil {
			return errors.Errorf("nil proxy")
		}
		s.proxy = p
		return nil
	}
}

// New returns a new server instance
func New(opts ...Option) (*Server, error) {
	p, _ := proxy.New()
	s := &Server{
		server: &http.Server{
			Addr: ":65001",
		},
		proxy: p,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return s, err
		}
	}

	return s, nil
}

// NewFromConfig returns a new server instance from given configuration
func NewFromConfig(cfg config.Properties, o options.Options) (*Server, error) {
	p, err := proxy.NewFromConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "create proxy")
	}

	opts := []Option{
		WithProxy(p),
		WithAddr(fmt.Sprintf(":%d", o.Port)),
	}

	if o.CertFile != "" && o.KeyFile != "" {
		opts = append(opts, WithTLSFromFile(o.CertFile, o.KeyFile))
	}

	return New(opts...)
}

// Start runs dfdaemon's http server
func (s *Server) Start() error {
	proxy.WithDirectHandler(handler.New())(s.proxy)
	s.server.Handler = s.proxy
	if s.server.TLSConfig != nil {
		logrus.Infof("start dfdaemon https server on %s", s.server.Addr)
	} else {
		logrus.Infof("start dfdaemon http server on %s", s.server.Addr)
	}
	return s.server.ListenAndServe()
}

// Stop gracefully stops the dfdaemon http server
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
