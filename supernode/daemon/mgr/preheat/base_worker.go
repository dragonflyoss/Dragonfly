package preheat

import (
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

const TIMEOUT = 30 * 60;

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
