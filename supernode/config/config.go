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

package config

import (
	"fmt"

	"net/rpc"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// NewConfig create an instant with default values.
func NewConfig() *Config {
	return &Config{
		BaseProperties: NewBaseProperties(),
	}
}

// Config contains all configuration of supernode.
type Config struct {
	*BaseProperties `yaml:"base"`
	Plugins         map[PluginType][]*PluginProperties `yaml:"plugins"`
	Storages        map[string]interface{}             `yaml:"storages"`
}

// SupernodeInfo store the supernode's info get from etcd.
type SupernodeInfo struct {
	IP           string
	ListenPort   int
	DownloadPort int
	RPCPort      int
	HostName     strfmt.Hostname
	PID          string
	RPCClient    *rpc.Client
}

// Load loads config properties from the giving file.
func (c *Config) Load(path string) error {
	return fileutils.LoadYaml(path, c)
}

func (c *Config) String() string {
	if out, err := yaml.Marshal(c); err == nil {
		return string(out)
	}
	return ""
}

// SetCIDPrefix sets a string as the prefix for supernode CID
// which used to distinguish from the other peer nodes.
func (c *Config) SetCIDPrefix(ip string) {
	c.cIDPrefix = fmt.Sprintf("cdnnode:%s~", ip)
}

// GetSuperCID returns the cid string for taskID.
func (c *Config) GetSuperCID(taskID string) string {
	return fmt.Sprintf("%s%s", c.cIDPrefix, taskID)
}

// IsSuperCID returns whether the clientID represents supernode.
func (c *Config) IsSuperCID(clientID string) bool {
	return strings.HasPrefix(clientID, c.cIDPrefix)
}

// SetSuperPID sets the value of supernode PID.
func (c *Config) SetSuperPID(pid string) {
	c.superNodePID = pid
}

// GetSuperPID returns the pid string for supernode.
func (c *Config) GetSuperPID() string {
	return c.superNodePID
}

// IsSuperPID returns whether the peerID represents supernode.
func (c *Config) IsSuperPID(peerID string) bool {
	return peerID == c.superNodePID
}

// GetOtherSupernodeInfo gets the other supernodes information in supernode ha cluster
func (c *Config) GetOtherSupernodeInfo() []SupernodeInfo {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return *c.OtherSupernodes
}

// SetOtherSupernodeInfo sets the other supernodes information in supernode ha cluster
func (c *Config) SetOtherSupernodeInfo(otherSupernodes []SupernodeInfo) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.OtherSupernodes = &otherSupernodes
}

// GetOtherSupernodeInfoByPID get other supernode's info by peerID
func (c *Config) GetOtherSupernodeInfoByPID(peerID string) (*SupernodeInfo, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, node := range *c.OtherSupernodes {
		if node.PID == peerID {
			return &node, nil
			break
		}
	}
	return nil, errors.Wrapf(errortypes.ErrDataNotFound, "peerID: %s", peerID)
}

// NewBaseProperties create an instant with default values.
func NewBaseProperties() *BaseProperties {
	home := filepath.Join(string(filepath.Separator), "home", "admin", "supernode")
	return &BaseProperties{
		ListenPort:              8002,
		DownloadPort:            8001,
		HomeDir:                 home,
		SchedulerCorePoolSize:   10,
		DownloadPath:            filepath.Join(home, "repo", "download"),
		PeerUpLimit:             5,
		PeerDownLimit:           5,
		EliminationLimit:        5,
		FailureCountLimit:       5,
		LinkLimit:               20,
		SystemReservedBandwidth: 20,
		MaxBandwidth:            200,
		EnableProfiler:          false,
		Debug:                   false,
		FailAccessInterval:      DefaultFailAccessInterval,
		GCInitialDelay:          DefaultGCInitialDelay,
		GCMetaInterval:          DefaultGCMetaInterval,
		TaskExpireTime:          DefaultTaskExpireTime,
		PeerGCDelay:             DefaultPeerGCDelay,
		UseHA:                   false,
		HAConfig:                []string{"127.0.0.1:2379"},
		HARpcPort:               9000,
	}
}

