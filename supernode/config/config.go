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
	"path/filepath"
	"strings"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/dflog"
	"github.com/dragonflyoss/Dragonfly/pkg/fileutils"
	"github.com/dragonflyoss/Dragonfly/pkg/rate"

	"gopkg.in/yaml.v2"
)

// NewConfig creates an instant with default values.
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
	c.cIDPrefix = fmt.Sprintf("%s%s~", SuperNodeCIdPrefix, ip)
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

// NewBaseProperties creates an instant with default values.
func NewBaseProperties() *BaseProperties {
	home := filepath.Join(string(filepath.Separator), "home", "admin", "supernode")
	return &BaseProperties{
		ListenPort:              DefaultListenPort,
		DownloadPort:            DefaultDownloadPort,
		HomeDir:                 home,
		SchedulerCorePoolSize:   DefaultSchedulerCorePoolSize,
		DownloadPath:            filepath.Join(home, "repo", "download"),
		PeerUpLimit:             DefaultPeerUpLimit,
		PeerDownLimit:           DefaultPeerDownLimit,
		EliminationLimit:        DefaultEliminationLimit,
		FailureCountLimit:       DefaultFailureCountLimit,
		LinkLimit:               DefaultLinkLimit,
		SystemReservedBandwidth: DefaultSystemReservedBandwidth,
		MaxBandwidth:            DefaultMaxBandwidth,
		EnableProfiler:          false,
		Debug:                   false,
		FailAccessInterval:      DefaultFailAccessInterval,
		GCInitialDelay:          DefaultGCInitialDelay,
		GCMetaInterval:          DefaultGCMetaInterval,
		GCDiskInterval:          DefaultGCDiskInterval,
		YoungGCThreshold:        DefaultYoungGCThreshold,
		FullGCThreshold:         DefaultFullGCThreshold,
		IntervalThreshold:       DefaultIntervalThreshold,
		TaskExpireTime:          DefaultTaskExpireTime,
		PeerGCDelay:             DefaultPeerGCDelay,
		CleanRatio:              DefaultCleanRatio,
	}
}

type CDNPattern string

const (
	CDNPatternLocal  = "local"
	CDNPatternSource = "source"
)

// BaseProperties contains all basic properties of supernode.
type BaseProperties struct {
	// CDNPattern cdn pattern which must be in ["local", "source"].
	// default: CDNPatternLocal
	CDNPattern CDNPattern `yaml:"cdnPattern"`

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
	DownloadPath string

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

	// LinkLimit is set for supernode to limit every piece download network speed.
	// default: 20 MB, in format of G(B)/g/M(B)/m/K(B)/k/B, pure number will also be parsed as Byte.
	LinkLimit rate.Rate `yaml:"linkLimit"`

	// SystemReservedBandwidth is the network bandwidth reserved for system software.
	// default: 20 MB, in format of G(B)/g/M(B)/m/K(B)/k/B, pure number will also be parsed as Byte.
	SystemReservedBandwidth rate.Rate `yaml:"systemReservedBandwidth"`

	// MaxBandwidth is the network bandwidth that supernode can use.
	// default: 200 MB, in format of G(B)/g/M(B)/m/K(B)/k/B, pure number will also be parsed as Byte.
	MaxBandwidth rate.Rate `yaml:"maxBandwidth"`

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

	// cIDPrefix is a prefix string used to indicate that the CID is supernode.
	cIDPrefix string

	// superNodePID is the ID of supernode, which is the same as peer ID of dfget.
	superNodePID string

	// gc related

	// GCInitialDelay is the delay time from the start to the first GC execution.
	// default: 6s
	GCInitialDelay time.Duration `yaml:"gcInitialDelay"`

	// GCMetaInterval is the interval time to execute GC meta.
	// default: 2min
	GCMetaInterval time.Duration `yaml:"gcMetaInterval"`

	// TaskExpireTime when a task is not accessed within the taskExpireTime,
	// and it will be treated to be expired.
	// default: 3min
	TaskExpireTime time.Duration `yaml:"taskExpireTime"`

	// PeerGCDelay is the delay time to execute the GC after the peer has reported the offline.
	// default: 3min
	PeerGCDelay time.Duration `yaml:"peerGCDelay"`

	// GCDiskInterval is the interval time to execute GC disk.
	// default: 15s
	GCDiskInterval time.Duration `yaml:"gcDiskInterval"`

	// YoungGCThreshold if the available disk space is more than YoungGCThreshold
	// and there is no need to GC disk.
	//
	// default: 100GB
	YoungGCThreshold fileutils.Fsize `yaml:"youngGCThreshold"`

	// FullGCThreshold if the available disk space is less than FullGCThreshold
	// and the supernode should gc all task files which are not being used.
	//
	// default: 5GB
	FullGCThreshold fileutils.Fsize `yaml:"fullGCThreshold"`

	// IntervalThreshold is the threshold of the interval at which the task file is accessed.
	// default: 2h
	IntervalThreshold time.Duration `yaml:"IntervalThreshold"`

	// CleanRatio is the ratio to clean the disk and it is based on 10.
	// It means the value of CleanRatio should be [1-10].
	//
	// default: 1
	CleanRatio int

	LogConfig dflog.LogConfig `yaml:"logConfig" json:"logConfig"`
}
