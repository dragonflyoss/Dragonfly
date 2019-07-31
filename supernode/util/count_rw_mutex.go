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

package util

import (
	"sync"

	"github.com/dragonflyoss/Dragonfly/pkg/atomiccount"
)

type countRWMutex struct {
	count *atomiccount.AtomicInt
	sync.RWMutex
}

func newCountRWMutex() *countRWMutex {
	return &countRWMutex{
		count: atomiccount.NewAtomicInt(0),
	}
}

func (cr *countRWMutex) reset() {
	cr.count.Set(0)
}

func (cr *countRWMutex) increaseCount() int32 {
	cr.count.Add(1)
	return cr.count.Get()
}

func (cr *countRWMutex) decreaseCount() int32 {
	cr.count.Add(-1)
	return cr.count.Get()
}

func (cr *countRWMutex) lock(ro bool) {
	if ro {
		cr.RLock()
		return
	}
	cr.Lock()
}

func (cr *countRWMutex) unlock(ro bool) {
	if ro {
		cr.RUnlock()
		return
	}
	cr.Unlock()
}
