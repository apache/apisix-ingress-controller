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
package translation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestTranslateUpstreamHealthCheckV2(t *testing.T) {
	tr := &translator{}
	hc := &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type:        apisixv1.HealthCheckHTTP,
			Timeout:     5 * time.Second,
			Concurrency: 2,
			HTTPPath:    "/healthz",
			Unhealthy: &configv2.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv2.PassiveHealthCheckUnhealthy{
					HTTPCodes: []int{500, 502, 504},
				},
				Interval: metav1.Duration{Duration: time.Second},
			},
			Healthy: &configv2.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv2.PassiveHealthCheckHealthy{
					HTTPCodes: []int{200},
					Successes: 2,
				},
				Interval: metav1.Duration{Duration: 3 * time.Second},
			},
		},
		Passive: &configv2.PassiveHealthCheck{
			Type: apisixv1.HealthCheckHTTP,
			Healthy: &configv2.PassiveHealthCheckHealthy{
				HTTPCodes: []int{200},
				Successes: 2,
			},
			Unhealthy: &configv2.PassiveHealthCheckUnhealthy{
				HTTPCodes: []int{500},
			},
		},
	}

	var ups apisixv1.Upstream
	err := tr.translateUpstreamHealthCheckV2(hc, &ups)
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

func TestTranslateUpstreamPassiveHealthCheckUnusuallyV2(t *testing.T) {
	tr := &translator{}

	// invalid passive health check type
	hc := &configv2.HealthCheck{
		Passive: &configv2.PassiveHealthCheck{
			Type: "redis",
		},
	}

	err := tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.passive.Type",
		Reason: "invalid value",
	}, err)

	// invalid passive health check healthy successes
	hc = &configv2.HealthCheck{
		Passive: &configv2.PassiveHealthCheck{
			Type: "http",
			Healthy: &configv2.PassiveHealthCheckHealthy{
				Successes: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.passive.healthy.successes",
		Reason: "invalid value",
	}, err)

	// empty passive health check healthy httpCodes.
	hc = &configv2.HealthCheck{
		Passive: &configv2.PassiveHealthCheck{
			Type: "http",
			Healthy: &configv2.PassiveHealthCheckHealthy{
				Successes: 1,
				HTTPCodes: []int{},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.passive.healthy.httpCodes",
		Reason: "empty",
	}, err)

	// empty passive health check unhealthy httpFailures.
	hc = &configv2.HealthCheck{
		Passive: &configv2.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv2.PassiveHealthCheckUnhealthy{
				HTTPFailures: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.passive.unhealthy.httpFailures",
		Reason: "invalid value",
	}, err)

	// empty passive health check unhealthy tcpFailures.
	hc = &configv2.HealthCheck{
		Passive: &configv2.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv2.PassiveHealthCheckUnhealthy{
				TCPFailures: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.passive.unhealthy.tcpFailures",
		Reason: "invalid value",
	}, err)

	// empty passive health check unhealthy httpCodes.
	hc = &configv2.HealthCheck{
		Passive: &configv2.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv2.PassiveHealthCheckUnhealthy{
				HTTPCodes: []int{},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.passive.unhealthy.httpCodes",
		Reason: "empty",
	}, err)
}

func TestTranslateUpstreamActiveHealthCheckUnusuallyV2(t *testing.T) {
	tr := &translator{}

	// invalid active health check type
	hc := &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "redis",
		},
	}
	err := tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.Type",
		Reason: "invalid value",
	}, err)

	// invalid active health check port value
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "http",
			Port: 65536,
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.port",
		Reason: "invalid value",
	}, err)

	// invalid active health check concurrency value
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type:        "https",
			Concurrency: -1,
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.concurrency",
		Reason: "invalid value",
	}, err)

	// invalid active health check healthy successes value
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv2.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv2.PassiveHealthCheckHealthy{
					Successes: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.healthy.successes",
		Reason: "invalid value",
	}, err)

	// invalid active health check healthy successes value
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv2.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv2.PassiveHealthCheckHealthy{
					HTTPCodes: []int{},
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.healthy.httpCodes",
		Reason: "empty",
	}, err)

	// invalid active health check healthy interval
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv2.ActiveHealthCheckHealthy{
				Interval: metav1.Duration{Duration: 500 * time.Millisecond},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.healthy.interval",
		Reason: "invalid value",
	}, err)

	// missing active health check healthy interval
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type:    "https",
			Healthy: &configv2.ActiveHealthCheckHealthy{},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.healthy.interval",
		Reason: "invalid value",
	}, err)

	// invalid active health check unhealthy httpFailures
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv2.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv2.PassiveHealthCheckUnhealthy{
					HTTPFailures: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.unhealthy.httpFailures",
		Reason: "invalid value",
	}, err)

	// invalid active health check unhealthy tcpFailures
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv2.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv2.PassiveHealthCheckUnhealthy{
					TCPFailures: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.unhealthy.tcpFailures",
		Reason: "invalid value",
	}, err)

	// invalid active health check unhealthy httpCodes
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv2.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv2.PassiveHealthCheckUnhealthy{
					HTTPCodes: []int{},
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.unhealthy.httpCodes",
		Reason: "empty",
	}, err)

	// invalid active health check unhealthy interval
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv2.ActiveHealthCheckUnhealthy{
				Interval: metav1.Duration{Duration: 500 * time.Millisecond},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.unhealthy.interval",
		Reason: "invalid value",
	}, err)

	// missing active health check unhealthy interval
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type:      "https",
			Unhealthy: &configv2.ActiveHealthCheckUnhealthy{},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &TranslateError{
		Field:  "healthCheck.active.unhealthy.interval",
		Reason: "invalid value",
	}, err)
}

