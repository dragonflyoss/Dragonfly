/*
 *     Copyright 2025 The Dragonfly Authors
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

package persistentcache

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"d7y.io/dragonfly/v2/pkg/container/set"
	pkggc "d7y.io/dragonfly/v2/pkg/gc"
	pkgredis "d7y.io/dragonfly/v2/pkg/redis"
	"d7y.io/dragonfly/v2/scheduler/config"
)

var mockHostManagerConfig = &config.Config{
	Manager:   config.ManagerConfig{SchedulerClusterID: 1},
	Scheduler: config.SchedulerConfig{GC: config.GCConfig{HostGCInterval: 10 * time.Minute}},
}

func mockRawHostFields(hostID string) map[string]string {
	return map[string]string{
		"id":                                  hostID,
		"type":                                mockRawHost.Type.Name(),
		"name":                                mockRawHost.Name,
		"hostname":                            mockRawHost.Hostname,
		"ip":                                  mockRawHost.IP,
		"port":                                strconv.FormatInt(int64(mockRawHost.Port), 10),
		"download_port":                       strconv.FormatInt(int64(mockRawHost.DownloadPort), 10),
		"proxy_port":                          strconv.FormatInt(int64(mockRawHost.ProxyPort), 10),
		"disable_shared":                      strconv.FormatBool(mockRawHost.DisableShared),
		"os":                                  mockRawHost.OS,
		"platform":                            mockRawHost.Platform,
		"platform_family":                     mockRawHost.PlatformFamily,
		"platform_version":                    mockRawHost.PlatformVersion,
		"kernel_version":                      mockRawHost.KernelVersion,
		"cpu_logical_count":                   strconv.FormatUint(uint64(mockRawHost.CPU.LogicalCount), 10),
		"cpu_physical_count":                  strconv.FormatUint(uint64(mockRawHost.CPU.PhysicalCount), 10),
		"cpu_percent":                         strconv.FormatFloat(mockRawHost.CPU.Percent, 'f', -1, 64),
		"cpu_process_percent":                 strconv.FormatFloat(mockRawHost.CPU.ProcessPercent, 'f', -1, 64),
		"cpu_times_user":                      strconv.FormatFloat(mockRawHost.CPU.Times.User, 'f', -1, 64),
		"cpu_times_system":                    strconv.FormatFloat(mockRawHost.CPU.Times.System, 'f', -1, 64),
		"cpu_times_idle":                      strconv.FormatFloat(mockRawHost.CPU.Times.Idle, 'f', -1, 64),
		"cpu_times_nice":                      strconv.FormatFloat(mockRawHost.CPU.Times.Nice, 'f', -1, 64),
		"cpu_times_iowait":                    strconv.FormatFloat(mockRawHost.CPU.Times.Iowait, 'f', -1, 64),
		"cpu_times_irq":                       strconv.FormatFloat(mockRawHost.CPU.Times.Irq, 'f', -1, 64),
		"cpu_times_softirq":                   strconv.FormatFloat(mockRawHost.CPU.Times.Softirq, 'f', -1, 64),
		"cpu_times_steal":                     strconv.FormatFloat(mockRawHost.CPU.Times.Steal, 'f', -1, 64),
		"cpu_times_guest":                     strconv.FormatFloat(mockRawHost.CPU.Times.Guest, 'f', -1, 64),
		"cpu_times_guest_nice":                strconv.FormatFloat(mockRawHost.CPU.Times.GuestNice, 'f', -1, 64),
		"memory_total":                        strconv.FormatUint(mockRawHost.Memory.Total, 10),
		"memory_available":                    strconv.FormatUint(mockRawHost.Memory.Available, 10),
		"memory_used":                         strconv.FormatUint(mockRawHost.Memory.Used, 10),
		"memory_used_percent":                 strconv.FormatFloat(mockRawHost.Memory.UsedPercent, 'f', -1, 64),
		"memory_process_used_percent":         strconv.FormatFloat(mockRawHost.Memory.ProcessUsedPercent, 'f', -1, 64),
		"memory_free":                         strconv.FormatUint(mockRawHost.Memory.Free, 10),
		"network_tcp_connection_count":        strconv.FormatUint(uint64(mockRawHost.Network.TCPConnectionCount), 10),
		"network_upload_tcp_connection_count": strconv.FormatUint(uint64(mockRawHost.Network.UploadTCPConnectionCount), 10),
		"network_location":                    mockRawHost.Network.Location,
		"network_idc":                         mockRawHost.Network.IDC,
		"network_rx_bandwidth":                strconv.FormatUint(mockRawHost.Network.RxBandwidth, 10),
		"network_max_rx_bandwidth":            strconv.FormatUint(mockRawHost.Network.MaxRxBandwidth, 10),
		"network_tx_bandwidth":                strconv.FormatUint(mockRawHost.Network.TxBandwidth, 10),
		"network_max_tx_bandwidth":            strconv.FormatUint(mockRawHost.Network.MaxTxBandwidth, 10),
		"disk_total":                          strconv.FormatUint(mockRawHost.Disk.Total, 10),
		"disk_free":                           strconv.FormatUint(mockRawHost.Disk.Free, 10),
		"disk_used":                           strconv.FormatUint(mockRawHost.Disk.Used, 10),
		"disk_used_percent":                   strconv.FormatFloat(mockRawHost.Disk.UsedPercent, 'f', -1, 64),
		"disk_inodes_total":                   strconv.FormatUint(mockRawHost.Disk.InodesTotal, 10),
		"disk_inodes_used":                    strconv.FormatUint(mockRawHost.Disk.InodesUsed, 10),
		"disk_inodes_free":                    strconv.FormatUint(mockRawHost.Disk.InodesFree, 10),
		"disk_inodes_used_percent":            strconv.FormatFloat(mockRawHost.Disk.InodesUsedPercent, 'f', -1, 64),
		"disk_write_bandwidth":                strconv.FormatUint(mockRawHost.Disk.WriteBandwidth, 10),
		"disk_read_bandwidth":                 strconv.FormatUint(mockRawHost.Disk.ReadBandwidth, 10),
		"build_git_version":                   mockRawHost.Build.GitVersion,
		"build_git_commit":                    mockRawHost.Build.GitCommit,
		"build_go_version":                    mockRawHost.Build.GoVersion,
		"build_platform":                      mockRawHost.Build.Platform,
		"scheduler_cluster_id":                strconv.FormatUint(mockRawHost.SchedulerClusterID, 10),
		"announce_interval":                   strconv.FormatInt(mockRawHost.AnnounceInterval.Nanoseconds(), 10),
		"created_at":                          mockRawHost.CreatedAt.Format(time.RFC3339),
		"updated_at":                          mockRawHost.UpdatedAt.Format(time.RFC3339),
	}
}

func mockStoreHostScript(mock redismock.ClientMock) *redismock.ExpectedCmd {
	return mock.CustomMatch(matchScriptArgs).ExpectEvalSha("", []string{
		pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID),
		pkgredis.MakePersistentCacheHostsInScheduler(1),
	},
		mockRawHost.ID,
		mockRawHost.Type.Name(),
		mockRawHost.Name,
		mockRawHost.Hostname,
		mockRawHost.IP,
		mockRawHost.Port,
		mockRawHost.DownloadPort,
		mockRawHost.ProxyPort,
		mockRawHost.DisableShared,
		mockRawHost.OS,
		mockRawHost.Platform,
		mockRawHost.PlatformFamily,
		mockRawHost.PlatformVersion,
		mockRawHost.KernelVersion,
		mockRawHost.CPU.LogicalCount,
		mockRawHost.CPU.PhysicalCount,
		mockRawHost.CPU.Percent,
		mockRawHost.CPU.ProcessPercent,
		mockRawHost.CPU.Times.User,
		mockRawHost.CPU.Times.System,
		mockRawHost.CPU.Times.Idle,
		mockRawHost.CPU.Times.Nice,
		mockRawHost.CPU.Times.Iowait,
		mockRawHost.CPU.Times.Irq,
		mockRawHost.CPU.Times.Softirq,
		mockRawHost.CPU.Times.Steal,
		mockRawHost.CPU.Times.Guest,
		mockRawHost.CPU.Times.GuestNice,
		mockRawHost.Memory.Total,
		mockRawHost.Memory.Available,
		mockRawHost.Memory.Used,
		mockRawHost.Memory.UsedPercent,
		mockRawHost.Memory.ProcessUsedPercent,
		mockRawHost.Memory.Free,
		mockRawHost.Network.TCPConnectionCount,
		mockRawHost.Network.UploadTCPConnectionCount,
		mockRawHost.Network.Location,
		mockRawHost.Network.IDC,
		mockRawHost.Network.RxBandwidth,
		mockRawHost.Network.MaxRxBandwidth,
		mockRawHost.Network.TxBandwidth,
		mockRawHost.Network.MaxTxBandwidth,
		mockRawHost.Disk.Total,
		mockRawHost.Disk.Free,
		mockRawHost.Disk.Used,
		mockRawHost.Disk.UsedPercent,
		mockRawHost.Disk.InodesTotal,
		mockRawHost.Disk.InodesUsed,
		mockRawHost.Disk.InodesFree,
		mockRawHost.Disk.InodesUsedPercent,
		mockRawHost.Disk.WriteBandwidth,
		mockRawHost.Disk.ReadBandwidth,
		mockRawHost.Build.GitVersion,
		mockRawHost.Build.GitCommit,
		mockRawHost.Build.GoVersion,
		mockRawHost.Build.Platform,
		mockRawHost.SchedulerClusterID,
		mockRawHost.AnnounceInterval.Nanoseconds(),
		mockRawHost.CreatedAt.Format(time.RFC3339),
		mockRawHost.UpdatedAt.Format(time.RFC3339),
	)
}

func mockDeleteHostScript(mock redismock.ClientMock, hostID string) *redismock.ExpectedCmd {
	return mock.CustomMatch(matchScriptArgs).ExpectEvalSha("", []string{
		pkgredis.MakePersistentCacheHostKeyInScheduler(1, hostID),
		pkgredis.MakePersistentCacheHostsInScheduler(1),
	}, hostID)
}

func TestNewHostManager(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(mock *pkggc.MockGCMockRecorder)
		expect func(t *testing.T, hostManager HostManager, err error)
	}{
		{
			name: "registers host gc task with configured interval",
			mock: func(mock *pkggc.MockGCMockRecorder) {
				mock.Add(gomock.Cond(func(task pkggc.Task) bool {
					return task.ID == GCHostID &&
						task.Interval == mockHostManagerConfig.Scheduler.GC.HostGCInterval &&
						task.Timeout == mockHostManagerConfig.Scheduler.GC.HostGCInterval &&
						task.Runner != nil
				})).Return(nil).Times(1)
			},
			expect: func(t *testing.T, hostManager HostManager, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.NotNil(hostManager)
			},
		},
		{
			name: "gc registration fails",
			mock: func(mock *pkggc.MockGCMockRecorder) {
				mock.Add(gomock.Any()).Return(errors.New("gc error")).Times(1)
			},
			expect: func(t *testing.T, hostManager HostManager, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Nil(hostManager)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			gc := pkggc.NewMockGC(ctrl)
			tc.mock(gc.EXPECT())

			rdb, _ := redismock.NewClientMock()
			hostManager, err := newHostManager(mockHostManagerConfig, gc, rdb)
			tc.expect(t, hostManager, err)
		})
	}
}

func TestHostManager_Load(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(mock redismock.ClientMock)
		expect func(t *testing.T, host *Host, loaded bool)
	}{
		{
			name: "redis error",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetErr(errors.New("redis error"))
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "host not found",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(map[string]string{})
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "missing cpu_process_percent and memory_process_used_percent default to zero",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				delete(fields, "cpu_process_percent")
				delete(fields, "memory_process_used_percent")
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.True(loaded)
				assert.Equal(float64(0), host.CPU.ProcessPercent)
				assert.Equal(float64(0), host.Memory.ProcessUsedPercent)
			},
		},
		{
			name: "invalid port value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["port"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid download_port value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["download_port"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid proxy_port value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["proxy_port"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid scheduler_cluster_id value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["scheduler_cluster_id"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disable_shared value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disable_shared"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_logical_count value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_logical_count"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_physical_count value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_physical_count"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_percent value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_percent"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_process_percent value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_process_percent"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_user value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_user"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_system value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_system"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_idle value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_idle"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_nice value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_nice"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_iowait value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_iowait"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_irq value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_irq"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_softirq value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_softirq"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_steal value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_steal"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_guest value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_guest"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid cpu_times_guest_nice value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["cpu_times_guest_nice"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid memory_total value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["memory_total"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid memory_available value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["memory_available"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid memory_used value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["memory_used"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid memory_used_percent value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["memory_used_percent"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid memory_process_used_percent value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["memory_process_used_percent"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid memory_free value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["memory_free"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid network_tcp_connection_count value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["network_tcp_connection_count"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid network_upload_tcp_connection_count value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["network_upload_tcp_connection_count"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid network_rx_bandwidth value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["network_rx_bandwidth"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid network_max_rx_bandwidth value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["network_max_rx_bandwidth"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid network_tx_bandwidth value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["network_tx_bandwidth"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid network_max_tx_bandwidth value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["network_max_tx_bandwidth"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_total value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_total"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_free value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_free"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_used value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_used"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_used_percent value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_used_percent"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_inodes_total value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_inodes_total"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_inodes_used value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_inodes_used"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_inodes_free value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_inodes_free"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_inodes_used_percent value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_inodes_used_percent"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_write_bandwidth value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_write_bandwidth"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid disk_read_bandwidth value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["disk_read_bandwidth"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid announce_interval value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["announce_interval"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid created_at value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["created_at"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "invalid updated_at value",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields(mockRawHost.ID)
				fields["updated_at"] = mockInvalidFieldValue
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(fields)
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.False(loaded)
				assert.Nil(host)
			},
		},
		{
			name: "successful load",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, mockRawHost.ID)).SetVal(mockRawHostFields(mockRawHost.ID))
			},
			expect: func(t *testing.T, host *Host, loaded bool) {
				assert := assert.New(t)
				assert.True(loaded)
				assert.Equal(mockRawHost.ID, host.ID)
				assert.Equal(mockRawHost.Type, host.Type)
				assert.Equal(mockRawHost.Name, host.Name)
				assert.Equal(mockRawHost.Hostname, host.Hostname)
				assert.Equal(mockRawHost.IP, host.IP)
				assert.Equal(mockRawHost.Port, host.Port)
				assert.Equal(mockRawHost.DownloadPort, host.DownloadPort)
				assert.Equal(mockRawHost.ProxyPort, host.ProxyPort)
				assert.Equal(mockRawHost.DisableShared, host.DisableShared)
				assert.Equal(mockRawHost.OS, host.OS)
				assert.Equal(mockRawHost.Platform, host.Platform)
				assert.Equal(mockRawHost.PlatformFamily, host.PlatformFamily)
				assert.Equal(mockRawHost.PlatformVersion, host.PlatformVersion)
				assert.Equal(mockRawHost.KernelVersion, host.KernelVersion)
				assert.Equal(mockRawHost.CPU, host.CPU)
				assert.Equal(mockRawHost.Memory, host.Memory)
				assert.Equal(mockRawHost.Network, host.Network)
				assert.Equal(mockRawHost.Disk, host.Disk)
				assert.Equal(mockRawHost.Build, host.Build)
				assert.Equal(mockRawHost.SchedulerClusterID, host.SchedulerClusterID)
				assert.Equal(mockRawHost.AnnounceInterval, host.AnnounceInterval)
				assert.Equal(mockRawHost.CreatedAt.Format(time.RFC3339), host.CreatedAt.Format(time.RFC3339))
				assert.Equal(mockRawHost.UpdatedAt.Format(time.RFC3339), host.UpdatedAt.Format(time.RFC3339))
				assert.NotNil(host.Log)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			rdb, mock := redismock.NewClientMock()
			tc.mock(mock)

			h := &hostManager{config: mockHostManagerConfig, rdb: rdb}
			host, loaded := h.Load(context.Background(), mockRawHost.ID)
			tc.expect(t, host, loaded)
			assert.NoError(mock.ExpectationsWereMet())
		})
	}
}

func TestHostManager_Store(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(mock redismock.ClientMock)
		expect func(t *testing.T, err error)
	}{
		{
			name: "store succeeds",
			mock: func(mock redismock.ClientMock) {
				mockStoreHostScript(mock).SetVal(true)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "redis error",
			mock: func(mock redismock.ClientMock) {
				mockStoreHostScript(mock).SetErr(errors.New("redis error"))
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			rdb, mock := redismock.NewClientMock()
			tc.mock(mock)

			h := &hostManager{config: mockHostManagerConfig, rdb: rdb}
			tc.expect(t, h.Store(context.Background(), &mockRawHost))
			assert.NoError(mock.ExpectationsWereMet())
		})
	}
}

func TestHostManager_Delete(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(mock redismock.ClientMock)
		expect func(t *testing.T, err error)
	}{
		{
			name: "delete succeeds",
			mock: func(mock redismock.ClientMock) {
				mockDeleteHostScript(mock, mockRawHost.ID).SetVal(true)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "redis error",
			mock: func(mock redismock.ClientMock) {
				mockDeleteHostScript(mock, mockRawHost.ID).SetErr(errors.New("redis error"))
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			rdb, mock := redismock.NewClientMock()
			tc.mock(mock)

			h := &hostManager{config: mockHostManagerConfig, rdb: rdb}
			tc.expect(t, h.Delete(context.Background(), mockRawHost.ID))
			assert.NoError(mock.ExpectationsWereMet())
		})
	}
}

func TestHostManager_LoadAll(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(mock redismock.ClientMock)
		expect func(t *testing.T, hosts []*Host, err error)
	}{
		{
			name: "scan fails",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetErr(errors.New("redis scan error"))
			},
			expect: func(t *testing.T, hosts []*Host, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Nil(hosts)
			},
		},
		{
			name: "host that fails to load is skipped",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host1", "host2"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetVal(mockRawHostFields("host1"))
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host2")).SetErr(errors.New("redis hgetall error"))
			},
			expect: func(t *testing.T, hosts []*Host, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(hosts, 1)
				assert.Equal("host1", hosts[0].ID)
			},
		},
		{
			name: "hosts spanning multiple scan cursors are all loaded",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host3"}, 123)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host3")).SetVal(mockRawHostFields("host3"))
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 123, "*", 10).SetVal([]string{"host4"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host4")).SetVal(mockRawHostFields("host4"))
			},
			expect: func(t *testing.T, hosts []*Host, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(hosts, 2)
				assert.Equal("host3", hosts[0].ID)
				assert.Equal("host4", hosts[1].ID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			rdb, mock := redismock.NewClientMock()
			tc.mock(mock)

			h := &hostManager{config: mockHostManagerConfig, rdb: rdb}
			hosts, err := h.LoadAll(context.Background())
			tc.expect(t, hosts, err)
			assert.NoError(mock.ExpectationsWereMet())
		})
	}
}

func TestHostManager_LoadRandom(t *testing.T) {
	tests := []struct {
		name      string
		n         int
		blocklist set.SafeSet[string]
		mock      func(mock redismock.ClientMock)
		expect    func(t *testing.T, hosts []*Host, err error)
	}{
		{
			name:      "smembers fails",
			n:         2,
			blocklist: set.NewSafeSet[string](),
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSMembers(pkgredis.MakePersistentCacheHostsInScheduler(1)).SetErr(errors.New("redis error"))
			},
			expect: func(t *testing.T, hosts []*Host, err error) {
				assert := assert.New(t)
				assert.Error(err)
				assert.Nil(hosts)
			},
		},
		{
			name: "every host in blocklist is skipped",
			n:    3,
			blocklist: func() set.SafeSet[string] {
				s := set.NewSafeSet[string]()
				s.Add("host1")
				s.Add("host2")
				s.Add("host3")
				return s
			}(),
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSMembers(pkgredis.MakePersistentCacheHostsInScheduler(1)).SetVal([]string{"host1", "host2", "host3"})
			},
			expect: func(t *testing.T, hosts []*Host, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Empty(hosts)
			},
		},
		{
			name: "blocklisted host is skipped and remaining host is loaded",
			n:    2,
			blocklist: func() set.SafeSet[string] {
				s := set.NewSafeSet[string]()
				s.Add("host1")
				return s
			}(),
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSMembers(pkgredis.MakePersistentCacheHostsInScheduler(1)).SetVal([]string{"host1", "host2"})
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host2")).SetVal(mockRawHostFields("host2"))
			},
			expect: func(t *testing.T, hosts []*Host, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Len(hosts, 1)
				assert.Equal("host2", hosts[0].ID)
			},
		},
		{
			name:      "host that fails to load is skipped",
			n:         1,
			blocklist: set.NewSafeSet[string](),
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSMembers(pkgredis.MakePersistentCacheHostsInScheduler(1)).SetVal([]string{"host1"})
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetErr(errors.New("redis hgetall error"))
			},
			expect: func(t *testing.T, hosts []*Host, err error) {
				assert := assert.New(t)
				assert.NoError(err)
				assert.Empty(hosts)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			rdb, mock := redismock.NewClientMock()
			tc.mock(mock)

			h := &hostManager{config: mockHostManagerConfig, rdb: rdb}
			hosts, err := h.LoadRandom(context.Background(), tc.n, tc.blocklist)
			tc.expect(t, hosts, err)
			assert.NoError(mock.ExpectationsWereMet())
		})
	}
}

func TestHostManager_RunGC(t *testing.T) {
	tests := []struct {
		name   string
		mock   func(mock redismock.ClientMock)
		expect func(t *testing.T, err error)
	}{
		{
			name: "load all fails",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetErr(errors.New("redis error"))
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.Error(err)
			},
		},
		{
			name: "recently announced host is kept",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields("host1")
				fields["announce_interval"] = strconv.FormatInt(mockAnnounceInterval.Nanoseconds(), 10)
				fields["updated_at"] = time.Now().Format(time.RFC3339)
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host1"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetVal(fields)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "host without announce interval is never reclaimed",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields("host1")
				fields["announce_interval"] = "0"
				fields["updated_at"] = time.Now().Add(-time.Hour).Format(time.RFC3339)
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host1"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetVal(fields)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "host silent for more than twice the announce interval is reclaimed",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields("host1")
				fields["announce_interval"] = strconv.FormatInt(mockAnnounceInterval.Nanoseconds(), 10)
				fields["updated_at"] = time.Now().Add(-3 * mockAnnounceInterval).Format(time.RFC3339)
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host1"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetVal(fields)
				mockDeleteHostScript(mock, "host1").SetVal(true)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "failure to reclaim a stale host does not fail gc",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields("host1")
				fields["announce_interval"] = strconv.FormatInt(mockAnnounceInterval.Nanoseconds(), 10)
				fields["updated_at"] = time.Now().Add(-3 * mockAnnounceInterval).Format(time.RFC3339)
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host1"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetVal(fields)
				mockDeleteHostScript(mock, "host1").SetErr(errors.New("redis error"))
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "host that can not be parsed is reclaimed",
			mock: func(mock redismock.ClientMock) {
				fields := mockRawHostFields("host1")
				fields["port"] = mockInvalidFieldValue
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host1"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetVal(fields)
				mockDeleteHostScript(mock, "host1").SetVal(true)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "host id without host hash is reclaimed",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host1"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetVal(map[string]string{})
				mockDeleteHostScript(mock, "host1").SetVal(true)
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
		{
			name: "host that fails to load because of a redis error is kept",
			mock: func(mock redismock.ClientMock) {
				mock.ExpectSScan(pkgredis.MakePersistentCacheHostsInScheduler(1), 0, "*", 10).SetVal([]string{"host1"}, 0)
				mock.ExpectHGetAll(pkgredis.MakePersistentCacheHostKeyInScheduler(1, "host1")).SetErr(errors.New("redis error"))
			},
			expect: func(t *testing.T, err error) {
				assert := assert.New(t)
				assert.NoError(err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			rdb, mock := redismock.NewClientMock()
			tc.mock(mock)

			h := &hostManager{config: mockHostManagerConfig, rdb: rdb}
			tc.expect(t, h.RunGC(context.Background()))
			assert.NoError(mock.ExpectationsWereMet())
		})
	}
}
