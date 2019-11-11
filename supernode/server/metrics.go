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

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/metricsutils"
	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metrics defines some prometheus metrics for monitoring supernode.
type metrics struct {
	// server http related metrics
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize     *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec

	// dfget metrics
	dfgetDownloadDuration  *prometheus.HistogramVec
	dfgetDownloadFileSize  *prometheus.CounterVec
	dfgetDownloadCount     *prometheus.CounterVec
	dfgetDownloadFailCount *prometheus.CounterVec

	pieceDownloadedBytes *prometheus.CounterVec
}

func newMetrics(register prometheus.Registerer) *metrics {
	return &metrics{
		requestCounter: metricsutils.NewCounter(config.SubsystemSupernode, "http_requests_total",
			"Counter of HTTP requests.", []string{"code", "handler"}, register,
		),
		requestDuration: metricsutils.NewHistogram(config.SubsystemSupernode, "http_request_duration_seconds",
			"Histogram of latencies for HTTP requests.", []string{"handler"},
			[]float64{.1, .2, .4, 1, 3, 8, 20, 60, 120}, register,
		),
		requestSize: metricsutils.NewHistogram(config.SubsystemSupernode, "http_request_size_bytes",
			"Histogram of request size for HTTP requests.", []string{"handler"},
			prometheus.ExponentialBuckets(100, 10, 8), register,
		),
		responseSize: metricsutils.NewHistogram(config.SubsystemSupernode, "http_response_size_bytes",
			"Histogram of response size for HTTP requests.", []string{"handler"},
			prometheus.ExponentialBuckets(100, 10, 8), register,
		),
		pieceDownloadedBytes: metricsutils.NewCounter(config.SubsystemSupernode, "pieces_downloaded_size_bytes_total",
			"total file size of pieces downloaded from supernode in bytes", []string{}, register,
		),
		dfgetDownloadDuration: metricsutils.NewHistogram(config.SubsystemDfget, "download_duration_seconds",
			"Histogram of duration for dfget download.", []string{"callsystem", "peer"},
			[]float64{10, 30, 60, 120, 300, 600}, register,
		),
		dfgetDownloadFileSize: metricsutils.NewCounter(config.SubsystemDfget, "download_size_bytes_total",
			"Total file size downloaded by dfget in bytes", []string{"callsystem", "peer"}, register,
		),
		dfgetDownloadCount: metricsutils.NewCounter(config.SubsystemDfget, "download_total",
			"Total times of dfget download", []string{"callsystem", "peer"}, register,
		),
		dfgetDownloadFailCount: metricsutils.NewCounter(config.SubsystemDfget, "download_failed_total",
			"Total failure times of dfget download", []string{"callsystem", "peer", "reason"}, register,
		),
	}
}

// instrumentHandler will update metrics for every http request.
func (m *metrics) instrumentHandler(handlerName string, handler http.HandlerFunc) http.HandlerFunc {
	return promhttp.InstrumentHandlerDuration(
		m.requestDuration.MustCurryWith(prometheus.Labels{"handler": handlerName}),
		promhttp.InstrumentHandlerCounter(
			m.requestCounter.MustCurryWith(prometheus.Labels{"handler": handlerName}),
			promhttp.InstrumentHandlerRequestSize(
				m.requestSize.MustCurryWith(prometheus.Labels{"handler": handlerName}),
				promhttp.InstrumentHandlerResponseSize(
					m.responseSize.MustCurryWith(prometheus.Labels{"handler": handlerName}),
					handler,
				),
			),
		),
	)
}

func (m *metrics) handleMetricsReport(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
	reader := req.Body
	request := &types.TaskMetricsRequest{}
	if err := json.NewDecoder(reader).Decode(request); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}

	if err := request.Validate(strfmt.NewFormats()); err != nil {
		return errors.Wrap(errortypes.ErrInvalidValue, err.Error())
	}
	status := "success"
	if !request.Success {
		status = "failed"
	}

	dfgetLogger.Debugf("dfget peer %s download %s, taskid %s, callsystem %s, filelength %d, "+
		"backsource reason %s", request.IP+":"+strconv.Itoa(int(request.Port)), status, request.TaskID,
		request.CallSystem, request.FileLength, request.BacksourceReason)

	m.dfgetDownloadCount.WithLabelValues(request.CallSystem, request.IP).Inc()
	if request.Success {
		m.dfgetDownloadDuration.WithLabelValues(request.CallSystem, request.IP).Observe(request.Duration)
		m.dfgetDownloadFileSize.WithLabelValues(request.CallSystem, request.IP).Add(float64(request.FileLength))
	} else {
		m.dfgetDownloadFailCount.WithLabelValues(request.CallSystem, request.IP, request.BacksourceReason).Inc()
	}

	return EncodeResponse(rw, http.StatusOK, nil)
}
