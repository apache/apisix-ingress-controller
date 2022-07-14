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
package translation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestTranslateUpstreamHealthCheckV2beta3(t *testing.T) {
	tr := &translator{}
	hc := &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type:        apisixv1.HealthCheckHTTP,
			Timeout:     5 * time.Second,
			Concurrency: 2,
			HTTPPath:    "/healthz",
			Unhealthy: &configv2beta3.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv2beta3.PassiveHealthCheckUnhealthy{
					HTTPCodes: []int{500, 502, 504},
				},
				Interval: metav1.Duration{Duration: time.Second},
			},
			Healthy: &configv2beta3.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv2beta3.PassiveHealthCheckHealthy{
					HTTPCodes: []int{200},
					Successes: 2,
				},
				Interval: metav1.Duration{Duration: 3 * time.Second},
			},
		},
		Passive: &configv2beta3.PassiveHealthCheck{
			Type: apisixv1.HealthCheckHTTP,
			Healthy: &configv2beta3.PassiveHealthCheckHealthy{
				HTTPCodes: []int{200},
				Successes: 2,
			},
			Unhealthy: &configv2beta3.PassiveHealthCheckUnhealthy{
				HTTPCodes: []int{500},
			},
		},
	}

	var ups apisixv1.Upstream
	err := tr.translateUpstreamHealthCheckV2beta3(hc, &ups)
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

func TestTranslateUpstreamPassiveHealthCheckUnusuallyV2beta3(t *testing.T) {
	tr := &translator{}

	// invalid passive health check type
	hc := &configv2beta3.HealthCheck{
		Passive: &configv2beta3.PassiveHealthCheck{
			Type: "redis",
		},
	}

	err := tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.Type",
		reason: "invalid value",
	}, err)

	// invalid passive health check healthy successes
	hc = &configv2beta3.HealthCheck{
		Passive: &configv2beta3.PassiveHealthCheck{
			Type: "http",
			Healthy: &configv2beta3.PassiveHealthCheckHealthy{
				Successes: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.healthy.successes",
		reason: "invalid value",
	}, err)

	// empty passive health check healthy httpCodes.
	hc = &configv2beta3.HealthCheck{
		Passive: &configv2beta3.PassiveHealthCheck{
			Type: "http",
			Healthy: &configv2beta3.PassiveHealthCheckHealthy{
				Successes: 1,
				HTTPCodes: []int{},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.healthy.httpCodes",
		reason: "empty",
	}, err)

	// empty passive health check unhealthy httpFailures.
	hc = &configv2beta3.HealthCheck{
		Passive: &configv2beta3.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv2beta3.PassiveHealthCheckUnhealthy{
				HTTPFailures: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.unhealthy.httpFailures",
		reason: "invalid value",
	}, err)

	// empty passive health check unhealthy tcpFailures.
	hc = &configv2beta3.HealthCheck{
		Passive: &configv2beta3.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv2beta3.PassiveHealthCheckUnhealthy{
				TCPFailures: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.unhealthy.tcpFailures",
		reason: "invalid value",
	}, err)

	// empty passive health check unhealthy httpCodes.
	hc = &configv2beta3.HealthCheck{
		Passive: &configv2beta3.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv2beta3.PassiveHealthCheckUnhealthy{
				HTTPCodes: []int{},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.unhealthy.httpCodes",
		reason: "empty",
	}, err)
}

func TestTranslateUpstreamActiveHealthCheckUnusuallyV2beta3(t *testing.T) {
	tr := &translator{}

	// invalid active health check type
	hc := &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "redis",
		},
	}
	err := tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.Type",
		reason: "invalid value",
	}, err)

	// invalid active health check port value
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "http",
			Port: 65536,
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.port",
		reason: "invalid value",
	}, err)

	// invalid active health check concurrency value
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type:        "https",
			Concurrency: -1,
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.concurrency",
		reason: "invalid value",
	}, err)

	// invalid active health check healthy successes value
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv2beta3.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv2beta3.PassiveHealthCheckHealthy{
					Successes: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.healthy.successes",
		reason: "invalid value",
	}, err)

	// invalid active health check healthy successes value
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv2beta3.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv2beta3.PassiveHealthCheckHealthy{
					HTTPCodes: []int{},
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.healthy.httpCodes",
		reason: "empty",
	}, err)

	// invalid active health check healthy interval
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv2beta3.ActiveHealthCheckHealthy{
				Interval: metav1.Duration{Duration: 500 * time.Millisecond},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.healthy.interval",
		reason: "invalid value",
	}, err)

	// missing active health check healthy interval
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type:    "https",
			Healthy: &configv2beta3.ActiveHealthCheckHealthy{},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.healthy.interval",
		reason: "invalid value",
	}, err)

	// invalid active health check unhealthy httpFailures
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv2beta3.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv2beta3.PassiveHealthCheckUnhealthy{
					HTTPFailures: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.httpFailures",
		reason: "invalid value",
	}, err)

	// invalid active health check unhealthy tcpFailures
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv2beta3.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv2beta3.PassiveHealthCheckUnhealthy{
					TCPFailures: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.tcpFailures",
		reason: "invalid value",
	}, err)

	// invalid active health check unhealthy httpCodes
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv2beta3.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv2beta3.PassiveHealthCheckUnhealthy{
					HTTPCodes: []int{},
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.httpCodes",
		reason: "empty",
	}, err)

	// invalid active health check unhealthy interval
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv2beta3.ActiveHealthCheckUnhealthy{
				Interval: metav1.Duration{Duration: 500 * time.Millisecond},
			},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.interval",
		reason: "invalid value",
	}, err)

	// missing active health check unhealthy interval
	hc = &configv2beta3.HealthCheck{
		Active: &configv2beta3.ActiveHealthCheck{
			Type:      "https",
			Unhealthy: &configv2beta3.ActiveHealthCheckUnhealthy{},
		},
	}
	err = tr.translateUpstreamHealthCheckV2beta3(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.interval",
		reason: "invalid value",
	}, err)
}

