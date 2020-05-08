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

package mgr

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/apis/types"
)

// PreheatTask stores the detailed preheat task information.
type PreheatTask struct {
	ID         string
	URL        string
	Type       string
	Filter     string
	Identifier string
	Headers    map[string]string

	// ParentID records its parent preheat task id. Sometimes the current
	// preheat task is not created by user directly. Such as preheating an
	// image, it contains several layers that should be preheated together.
	// So the image preheat task is the parent of its layer preheat tasks.
	ParentID string
	Children []string

	Status     types.PreheatStatus
	StartTime  int64
	FinishTime int64
	ErrorMsg   string
}

// PreheatManager provides basic operations of preheat.
type PreheatManager interface {
	// Create creates a preheat task to cache data in supernode, thus accelerating the
	// p2p downloading.
	Create(ctx context.Context, task *types.PreheatCreateRequest) (preheatID string, err error)

	// Get gets detailed preheat task information by preheatID.
	Get(ctx context.Context, preheatID string) (preheatTask *PreheatTask, err error)

	// Delete deletes a preheat task by preheatID.
	Delete(ctx context.Context, preheatID string) (err error)

	// GetAll gets all preheat tasks that unexpired.
	GetAll(ctx context.Context) (preheatTask []*PreheatTask, err error)
}
