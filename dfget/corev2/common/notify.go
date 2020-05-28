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

package common

import (
	"fmt"
	"sync"

	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
)

// NewNotify creates Notify which implements basic.Notify.
func NewNotify() *Notify {
	return &Notify{}
}

// Notify is an implementation of basic.Notify.
type Notify struct {
	sync.RWMutex
	done   chan struct{}
	result basic.NotifyResult
}

func (notify *Notify) Done() <-chan struct{} {
	return notify.done
}

// Result returns the NotifyResult and only valid after Done channel is closed.
func (notify *Notify) Result() basic.NotifyResult {
	notify.RLock()
	defer notify.RUnlock()

	return notify.result
}

// Finish sets result and close done channel to notify work done.
func (notify *Notify) Finish(result basic.NotifyResult) error {
	notify.Lock()
	defer notify.Unlock()

	if notify.result != nil {
		return fmt.Errorf("result have been set once")
	}

	notify.result = result
	close(notify.done)
	return nil
}

// NotifyResult is an implementation of basic.NotifyResult.
type notifyResult struct {
	success bool
	err     error
	data    interface{}
}

func NewNotifyResult(success bool, err error, data interface{}) basic.NotifyResult {
	return &notifyResult{
		success: success,
		err:     err,
		data:    data,
	}
}

func (nr notifyResult) Success() bool {
	return nr.success
}

func (nr notifyResult) Error() error {
	return nr.err
}

func (nr notifyResult) Data() interface{} {
	return nr.data
}
