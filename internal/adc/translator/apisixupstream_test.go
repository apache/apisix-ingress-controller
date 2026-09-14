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

	"github.com/stretchr/testify/assert"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func TestTranslateUpstreamHealthCheckPreservesType(t *testing.T) {
	tests := []struct {
		name      string
		typeValue string
		expected  string
	}{
		{name: "default", expected: apiv2.HealthCheckHTTP},
		{name: "http", typeValue: apiv2.HealthCheckHTTP, expected: apiv2.HealthCheckHTTP},
		{name: "https", typeValue: apiv2.HealthCheckHTTPS, expected: apiv2.HealthCheckHTTPS},
		{name: "tcp", typeValue: apiv2.HealthCheckTCP, expected: apiv2.HealthCheckTCP},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			active, err := translateUpstreamActiveHealthCheck(&apiv2.ActiveHealthCheck{Type: test.typeValue})
			assert.NoError(t, err)
			assert.Equal(t, test.expected, active.Type)

			passive := translateUpstreamPassiveHealthCheck(&apiv2.PassiveHealthCheck{Type: test.typeValue})
			assert.Equal(t, test.expected, passive.Type)
		})
	}
}
