package server

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	peer "github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/peer"

	"github.com/sirupsen/logrus"
)

// Server is server instance.
type Server struct {
	Config  *config.Config
	PeerMgr mgr.PeerMgr
}

// New creates a brand new server instance.
func New(cfg *config.Config) (*Server, error) {
	peerMgr, err := peer.NewManager()
	if err != nil {
		return nil, err
	}
	return &Server{
		Config:  cfg,
		PeerMgr: peerMgr,
	}, nil
}

// Start runs
func (s *Server) Start() error {
	router := initRoute(s)

	address := fmt.Sprintf("0.0.0.0:%d", s.Config.ListenPort)

	l, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Errorf("failed to listen port %d: %v", s.Config.ListenPort, err)
		return err
	}

	server := &http.Server{
		Handler:           router,
		ReadTimeout:       time.Minute * 10,
		ReadHeaderTimeout: time.Minute * 10,
		IdleTimeout:       time.Minute * 10,
	}
	return server.Serve(l)
}
