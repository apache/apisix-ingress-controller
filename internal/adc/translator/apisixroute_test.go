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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	adc "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func TestBuildRoute_HostsNotSet(t *testing.T) {
	translator := NewTranslator(logr.Discard())

	ar := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "default",
		},
	}

	service := &adc.Service{}
	rule := apiv2.ApisixRouteHTTP{
		Name: "rule1",
		Match: apiv2.ApisixRouteHTTPMatch{
			Hosts: []string{"example.com", "foo.com"},
			Paths: []string{"/api/*"},
		},
	}

	var enableWebsocket *bool
	translator.buildRoute(ar, service, rule, nil, nil, nil, &enableWebsocket)

	assert.Len(t, service.Routes, 1)
	route := service.Routes[0]
	// route.Hosts should NOT be set — hosts belong on Service, not Route.
	// Setting hosts on Route causes false diffs in backends that don't
	// support route-level hosts, triggering unnecessary PUT requests.
	assert.Nil(t, route.Hosts, "route.Hosts should not be set; hosts should only be on Service")
	assert.Equal(t, []string{"/api/*"}, route.Uris)
}

func TestBuildService_HostsSet(t *testing.T) {
	translator := NewTranslator(logr.Discard())

	ar := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "default",
		},
	}

	rule := apiv2.ApisixRouteHTTP{
		Name: "rule1",
		Match: apiv2.ApisixRouteHTTPMatch{
			Hosts: []string{"example.com", "foo.com"},
			Paths: []string{"/api/*"},
		},
	}

	service := translator.buildService(ar, rule, 0)

	// service.Hosts SHOULD be set — this is the canonical location for hosts.
	assert.Equal(t, []string{"example.com", "foo.com"}, service.Hosts)
}
