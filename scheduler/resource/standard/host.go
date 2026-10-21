/*
 *     Copyright 2020 The Dragonfly Authors
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

package standard

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	pkgatomic "d7y.io/dragonfly/v2/pkg/atomic"
	"d7y.io/dragonfly/v2/pkg/types"
	"d7y.io/dragonfly/v2/scheduler/config"
)

// HostOption is a functional option for configuring the host.
type HostOption func(h *Host)

// WithSchedulerClusterID sets host's SchedulerClusterID.
func WithSchedulerClusterID(id uint64) HostOption {
	return func(h *Host) {
		h.SchedulerClusterID = id
	}
}

// WithObjectStoragePort sets host's ObjectStoragePort.
func WithObjectStoragePort(port int32) HostOption {
	return func(h *Host) {
		h.ObjectStoragePort = port
	}
}

// WithConcurrentUploadLimit sets host's ConcurrentUploadLimit.
func WithConcurrentUploadLimit(limit int32) HostOption {
	return func(h *Host) {
		h.ConcurrentUploadLimit.Store(limit)
	}
}

// WithDisableShared sets host's DisableShared.
func WithDisableShared(disableShared bool) HostOption {
	return func(h *Host) {
		h.DisableShared = disableShared
	}
}

// WithOS sets host's os.
func WithOS(os string) HostOption {
	return func(h *Host) {
		h.OS = os
	}
}

// WithPlatform sets host's platform.
func WithPlatform(platform string) HostOption {
	return func(h *Host) {
		h.Platform = platform
	}
}

// WithPlatformFamily sets host's platform family.
func WithPlatformFamily(platformFamily string) HostOption {
	return func(h *Host) {
		h.PlatformFamily = platformFamily
	}
}

// WithPlatformVersion sets host's platform version.
func WithPlatformVersion(platformVersion string) HostOption {
	return func(h *Host) {
		h.PlatformVersion = platformVersion
	}
}

// WithKernelVersion sets host's kernel version.
func WithKernelVersion(kernelVersion string) HostOption {
	return func(h *Host) {
		h.KernelVersion = kernelVersion
	}
}

// WithCPU sets host's cpu.
func WithCPU(cpu CPU) HostOption {
	return func(h *Host) {
		h.CPU = cpu
	}
}

// WithMemory sets host's memory.
func WithMemory(memory Memory) HostOption {
	return func(h *Host) {
		h.Memory = memory
	}
}

// WithNetwork sets host's network.
func WithNetwork(network Network) HostOption {
	return func(h *Host) {
		h.Network = network
	}
}

// WithDisk sets host's disk.
func WithDisk(disk Disk) HostOption {
	return func(h *Host) {
		h.Disk = disk
	}
}

// WithBuild sets host's build information.
func WithBuild(build Build) HostOption {
	return func(h *Host) {
		h.Build = build
	}
}

// WithAnnounceInterval sets host's announce interval.
func WithAnnounceInterval(announceInterval time.Duration) HostOption {
	return func(h *Host) {
		h.AnnounceInterval = announceInterval
	}
}

// Host contains content for host.
type Host struct {
	// ID is host id.
	ID string

	// Type is host type.
	Type types.HostType

	// Name is the instance name.
	Name string

	// Hostname is host name.
	Hostname string

	// IP is host ip.
	IP string

	// Port is grpc service port.
	Port int32

	// DownloadPort is piece downloading port.
	DownloadPort int32

	// ProxyPort is proxy server port.
	ProxyPort int32

	// ObjectStoragePort is object storage port.
	ObjectStoragePort int32

	// DisableShared is whether the host is disabled for
	// shared with other peers.
	DisableShared bool

	// Host OS.
	OS string

	// Host platform.
	Platform string

	// Host platform family.
	PlatformFamily string

	// Host platform version.
	PlatformVersion string

	// Host kernel version.
	KernelVersion string

	// CPU Stat.
	CPU CPU

	// Memory Stat.
	Memory Memory

	// Network Stat.
	Network Network

	// Dist Stat.
	Disk Disk

	// Build information.
	Build Build

	// SchedulerClusterID is the scheduler cluster id matched by scopes.
	SchedulerClusterID uint64

	// AnnounceInterval is the interval between host announces to scheduler.
	AnnounceInterval time.Duration

	// TxBandwidth is transmit rate of the host, unit is bps. The transmit bandwidth calculated
	// by the scheduler. When the task is uploading, the host's transmit bandwidth will be increased.
	TxBandwidth *atomic.Uint64

	// UploadContentLength is the total content length of tasks currently being uploaded by
	// this host, unit is byte.
	UploadContentLength *atomic.Uint64

	// ConcurrentPieceCount is concurrent piece count of tasks currently being uploaded by
	// this host.
	ConcurrentUploadPieceCount *atomic.Uint64

	// ConcurrentUploadLimit is concurrent upload limit count.
	ConcurrentUploadLimit *atomic.Int32

	// ConcurrentUploadCount is concurrent upload count.
	ConcurrentUploadCount *atomic.Int32

	// UploadCount is total upload count.
	UploadCount *atomic.Int64

	// UploadFailedCount is upload failed count.
	UploadFailedCount *atomic.Int64

	// ConcurrentRegisterCount tracks active peer registration requests.
	ConcurrentRegisterCount *atomic.Int32

	// Peer sync map.
	Peers *sync.Map

	// PeerCount is peer count.
	PeerCount *atomic.Int32

	// CreatedAt is host create time.
	CreatedAt *pkgatomic.Time

	// UpdatedAt is host update time.
	UpdatedAt *pkgatomic.Time

	// Host log.
	Log *logger.SugaredLoggerOnWith
}

// CPU contains content for cpu.
type CPU struct {
	// Number of logical cores in the system.
	LogicalCount uint32

	// Number of physical cores in the system.
	PhysicalCount uint32

	// Percent calculates the percentage of cpu used.
	Percent float64

	// Calculates the percentage of cpu used by process.
	ProcessPercent float64

	// Times contains the amounts of time the CPU has spent performing different kinds of work.
	Times CPUTimes
}

// CPUTimes contains content for cpu times.
type CPUTimes struct {
	// CPU time of user.
	User float64

	// CPU time of system.
	System float64

	// CPU time of idle.
	Idle float64

	// CPU time of nice.
	Nice float64

	// CPU time of iowait.
	Iowait float64

	// CPU time of irq.
	Irq float64

	// CPU time of softirq.
	Softirq float64

	// CPU time of steal.
	Steal float64

	// CPU time of guest.
	Guest float64

	// CPU time of guest nice.
	GuestNice float64
}

// Memory contains content for memory.
type Memory struct {
	// Total amount of RAM on this system.
	Total uint64

	// RAM available for programs to allocate.
	Available uint64

	// RAM used by programs.
	Used uint64

	// Percentage of RAM used by programs.
	UsedPercent float64

	// Calculates the percentage of memory used by process.
	ProcessUsedPercent float64

	// This is the kernel's notion of free memory.
	Free uint64
}

// Network contains content for network.
type Network struct {
	// Return count of tcp connections opened and status is ESTABLISHED.
	TCPConnectionCount uint32

	// Return count of upload tcp connections opened and status is ESTABLISHED.
	UploadTCPConnectionCount uint32

	// Location path(area|country|province|city|...).
	Location string

	// IDC where the peer host is located
	IDC string

	// RxBandwidth is receive rate of the host, unit is bps. The receive bandwidth based on
	// reported data from host.
	RxBandwidth uint64

	// MaxRxBandwidth is max receive rate of the host, unit is bps. The max receive bandwidth based on
	// reported data from host.
	MaxRxBandwidth uint64

	// TxBandwidth is transmit rate of the host, unit is bps. The transmit bandwidth based on
	// reported data from host.
	TxBandwidth uint64

	// MaxTxBandwidth is max transmit rate of the host, unit is bps. The max transmit bandwidth based on
	// reported data from host.
	MaxTxBandwidth uint64
}

// Build contains content for build.
type Build struct {
	// Git version.
	GitVersion string

	// Git commit.
	GitCommit string

	// Golang version.
	GoVersion string

	// Build platform.
	Platform string
}

// Disk contains content for disk.
type Disk struct {
	// Total amount of disk on the data path of dragonfly.
	Total uint64

	// Free amount of disk on the data path of dragonfly.
	Free uint64

	// Used amount of disk on the data path of dragonfly.
	Used uint64

	// Used percent of disk on the data path of dragonfly directory.
	UsedPercent float64

	// Total amount of indoes on the data path of dragonfly directory.
	InodesTotal uint64

	// Used amount of indoes on the data path of dragonfly directory.
	InodesUsed uint64

	// Free amount of indoes on the data path of dragonfly directory.
	InodesFree uint64

	// Used percent of indoes on the data path of dragonfly directory.
	InodesUsedPercent float64

	// Disk write bandwidth, unit is byte/s.
	WriteBandwidth uint64

	// Disk read bandwidth, unit is byte/s.
	ReadBandwidth uint64
}

// New host instance.
func NewHost(
	id, ip, name, hostname string, port, downloadPort, proxyPort int32,
	typ types.HostType, options ...HostOption,
) *Host {
	// Calculate default of the concurrent upload limit by host type.
	concurrentUploadLimit := config.DefaultSeedPeerConcurrentUploadLimit
	if typ == types.HostTypeNormal {
		concurrentUploadLimit = config.DefaultPeerConcurrentUploadLimit
	}

	h := &Host{
		ID:                         id,
		Type:                       typ,
		IP:                         ip,
		Name:                       name,
		Hostname:                   hostname,
		Port:                       port,
		DownloadPort:               downloadPort,
		ProxyPort:                  proxyPort,
		DisableShared:              false,
		TxBandwidth:                new(atomic.Uint64),
		UploadContentLength:        new(atomic.Uint64),
		ConcurrentUploadPieceCount: new(atomic.Uint64),
		ConcurrentUploadLimit:      new(atomic.Int32),
		ConcurrentUploadCount:      new(atomic.Int32),
		UploadCount:                new(atomic.Int64),
		UploadFailedCount:          new(atomic.Int64),
		ConcurrentRegisterCount:    new(atomic.Int32),
		Peers:                      &sync.Map{},
		PeerCount:                  new(atomic.Int32),
		CreatedAt:                  pkgatomic.NewTime(time.Now()),
		UpdatedAt:                  pkgatomic.NewTime(time.Now()),
		Log:                        logger.WithHost(id, hostname, ip),
	}
	h.ConcurrentUploadLimit.Store(int32(concurrentUploadLimit))

	for _, opt := range options {
		opt(h)
	}

	return h
}

// LoadPeer return peer for a key.
func (h *Host) LoadPeer(key string) (*Peer, bool) {
	rawPeer, loaded := h.Peers.Load(key)
	if !loaded {
		return nil, false
	}

	return rawPeer.(*Peer), loaded
}

// StorePeer set peer.
func (h *Host) StorePeer(peer *Peer) {
	h.Peers.Store(peer.ID, peer)
	h.PeerCount.Add(1)
}

// DeletePeer deletes peer for a key.
func (h *Host) DeletePeer(key string) {
	if _, loaded := h.Peers.LoadAndDelete(key); loaded {
		h.PeerCount.Add(-1)
	}
}

// LeavePeers set peer state to PeerStateLeave.
func (h *Host) LeavePeers() {
	h.Peers.Range(func(_, value any) bool {
		peer, ok := value.(*Peer)
		if !ok {
			h.Log.Error("invalid peer")
			return true
		}

		if peer.FSM.Is(PeerStateLeave) {
			return true
		}

		peer.Log.Info("host leaves peers, causing the peer to leave")
		if err := peer.FSM.Event(context.Background(), PeerEventLeave); err != nil {
			peer.Log.Errorf("peer fsm event failed: %s", err.Error())
			return true
		}

		return true
	})
}

// FreeUploadCount return free upload count of host.
func (h *Host) FreeUploadCount() int32 {
	return h.ConcurrentUploadLimit.Load() - h.ConcurrentUploadCount.Load()
}

// IsSeedPeer return whether the host is seed peer.
func (h *Host) IsSeedPeer() bool {
	return h.Type != types.HostTypeNormal
}
