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
	"fmt"
	"sync"

	"github.com/dragonflyoss/Dragonfly/supernode/config"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
	"github.com/dragonflyoss/Dragonfly/supernode/plugins"

	"gopkg.in/yaml.v2"
)

// ScheduleBuilder is a function that creates a new schedule plugin instant
// with the giving conf.
type ScheduleBuilder func(conf *config.Config, progressMgr mgr.ProgressMgr) (mgr.SchedulerMgr, error)

// Register defines an interface to register a driver with specified name.
// All drivers should call this function to register itself to the driverFactory.
func Register(name string, builder ScheduleBuilder) {
	var f plugins.Builder = func(conf plugins.BuilderConfig) (plugin plugins.Plugin, e error) {
		pro, ok := conf.Other.(mgr.ProgressMgr)
		if !ok {
			return nil, fmt.Errorf("get progress error: no progress input")
		}
		stringToConf := &config.Config{}
		if err := yaml.Unmarshal([]byte(conf.Conf), stringToConf); err != nil {
			return nil, fmt.Errorf("failed to parse config: %v", err)
		}
		return NewSchedule(name, builder, stringToConf, pro)
	}
	plugins.RegisterPlugin(config.SchedulerPlugin, name, f)
}

// Manager manages schedule.
type ScheduleDriverManager struct {
	cfg         *config.Config
	progressMgr mgr.ProgressMgr

	defaultSchedule *Schedule
	mutex           sync.Mutex
}

// NewManager creates a schedule manager.
func NewScheduleDriverManager(cfg *config.Config, progressMgr mgr.ProgressMgr) (*ScheduleDriverManager, error) {
	return &ScheduleDriverManager{
		cfg:         cfg,
		progressMgr: progressMgr,
	}, nil
}

// Get a schedule from manager with specified name.
func (sm *ScheduleDriverManager) Get(name string) (*Schedule, error) {
	v := plugins.GetPlugin(config.SchedulerPlugin, name)
	if v == nil {
		if name == LocalScheduleDriver {
			return sm.getDefaultSchedule()
		}
		return nil, fmt.Errorf("not existed schedule: %s", name)
	}
	if schedule, ok := v.(*Schedule); ok {
		return schedule, nil
	}
	return nil, fmt.Errorf("get schedule error: unknown reason")
}

func (sm *ScheduleDriverManager) getDefaultSchedule() (*Schedule, error) {
	if sm.defaultSchedule != nil {
		return sm.defaultSchedule, nil
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// check again to avoid initializing repeatedly
	if sm.defaultSchedule != nil {
		return sm.defaultSchedule, nil
	}

	if sm.cfg == nil {
		return nil, fmt.Errorf("cannot init local schedule without cfg")
	}
	if sm.progressMgr == nil {
		return nil, fmt.Errorf("cannot init local schedule without progressMgr")
	}

	s, err := NewSchedule(LocalScheduleDriver, NewLocalManager, sm.cfg, sm.progressMgr)
	if err != nil {
		return nil, err
	}
	sm.defaultSchedule = s
	return sm.defaultSchedule, nil
}
