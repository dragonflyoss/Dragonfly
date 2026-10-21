/*
 *     Copyright 2020 The Dragonfly Authors
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

package job

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	v2 "d7y.io/api/v2/pkg/apis/common/v2"
	dfdaemonv2 "d7y.io/api/v2/pkg/apis/dfdaemon/v2"
)

// PreheatRequest defines the request parameters for preheating.
type PreheatRequest struct {
	// DEPRECATED: Use URLs instead for V2 preheating and URL is only used for V1 preheating.
	URL                         string            `json:"url" validate:"omitempty,url"`
	URLs                        []string          `json:"urls" validate:"omitempty"`
	PieceLength                 *uint64           `json:"pieceLength" binding:"omitempty,gte=4194304"`
	Tag                         string            `json:"tag" validate:"omitempty"`
	FilteredQueryParams         string            `json:"filtered_query_params" validate:"omitempty"`
	Headers                     map[string]string `json:"headers" validate:"omitempty"`
	Application                 string            `json:"application" validate:"omitempty"`
	Priority                    int32             `json:"priority" validate:"omitempty"`
	Scope                       string            `json:"scope" validate:"omitempty"`
	IPs                         []string          `json:"ips" validate:"omitempty"`
	Percentage                  *uint32           `json:"percentage" validate:"omitempty,gte=1,lte=100"`
	Count                       *uint32           `json:"count" validate:"omitempty,gte=1,lte=200"`
	ConcurrentTaskCount         int64             `json:"concurrent_task_count" validate:"omitempty"`
	ConcurrentPeerCount         int64             `json:"concurrent_peer_count" validate:"omitempty"`
	CertificateChain            [][]byte          `json:"certificate_chain" validate:"omitempty"`
	InsecureSkipVerify          bool              `json:"insecure_skip_verify" validate:"omitempty"`
	Timeout                     time.Duration     `json:"timeout" validate:"omitempty"`
	GroupUUID                   string            `json:"group_uuid" validate:"omitempty"`
	TaskUUID                    string            `json:"task_uuid" validate:"omitempty"`
	ObjectStorage               *v2.ObjectStorage `json:"object_storage" validate:"omitempty"`
	Hdfs                        *v2.HDFS          `json:"hdfs" validate:"omitempty"`
	OutputPath                  *string           `json:"output_path" validate:"omitempty"`
	EnableTaskIDBasedBlobDigest bool              `json:"enable_task_id_based_blob_digest" validate:"omitempty"`
}

// PreheatResponse defines the response parameters for preheating.
type PreheatResponse struct {
	SuccessTasks       []*PreheatSuccessTask `json:"success_tasks"`
	FailureTasks       []*PreheatFailureTask `json:"failure_tasks"`
	SchedulerClusterID uint                  `json:"scheduler_cluster_id"`
}

// PreheatSuccessTask defines the response parameters for preheating successfully.
type PreheatSuccessTask struct {
	URL      string `json:"url"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

// PreheatFailureTask defines the response parameters for preheating failed.
type PreheatFailureTask struct {
	URL         string `json:"url"`
	Hostname    string `json:"hostname"`
	IP          string `json:"ip"`
	Description string `json:"description"`
}

// GetTaskRequest defines the request parameters for getting task.
type GetTaskRequest struct {
	TaskID              string        `json:"task_id" validate:"required"`
	Timeout             time.Duration `json:"timeout" validate:"omitempty"`
	GroupUUID           string        `json:"group_uuid" validate:"omitempty"`
	TaskUUID            string        `json:"task_uuid" validate:"omitempty"`
	ConcurrentPeerCount int64         `json:"concurrent_peer_count" validate:"omitempty"`
	Scope               string        `json:"scope" validate:"omitempty"`
}

// GetTaskResponse defines the response parameters for getting task.
type GetTaskResponse struct {
	Peers              []*Peer `json:"peers"`
	SchedulerClusterID uint    `json:"scheduler_cluster_id"`
}

// ListTaskEntriesRequest defines the request parameters for listing task entries.
type ListTaskEntriesRequest struct {
	TaskID           string               `json:"task_id" validate:"required"`
	Url              string               `json:"url" validate:"omitempty"`
	Timeout          *durationpb.Duration `json:"timeout" validate:"omitempty"`
	Header           map[string]string    `json:"header" validate:"omitempty"`
	CertificateChain [][]byte             `json:"certificate_chain" validate:"omitempty"`
	ObjectStorage    *v2.ObjectStorage    `json:"object_storage" validate:"omitempty"`
	Hdfs             *v2.HDFS             `json:"hdfs" validate:"omitempty"`
}

// ListTaskEntriesResponse defines the response parameters for listing task entries.
type ListTaskEntriesResponse struct {
	Entries     []*dfdaemonv2.Entry `json:"entries"`
	Recursive   bool                `json:"recursive"`
	SchedulerID uint                `json:"scheduler_id"`
}

// Peer represents the peer information.
type Peer struct {
	ID         string    `json:"id"`
	Hostname   string    `json:"hostname"`
	IP         string    `json:"ip"`
	HostType   string    `json:"host_type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	IsFinished bool      `json:"IsFinished"`
}

// DeleteTaskRequest defines the request parameters for deleting task.
type DeleteTaskRequest struct {
	TaskID    string        `json:"task_id" validate:"required"`
	Timeout   time.Duration `json:"timeout" validate:"omitempty"`
	GroupUUID string        `json:"group_uuid" validate:"omitempty"`
	TaskUUID  string        `json:"task_uuid" validate:"omitempty"`
}

// DeleteTaskResponse defines the response parameters for deleting task.
type DeleteTaskResponse struct {
	SuccessTasks       []*DeleteSuccessTask `json:"success_tasks"`
	FailureTasks       []*DeleteFailureTask `json:"failure_tasks"`
	SchedulerClusterID uint                 `json:"scheduler_cluster_id"`
}

// DeleteSuccessTask defines the response parameters for deleting peer successfully.
type DeleteSuccessTask struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	HostType string `json:"host_type"`
}

// DeleteFailureTask defines the response parameters for deleting peer failed.
type DeleteFailureTask struct {
	Hostname    string `json:"hostname"`
	IP          string `json:"ip"`
	HostType    string `json:"host_type"`
	Description string `json:"description"`
}

// SyncPeerRequest defines the request parameters for sync peer task.
type SyncPeerRequest struct {
	GroupUUID      string `json:"group_uuid"`
	GroupTaskCount int    `json:"group_task_count"`
	TotalGroupTask int    `json:"total_group_task"`
}
