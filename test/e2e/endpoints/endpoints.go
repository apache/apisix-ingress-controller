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

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("endpoints", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("ignore applied only if there is an ApisixRoute referenced", func() {
		time.Sleep(5 * time.Second)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(0), "checking number of upstreams")
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		ups := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
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
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ups))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")
	})

	ginkgo.It("upstream nodes should be reset to empty when Service/Endpoints was deleted", func() {
		ginkgo.Skip("now we don't handle endpoints delete event")
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths: /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

		// Now delete the backend httpbin service resource.
		assert.Nil(ginkgo.GinkgoT(), s.DeleteHTTPBINService())
		time.Sleep(3 * time.Second)
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusServiceUnavailable)
	})
})

var _ = ginkgo.Describe("port usage", func() {
	opts := &scaffold.Options{
		Name:                  "endpoints-port",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    8080,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("service port != target port", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
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
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		time.Sleep(5 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing APISIX upstreams")
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 1)

		// port in nodes is still the targetPort, not the service port
		assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[0].Port, 80)

		// scale HTTPBIN, so the endpoints controller has the opportunity to update upstream.
		assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(3))
		time.Sleep(15 * time.Second)
		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing APISIX upstreams")
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 3)

		// port in nodes is still the targetPort, not the service port
		assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[0].Port, 80)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[1].Port, 80)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Nodes[2].Port, 80)
	})
})
