// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func apisixBadStatusCodesTestHandler(t *testing.T, metrics []*io_prometheus_client.MetricFamily) func(*testing.T) {
	return func(t *testing.T) {
		metric := findMetric("apisix_ingress_controller_apisix_bad_status_codes", metrics)
		assert.NotNil(t, metric)
		assert.Equal(t, metric.Type.String(), "GAUGE")
		m := metric.GetMetric()
		assert.Len(t, m, 2)
		assert.Equal(t, *m[0].Gauge.Value, float64(1))
		assert.Equal(t, *m[0].Label[0].Name, "controller_namespace")
		assert.Equal(t, *m[0].Label[0].Value, "default")
		assert.Equal(t, *m[0].Label[1].Name, "controller_pod")
		assert.Equal(t, *m[0].Label[1].Value, "test")
		assert.Equal(t, *m[0].Label[2].Name, "resource")
		assert.Equal(t, *m[0].Label[2].Value, "route")
		assert.Equal(t, *m[0].Label[3].Name, "status_code")
		assert.Equal(t, *m[0].Label[3].Value, "404")

		assert.Equal(t, *m[1].Gauge.Value, float64(1))
		assert.Equal(t, *m[1].Label[0].Name, "controller_namespace")
		assert.Equal(t, *m[1].Label[0].Value, "default")
		assert.Equal(t, *m[1].Label[1].Name, "controller_pod")
		assert.Equal(t, *m[1].Label[1].Value, "test")
		assert.Equal(t, *m[1].Label[2].Name, "resource")
		assert.Equal(t, *m[1].Label[2].Value, "upstream")
		assert.Equal(t, *m[1].Label[3].Name, "status_code")
		assert.Equal(t, *m[1].Label[3].Value, "500")
	}
}

func isLeaderTestHandler(t *testing.T, metrics []*io_prometheus_client.MetricFamily) func(*testing.T) {
	return func(t *testing.T) {
		metric := findMetric("apisix_ingress_controller_is_leader", metrics)
		assert.NotNil(t, metric)
		assert.Equal(t, metric.Type.String(), "GAUGE")
		m := metric.GetMetric()
		assert.Len(t, m, 1)

		assert.Equal(t, *m[0].Gauge.Value, float64(1))
		assert.Equal(t, *m[0].Label[0].Name, "controller_namespace")
		assert.Equal(t, *m[0].Label[0].Value, "default")
		assert.Equal(t, *m[0].Label[1].Name, "controller_pod")
		assert.Equal(t, *m[0].Label[1].Value, "test")
	}
}

func apisixLatencyTestHandler(t *testing.T, metrics []*io_prometheus_client.MetricFamily) func(t *testing.T) {
	return func(t *testing.T) {
		metric := findMetric("apisix_ingress_controller_apisix_request_latencies", metrics)
		assert.NotNil(t, metric)
		assert.Equal(t, metric.Type.String(), "SUMMARY")
		m := metric.GetMetric()
		assert.Len(t, m, 1)

		assert.Equal(t, *m[0].Summary.SampleCount, uint64(1))
		assert.Equal(t, *m[0].Summary.SampleSum, float64((500 * time.Millisecond).Nanoseconds()))
		assert.Equal(t, *m[0].Label[0].Name, "controller_namespace")
		assert.Equal(t, *m[0].Label[0].Value, "default")
		assert.Equal(t, *m[0].Label[1].Name, "controller_pod")
		assert.Equal(t, *m[0].Label[1].Value, "test")
	}
}

func apisixRequestTestHandler(t *testing.T, metrics []*io_prometheus_client.MetricFamily) func(t *testing.T) {
	return func(t *testing.T) {
		metric := findMetric("apisix_ingress_controller_apisix_requests", metrics)
		assert.NotNil(t, metric)
		assert.Equal(t, metric.Type.String(), "COUNTER")
		m := metric.GetMetric()
		assert.Len(t, m, 2)

		assert.Equal(t, *m[0].Counter.Value, float64(2))
		assert.Equal(t, *m[0].Label[0].Name, "controller_namespace")
		assert.Equal(t, *m[0].Label[0].Value, "default")
		assert.Equal(t, *m[0].Label[1].Name, "controller_pod")
		assert.Equal(t, *m[0].Label[1].Value, "test")
		assert.Equal(t, *m[0].Label[2].Name, "resource")
		assert.Equal(t, *m[0].Label[2].Value, "route")

		assert.Equal(t, *m[1].Counter.Value, float64(1))
		assert.Equal(t, *m[1].Label[0].Name, "controller_namespace")
		assert.Equal(t, *m[1].Label[0].Value, "default")
		assert.Equal(t, *m[1].Label[1].Name, "controller_pod")
		assert.Equal(t, *m[1].Label[1].Value, "test")
		assert.Equal(t, *m[1].Label[2].Name, "resource")
		assert.Equal(t, *m[1].Label[2].Value, "upstream")
	}
}

func TestPrometheusCollector(t *testing.T) {
	c := NewPrometheusCollector("test", "default")
	c.ResetLeader(true)
	c.RecordAPISIXCode(404, "route")
	c.RecordAPISIXCode(500, "upstream")
	c.RecordAPISIXLatency(500 * time.Millisecond)
	c.IncrAPISIXRequest("route")
	c.IncrAPISIXRequest("route")
	c.IncrAPISIXRequest("upstream")

	metrics, err := prometheus.DefaultGatherer.Gather()
	assert.Nil(t, err)

	t.Run("apisix_bad_status_codes", apisixBadStatusCodesTestHandler(t, metrics))
	t.Run("is_leader", isLeaderTestHandler(t, metrics))
	t.Run("apisix_request_latencies", apisixLatencyTestHandler(t, metrics))
	t.Run("apisix_requests", apisixRequestTestHandler(t, metrics))
}

func findMetric(name string, metrics []*io_prometheus_client.MetricFamily) *io_prometheus_client.MetricFamily {
	for _, m := range metrics {
		if name == *m.Name {
			return m
		}
	}
	return nil
}
