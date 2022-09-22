// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package metrics

import (
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	_namespace = "apisix_ingress_controller"
)

// Collector defines all metrics for ingress apisix.
type Collector interface {
	// ResetLeader changes the role of ingress apisix instance (leader, follower).
	ResetLeader(bool)
	// RecordAPISIXCode records a status code returned by APISIX with the resource
	// type label.
	RecordAPISIXCode(int, string)
	// RecordAPISIXLatency records the latency for a round trip from ingress apisix
	// to apisix.
	RecordAPISIXLatency(time.Duration, string)
	// IncrAPISIXRequest increases the number of requests to apisix.
	IncrAPISIXRequest(string)
	// IncrCheckClusterHealth increases the number of cluster health check operations
	// with the cluster name label.
	IncrCheckClusterHealth(string)
	// IncrSyncOperation increases the number of sync operations with the resource
	// type label.
	IncrSyncOperation(string, string)
	// IncrCacheSyncOperation increases the number of cache sync operations with the
	// resource type label.
	IncrCacheSyncOperation(string)
	// IncrEvents increases the number of events handled by controllers with the
	// operation label.
	IncrEvents(string, string)
}

// collector contains necessary messages to collect Prometheus metrics.
type collector struct {
	isLeader           prometheus.Gauge
	apisixLatency      *prometheus.SummaryVec
	apisixRequests     *prometheus.CounterVec
	apisixCodes        *prometheus.GaugeVec
	checkClusterHealth *prometheus.CounterVec
	syncOperation      *prometheus.CounterVec
	cacheSyncOperation *prometheus.CounterVec
	controllerEvents   *prometheus.CounterVec
}

// NewPrometheusCollector creates the Prometheus metrics collector.
// It also registers all internal metric collector to prometheus,
// so do not call this function duplicately.
func NewPrometheusCollector() Collector {
	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	if podNamespace == "" {
		podNamespace = "default"
	}
	constLabels := prometheus.Labels{
		"controller_pod":       podName,
		"controller_namespace": podNamespace,
	}

	collector := &collector{
		isLeader: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "is_leader",
				Namespace:   _namespace,
				Help:        "Whether the role of controller instance is leader",
				ConstLabels: constLabels,
			},
		),
		apisixCodes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "apisix_status_codes",
				Namespace:   _namespace,
				Help:        "Status codes of requests to APISIX",
				ConstLabels: constLabels,
			},
			[]string{"resource", "status_code"},
		),
		apisixLatency: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:   _namespace,
				Name:        "apisix_request_latencies",
				Help:        "Request latencies with APISIX",
				ConstLabels: constLabels,
			},
			[]string{"operation"},
		),
		apisixRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   _namespace,
				Name:        "apisix_requests",
				Help:        "Number of requests to APISIX",
				ConstLabels: constLabels,
			},
			[]string{"resource"},
		),
		checkClusterHealth: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   _namespace,
				Name:        "check_cluster_health_total",
				Help:        "Number of cluster health check operations",
				ConstLabels: constLabels,
			},
			[]string{"name"},
		),
		syncOperation: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   _namespace,
				Name:        "sync_operation_total",
				Help:        "Number of sync operations",
				ConstLabels: constLabels,
			},
			[]string{"resource", "result"},
		),
		cacheSyncOperation: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   _namespace,
				Name:        "cache_sync_total",
				Help:        "Number of cache sync operations",
				ConstLabels: constLabels,
			},
			[]string{"result"},
		),
		controllerEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   _namespace,
				Name:        "events_total",
				Help:        "Number of events handled by the controller",
				ConstLabels: constLabels,
			},
			[]string{"operation", "resource"},
		),
	}

	// Since we use the DefaultRegisterer, in test cases, the metrics
	// might be registered duplicately, unregister them before re register.
	prometheus.Unregister(collector.isLeader)
	prometheus.Unregister(collector.apisixCodes)
	prometheus.Unregister(collector.apisixLatency)
	prometheus.Unregister(collector.apisixRequests)
	prometheus.Unregister(collector.checkClusterHealth)
	prometheus.Unregister(collector.syncOperation)
	prometheus.Unregister(collector.cacheSyncOperation)
	prometheus.Unregister(collector.controllerEvents)

	prometheus.MustRegister(
		collector.isLeader,
		collector.apisixCodes,
		collector.apisixLatency,
		collector.apisixRequests,
		collector.checkClusterHealth,
		collector.syncOperation,
		collector.cacheSyncOperation,
		collector.controllerEvents,
	)

	return collector
}

// ResetLeader resets the leader role.
func (c *collector) ResetLeader(leader bool) {
	if leader {
		c.isLeader.Set(1)
	} else {
		c.isLeader.Set(0)
	}
}

// RecordAPISIXCode records the status code (returned by APISIX)
// for the specific resource (e.g. Route, Upstream and etc).
func (c *collector) RecordAPISIXCode(code int, resource string) {
	c.apisixCodes.With(prometheus.Labels{
		"resource":    resource,
		"status_code": strconv.Itoa(code),
	}).Inc()
}

// RecordAPISIXLatency records the latency for a complete round trip
// from controller to APISIX.
func (c *collector) RecordAPISIXLatency(latency time.Duration, resource string) {
	c.apisixLatency.WithLabelValues(resource).Observe(float64(latency.Nanoseconds()))
}

// IncrAPISIXRequest increases the number of requests for specific
// resource to APISIX.
func (c *collector) IncrAPISIXRequest(resource string) {
	c.apisixRequests.WithLabelValues(resource).Inc()
}

// IncrCheckClusterHealth increases the number of cluster health check
// operations.
func (c *collector) IncrCheckClusterHealth(name string) {
	c.checkClusterHealth.WithLabelValues(name).Inc()
}

// IncrSyncOperation increases the number of sync operations for specific
// resource.
func (c *collector) IncrSyncOperation(resource, result string) {
	c.syncOperation.With(prometheus.Labels{
		"resource": resource,
		"result":   result,
	}).Inc()
}

// IncrCacheSyncOperation increases the number of cache sync operations for
// cluster.
func (c *collector) IncrCacheSyncOperation(result string) {
	c.cacheSyncOperation.WithLabelValues(result).Inc()
}

// IncrEvents increases the number of events handled by controllers for
// specific operation.
func (c *collector) IncrEvents(resource, operation string) {
	c.controllerEvents.With(prometheus.Labels{
		"operation": operation,
		"resource":  resource,
	}).Inc()
}

// Collect collects the prometheus.Collect.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.isLeader.Collect(ch)
	c.apisixLatency.Collect(ch)
	c.apisixRequests.Collect(ch)
	c.apisixCodes.Collect(ch)
	c.checkClusterHealth.Collect(ch)
	c.syncOperation.Collect(ch)
	c.cacheSyncOperation.Collect(ch)
	c.controllerEvents.Collect(ch)
}

// Describe describes the prometheus.Describe.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	c.isLeader.Describe(ch)
	c.apisixLatency.Describe(ch)
	c.apisixRequests.Describe(ch)
	c.apisixCodes.Describe(ch)
	c.checkClusterHealth.Describe(ch)
	c.syncOperation.Describe(ch)
	c.cacheSyncOperation.Describe(ch)
	c.controllerEvents.Describe(ch)
}
