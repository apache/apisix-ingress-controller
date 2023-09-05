// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-gateway: TCP Route", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("create TCPRoute", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		tcpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: httpbin-tcp-route
spec:
  rules:
    - backendRefs:
      - name: %s
        port: %d
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(tcpRoute), "creating TCPRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(1), "Checking number of stream_routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClientWithTCPProxy().
			GET("/get").
			Expect().
			Status(http.StatusOK)
	})

	ginkgo.It("update TCPRoute", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		tcpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: httpbin-tcp-route
spec:
  rules:
    - backendRefs:
      - name: %s
        port: %d
`, "httpbin", 80)

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(tcpRoute), "creating TCPRoute")

		time.Sleep(6 * time.Second)
		// Non existent k8s service, service not found
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(1), "The number of stream_routes should be 1")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(0), "The number of upstreams should be 0")

		tcpRoute = fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: httpbin-tcp-route
spec:
  rules:
    - backendRefs:
      - name: %s
        port: %d
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(tcpRoute), "creating TCPRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(2), "The number of stream_routes should be 2")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "The number of upstreams should be 1")

		_ = s.NewAPISIXClientWithTCPProxy().
			GET("/get").
			Expect().
			Status(http.StatusOK)
	})

	ginkgo.It("delete TCPRoute", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		tcpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: httpbin-tcp-route
spec:
  rules:
    - backendRefs:
      - name: %s
        port: %d
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(tcpRoute), "creating TCPRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(1), "The number of stream_routes should be 1")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "The number of upstreams should be 1")

		_ = s.NewAPISIXClientWithTCPProxy().
			GET("/get").
			Expect().
			Status(http.StatusOK)

		assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromString(tcpRoute), "deleting TCPRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(0), "The number of stream_routes should be 0")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(0), "The number of upstreams should be 0")
	})
})
