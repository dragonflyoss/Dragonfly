package preheat

import (
	"github.com/dragonflyoss/Dragonfly/supernode/daemon/mgr"
)

type Preheater interface {
	/**
	 * The type of this preheater
	 */
	Type() string

	/**
	 * Create a worker to preheat the task.
	 */
	NewWorker(task *mgr.PreheatTask , service *PreheatService ) IWorker

	/**
	 * cancel the running task
	 */
	Cancel(id string)

	/**
	 * remove a running preheat task
	 */
	Remove(id string)
}

func GetPreheater(typ string) Preheater {
	return preheaterMap[typ]
}

func RegisterPreheater(typ string, preheater Preheater) {
	preheaterMap[typ] = preheater
}

var preheaterMap = make(map[string]Preheater)