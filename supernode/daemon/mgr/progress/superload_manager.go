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

package progress

import (
	"context"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
)

const (
	// renewInterval is the interval time to check the superload.
	renewInterval = 2 * time.Second

	// renewDelayTime if the superload has not been changed after renewDelayTime, it should be renewed.
	renewDelayTime = 30 * time.Second
)

// UpdateSuperLoad updates the superload of taskID by adding the delta.
// The updated will be `false` if failed to do update operation.
//
// It's considered as a failure when then superload is greater than limit after adding delta.
func (pm *Manager) UpdateSuperLoad(ctx context.Context, taskID string, delta, limit int32) (updated bool, err error) {
	v, _ := pm.superLoad.LoadOrStore(taskID, newSuperLoadState())
	loadState, ok := v.(*superLoadState)
	if !ok {
		return false, errortypes.ErrConvertFailed
	}

	if loadState.loadValue.Add(delta) > limit && limit > 0 {
		loadState.loadValue.Add(-delta)
		return false, nil
	}
	loadState.loadModTime = time.Now()

	return true, nil
}

// startMonitorSuperLoad starts a new goroutine to check the superload periodically and
// reset the superload to zero if there is no update for a long time for one task to
// avoid being occupied when supernode doesn't receive the message from peers that downloading piece from supernode for a variety of reasons.
//
func (pm *Manager) startMonitorSuperLoad() {
	go func() {
		ticker := time.NewTicker(renewInterval)
		for range ticker.C {
			pm.renewSuperLoad()
		}
	}()
}

func (pm *Manager) renewSuperLoad() {
	rangeFunc := func(key, value interface{}) bool {
		if v, ok := value.(*superLoadState); ok {
			if time.Since(v.loadModTime) > renewDelayTime {
				v.loadValue.Set(0)
			}
		}
		return true
	}

	pm.superLoad.Range(rangeFunc)
}
