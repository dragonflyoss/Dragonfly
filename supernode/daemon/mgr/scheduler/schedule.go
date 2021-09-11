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

package scheduler

import (
	"context"
	"fmt"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

// Schedule is a wrapper of the schedule which implements the interface of ScheduleMgr.
type Schedule struct {
	// name is a unique identifier, you can also name it ID.
	driverName string
	// config is used to init schedule driver.
	config interface{}
	// driver holds a schedule which implements the interface of ScheduleDriver.
	driver mgr.SchedulerMgr
}

// NewSchedule creates a new Schedule instance.
func NewSchedule(name string, builder ScheduleBuilder, cfg *config.Config, progress mgr.ProgressMgr) (*Schedule, error) {
	if name == "" || builder == nil {
		return nil, fmt.Errorf("plugin name or builder cannot be nil")
	}

	// init driver with specific config
	driver, err := builder(cfg, progress)
	if err != nil {
		return nil, fmt.Errorf("failed to init schedule driver %s: %v", name, err)
	}

	return &Schedule{
		driverName: name,
		config:     cfg,
		driver:     driver,
	}, nil
}

// Type returns the plugin type: SchedulePlugin.
func (s *Schedule) Type() config.PluginType {
	return config.SchedulerPlugin
}

// Name returns the plugin name.
func (s *Schedule) Name() string {
	return s.driverName
}

// Schedule gets scheduler result with specified taskID, clientID and peerID through some rules.
func (s *Schedule) Schedule(ctx context.Context, taskID, clientID, peerID string) ([]*mgr.PieceResult, error) {
	return s.driver.Schedule(ctx, taskID, clientID, peerID)
}
