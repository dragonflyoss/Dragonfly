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

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"d7y.io/dragonfly/v2/manager/middlewares"
	"d7y.io/dragonfly/v2/manager/models"
	"d7y.io/dragonfly/v2/manager/service/mocks"
	"d7y.io/dragonfly/v2/manager/types"
)

var (
	mockPreheatJobReqBody = `
		{
			"type": "preheat",
			"user_id": 4,
			"bio": "bio"
		}`

	mockGetTaskJobReqBody = `
		{
			"type": "get_task",
			"user_id": 4,
			"bio": "bio",
			"args": {
				"task_id": "7575d21d69495905a4709bf4e10d0e5cffcf7fd1e76e93171e0ef6e0abcf07a8"
			}
		}`

	mockDeleteTaskJobReqBody = `
		{
			"type": "delete_task",
			"user_id": 4,
			"bio": "bio",
			"args": {
				"task_id": "04a29122b0c4d0affde2d577fb36bb956caa3da10e9130375623c24a5f865a49"
			}
		}`

	mockOtherJobReqBody = `
		{
			"type": "others",
			"user_id": 4,
			"bio": "bio"
		}`

	mockPreheatJobWithInvalidArgsReqBody = `
		{
			"type": "preheat",
			"user_id": 4,
			"bio": "bio",
			"args": {
				"type": "others"
			}
		}`

	mockSyncPeersJobReqBody = `
		{
			"type": "sync_peers",
			"user_id": 4,
			"bio": "bio",
			"scheduler_cluster_ids": [1]
		}`

	mockGetTaskJobWithoutArgsReqBody = `
		{
			"type": "get_task",
			"user_id": 4,
			"bio": "bio"
		}`

	mockGetImageDistributionJobReqBody = `
		{
			"type": "get_image_distribution",
			"user_id": 4,
			"bio": "bio",
			"args": {
				"url": "http://example.com/v2/foo/manifests/latest"
			}
		}`

	mockGetImageDistributionJobWithoutURLReqBody = `
		{
			"type": "get_image_distribution",
			"user_id": 4,
			"bio": "bio",
			"args": {
				"tag": "foo"
			}
		}`

	mockDeleteTaskJobWithoutArgsReqBody = `
		{
			"type": "delete_task",
			"user_id": 4,
			"bio": "bio"
		}`

	mockGCJobReqBody = `
		{
			"type": "gc",
			"user_id": 4,
			"bio": "bio",
			"args": {
				"type": "audit"
			}
		}`

	mockGCJobWithInvalidArgsReqBody = `
		{
			"type": "gc",
			"user_id": 4,
			"bio": "bio",
			"args": {
				"type": "others"
			}
		}`

	mockPreheatCreateJobRequest = types.CreatePreheatJobRequest{
		UserID: 4,
		Type:   "preheat",
		BIO:    "bio",
	}

	mockCreateGetTaskJobRequest = types.CreateGetTaskJobRequest{
		UserID: 4,
		Type:   "get_task",
		BIO:    "bio",
		Args:   types.GetTaskArgs{TaskID: "7575d21d69495905a4709bf4e10d0e5cffcf7fd1e76e93171e0ef6e0abcf07a8"},
	}

	mockCreateDeleteTaskJobRequest = types.CreateDeleteTaskJobRequest{
		UserID: 4,
		Type:   "delete_task",
		BIO:    "bio",
		Args: types.DeleteTaskArgs{
			TaskID: "04a29122b0c4d0affde2d577fb36bb956caa3da10e9130375623c24a5f865a49",
		},
	}

	mockCreateSyncPeersJobRequest = types.CreateSyncPeersJobRequest{
		UserID:              4,
		Type:                "sync_peers",
		BIO:                 "bio",
		SchedulerClusterIDs: []uint{1},
	}

	mockCreateGetImageDistributionJobRequest = types.CreateGetImageDistributionJobRequest{
		UserID: 4,
		Type:   "get_image_distribution",
		BIO:    "bio",
		Args:   types.GetImageDistributionArgs{URL: "http://example.com/v2/foo/manifests/latest"},
	}

	mockCreateGCJobRequest = types.CreateGCJobRequest{
		UserID: 4,
		Type:   "gc",
		BIO:    "bio",
		Args:   types.GCArgs{Type: "audit"},
	}

	mockUpdateJobRequest = types.UpdateJobRequest{
		UserID: 4,
		BIO:    "bio",
	}

	mockPreheatJobModel = &models.Job{
		BaseModel: mockBaseModel,
		UserID:    4,
		Type:      "preheat",
		BIO:       "bio",
		TaskID:    "dec6fe878785cea844dcecdf2ea25e19156822201016455733e47e9f0bfab563",
		Args: models.JSONMap{
			"url":      "https://registry.example.com/v2/library/busybox/manifests/latest",
			"username": "robot$dragonfly",
			"password": "super-secret",
			"headers": map[string]any{
				"Authorization": "Bearer abc123",
				"Accept":        "application/json",
			},
			"object_storage": map[string]any{
				"endpoint":          "https://s3.example.com",
				"access_key_id":     "access-key",
				"access_key_secret": "secret-key",
				"session_token":     "session-token",
				"security_token":    "security-token",
			},
			"hdfs": map[string]any{
				"delegation_token": "delegation-token",
			},
		},
	}
	mockSanitizedPreheatJobModel = &models.Job{
		BaseModel: mockBaseModel,
		UserID:    4,
		Type:      "preheat",
		BIO:       "bio",
		TaskID:    "dec6fe878785cea844dcecdf2ea25e19156822201016455733e47e9f0bfab563",
		Args: models.JSONMap{
			"url":      "https://registry.example.com/v2/library/busybox/manifests/latest",
			"username": "robot$dragonfly",
			"object_storage": map[string]any{
				"endpoint": "https://s3.example.com",
			},
			"hdfs": map[string]any{},
		},
	}

	mockGetTaskJobModel = &models.Job{
		BaseModel: mockBaseModel,
		UserID:    4,
		Type:      "get_task",
		BIO:       "bio",
		TaskID:    "7575d21d69495905a4709bf4e10d0e5cffcf7fd1e76e93171e0ef6e0abcf07a8",
	}

	mockDeleteTaskJobModel = &models.Job{
		BaseModel: mockBaseModel,
		UserID:    4,
		Type:      "delete_task",
		BIO:       "bio",
		TaskID:    "04a29122b0c4d0affde2d577fb36bb956caa3da10e9130375623c24a5f865a49",
	}

	mockGCJobModel = &models.Job{
		BaseModel: mockBaseModel,
		UserID:    4,
		Type:      "gc",
		BIO:       "bio",
	}

	mockGetImageDistributionJobResponse = &types.CreateGetImageDistributionJobResponse{
		Image: types.Image{
			Layers: []types.Layer{{URL: "http://example.com/v2/foo/blobs/sha256:foo"}},
		},
		Peers: []types.Peer{
			{
				IP:                 "127.0.0.1",
				Hostname:           "foo",
				CachedLayers:       []types.Layer{{URL: "http://example.com/v2/foo/blobs/sha256:foo"}},
				SchedulerClusterID: 1,
			},
		},
	}
)

