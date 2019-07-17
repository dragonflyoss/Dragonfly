package server

import (
	"net/http"

	"github.com/dragonflyoss/Dragonfly/supernode/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metrics defines three prometheus metrics for monitoring http handler status
type metrics struct {
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize     *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
}

func newMetrics() *metrics {
	m := &metrics{
		requestCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "http_requests_total",
				Help:      "Counter of HTTP requests.",
			},
			[]string{"code", "handler", "method"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "Histogram of latencies for HTTP requests.",
				Buckets:   []float64{.1, .2, .4, 1, 3, 8, 20, 60, 120},
			},
			[]string{"code", "handler", "method"},
		),
		requestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "http_request_size_bytes",
				Help:      "Histogram of request size for HTTP requests.",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"code", "handler", "method"},
		),
		responseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "http_response_size_bytes",
				Help:      "Histogram of response size for HTTP requests.",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"code", "handler", "method"},
		),
	}

	return m
}

// instrumentHandler will update metrics for every http request
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
