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

package seed

import (
	"math/rand"

	"github.com/dragonflyoss/Dragonfly/dfget/corev2/pattern/seed/config"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"

	"github.com/pkg/errors"
)

type taskStatePerNode struct {
	peerID string
	info   *config.TaskFetchInfo
}

type taskState struct {
	// key is peerID, value is taskStatePerNode
	peerContainer *syncmap.SyncMap
}

func newTaskState() *taskState {
	return &taskState{
		peerContainer: syncmap.NewSyncMap(),
	}
}

func (ts *taskState) add(peerID string, info *config.TaskFetchInfo) error {
	if stringutils.IsEmptyStr(peerID) {
		return errors.Wrap(errortypes.ErrEmptyValue, "peerID")
	}

	_, err := ts.peerContainer.Get(peerID)
	if err != nil && !errortypes.IsDataNotFound(err) {
		return err
	}

	item := &taskStatePerNode{
		peerID: peerID,
		info:   info,
	}

	return ts.peerContainer.Add(peerID, item)
}

// getPeersByLoad return the peers which satisfy the request, and order by load
// the number of peers should not more than maxCount;
func (ts *taskState) getPeersByLoad(maxCount int, filters map[string]map[string]bool) []*taskStatePerNode {
	result := []*taskStatePerNode{}

	ts.peerContainer.Range(func(key, value interface{}) bool {
		pn := value.(*taskStatePerNode)
		if filters != nil && FilterMatch(filters, "taskFetchInfo", "allowSeedDownload", "true") {
			if !pn.info.AllowSeedDownload {
				return true
			}
		}
		result = append(result, pn)
		return true
	})

	rand.Shuffle(len(result), func(i, j int) {
		tmp := result[j]
		result[j] = result[i]
		result[i] = tmp
	})

	if maxCount > len(result) {
		return result
	}

	return result[:maxCount]
}
