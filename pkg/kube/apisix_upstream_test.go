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
package kube

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
)

func TestTranslateUpstreamHealthCheck(t *testing.T) {
	tr := &translator{}
	hc := &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type:        apisixv1.HealthCheckHTTP,
			Timeout:     5 * time.Second,
			Concurrency: 2,
			HTTPPath:    "/healthz",
			Unhealthy: &configv1.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv1.PassiveHealthCheckUnhealthy{
					HTTPCodes: []int{500, 502, 504},
				},
				Interval: time.Second,
			},
			Healthy: &configv1.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv1.PassiveHealthCheckHealthy{
					HTTPCodes: []int{200},
					Successes: 2,
				},
				Interval: 3 * time.Second,
			},
		},
		Passive: &configv1.PassiveHealthCheck{
			Type: apisixv1.HealthCheckHTTP,
			Healthy: &configv1.PassiveHealthCheckHealthy{
				HTTPCodes: []int{200},
				Successes: 2,
			},
			Unhealthy: &configv1.PassiveHealthCheckUnhealthy{
				HTTPCodes: []int{500},
			},
		},
	}

	var ups apisixv1.Upstream
	err := tr.translateUpstreamHealthCheck(hc, &ups)
	assert.Nil(t, err, "translating upstream health check")
	assert.Equal(t, ups.Checks.Active, &apisixv1.UpstreamActiveHealthCheck{
		Type:            apisixv1.HealthCheckHTTP,
		Timeout:         5,
		Concurrency:     2,
		HTTPPath:        "/healthz",
		HTTPSVerifyCert: true,
		Healthy: apisixv1.UpstreamActiveHealthCheckHealthy{
			Interval: 3,
			UpstreamPassiveHealthCheckHealthy: apisixv1.UpstreamPassiveHealthCheckHealthy{
				HTTPStatuses: []int{200},
				Successes:    2,
			},
		},
		Unhealthy: apisixv1.UpstreamActiveHealthCheckUnhealthy{
			Interval: 1,
			UpstreamPassiveHealthCheckUnhealthy: apisixv1.UpstreamPassiveHealthCheckUnhealthy{
				HTTPStatuses: []int{500, 502, 504},
			},
		},
	})
	assert.Equal(t, ups.Checks.Passive, &apisixv1.UpstreamPassiveHealthCheck{
		Type: apisixv1.HealthCheckHTTP,
		Healthy: apisixv1.UpstreamPassiveHealthCheckHealthy{
			Successes:    2,
			HTTPStatuses: []int{200},
		},
		Unhealthy: apisixv1.UpstreamPassiveHealthCheckUnhealthy{
			HTTPStatuses: []int{500},
		},
	})
}
