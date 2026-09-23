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

package translator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func TestTranslateApisixUpstreamRetriesAndTimeoutPreservesSubsecondValues(t *testing.T) {
	config := &apiv2.ApisixUpstreamConfig{
		Timeout: &apiv2.UpstreamTimeout{
			Connect: metav1.Duration{Duration: 500 * time.Millisecond},
			Read:    metav1.Duration{Duration: 1500 * time.Millisecond},
			Send:    metav1.Duration{Duration: 2500 * time.Millisecond},
		},
	}
	upstream := adc.NewDefaultUpstream()

	err := translateApisixUpstreamRetriesAndTimeout(config, upstream)

	require.NoError(t, err)
	assert.Equal(t, 0.5, upstream.Timeout.Connect)
	assert.Equal(t, 1.5, upstream.Timeout.Read)
	assert.Equal(t, 2.5, upstream.Timeout.Send)
}

func TestBuildTimeoutPreservesSubsecondValues(t *testing.T) {
	rule := apiv2.ApisixRouteHTTP{
		Timeout: &apiv2.UpstreamTimeout{
			Connect: metav1.Duration{Duration: 500 * time.Millisecond},
			Read:    metav1.Duration{Duration: 1500 * time.Millisecond},
			Send:    metav1.Duration{Duration: 2500 * time.Millisecond},
		},
	}

	timeout := (&Translator{}).buildTimeout(rule)

	assert.Equal(t, 0.5, timeout.Connect)
	assert.Equal(t, 1.5, timeout.Read)
	assert.Equal(t, 2.5, timeout.Send)
}

func TestAttachBackendTrafficPolicyToUpstreamPreservesSubsecondValues(t *testing.T) {
	policy := &v1alpha1.BackendTrafficPolicy{
		Spec: v1alpha1.BackendTrafficPolicySpec{
			Timeout: &v1alpha1.Timeout{
				Connect: metav1.Duration{},
				Read:    metav1.Duration{Duration: 1500 * time.Millisecond},
				Send:    metav1.Duration{Duration: 2500 * time.Millisecond},
			},
		},
	}
	upstream := &adc.Upstream{}

	(&Translator{}).attachBackendTrafficPolicyToUpstream(policy, upstream)

	assert.Equal(t, 60.0, upstream.Timeout.Connect)
	assert.Equal(t, 1.5, upstream.Timeout.Read)
	assert.Equal(t, 2.5, upstream.Timeout.Send)
}
