package preheat

import (
	"sync"

	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

var _ Preheater = &BasePreheater{}

type BasePreheater struct {}

/**
 * The type of this preheater
 */
func (p *BasePreheater) Type() string {
	panic("not implement")
}

/**
 * Create a worker to preheat the task.
 */
func (p *BasePreheater) NewWorker(task *mgr.PreheatTask , service *PreheatService) IWorker {
	panic("not implement")
}

/**
 * cancel the running task
 */
func (p *BasePreheater) Cancel(id string) {
	woker, ok := workerMap.Load(id)
	if !ok {
		return
	}
	woker.(IWorker).Stop()
}

/**
 * remove a running preheat task
 */
func (p *BasePreheater) Remove(id string) {
	p.Cancel(id)
	workerMap.Delete(id)
}

/**
 * add a worker to workerMap.
 */
func (p *BasePreheater) addWorker(id string, worker IWorker) {
	workerMap.Store(id, worker)
}

var workerMap = new(sync.Map)