func mockJobRouter(h *Handlers) *gin.Engine {
	r := gin.Default()
	r.Use(middlewares.Error())
	oapiv1 := r.Group("/oapi/v1")
	ojob := oapiv1.Group("/jobs")
	ojob.POST("", h.CreateJob)
	ojob.DELETE(":id", h.DestroyJob)
	ojob.PATCH(":id", h.UpdateJob)
	ojob.GET(":id", h.GetJob)
	ojob.GET("", h.GetJobs)
	return r
}

func TestHandlers_CreateJob(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		mock   func(ms *mocks.MockServiceMockRecorder)
		expect func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "unprocessable entity by body",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "unprocessable entity by type",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockOtherJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "unprocessable entity by preheat job args",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockPreheatJobWithInvalidArgsReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "create preheat job success",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockPreheatJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreatePreheatJob(gomock.Any(), gomock.Eq(mockPreheatCreateJobRequest)).Return(mockPreheatJobModel, nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				job := models.Job{}
				err := json.Unmarshal(w.Body.Bytes(), &job)
				assert.NoError(err)
				assert.Equal(mockSanitizedPreheatJobModel, &job)
			},
		},
		{
			name: "create preheat job internal server error",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockPreheatJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreatePreheatJob(gomock.Any(), gomock.Eq(mockPreheatCreateJobRequest)).Return(nil, errMockService).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "create sync peers job success",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockSyncPeersJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateSyncPeersJob(gomock.Any(), gomock.Eq(mockCreateSyncPeersJobRequest)).Return(nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				assert.Equal(`"OK"`, w.Body.String())
			},
		},
		{
			name: "create sync peers job internal server error",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockSyncPeersJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateSyncPeersJob(gomock.Any(), gomock.Eq(mockCreateSyncPeersJobRequest)).Return(errMockService).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "unprocessable entity by get task job args",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGetTaskJobWithoutArgsReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "create get task job success",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGetTaskJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateGetTaskJob(gomock.Any(), gomock.Eq(mockCreateGetTaskJobRequest)).Return(mockGetTaskJobModel, nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				job := models.Job{}
				err := json.Unmarshal(w.Body.Bytes(), &job)
				assert.NoError(err)
				assert.Equal(mockGetTaskJobModel, &job)
			},
		},
		{
			name: "create get task job internal server error",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGetTaskJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateGetTaskJob(gomock.Any(), gomock.Eq(mockCreateGetTaskJobRequest)).Return(nil, errMockService).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "unprocessable entity by get image distribution job args",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGetImageDistributionJobWithoutURLReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "create get image distribution job success",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGetImageDistributionJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateGetImageDistributionJob(gomock.Any(), gomock.Eq(mockCreateGetImageDistributionJobRequest)).Return(mockGetImageDistributionJobResponse, nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				resp := types.CreateGetImageDistributionJobResponse{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(err)
				assert.Equal(mockGetImageDistributionJobResponse, &resp)
			},
		},
		{
			name: "create get image distribution job internal server error",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGetImageDistributionJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateGetImageDistributionJob(gomock.Any(), gomock.Eq(mockCreateGetImageDistributionJobRequest)).Return(nil, errMockService).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "unprocessable entity by delete task job args",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockDeleteTaskJobWithoutArgsReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "create delete task job success",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockDeleteTaskJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateDeleteTaskJob(gomock.Any(), gomock.Eq(mockCreateDeleteTaskJobRequest)).Return(mockDeleteTaskJobModel, nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				job := models.Job{}
				err := json.Unmarshal(w.Body.Bytes(), &job)
				assert.NoError(err)
				assert.Equal(mockDeleteTaskJobModel, &job)
			},
		},
		{
			name: "create delete task job internal server error",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockDeleteTaskJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateDeleteTaskJob(gomock.Any(), gomock.Eq(mockCreateDeleteTaskJobRequest)).Return(nil, errMockService).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "unprocessable entity by gc job args",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGCJobWithInvalidArgsReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "create gc job success",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGCJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateGCJob(gomock.Any(), gomock.Eq(mockCreateGCJobRequest)).Return(mockGCJobModel, nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				job := models.Job{}
				err := json.Unmarshal(w.Body.Bytes(), &job)
				assert.NoError(err)
				assert.Equal(mockGCJobModel, &job)
			},
		},
		{
			name: "create gc job internal server error",
			req:  httptest.NewRequest(http.MethodPost, "/oapi/v1/jobs", strings.NewReader(mockGCJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.CreateGCJob(gomock.Any(), gomock.Eq(mockCreateGCJobRequest)).Return(nil, errMockService).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusInternalServerError, w.Code)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			svc := mocks.NewMockService(ctl)
			w := httptest.NewRecorder()
			h := New(svc)
			mockRouter := mockJobRouter(h)

			tc.mock(svc.EXPECT())
			mockRouter.ServeHTTP(w, tc.req)
			tc.expect(t, w)
		})
	}
}

