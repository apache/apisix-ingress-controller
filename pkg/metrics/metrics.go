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
	"strconv"

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

	// ADC client Sync call duration histogram. Distinct in scope from ADCSyncDuration:
	// that one covers a whole logical sync, which for apisix-standalone recovering from a
	// stale conf_version can drive more than one underlying call; this one is exactly one
	// Client.Sync call, one HTTP round trip to ADC. A retry made above the client is a
	// second, independent sample here.
	ADCClientSyncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apisix_ingress_adc_client_sync_duration_seconds",
			Help:    "Time spent on a single adc client Sync call (one HTTP round trip to ADC)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"config_name", "status"},
	)

	// ADC client Sync call errors counter. Same config_name/error_type label shape as
	// ADCExecutionErrors, but error_type here is the raw HTTP status ADC answered the
	// call with ("0" when the call never got a response at all, e.g. a transport
	// failure) rather than a semantic category: this is the client's own per-call view,
	// entirely internal to how Client.Sync went, and never leaves the adc client package.
	ADCClientSyncErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apisix_ingress_adc_client_sync_errors",
			Help: "Total number of adc client Sync call failures, by the raw ADC HTTP status code",
		},
		[]string{"config_name", "error_type"},
	)

	// Status update channel queue length gauge
	StatusUpdateQueueLength = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "apisix_ingress_status_update_queue_length",
			Help: "Current length of the status update queue",
		},
	)
)

// init registers all metrics with the global prometheus registry
func init() {
	// Register metrics with controller-runtime's metrics registry
	metrics.Registry.MustRegister(
		ADCSyncDuration,
		ADCSyncTotal,
		ADCExecutionErrors,
		ADCClientSyncDuration,
		ADCClientSyncErrors,
		StatusUpdateQueueLength,
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

// RecordClientSyncDuration records the duration of a single adc client Sync call.
func RecordClientSyncDuration(configName, status string, duration float64) {
	ADCClientSyncDuration.WithLabelValues(configName, status).Observe(duration)
}

// RecordClientSyncError records a single adc client Sync call failure, by the raw HTTP
// status ADC answered with (0 when no response was received at all).
func RecordClientSyncError(configName string, statusCode int) {
	ADCClientSyncErrors.WithLabelValues(configName, strconv.Itoa(statusCode)).Inc()
}

// UpdateStatusQueueLength updates the status update queue length gauge
func UpdateStatusQueueLength(length float64) {
	StatusUpdateQueueLength.Set(length)
}

// IncStatusQueueLength increments the status update queue length gauge by 1
func IncStatusQueueLength() {
	StatusUpdateQueueLength.Inc()
}

// DecStatusQueueLength decrements the status update queue length gauge by 1
func DecStatusQueueLength() {
	StatusUpdateQueueLength.Dec()
}
