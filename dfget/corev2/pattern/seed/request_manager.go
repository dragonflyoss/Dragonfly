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
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/queue"

	"github.com/pkg/errors"
)

// requestManager manage the recent the requests, it provides the
type requestManager struct {
	q *queue.LRUQueue
}

func newRequestManager() *requestManager {
	return &requestManager{
		q: queue.NewLRUQueue(100),
	}
}

func (rm *requestManager) addRequest(url string) error {
	data, err := rm.q.GetItemByKey(url)
	if err != nil && err != errortypes.ErrDataNotFound {
		return err
	}

	var rs *requestState = nil
	if err == errortypes.ErrDataNotFound {
		rs = newRequestState(url)
	} else {
		ors, ok := data.(*requestState)
		if !ok {
			return errors.Wrapf(errortypes.ErrConvertFailed, "value: %v", data)
		}

		rs = ors.copy()
		rs.updateRecentTime()
	}

	rs.updateRecentTime()
	rm.q.Put(url, rs)

	return nil
}

// getRecentRequest will return 50 of the recent request url
func (rm *requestManager) getRecentRequest(count int) []string {
	if count == 0 {
		count = 50
	}
	arr := rm.q.GetFront(count)
	result := []string{}

	for _, d := range arr {
		if rs, ok := d.(*requestState); ok {
			result = append(result, rs.url)
		}
	}

	return result
}

func (rm *requestManager) getRequestState(url string) (*requestState, error) {
	data, err := rm.q.GetItemByKey(url)
	if err != nil {
		return nil, err
	}

	if rs, ok := data.(*requestState); ok {
		return rs, nil
	}

	return nil, errors.Wrapf(errortypes.ErrConvertFailed, "value: %v", data)
}
