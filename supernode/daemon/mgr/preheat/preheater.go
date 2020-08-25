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