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
package endpoints

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

func ensureNumListUpstreamNodes(s *scaffold.Scaffold, upsNum int, upsNodesNum int) error {
	condFunc := func() (bool, error) {
		ups, err := s.ListApisixUpstreams()
		if err != nil || len(ups) != upsNum || len(ups[0].Nodes) != upsNodesNum {
			return false, fmt.Errorf("ensureNumListUpstreamNodes failed")
		}
		return true, nil
	}
	return wait.Poll(2*time.Second, 40*time.Second, condFunc)
}

var _ = ginkgo.Describe("suite-endpoints: endpoints", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("ignore applied only if there is an ApisixRoute referenced", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
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
apiVersion: apisix.apache.org/v2beta3
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
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

			// Now delete the backend httpbin service resource.
			assert.Nil(ginkgo.GinkgoT(), s.DeleteHTTPBINService())
			assert.Nil(ginkgo.GinkgoT(), ensureNumListUpstreamNodes(s, 1, 0))
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusServiceUnavailable)
		})

		ginkgo.It("when endpoint is 0, upstream nodes is also 0", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
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
			assert.Nil(ginkgo.GinkgoT(), ensureNumListUpstreamNodes(s, 1, 1))

			// scale HTTPBIN, so the endpoints controller has the opportunity to update upstream.
			assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(0))
			assert.Nil(ginkgo.GinkgoT(), ensureNumListUpstreamNodes(s, 1, 0))
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusServiceUnavailable)
		})
	}
	ginkgo.Describe("suite-endpoints: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold())
	})
	ginkgo.Describe("suite-endpoints: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold())
	})
})

var _ = ginkgo.Describe("suite-endpoints: port usage", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("service port != target port", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
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
			assert.Nil(ginkgo.GinkgoT(), ensureNumListUpstreamNodes(s, 1, 1))

			// port in nodes is still the targetPort, not the service port
			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err, "listing APISIX upstreams")
			assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[0].Port, 80)

			// scale HTTPBIN, so the endpoints controller has the opportunity to update upstream.
			assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(3))
			assert.Nil(ginkgo.GinkgoT(), ensureNumListUpstreamNodes(s, 1, 3))

			// port in nodes is still the targetPort, not the service port
			ups, err = s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err, "listing APISIX upstreams")
			assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[0].Port, 80)
			assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[1].Port, 80)
			assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[2].Port, 80)
		})
	}
	ginkgo.Describe("suite-endpoints: scaffold v2beta3", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "endpoints-port",
			IngressAPISIXReplicas: 1,
			HTTPBinServicePort:    8080,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2beta3,
		}))
	})
	ginkgo.Describe("suite-endpoints: scaffold v2", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "endpoints-port",
			IngressAPISIXReplicas: 1,
			HTTPBinServicePort:    8080,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
		}))
	})
})
