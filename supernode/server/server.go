package server

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/cdn"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/dfgettask"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/peer"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/progress"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/scheduler"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr/task"
	"github.com/dragonflyoss/Dragonfly/supernode/httpclient"
	"github.com/dragonflyoss/Dragonfly/supernode/store"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/sirupsen/logrus"
)

// Server is supernode server struct.
type Server struct {
	Config       *config.Config
	PeerMgr      mgr.PeerMgr
	TaskMgr      mgr.TaskMgr
	DfgetTaskMgr mgr.DfgetTaskMgr
	ProgressMgr  mgr.ProgressMgr
	OriginClient httpclient.OriginHTTPClient
}

// New creates a brand new server instance.
func New(cfg *config.Config) (*Server, error) {
	sm, err := store.NewManager(cfg)
	if err != nil {
		return nil, err
	}
	storeLocal, err := sm.Get(store.LocalStorageDriver)
	if err != nil {
		return nil, err
	}

	originClient := httpclient.NewOriginClient()

	peerMgr, err := peer.NewManager()
	if err != nil {
		return nil, err
	}

	dfgetTaskMgr, err := dfgettask.NewManager()
	if err != nil {
		return nil, err
	}

	progressMgr, err := progress.NewManager(cfg)
	if err != nil {
		return nil, err
	}

	schedulerMgr, err := scheduler.NewManager(cfg, progressMgr)
	if err != nil {
		return nil, err
	}

	cdnMgr, err := cdn.NewManager(cfg, storeLocal, progressMgr, originClient)
	if err != nil {
		return nil, err
	}

	taskMgr, err := task.NewManager(cfg, peerMgr, dfgetTaskMgr, progressMgr, cdnMgr, schedulerMgr, originClient)
	if err != nil {
		return nil, err
	}

	return &Server{
		Config:       cfg,
		PeerMgr:      peerMgr,
		TaskMgr:      taskMgr,
		DfgetTaskMgr: dfgetTaskMgr,
		ProgressMgr:  progressMgr,
		OriginClient: originClient,
	}, nil
}

// Start runs supernode server.
func (s *Server) Start() error {
	router := initRoute(s)

	address := fmt.Sprintf("0.0.0.0:%d", s.Config.ListenPort)

	l, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Errorf("failed to listen port %d: %v", s.Config.ListenPort, err)
		return err
	}

	// register supernode build information
	version.NewBuildInfo("supernode")
	server := &http.Server{
		Handler:           router,
		ReadTimeout:       time.Minute * 10,
		ReadHeaderTimeout: time.Minute * 10,
		IdleTimeout:       time.Minute * 10,
	}
	return server.Serve(l)
}
