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
package chore

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-chore: endpoints", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("ignore applied only if there is an ApisixRoute referenced", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")

			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		})

		ginkgo.It("upstream nodes should be reset to empty when Service/Endpoints was deleted", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

			// Now delete the backend httpbin service resource.
			assert.Nil(ginkgo.GinkgoT(), s.DeleteHTTPBINService())
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumListUpstreamNodesNth(1, 0))
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusServiceUnavailable)
		})

		ginkgo.It("when endpoint is 0, upstream nodes is also 0", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths: 
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of upstreams")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumListUpstreamNodesNth(1, 1))

			// scale HTTPBIN, so the endpoints controller has the opportunity to update upstream.
			assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(0))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumListUpstreamNodesNth(1, 0))
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusServiceUnavailable)
		})
	}
	ginkgo.Describe("suite-chore: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold())
	})
})

var _ = ginkgo.Describe("suite-chore: port usage", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("service port != target port", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: httpbin.com
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumListUpstreamNodesNth(1, 1))

		// port in nodes is still the targetPort, not the service port
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing APISIX upstreams")
		assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[0].Port, 80)

		// scale HTTPBIN, so the endpoints controller has the opportunity to update upstream.
		assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(3))
		// s.ScaleHTTPBIN(3) process will be slow, and need time.
		time.Sleep(15 * time.Second)
		ups, err = s.ListApisixUpstreams()
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 3)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumListUpstreamNodesNth(1, 3))

		// port in nodes is still the targetPort, not the service port
		assert.Nil(ginkgo.GinkgoT(), err, "listing APISIX upstreams")
		assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[0].Port, 80)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[1].Port, 80)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[2].Port, 80)
	})
})
