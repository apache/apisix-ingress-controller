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

package features

import (
	"fmt"
	"net/http"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: external service discovery", func() {

	PhaseCreateApisixRoute := func(s *scaffold.Scaffold, name, upstream string) {
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
        - /*
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: bar
    upstreams:
    - name: %s
`, name, upstream)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
	}

	PhaseCreateApisixUpstream := func(s *scaffold.Scaffold, name, discoveryType, serviceName string) {
		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  discovery:
  - type: %s
    serviceName: %s
`, name, discoveryType, serviceName)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(au))
	}

	PhaseValidateNoUpstreams := func(s *scaffold.Scaffold) {
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 0, "upstream count")
	}

	PhaseValidateNoRoutes := func(s *scaffold.Scaffold) {
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 0, "route count")
	}

	PhaseValidateFirstUpstream := func(s *scaffold.Scaffold, length int, node string, port, weight int) string {
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, length, "upstream count")
		upstream := ups[0]
		assert.Len(ginkgo.GinkgoT(), upstream.Nodes, 1)
		assert.Equal(ginkgo.GinkgoT(), node, upstream.Nodes[0].Host)
		assert.Equal(ginkgo.GinkgoT(), port, upstream.Nodes[0].Port)
		assert.Equal(ginkgo.GinkgoT(), weight, upstream.Nodes[0].Weight)

		return upstream.ID
	}

	PhaseValidateRouteAccess := func(s *scaffold.Scaffold, upstreamId string) {
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 1, "route count")
		assert.Equal(ginkgo.GinkgoT(), upstreamId, routes[0].UpstreamId)

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "bar").
			Expect().
			Status(http.StatusOK)
	}

	PhaseValidateRouteAccessCode := func(s *scaffold.Scaffold, upstreamId string, code int) {
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 1, "route count")
		assert.Equal(ginkgo.GinkgoT(), upstreamId, routes[0].UpstreamId)

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "bar").
			Expect().
			Status(code)
	}
})
