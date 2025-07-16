// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ADC sync operation duration histogram
	ADCSyncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apisix_ingress_adc_sync_duration_seconds",
			Help:    "Time spent on ADC sync operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"config_name", "resource_type", "status"},
	)

	// ADC sync operation counter
	ADCSyncTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apisix_ingress_adc_sync_total",
			Help: "Total number of ADC sync operations",
		},
		[]string{"config_name", "resource_type", "status"},
	)

	// ADC execution errors counter
	ADCExecutionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apisix_ingress_adc_execution_errors_total",
			Help: "Total number of ADC execution errors",
		},
		[]string{"config_name", "error_type"},
	)

	// Resource sync gauge
	ResourceSyncGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apisix_ingress_resources_synced",
			Help: "Number of resources currently synced",
		},
		[]string{"config_name", "resource_type"},
	)
)

// init registers all metrics with the global prometheus registry
func init() {
	// Register metrics with controller-runtime's metrics registry
	metrics.Registry.MustRegister(
		ADCSyncDuration,
		ADCSyncTotal,
		ADCExecutionErrors,
		ResourceSyncGauge,
	)
}

// RecordSyncDuration records the duration of an ADC sync operation
func RecordSyncDuration(configName, resourceType, status string, duration float64) {
	ADCSyncDuration.WithLabelValues(configName, resourceType, status).Observe(duration)
	ADCSyncTotal.WithLabelValues(configName, resourceType, status).Inc()
}

// RecordExecutionError records an ADC execution error
func RecordExecutionError(configName, errorType string) {
	ADCExecutionErrors.WithLabelValues(configName, errorType).Inc()
}

// UpdateResourceGauge updates the resource sync gauge
func UpdateResourceGauge(configName, resourceType string, count float64) {
	ResourceSyncGauge.WithLabelValues(configName, resourceType).Set(count)
}
