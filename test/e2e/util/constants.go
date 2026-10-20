/*
 *     Copyright 2024 The Dragonfly Authors
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

const (
	DragonflyNamespace = "dragonfly-system"
)

const (
	ManagerServerName    = "manager"
	SchedulerServerName  = "scheduler"
	SeedClientServerName = "seed-client"
	ClientServerName     = "client"
)

const (
	// SchedulerStatefulSetName is the name of the scheduler statefulset installed
	// by the dragonfly chart.
	SchedulerStatefulSetName = "dragonfly-scheduler"
)

type server struct {
	Name       string
	Namespace  string
	LogDirName string
	Replicas   int
}

var Servers = map[string]server{
	ManagerServerName: {
		Name:       ManagerServerName,
		Namespace:  DragonflyNamespace,
		LogDirName: ManagerServerName,
		Replicas:   1,
	},
	SchedulerServerName: {
		Name:       SchedulerServerName,
		Namespace:  DragonflyNamespace,
		LogDirName: SchedulerServerName,
		Replicas:   3,
	},
	SeedClientServerName: {
		Name:       SeedClientServerName,
		Namespace:  DragonflyNamespace,
		LogDirName: "dfdaemon",
		Replicas:   3,
	},
	ClientServerName: {
		Name:       ClientServerName,
		Namespace:  DragonflyNamespace,
		LogDirName: "dfdaemon",
		Replicas:   2,
	},
}
