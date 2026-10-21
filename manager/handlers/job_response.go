/*
 *     Copyright 2026 The Dragonfly Authors
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

package handlers

import "d7y.io/dragonfly/v2/manager/models"

var (
	jobArgsSecretKeys = map[string]struct{}{
		"password": {},
		"headers":  {},
	}

	objectStorageSecretKeys = map[string]struct{}{
		"access_key_id":     {},
		"access_key_secret": {},
		"session_token":     {},
		"security_token":    {},
	}

	hdfsSecretKeys = map[string]struct{}{
		"delegation_token": {},
	}
)

func sanitizeJobForResponse(job *models.Job) models.Job {
	if job == nil {
		return models.Job{}
	}

	sanitized := *job
	sanitized.Args = sanitizeJobArgs(job.Args)
	return sanitized
}

func sanitizeJobsForResponse(jobs []models.Job) []models.Job {
	sanitized := make([]models.Job, 0, len(jobs))
	for i := range jobs {
		sanitized = append(sanitized, sanitizeJobForResponse(&jobs[i]))
	}

	return sanitized
}

func sanitizeJobArgs(args models.JSONMap) models.JSONMap {
	if args == nil {
		return nil
	}

	sanitized := make(models.JSONMap, len(args))
	for key, value := range args {
		if _, ok := jobArgsSecretKeys[key]; ok {
			continue
		}

		switch key {
		case "object_storage":
			sanitized[key] = sanitizeNestedMap(value, objectStorageSecretKeys)
		case "hdfs":
			sanitized[key] = sanitizeNestedMap(value, hdfsSecretKeys)
		default:
			sanitized[key] = value
		}
	}

	return sanitized
}

func sanitizeNestedMap(value any, secretKeys map[string]struct{}) any {
	nested, ok := value.(map[string]any)
	if !ok {
		return value
	}

	sanitized := make(map[string]any, len(nested))
	for key, nestedValue := range nested {
		if _, ok := secretKeys[key]; ok {
			continue
		}

		sanitized[key] = nestedValue
	}

	return sanitized
}