func TestUpstreamRetriesAndTimeoutV2beta3(t *testing.T) {
	tr := &translator{}
	retries := -1
	err := tr.translateUpstreamRetriesAndTimeoutV2beta3(&retries, nil, nil)
	assert.Equal(t, &translateError{
		field:  "retries",
		reason: "invalid value",
	}, err)

	var ups apisixv1.Upstream
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeoutV2beta3(&retries, nil, &ups)
	assert.Nil(t, err)
	assert.Equal(t, *ups.Retries, 3)

	timeout := &configv2beta3.UpstreamTimeout{
		Connect: metav1.Duration{Duration: time.Second},
		Read:    metav1.Duration{Duration: -1},
	}
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeoutV2beta3(&retries, timeout, &ups)
	assert.Equal(t, &translateError{
		field:  "timeout.read",
		reason: "invalid value",
	}, err)

	timeout = &configv2beta3.UpstreamTimeout{
		Connect: metav1.Duration{Duration: time.Second},
		Read:    metav1.Duration{Duration: 15 * time.Second},
	}
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeoutV2beta3(&retries, timeout, &ups)
	assert.Nil(t, err)
	assert.Equal(t, &apisixv1.UpstreamTimeout{
		Connect: 1,
		Send:    60,
		Read:    15,
	}, ups.Timeout)
}

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
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.Type",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.healthy.successes",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.healthy.httpCodes",
		reason: "empty",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.unhealthy.httpFailures",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.unhealthy.tcpFailures",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.passive.unhealthy.httpCodes",
		reason: "empty",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.Type",
		reason: "invalid value",
	}, err)

	// invalid active health check port value
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type: "http",
			Port: 65536,
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.port",
		reason: "invalid value",
	}, err)

	// invalid active health check concurrency value
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type:        "https",
			Concurrency: -1,
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.concurrency",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.healthy.successes",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.healthy.httpCodes",
		reason: "empty",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.healthy.interval",
		reason: "invalid value",
	}, err)

	// missing active health check healthy interval
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type:    "https",
			Healthy: &configv2.ActiveHealthCheckHealthy{},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.healthy.interval",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.httpFailures",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.tcpFailures",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.httpCodes",
		reason: "empty",
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
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.interval",
		reason: "invalid value",
	}, err)

	// missing active health check unhealthy interval
	hc = &configv2.HealthCheck{
		Active: &configv2.ActiveHealthCheck{
			Type:      "https",
			Unhealthy: &configv2.ActiveHealthCheckUnhealthy{},
		},
	}
	err = tr.translateUpstreamHealthCheckV2(hc, nil)
	assert.Equal(t, &translateError{
		field:  "healthCheck.active.unhealthy.interval",
		reason: "invalid value",
	}, err)
}

func TestUpstreamRetriesAndTimeoutV2(t *testing.T) {
	tr := &translator{}
	retries := -1
	err := tr.translateUpstreamRetriesAndTimeoutV2(&retries, nil, nil)
	assert.Equal(t, &translateError{
		field:  "retries",
		reason: "invalid value",
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
	assert.Equal(t, &translateError{
		field:  "timeout.read",
		reason: "invalid value",
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