func TestHandlers_DestroyJob(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		mock   func(ms *mocks.MockServiceMockRecorder)
		expect func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "unprocessable entity",
			req:  httptest.NewRequest(http.MethodDelete, "/oapi/v1/jobs/test", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "success",
			req:  httptest.NewRequest(http.MethodDelete, "/oapi/v1/jobs/2", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.DestroyJob(gomock.Any(), gomock.Eq(uint(2))).Return(nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			svc := mocks.NewMockService(ctl)
			w := httptest.NewRecorder()
			h := New(svc)
			mockRouter := mockJobRouter(h)

			tc.mock(svc.EXPECT())
			mockRouter.ServeHTTP(w, tc.req)
			tc.expect(t, w)
		})
	}
}

func TestHandlers_UpdateJob(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		mock   func(ms *mocks.MockServiceMockRecorder)
		expect func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "unprocessable entity caused by uri",
			req:  httptest.NewRequest(http.MethodPatch, "/oapi/v1/jobs/test", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "unprocessable entity caused by body",
			req:  httptest.NewRequest(http.MethodPatch, "/oapi/v1/jobs/2", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "success",
			req:  httptest.NewRequest(http.MethodPatch, "/oapi/v1/jobs/2", strings.NewReader(mockPreheatJobReqBody)),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.UpdateJob(gomock.Any(), gomock.Eq(uint(2)), gomock.Eq(mockUpdateJobRequest)).Return(mockPreheatJobModel, nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				job := models.Job{}
				err := json.Unmarshal(w.Body.Bytes(), &job)
				assert.NoError(err)
				assert.Equal(mockSanitizedPreheatJobModel, &job)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			svc := mocks.NewMockService(ctl)
			w := httptest.NewRecorder()
			h := New(svc)
			mockRouter := mockJobRouter(h)

			tc.mock(svc.EXPECT())
			mockRouter.ServeHTTP(w, tc.req)
			tc.expect(t, w)
		})
	}
}

