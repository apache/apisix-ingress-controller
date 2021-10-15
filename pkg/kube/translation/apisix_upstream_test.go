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

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
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
				Interval: metav1.Duration{Duration: time.Second},
			},
			Healthy: &configv1.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv1.PassiveHealthCheckHealthy{
					HTTPCodes: []int{200},
					Successes: 2,
				},
				Interval: metav1.Duration{Duration: 3 * time.Second},
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

func TestTranslateUpstreamPassiveHealthCheckUnusually(t *testing.T) {
	tr := &translator{}

	// invalid passive health check type
	hc := &configv1.HealthCheck{
		Passive: &configv1.PassiveHealthCheck{
			Type: "redis",
		},
	}

	err := tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.passive.Type",
		reason: "invalid value",
	})

	// invalid passive health check healthy successes
	hc = &configv1.HealthCheck{
		Passive: &configv1.PassiveHealthCheck{
			Type: "http",
			Healthy: &configv1.PassiveHealthCheckHealthy{
				Successes: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.passive.healthy.successes",
		reason: "invalid value",
	})

	// empty passive health check healthy httpCodes.
	hc = &configv1.HealthCheck{
		Passive: &configv1.PassiveHealthCheck{
			Type: "http",
			Healthy: &configv1.PassiveHealthCheckHealthy{
				Successes: 1,
				HTTPCodes: []int{},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.passive.healthy.httpCodes",
		reason: "empty",
	})

	// empty passive health check unhealthy httpFailures.
	hc = &configv1.HealthCheck{
		Passive: &configv1.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv1.PassiveHealthCheckUnhealthy{
				HTTPFailures: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.passive.unhealthy.httpFailures",
		reason: "invalid value",
	})

	// empty passive health check unhealthy tcpFailures.
	hc = &configv1.HealthCheck{
		Passive: &configv1.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv1.PassiveHealthCheckUnhealthy{
				TCPFailures: -1,
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.passive.unhealthy.tcpFailures",
		reason: "invalid value",
	})

	// empty passive health check unhealthy httpCodes.
	hc = &configv1.HealthCheck{
		Passive: &configv1.PassiveHealthCheck{
			Type: "http",
			Unhealthy: &configv1.PassiveHealthCheckUnhealthy{
				HTTPCodes: []int{},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.passive.unhealthy.httpCodes",
		reason: "empty",
	})
}

func TestTranslateUpstreamActiveHealthCheckUnusually(t *testing.T) {
	tr := &translator{}

	// invalid active health check type
	hc := &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "redis",
		},
	}
	err := tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.Type",
		reason: "invalid value",
	})

	// invalid active health check port value
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "http",
			Port: 65536,
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.port",
		reason: "invalid value",
	})

	// invalid active health check concurrency value
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type:        "https",
			Concurrency: -1,
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.concurrency",
		reason: "invalid value",
	})

	// invalid active health check healthy successes value
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv1.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv1.PassiveHealthCheckHealthy{
					Successes: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.healthy.successes",
		reason: "invalid value",
	})

	// invalid active health check healthy successes value
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv1.ActiveHealthCheckHealthy{
				PassiveHealthCheckHealthy: configv1.PassiveHealthCheckHealthy{
					HTTPCodes: []int{},
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.healthy.httpCodes",
		reason: "empty",
	})

	// invalid active health check healthy interval
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "https",
			Healthy: &configv1.ActiveHealthCheckHealthy{
				Interval: metav1.Duration{Duration: 500 * time.Millisecond},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.healthy.interval",
		reason: "invalid value",
	})

	// missing active health check healthy interval
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type:    "https",
			Healthy: &configv1.ActiveHealthCheckHealthy{},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.healthy.interval",
		reason: "invalid value",
	})

	// invalid active health check unhealthy httpFailures
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv1.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv1.PassiveHealthCheckUnhealthy{
					HTTPFailures: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.unhealthy.httpFailures",
		reason: "invalid value",
	})

	// invalid active health check unhealthy tcpFailures
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv1.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv1.PassiveHealthCheckUnhealthy{
					TCPFailures: -1,
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.unhealthy.tcpFailures",
		reason: "invalid value",
	})

	// invalid active health check unhealthy httpCodes
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv1.ActiveHealthCheckUnhealthy{
				PassiveHealthCheckUnhealthy: configv1.PassiveHealthCheckUnhealthy{
					HTTPCodes: []int{},
				},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.unhealthy.httpCodes",
		reason: "empty",
	})

	// invalid active health check unhealthy interval
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type: "https",
			Unhealthy: &configv1.ActiveHealthCheckUnhealthy{
				Interval: metav1.Duration{Duration: 500 * time.Millisecond},
			},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.unhealthy.interval",
		reason: "invalid value",
	})

	// missing active health check unhealthy interval
	hc = &configv1.HealthCheck{
		Active: &configv1.ActiveHealthCheck{
			Type:      "https",
			Unhealthy: &configv1.ActiveHealthCheckUnhealthy{},
		},
	}
	err = tr.translateUpstreamHealthCheck(hc, nil)
	assert.Equal(t, err, &translateError{
		field:  "healthCheck.active.unhealthy.interval",
		reason: "invalid value",
	})
}

func TestUpstreamRetriesAndTimeout(t *testing.T) {
	tr := &translator{}
	retries := -1
	err := tr.translateUpstreamRetriesAndTimeout(&retries, nil, nil)
	assert.Equal(t, err, &translateError{
		field:  "retries",
		reason: "invalid value",
	})

	var ups apisixv1.Upstream
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeout(&retries, nil, &ups)
	assert.Nil(t, err)
	assert.Equal(t, *ups.Retries, 3)

	timeout := &configv1.UpstreamTimeout{
		Connect: metav1.Duration{Duration: time.Second},
		Read:    metav1.Duration{Duration: -1},
	}
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeout(&retries, timeout, &ups)
	assert.Equal(t, err, &translateError{
		field:  "timeout.read",
		reason: "invalid value",
	})

	timeout = &configv1.UpstreamTimeout{
		Connect: metav1.Duration{Duration: time.Second},
		Read:    metav1.Duration{Duration: 15 * time.Second},
	}
	retries = 3
	err = tr.translateUpstreamRetriesAndTimeout(&retries, timeout, &ups)
	assert.Nil(t, err)
	assert.Equal(t, ups.Timeout, &apisixv1.UpstreamTimeout{
		Connect: 1,
		Send:    60,
		Read:    15,
	})
}
