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

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"d7y.io/dragonfly/v2/internal/job"
	"d7y.io/dragonfly/v2/manager/metrics"
	_ "d7y.io/dragonfly/v2/manager/models" // nolint
	"d7y.io/dragonfly/v2/manager/types"
)

// @Summary Create Job
// @Description Create by json config
// @Tags Job
// @Accept json
// @Produce json
// @Param Job body types.CreateJobRequest true "Job"
// @Success 200 {object} models.Job
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /api/v1/jobs [post]
// @Router /oapi/v1/jobs [post]
func (h *Handlers) CreateJob(ctx *gin.Context) {
	var json types.CreateJobRequest
	if err := ctx.ShouldBindBodyWith(&json, binding.JSON); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
		return
	}

	// Collect CreateJobCount metrics.
	metrics.CreateJobCount.WithLabelValues(json.Type).Inc()
	switch json.Type {
	case job.PreheatJob:
		var json types.CreatePreheatJobRequest
		if err := ctx.ShouldBindBodyWith(&json, binding.JSON); err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
			return
		}

		job, err := h.service.CreatePreheatJob(ctx.Request.Context(), json)
		if err != nil {
			ctx.Error(err) // nolint: errcheck
			return
		}

		ctx.JSON(http.StatusOK, sanitizeJobForResponse(job))
	case job.SyncPeersJob:
		var json types.CreateSyncPeersJobRequest
		if err := ctx.ShouldBindBodyWith(&json, binding.JSON); err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
			return
		}

		// CreateSyncPeersJob is a sync operation, so don't need to return the job id,
		// and not record the job information in the database. If return success, need to
		// query the peers table to get the latest data.
		if err := h.service.CreateSyncPeersJob(ctx.Request.Context(), json); err != nil {
			ctx.Error(err) // nolint: errcheck
			return
		}

		ctx.JSON(http.StatusOK, http.StatusText(http.StatusOK))
	case job.GetTaskJob:
		var json types.CreateGetTaskJobRequest
		if err := ctx.ShouldBindBodyWith(&json, binding.JSON); err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
			return
		}

		if json.Args.TaskID == "" && json.Args.URL == "" && json.Args.ContentForCalculatingTaskID == nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": "invalid params: task_id, content_for_calculating_task_id or url is required"})
			return
		}

		job, err := h.service.CreateGetTaskJob(ctx.Request.Context(), json)
		if err != nil {
			ctx.Error(err) // nolint: errcheck
			return
		}

		ctx.JSON(http.StatusOK, sanitizeJobForResponse(job))
	case job.GetImageDistributionJob:
		var json types.CreateGetImageDistributionJobRequest
		if err := ctx.ShouldBindBodyWith(&json, binding.JSON); err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
			return
		}

		job, err := h.service.CreateGetImageDistributionJob(ctx.Request.Context(), json)
		if err != nil {
			ctx.Error(err) // nolint: errcheck
			return
		}

		ctx.JSON(http.StatusOK, job)
	case job.DeleteTaskJob:
		var json types.CreateDeleteTaskJobRequest
		if err := ctx.ShouldBindBodyWith(&json, binding.JSON); err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
			return
		}

		if json.Args.TaskID == "" && json.Args.URL == "" && json.Args.ContentForCalculatingTaskID == nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": "invalid params: task_id, content_for_calculating_task_id or url is required"})
			return
		}

		job, err := h.service.CreateDeleteTaskJob(ctx.Request.Context(), json)
		if err != nil {
			ctx.Error(err) // nolint: errcheck
			return
		}

		ctx.JSON(http.StatusOK, sanitizeJobForResponse(job))
	case job.GCJob:
		var json types.CreateGCJobRequest
		if err := ctx.ShouldBindBodyWith(&json, binding.JSON); err != nil {
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
			return
		}

		job, err := h.service.CreateGCJob(ctx.Request.Context(), json)
		if err != nil {
			ctx.Error(err) // nolint: errcheck
			return
		}

		ctx.JSON(http.StatusOK, sanitizeJobForResponse(job))
	default:
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": "Unknow type"})
	}
}

// @Summary Destroy Job
// @Description Destroy by id
// @Tags Job
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /api/v1/jobs/{id} [delete]
// @Router /oapi/v1/jobs/{id} [delete]
func (h *Handlers) DestroyJob(ctx *gin.Context) {
	var params types.JobParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
		return
	}

	if err := h.service.DestroyJob(ctx.Request.Context(), params.ID); err != nil {
		ctx.Error(err) // nolint: errcheck
		return
	}

	ctx.Status(http.StatusOK)
}

// @Summary Update Job
// @Description Update by json config
// @Tags Job
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Param Job body types.UpdateJobRequest true "Job"
// @Success 200 {object} models.Job
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /api/v1/jobs/{id} [patch]
// @Router /oapi/v1/jobs/{id} [patch]
func (h *Handlers) UpdateJob(ctx *gin.Context) {
	var params types.JobParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
		return
	}

	var json types.UpdateJobRequest
	if err := ctx.ShouldBindJSON(&json); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
		return
	}

	job, err := h.service.UpdateJob(ctx.Request.Context(), params.ID, json)
	if err != nil {
		ctx.Error(err) // nolint: errcheck
		return
	}

	ctx.JSON(http.StatusOK, sanitizeJobForResponse(job))
}

// @Summary Get Job
// @Description Get Job by id
// @Tags Job
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} models.Job
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /api/v1/jobs/{id} [get]
// @Router /oapi/v1/jobs/{id} [get]
func (h *Handlers) GetJob(ctx *gin.Context) {
	var params types.JobParams
	if err := ctx.ShouldBindUri(&params); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
		return
	}

	job, err := h.service.GetJob(ctx.Request.Context(), params.ID)
	if err != nil {
		ctx.Error(err) // nolint: errcheck
		return
	}

	ctx.JSON(http.StatusOK, sanitizeJobForResponse(job))
}

// @Summary Get Jobs
// @Description Get Jobs
// @Tags Job
// @Accept json
// @Produce json
// @Param page query int true "current page" default(0)
// @Param per_page query int true "return max item count, default 10, max 50" default(10) minimum(2) maximum(50)
// @Success 200 {object} []models.Job
// @Failure 400
// @Failure 404
// @Failure 500
// @Router /api/v1/jobs [get]
// @Router /oapi/v1/jobs [get]
func (h *Handlers) GetJobs(ctx *gin.Context) {
	var query types.GetJobsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"errors": err.Error()})
		return
	}

	h.setPaginationDefault(&query.Page, &query.PerPage)
	jobs, count, err := h.service.GetJobs(ctx.Request.Context(), query)
	if err != nil {
		ctx.Error(err) // nolint: errcheck
		return
	}

	h.setPaginationLinkHeader(ctx, query.Page, query.PerPage, int(count))
	ctx.JSON(http.StatusOK, sanitizeJobsForResponse(jobs))
}
