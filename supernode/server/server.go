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
	"github.com/dragonflyoss/Dragonfly/supernode/store"

	"github.com/sirupsen/logrus"
)

// Server is server instance.
type Server struct {
	Config       *config.Config
	PeerMgr      mgr.PeerMgr
	TaskMgr      mgr.TaskMgr
	DfgetTaskMgr mgr.DfgetTaskMgr
	ProgressMgr  mgr.ProgressMgr
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

	cdnMgr, err := cdn.NewManager(cfg, storeLocal, progressMgr)
	if err != nil {
		return nil, err
	}

	taskMgr, err := task.NewManager(cfg, peerMgr, dfgetTaskMgr, progressMgr, cdnMgr, schedulerMgr)
	if err != nil {
		return nil, err
	}

	return &Server{
		Config:       cfg,
		PeerMgr:      peerMgr,
		TaskMgr:      taskMgr,
		DfgetTaskMgr: dfgetTaskMgr,
		ProgressMgr:  progressMgr,
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
