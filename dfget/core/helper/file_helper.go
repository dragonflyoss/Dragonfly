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

package helper

import (
	"path/filepath"
	"strings"
)

// GetTaskFile returns file path of task file.
func GetTaskFile(taskFileName, dataDir string) string {
	return filepath.Join(dataDir, taskFileName)
}

// GetServiceFile returns file path of service file.
func GetServiceFile(taskFileName, dataDir string) string {
	return GetTaskFile(taskFileName, dataDir) + ".service"
}

// GetTaskName extracts and returns task name from serviceFile.
func GetTaskName(serviceFile string) string {
	if idx := strings.LastIndex(serviceFile, ".service"); idx != -1 {
		return serviceFile[:idx]
	}
	return serviceFile
}