func TestUpstreamRetriesAndTimeoutV2(t *testing.T) {
	tr := &translator{}
	retries := -1
	err := tr.translateUpstreamRetriesAndTimeoutV2(&retries, nil, nil)
	assert.Equal(t, &TranslateError{
		Field:  "retries",
		Reason: "invalid value",
	}, err)

	var ups apisixv1.Upstream
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeoutV2(&retries, nil, &ups)
	assert.Nil(t, err)
	assert.Equal(t, *ups.Retries, 3)

	timeout := &configv2.UpstreamTimeout{
		Connect: metav1.Duration{Duration: time.Second},
		Read:    metav1.Duration{Duration: -1},
	}
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeoutV2(&retries, timeout, &ups)
	assert.Equal(t, &TranslateError{
		Field:  "timeout.read",
		Reason: "invalid value",
	}, err)

	timeout = &configv2.UpstreamTimeout{
		Connect: metav1.Duration{Duration: time.Second},
		Read:    metav1.Duration{Duration: 15 * time.Second},
	}
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeoutV2(&retries, timeout, &ups)
	assert.Nil(t, err)
	assert.Equal(t, &apisixv1.UpstreamTimeout{
		Connect: 1,
		Send:    60,
		Read:    15,
	}, ups.Timeout)
}

func TestUpstreamPassHost(t *testing.T) {
	tr := &translator{}
	tests := []struct {
		name     string
		phc      *passHostConfig
		wantFunc func(t *testing.T, err error, ups *apisixv1.Upstream, phc *passHostConfig)
	}{
		{
			name: "should be empty when settings not set explicitly",
			phc:  &passHostConfig{},
			wantFunc: func(t *testing.T, err error, ups *apisixv1.Upstream, phc *passHostConfig) {
				assert.Nil(t, err)
				assert.Empty(t, ups.PassHost)
				assert.Empty(t, ups.UpstreamHost)
			},
		},
		{
			name: "should set passHost to pass",
			phc:  &passHostConfig{passHost: apisixv1.PassHostPass},
			wantFunc: func(t *testing.T, err error, ups *apisixv1.Upstream, phc *passHostConfig) {
				assert.Nil(t, err)
				assert.Equal(t, phc.passHost, ups.PassHost)
				assert.Empty(t, ups.UpstreamHost)
			},
		},
		{
			name: "should fail when passHost set to invalid value",
			phc:  &passHostConfig{passHost: "unknown"},
			wantFunc: func(t *testing.T, err error, ups *apisixv1.Upstream, phc *passHostConfig) {
				assert.Equal(t, &TranslateError{
					Field:  "passHost",
					Reason: "invalid value",
				}, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ups := apisixv1.NewDefaultUpstream()
			err := tr.translatePassHost(tt.phc, ups)

			tt.wantFunc(t, err, ups, tt.phc)
		})
	}
}