func TestHandlers_GetJob(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		mock   func(ms *mocks.MockServiceMockRecorder)
		expect func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "unprocessable entity",
			req:  httptest.NewRequest(http.MethodGet, "/oapi/v1/jobs/test", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "success",
			req:  httptest.NewRequest(http.MethodGet, "/oapi/v1/jobs/2", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.GetJob(gomock.Any(), gomock.Eq(uint(2))).Return(mockPreheatJobModel, nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				job := models.Job{}
				err := json.Unmarshal(w.Body.Bytes(), &job)
				assert.NoError(err)
				assert.Equal(mockSanitizedPreheatJobModel, &job)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			svc := mocks.NewMockService(ctl)
			w := httptest.NewRecorder()
			h := New(svc)
			mockRouter := mockJobRouter(h)

			tc.mock(svc.EXPECT())
			mockRouter.ServeHTTP(w, tc.req)
			tc.expect(t, w)
		})
	}
}

func TestHandlers_GetJobs(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		mock   func(ms *mocks.MockServiceMockRecorder)
		expect func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "unprocessable entity",
			req:  httptest.NewRequest(http.MethodGet, "/oapi/v1/jobs?page=-1", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusUnprocessableEntity, w.Code)
			},
		},
		{
			name: "success",
			req:  httptest.NewRequest(http.MethodGet, "/oapi/v1/jobs?user_id=4", nil),
			mock: func(ms *mocks.MockServiceMockRecorder) {
				ms.GetJobs(gomock.Any(), gomock.Eq(types.GetJobsQuery{
					UserID:  4,
					Page:    1,
					PerPage: 10,
				})).Return([]models.Job{*mockPreheatJobModel}, int64(1), nil).Times(1)
			},
			expect: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert := assert.New(t)
				assert.Equal(http.StatusOK, w.Code)
				job := models.Job{}
				err := json.Unmarshal(w.Body.Bytes()[1:w.Body.Len()-1], &job)
				assert.NoError(err)
				assert.Equal(mockSanitizedPreheatJobModel, &job)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctl := gomock.NewController(t)
			defer ctl.Finish()
			svc := mocks.NewMockService(ctl)
			w := httptest.NewRecorder()
			h := New(svc)
			mockRouter := mockJobRouter(h)

			tc.mock(svc.EXPECT())
			mockRouter.ServeHTTP(w, tc.req)
			tc.expect(t, w)
		})
	}
}
