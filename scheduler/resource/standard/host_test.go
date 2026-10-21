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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	commonv2 "d7y.io/api/v2/pkg/apis/common/v2"

	logger "d7y.io/dragonfly/v2/internal/dflog"
	pkgatomic "d7y.io/dragonfly/v2/pkg/atomic"
	"d7y.io/dragonfly/v2/pkg/idgen"
	"d7y.io/dragonfly/v2/pkg/types"
	"d7y.io/dragonfly/v2/scheduler/config"
)

var (
	mockRawHost = Host{
		ID:              mockHostID,
		Type:            types.HostTypeNormal,
		Name:            "default-host",
		Hostname:        "foo",
		IP:              "127.0.0.1",
		Port:            8003,
		DownloadPort:    8001,
		ProxyPort:       8004,
		OS:              "darwin",
		Platform:        "darwin",
		PlatformFamily:  "Standalone Workstation",
		PlatformVersion: "11.1",
		KernelVersion:   "20.2.0",
		CPU:             mockCPU,
		Memory:          mockMemory,
		Network:         mockNetwork,
		Disk:            mockDisk,
		Build:           mockBuild,
		CreatedAt:       pkgatomic.NewTime(time.Now()),
		UpdatedAt:       pkgatomic.NewTime(time.Now()),
	}

	mockRawSeedHost = Host{
		ID:              mockSeedHostID,
		Type:            types.HostTypeSuperSeed,
		Name:            "default-seed-host",
		Hostname:        "bar",
		IP:              "127.0.0.1",
		Port:            8003,
		DownloadPort:    8001,
		ProxyPort:       8004,
		OS:              "darwin",
		Platform:        "darwin",
		PlatformFamily:  "Standalone Workstation",
		PlatformVersion: "11.1",
		KernelVersion:   "20.2.0",
		CPU:             mockCPU,
		Memory:          mockMemory,
		Network:         mockNetwork,
		Disk:            mockDisk,
		Build:           mockBuild,
		CreatedAt:       pkgatomic.NewTime(time.Now()),
		UpdatedAt:       pkgatomic.NewTime(time.Now()),
	}

	mockCPU = CPU{
		LogicalCount:   4,
		PhysicalCount:  2,
		Percent:        1,
		ProcessPercent: 0.5,
		Times: CPUTimes{
			User:      240662.2,
			System:    317950.1,
			Idle:      3393691.3,
			Nice:      0,
			Iowait:    0,
			Irq:       0,
			Softirq:   0,
			Steal:     0,
			Guest:     0,
			GuestNice: 0,
		},
	}

	mockMemory = Memory{
		Total:              17179869184,
		Available:          5962813440,
		Used:               11217055744,
		UsedPercent:        65.291858,
		ProcessUsedPercent: 41.525125,
		Free:               2749598908,
	}

	mockNetwork = Network{
		TCPConnectionCount:       10,
		UploadTCPConnectionCount: 1,
		Location:                 mockHostLocation,
		IDC:                      mockHostIDC,
		RxBandwidth:              100,
		MaxRxBandwidth:           200,
		TxBandwidth:              100,
		MaxTxBandwidth:           200,
	}

	mockDisk = Disk{
		Total:             499963174912,
		Free:              37226479616,
		Used:              423809622016,
		UsedPercent:       91.92547406065952,
		InodesTotal:       4882452880,
		InodesUsed:        7835772,
		InodesFree:        4874617108,
		InodesUsedPercent: 0.1604884305611568,
	}

	mockBuild = Build{
		GitVersion: "v1.0.0",
		GitCommit:  "221176b117c6d59366d68f2b34d38be50c935883",
		GoVersion:  "1.18",
		Platform:   "darwin",
	}

	mockAnnounceInterval = 5 * time.Minute

	mockHostID       = idgen.HostID("127.0.0.1", "foo", false)
	mockSeedHostID   = idgen.HostID("127.0.0.1", "bar", true)
	mockHostLocation = "baz"
	mockHostIDC      = "bas"
)