// BaseProperties contains all basic properties of supernode.
type BaseProperties struct {
	// ListenPort is the port supernode server listens on.
	// default: 8002
	ListenPort int `yaml:"listenPort"`

	// DownloadPort is the port for download files from supernode.
	// default: 8001
	DownloadPort int `yaml:"downloadPort"`

	// HomeDir is working directory of supernode.
	// default: /home/admin/supernode
	HomeDir string `yaml:"homeDir"`

	// the core pool size of ScheduledExecutorService.
	// When a request to start a download task, supernode will construct a thread concurrent pool
	// to download pieces of source file and write to specified storage.
	// Note: source file downloading is into pieces via range attribute set in HTTP header.
	// default: 10
	SchedulerCorePoolSize int `yaml:"schedulerCorePoolSize"`

	// DownloadPath specifies the path where to store downloaded files from source address.
	// This path can be set beyond BaseDir, such as taking advantage of a different disk from BaseDir's.
	// default: $BaseDir/downloads
	DownloadPath string `yaml:"downloadPath"`

	// PeerUpLimit is the upload limit of a peer. When dfget starts to play a role of peer,
	// it can only stand PeerUpLimit upload tasks from other peers.
	// default: 5
	PeerUpLimit int `yaml:"peerUpLimit"`

	// PeerDownLimit is the download limit of a peer. When a peer starts to download a file/image,
	// it will download file/image in the form of pieces. PeerDownLimit mean that a peer can only
	// stand starting PeerDownLimit concurrent downloading tasks.
	// default: 4
	PeerDownLimit int `yaml:"peerDownLimit"`

	// When dfget node starts to play a role of peer, it will provide services for other peers
	// to pull pieces. If it runs into an issue when providing services for a peer, its self failure
	// increases by 1. When the failure limit reaches EliminationLimit, the peer will isolate itself
	// as a unhealthy state. Then this dfget will be no longer called by other peers.
	// default: 5
	EliminationLimit int `yaml:"eliminationLimit"`

	// FailureCountLimit is the failure count limit set in supernode for dfget client.
	// When a dfget client takes part in the peer network constructed by supernode,
	// supernode will command the peer to start distribution task.
	// When dfget client fails to finish distribution task, the failure count of client
	// increases by 1. When failure count of client reaches to FailureCountLimit(default 5),
	// dfget client will be moved to blacklist of supernode to stop playing as a peer.
	// default: 5
	FailureCountLimit int `yaml:"failureCountLimit"`

	// LinkLimit is set for supernode to limit every piece download network speed (unit: MB/s).
	// default: 20
	LinkLimit int `yaml:"linkLimit"`

	// SystemReservedBandwidth is the network bandwidth reserved for system software.
	// unit: MB/s
	// default: 20
	SystemReservedBandwidth int `yaml:"systemReservedBandwidth"`

	// MaxBandwidth is the network bandwidth that supernode can use.
	// unit: MB/s
	// default: 200
	MaxBandwidth int `yaml:"maxBandwidth"`

	// Whether to enable profiler
	// default: false
	EnableProfiler bool `yaml:"enableProfiler"`

	// Whether to open DEBUG level
	// default: false
	Debug bool `yaml:"debug"`

	// AdvertiseIP is used to set the ip that we advertise to other peer in the p2p-network.
	// By default, the first non-loop address is advertised.
	AdvertiseIP string `yaml:"advertiseIP"`

	// FailAccessInterval is the interval time after failed to access the URL.
	// unit: minutes
	// default: 3
	FailAccessInterval time.Duration `yaml:"failAccessInterval"`

	// GCInitialDelay is the delay time from the start to the first GC execution.
	GCInitialDelay time.Duration `yaml:"gcInitialDelay"`

	// GCMetaInterval is the interval time to execute the GC meta.
	GCMetaInterval time.Duration `yaml:"gcMetaInterval"`

	// TaskExpireTime when a task is not accessed within the taskExpireTime,
	// and it will be treated to be expired.
	TaskExpireTime time.Duration `yaml:"taskExpireTime"`

	// PeerGCDelay is the delay time to execute the GC after the peer has reported the offline.
	PeerGCDelay time.Duration `yaml:"peerGCDelay"`

	// cIDPrefix s a prefix string used to indicate that the CID is supernode.
	cIDPrefix string

	// superNodePID is the ID of supernode, which is the same as peer ID of dfget.
	superNodePID string

	// UseHA is the mark of whether the supernode use the ha model.
	// ha means if the active supernode is off,the standby supernode can take over active supernode's work.
	// and the whole system can work as before.
	// default:false.
	UseHA bool `yaml:"useHa"`

	// HAConfig is available when UseHa is true.
	// HAConfig configs the tool's ip and port we use to implement ha.
	// default:[] string {127.0.0.1:2379}.
	HAConfig []string `yaml:"haConfig"`

	//HAStandbyPort is available when UseHa is true.
	//HAStandbyPort configs the port the supernode use when its status is standby
	//HAStandbyPort is the port to receive the active supernode's request copy to implement state synchronization
	HARpcPort int `yaml:"haStandbyPort"`

	//OtherSupernodes records other supernode info in the p2p System.
	OtherSupernodes *[]SupernodeInfo `yaml:"otherSupernode"`

	//lock is thr read-write lock to use change supernodeInfo,
	//if OtherSupernode changes when use,supernode may panic
	lock sync.RWMutex `yaml:"lock"`
}

// TransLimit trans rateLimit from MB/s to B/s.
func TransLimit(rateLimit int) int {
	return rateLimit * 1024 * 1024
}
