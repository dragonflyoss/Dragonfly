package daemon

import (
	"context"
	"fmt"
	"github.com/dragonflyoss/Dragonfly/common/constants"
	"github.com/sirupsen/logrus"
	"os"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/plugins"
	"github.com/dragonflyoss/Dragonfly/supernode/server"

	"github.com/go-openapi/strfmt"
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
	//TODO(yunfeiyangbuaa): handle these log,errors and exceptions in the future
	//TODO(yunfeiyangbuaa):the following code is just a test
	if d.server.Config.UseHA == false {
		if changOk := d.server.HaMgr.CompareAndSetSupernodeStatus(constants.SupernodeUseHaInit, constants.SupernodeUseHaActive); changOk == false {
			fmt.Println(d.server.Config.AdvertiseIP, "change from initial to active failed")
		}

		fmt.Println(d.server.Config.AdvertiseIP, "don't use ha,now the supernode status change from initial to active")

		if err := d.server.Start(d.config.ListenPort); err != nil {
			logrus.Errorf("failed to start HTTP server: %v", err)
			return err
		}

	} else {
		if changOk := d.server.HaMgr.CompareAndSetSupernodeStatus(constants.SupernodeUseHaInit, constants.SupernodeUseHaStandby); changOk == false {
			fmt.Println(d.server.Config.AdvertiseIP, "change from initial to active failed")
		}
		fmt.Println(d.server.Config.AdvertiseIP, "use ha,change from initial to standby,and try to obtain active status!")
		change := make(chan int, 10)
		go d.server.HaMgr.ElectDaemon(change)
		for {
			if ch, ok := <-change; ok {
				fmt.Println("change:", ch)
				if ch == constants.SupernodeUseHaActive {
					fmt.Println("server port:", d.server.ServerPort, "config port:", d.config.ListenPort)
					if d.server.ServerPort != d.config.ListenPort && d.server.ServerPort != server.ServerClose {
						d.server.Close()
					}
					go func() {
						if err := d.server.Start(d.config.ListenPort); err != nil {
							logrus.Errorf("failed to start HTTP server: %v", err)
						}
					}()
				} else if ch == constants.SupernodeUsehakill {
					d.server.Close()
					fmt.Println("game over")

				} else if ch == constants.SupernodeUseHaStandby {
					fmt.Println("server port:", d.server.ServerPort)
					if d.server.ServerPort == d.config.ListenPort && d.server.ServerPort != server.ServerClose {
						d.server.Close()
					}
					go func() {
						if err := d.server.Start(8003); err != nil {
							logrus.Errorf("failed to start HTTP server: %v", err)
						}
					}()
				}
			}
		}

		//defer d.server.HaMgr.CloseHaManager()
	}
	defer d.server.Close()
	return nil
}