func TestHost_NewHost(t *testing.T) {
	tests := []struct {
		name    string
		rawHost Host
		options []HostOption
		expect  func(t *testing.T, host *Host)
	}{
		{
			name:    "new host",
			rawHost: mockRawHost,
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.False(host.DisableShared)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new seed host",
			rawHost: mockRawSeedHost,
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawSeedHost.ID, host.ID)
				assert.Equal(mockRawSeedHost.Type, host.Type)
				assert.Equal(mockRawSeedHost.Name, host.Name)
				assert.Equal(mockRawSeedHost.Hostname, host.Hostname)
				assert.Equal(mockRawSeedHost.IP, host.IP)
				assert.Equal(mockRawSeedHost.Port, host.Port)
				assert.Equal(mockRawSeedHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawSeedHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.False(host.DisableShared)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultSeedPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set scheduler cluster id",
			rawHost: mockRawHost,
			options: []HostOption{WithSchedulerClusterID(1)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(uint64(1), host.SchedulerClusterID)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set object storage port",
			rawHost: mockRawHost,
			options: []HostOption{WithObjectStoragePort(1)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(1), host.ObjectStoragePort)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set upload loadlimit",
			rawHost: mockRawHost,
			options: []HostOption{WithConcurrentUploadLimit(300)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(300), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set disable shared",
			rawHost: mockRawHost,
			options: []HostOption{WithDisableShared(true)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.True(host.DisableShared)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set os",
			rawHost: mockRawHost,
			options: []HostOption{WithOS("linux")},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.Equal("linux", host.OS)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set platform",
			rawHost: mockRawHost,
			options: []HostOption{WithPlatform("ubuntu")},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.Equal("ubuntu", host.Platform)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set platform family",
			rawHost: mockRawHost,
			options: []HostOption{WithPlatformFamily("debian")},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.Equal("debian", host.PlatformFamily)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set platform version",
			rawHost: mockRawHost,
			options: []HostOption{WithPlatformVersion("22.04")},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.Equal("22.04", host.PlatformVersion)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set kernel version",
			rawHost: mockRawHost,
			options: []HostOption{WithKernelVersion("5.15.0-27-generic")},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.Equal("5.15.0-27-generic", host.KernelVersion)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set cpu",
			rawHost: mockRawHost,
			options: []HostOption{WithCPU(mockCPU)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.EqualValues(mockCPU, host.CPU)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set memory",
			rawHost: mockRawHost,
			options: []HostOption{WithMemory(mockMemory)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.EqualValues(mockMemory, host.Memory)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set network",
			rawHost: mockRawHost,
			options: []HostOption{WithNetwork(mockNetwork)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.EqualValues(mockNetwork, host.Network)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set disk",
			rawHost: mockRawHost,
			options: []HostOption{WithDisk(mockDisk)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.EqualValues(mockDisk, host.Disk)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set build",
			rawHost: mockRawHost,
			options: []HostOption{WithBuild(mockBuild)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.EqualValues(mockBuild, host.Build)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(time.Duration(0), host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
		{
			name:    "new host and set announce interval",
			rawHost: mockRawHost,
			options: []HostOption{WithAnnounceInterval(mockAnnounceInterval)},
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(types.HostTypeNormal, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(int32(0), host.ObjectStoragePort)
				assert.Equal(uint64(0), host.SchedulerClusterID)
				assert.Equal(mockAnnounceInterval, host.AnnounceInterval)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.ConcurrentUploadLimit.Load())
				assert.Equal(int32(0), host.ConcurrentUploadCount.Load())
				assert.Equal(int64(0), host.UploadCount.Load())
				assert.Equal(int64(0), host.UploadFailedCount.Load())
				assert.NotNil(host.Peers)
				assert.Equal(int32(0), host.PeerCount.Load())
				assert.NotEmpty(host.CreatedAt.Load())
				assert.NotEmpty(host.UpdatedAt.Load())
				assert.NotNil(host.Log)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.expect(t, NewHost(
				tc.rawHost.ID, tc.rawHost.IP, tc.rawHost.Name, tc.rawHost.Hostname,
				tc.rawHost.Port, tc.rawHost.DownloadPort, tc.rawHost.ProxyPort, tc.rawHost.Type,
				tc.options...))
		})
	}
}

func TestHost_LoadPeer(t *testing.T) {
	tests := []struct {
		name   string
		peerID string
		expect func(t *testing.T, peer *Peer, loaded bool)
	}{
		{
			name:   "load peer",
			peerID: mockPeerID,
			expect: func(t *testing.T, peer *Peer, loaded bool) {
				assert := assert.New(t)
				assert.True(loaded)
				assert.Equal(mockPeerID, peer.ID)
			},
		},
		{
			name:   "peer does not exist",
			peerID: idgen.PeerID(),
			expect: func(t *testing.T, peer *Peer, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
			},
		},
		{
			name:   "load key is empty",
			peerID: "",
			expect: func(t *testing.T, peer *Peer, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, WithDigest(mockTaskDigest))
			mockPeer := NewPeer(mockPeerID, mockTask, host)

			host.StorePeer(mockPeer)
			peer, loaded := host.LoadPeer(tc.peerID)
			tc.expect(t, peer, loaded)
		})
	}
}

func TestHost_StorePeer(t *testing.T) {
	tests := []struct {
		name   string
		peerID string
		expect func(t *testing.T, host *Host, peer *Peer, loaded bool)
	}{
		{
			name:   "store peer",
			peerID: mockPeerID,
			expect: func(t *testing.T, host *Host, peer *Peer, loaded bool) {
				assert := assert.New(t)
				assert.True(loaded)
				assert.Equal(mockPeerID, peer.ID)
				assert.Equal(int32(1), host.PeerCount.Load())
			},
		},
		{
			name:   "store key is empty",
			peerID: "",
			expect: func(t *testing.T, host *Host, peer *Peer, loaded bool) {
				assert := assert.New(t)
				assert.True(loaded)
				assert.Equal("", peer.ID)
				assert.Equal(int32(1), host.PeerCount.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, WithDigest(mockTaskDigest))
			mockPeer := NewPeer(tc.peerID, mockTask, host)

			host.StorePeer(mockPeer)
			peer, loaded := host.LoadPeer(tc.peerID)
			tc.expect(t, host, peer, loaded)
		})
	}
}

func TestHost_DeletePeer(t *testing.T) {
	tests := []struct {
		name   string
		peerID string
		expect func(t *testing.T, host *Host)
	}{
		{
			name:   "delete peer",
			peerID: mockPeerID,
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				_, loaded := host.LoadPeer(mockPeerID)
				assert.False(loaded)
				assert.Equal(int32(0), host.PeerCount.Load())
			},
		},
		{
			name:   "delete key is empty",
			peerID: "",
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				peer, loaded := host.LoadPeer(mockPeerID)
				assert.True(loaded)
				assert.Equal(mockPeerID, peer.ID)
				assert.Equal(int32(1), host.PeerCount.Load())
			},
		},
		{
			name:   "delete key does not exist",
			peerID: idgen.PeerID(),
			expect: func(t *testing.T, host *Host) {
				assert := assert.New(t)
				_, loaded := host.LoadPeer(mockPeerID)
				assert.True(loaded)
				assert.Equal(int32(1), host.PeerCount.Load())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, WithDigest(mockTaskDigest))
			mockPeer := NewPeer(mockPeerID, mockTask, host)

			host.StorePeer(mockPeer)
			host.DeletePeer(tc.peerID)
			tc.expect(t, host)
		})
	}
}

func TestHost_LeavePeers(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, host *Host, mockPeer *Peer)
	}{
		{
			name: "leave peers",
			expect: func(t *testing.T, host *Host, mockPeer *Peer) {
				assert := assert.New(t)
				host.StorePeer(mockPeer)
				assert.Equal(int32(1), host.PeerCount.Load())
				host.LeavePeers()
				host.Peers.Range(func(_, value any) bool {
					peer := value.(*Peer)
					assert.True(peer.FSM.Is(PeerStateLeave))
					return true
				})
			},
		},
		{
			name: "leave peers keeps peers that already left",
			expect: func(t *testing.T, host *Host, mockPeer *Peer) {
				assert := assert.New(t)
				mockPeer.FSM.SetState(PeerStateLeave)
				host.StorePeer(mockPeer)
				host.LeavePeers()
				peer, loaded := host.LoadPeer(mockPeer.ID)
				assert.True(loaded)
				assert.True(peer.FSM.Is(PeerStateLeave))
				assert.Equal(int32(1), host.PeerCount.Load())
			},
		},
		{
			name: "leave peers does not log an error for peers already left",
			expect: func(t *testing.T, host *Host, mockPeer *Peer) {
				mockPeer.FSM.SetState(PeerStateLeave)
				host.StorePeer(mockPeer)

				core, logs := observer.New(zap.ErrorLevel)
				original := logger.CoreLogger
				logger.SetCoreLogger(zap.New(core).Sugar())
				defer logger.SetCoreLogger(original)

				host.LeavePeers()
				assert.Empty(t, logs.All())
			},
		},
		{
			name: "peers is empty",
			expect: func(t *testing.T, host *Host, mockPeer *Peer) {
				assert := assert.New(t)
				assert.Equal(int32(0), host.PeerCount.Load())
				host.LeavePeers()
				var count int
				host.Peers.Range(func(_, _ any) bool {
					count++
					return true
				})

				assert.Zero(count)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, WithDigest(mockTaskDigest))
			mockPeer := NewPeer(mockPeerID, mockTask, host)

			tc.expect(t, host, mockPeer)
		})
	}
}

func TestHost_FreeUploadCount(t *testing.T) {
	tests := []struct {
		name   string
		expect func(t *testing.T, host *Host, mockTask *Task, mockPeer *Peer)
	}{
		{
			name: "get free upload load",
			expect: func(t *testing.T, host *Host, mockTask *Task, mockPeer *Peer) {
				assert := assert.New(t)
				mockSeedPeer := NewPeer(mockSeedPeerID, mockTask, host)
				mockPeer.Task.StorePeer(mockSeedPeer)
				mockPeer.Task.StorePeer(mockPeer)
				err := mockPeer.Task.AddPeerEdge(mockSeedPeer, mockPeer)
				assert.NoError(err)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit-1), host.FreeUploadCount())
				err = mockTask.DeletePeerInEdges(mockPeer.ID)
				assert.NoError(err)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.FreeUploadCount())
				err = mockPeer.Task.AddPeerEdge(mockSeedPeer, mockPeer)
				assert.NoError(err)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit-1), host.FreeUploadCount())
				err = mockTask.DeletePeerOutEdges(mockSeedPeer.ID)
				assert.NoError(err)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.FreeUploadCount())
			},
		},
		{
			name: "upload peer does not exist",
			expect: func(t *testing.T, host *Host, mockTask *Task, mockPeer *Peer) {
				assert := assert.New(t)
				assert.Equal(int32(config.DefaultPeerConcurrentUploadLimit), host.FreeUploadCount())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, mockRawHost.Type)
			mockTask := NewTask(mockTaskID, mockTaskURL, mockTaskTag, mockTaskApplication, commonv2.TaskType_STANDARD, mockTaskFilteredQueryParams, mockTaskHeader, mockTaskBackToSourceLimit, WithDigest(mockTaskDigest))
			mockPeer := NewPeer(mockPeerID, mockTask, host)

			tc.expect(t, host, mockTask, mockPeer)
		})
	}
}

func TestHost_IsSeedPeer(t *testing.T) {
	tests := []struct {
		name   string
		typ    types.HostType
		expect func(t *testing.T, isSeedPeer bool)
	}{
		{
			name: "normal host",
			typ:  types.HostTypeNormal,
			expect: func(t *testing.T, isSeedPeer bool) {
				assert := assert.New(t)
				assert.False(isSeedPeer)
			},
		},
		{
			name: "super seed host",
			typ:  types.HostTypeSuperSeed,
			expect: func(t *testing.T, isSeedPeer bool) {
				assert := assert.New(t)
				assert.True(isSeedPeer)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := NewHost(
				mockRawHost.ID, mockRawHost.IP, mockRawHost.Name, mockRawHost.Hostname,
				mockRawHost.Port, mockRawHost.DownloadPort, mockRawHost.ProxyPort, tc.typ)
			tc.expect(t, host.IsSeedPeer())
		})
	}
}
