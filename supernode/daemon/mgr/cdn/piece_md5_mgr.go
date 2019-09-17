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

package cdn

import (
	"sort"
	"strconv"

	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/syncmap"
)

type pieceMD5Mgr struct {
	taskPieceMD5s *syncmap.SyncMap
}

func newpieceMD5Mgr() *pieceMD5Mgr {
	return &pieceMD5Mgr{
		taskPieceMD5s: syncmap.NewSyncMap(),
	}
}

// getPieceMD5 returns the md5 of pieceRange for taskID.
func (pmm *pieceMD5Mgr) getPieceMD5(taskID string, pieceNum int) (pieceMD5 string, err error) {
	pieceMD5s, err := pmm.taskPieceMD5s.GetAsMap(taskID)
	if err != nil {
		return "", err
	}

	return pieceMD5s.GetAsString(strconv.Itoa(pieceNum))
}

// setPieceMD5 sets the md5 for pieceRange of taskID.
func (pmm *pieceMD5Mgr) setPieceMD5(taskID string, pieceNum int, pieceMD5 string) (err error) {
	pieceMD5s, err := pmm.taskPieceMD5s.GetAsMap(taskID)
	if err != nil && !errortypes.IsDataNotFound(err) {
		return err
	}

	if pieceMD5s == nil {
		pieceMD5s = syncmap.NewSyncMap()
		pmm.taskPieceMD5s.Add(taskID, pieceMD5s)
	}

	return pieceMD5s.Add(strconv.Itoa(pieceNum), pieceMD5)
}

// getPieceMD5sByTaskID returns all pieceMD5s as a string slice.
func (pmm *pieceMD5Mgr) getPieceMD5sByTaskID(taskID string) (pieceMD5s []string, err error) {
	pieceMD5sMap, err := pmm.taskPieceMD5s.GetAsMap(taskID)
	if err != nil {
		return nil, err
	}
	pieceNums := pieceMD5sMap.ListKeyAsIntSlice()
	sort.Ints(pieceNums)

	for i := 0; i < len(pieceNums); i++ {
		pieceMD5, err := pieceMD5sMap.GetAsString(strconv.Itoa(pieceNums[i]))
		if err != nil {
			return nil, err
		}
		pieceMD5s = append(pieceMD5s, pieceMD5)
	}
	return pieceMD5s, nil
}

func (pmm *pieceMD5Mgr) removePieceMD5sByTaskID(taskID string) error {
	return pmm.taskPieceMD5s.Remove(taskID)
}
