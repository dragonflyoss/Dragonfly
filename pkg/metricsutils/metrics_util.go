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

package metricsutils

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "dragonfly"
)

// NewCounter will register a Counter metric to specified registry and return it.
// If registry is not specified, it will register metric to default prometheus registry.
func NewCounter(subsystem, name, help string, labels []string, register prometheus.Registerer) *prometheus.CounterVec {
	if register == nil {
		register = prometheus.DefaultRegisterer
	}
	m := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
	register.MustRegister(m)
	return m
}

// NewGauge will register a Gauge metric to specified registry and return it.
// If registry is not specified, it will register metric to default prometheus registry.
func NewGauge(subsystem, name, help string, labels []string, register prometheus.Registerer) *prometheus.GaugeVec {
	if register == nil {
		register = prometheus.DefaultRegisterer
	}
	m := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)
	register.MustRegister(m)
	return m
}

// NewSummary will register a Summary metric to specified registry and return it.
// If registry is not specified, it will register metric to default prometheus registry.
func NewSummary(subsystem, name, help string, labels []string, objectives map[float64]float64, register prometheus.Registerer) *prometheus.SummaryVec {
	if register == nil {
		register = prometheus.DefaultRegisterer
	}
	m := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labels,
	)
	register.MustRegister(m)
	return m
}

// NewHistogram will register a Histogram metric to specified registry and return it.
// If registry is not specified, it will register metric to default prometheus registry.
func NewHistogram(subsystem, name, help string, labels []string, buckets []float64, register prometheus.Registerer) *prometheus.HistogramVec {
	if register == nil {
		register = prometheus.DefaultRegisterer
	}
	m := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)
	register.MustRegister(m)
	return m
}
