package util

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "dragonfly"
)

// NewCounter will auto-register a Counter metric to prometheus default registry and return it.
func NewCounter(subsystem, name, help string, labels []string) *prometheus.CounterVec {
	return promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

// NewGauge will auto-register a Gauge metric to prometheus default registry and return it.
func NewGauge(subsystem, name, help string, labels []string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

// NewSummary will auto-register a Summary metric to prometheus default registry and return it.
func NewSummary(subsystem, name, help string, labels []string, objectives map[float64]float64) *prometheus.SummaryVec {
	return promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)
}

// NewHistogram will auto-register a Histogram metric to prometheus default registry and return it.
func NewHistogram(subsystem, name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)
}
