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
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

func init() {
	RegisterPreheater("file", &FilePreheat{BasePreheater:new(BasePreheater)})
	logrus.StandardLogger().SetLevel(logrus.DebugLevel)
}

type FilePreheat struct {
	*BasePreheater
}

func (p *FilePreheat) Type() string {
	return "file"
}

/**
 * Create a worker to preheat the task.
 */
func (p *FilePreheat) NewWorker(task *mgr.PreheatTask , service *PreheatService) IWorker {
	worker := &FileWorker{BaseWorker: newBaseWorker(task, p, service)}
	worker.worker = worker
	p.addWorker(task.ID, worker)
	return worker
}

type FileWorker struct {
	*BaseWorker
	progress *PreheatProgress
}

func (w *FileWorker) preRun() bool {
	w.Task.Status = types.PreheatStatusRUNNING
	w.PreheatService.Update(w.Task.ID, w.Task)
	var err error
	w.progress, err = w.PreheatService.ExecutePreheat(w.Task)
	if err != nil {
		w.failed(err.Error())
		return false
	}
	return true
}

func (w *FileWorker) afterRun() {
	if w.progress != nil {
		w.progress.cmd.Process.Kill()
	}
	w.BaseWorker.afterRun()
}

func (w *FileWorker) query() chan error {
	result := make(chan error, 1)
	go func(){
		time.Sleep(time.Second*2)
		for w.isRunning() {
			if w.Task.FinishTime > 0 {
				w.Preheater.Cancel(w.Task.ID)
				return
			}
			if w.progress == nil {
				w.succeed()
				return
			}
			status := w.progress.cmd.ProcessState
			if status != nil && status.Exited() {
				if !status.Success() {
					errMsg := fmt.Sprintf("dfget failed: %s err: %s",  status.String(), w.progress.errmsg.String())
					w.failed(errMsg)
					w.Preheater.Cancel(w.Task.ID)
					result <- errors.New(errMsg)
					return
				} else {
					w.succeed()
					w.Preheater.Cancel(w.Task.ID)
					result <- nil
					return
				}
			}

			time.Sleep(time.Second*10)
		}
	}()
	return result
}

