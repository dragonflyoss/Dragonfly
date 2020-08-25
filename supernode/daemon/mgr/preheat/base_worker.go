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
package preheat

import (
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

const TIMEOUT = 30 * 60

var _ IWorker = &BaseWorker{}

type IWorker interface{
	Run()
	Stop()
	query() chan error
	preRun() bool
	failed(errMsg string)
	afterRun()
}

type BaseWorker struct {
	Task *mgr.PreheatTask
	Preheater Preheater
	PreheatService *PreheatService
	stop *atomic.Value
	worker IWorker
}

func newBaseWorker(task *mgr.PreheatTask, preheater Preheater, preheatService *PreheatService) *BaseWorker {
	worker := &BaseWorker{
		Task: task,
		Preheater: preheater,
		PreheatService: preheatService,
		stop: new(atomic.Value),
	}
	worker.worker = worker
	return worker
}

func (w *BaseWorker) Run() {
	go func() {
		defer func(){
			e := recover()
			if e != nil {
				debug.PrintStack()
			}
		}()

		if w.worker.preRun() {
			timer := time.NewTimer(time.Second*TIMEOUT)
			ch := w.worker.query()
			select {
			case <-timer.C:
				w.worker.failed("timeout")
			case err := <-ch:
				if err != nil {
					w.worker.failed(err.Error())
				}
			}
		}
		w.worker.afterRun()
	}()
}

func (w *BaseWorker) Stop() {
	w.stop.Store(true)
}

func (w *BaseWorker) isRunning() bool {
	return w.stop.Load() == nil
}

func (w *BaseWorker) preRun() bool {
	panic("not implement")
}

func (w *BaseWorker) afterRun() {
	w.Preheater.Remove(w.Task.ID)
}

func (w *BaseWorker) query() chan error {
	panic("not implement")
}

func (w *BaseWorker) succeed() {
	w.Task.FinishTime = time.Now().UnixNano()/int64(time.Millisecond)
	w.Task.Status = types.PreheatStatusSUCCESS
	w.PreheatService.Update(w.Task.ID, w.Task)
}

func (w *BaseWorker) failed(errMsg string) {
	w.Task.FinishTime = time.Now().UnixNano()/int64(time.Millisecond)
	w.Task.Status = types.PreheatStatusFAILED
	w.Task.ErrorMsg = errMsg
	w.PreheatService.Update(w.Task.ID, w.Task)
}
