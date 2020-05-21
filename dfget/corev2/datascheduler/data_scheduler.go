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

package datascheduler

import (
	"context"

	"github.com/dragonflyoss/Dragonfly/dfget/corev2/basic"
)

// SchedulerResult defines the schedule result of request range.
// For some implementation, developer could do more than one schedule for the same request range.
type SchedulerResult interface {
	// Result get the schedule result for range data which may not include all data of request range.
	Result() []*basic.SchedulePieceDataResult

	// State gets the temporary states of this schedule which binds to range request.
	State() ScheduleState
}

// ScheduleState defines the state of this schedule.
type ScheduleState interface {
	// Continue tells user if reschedule the request range again.
	Continue() bool
}

// DataScheduler defines how to schedule peers for range request.
type DataScheduler interface {
	// state should be got from SchedulerResult which is got from last caller for the same range request.
	Schedule(ctx context.Context, rr basic.RangeRequest, state ScheduleState) (SchedulerResult, error)
}
