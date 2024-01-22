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

var _ = ginkgo.Describe("suite-gateway: HTTPRoute", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("Basic HTTPRoute with 1 Hosts 1 Rule 1 Match 1 BackendRef", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		time.Sleep(time.Second * 15)
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: basic-http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /ip
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/notfound").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusNotFound)
	})

	ginkgo.It("Basic HTTPRoute with 2 Hosts 1 Rule 1 Match 1 BackendRef", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		time.Sleep(time.Second * 15)
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: basic-http-route
spec:
  hostnames: ["httpbin.org", "good.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /ip
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "good.org").
			Expect().
			Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "bad.org").
			Expect().
			Status(http.StatusNotFound)
	})

	ginkgo.It("Basic HTTPRoute with 1 Hosts 1 Rule 2 Match 1 BackendRef", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: basic-http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /ip
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/notfound").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusNotFound)
	})

	ginkgo.It("Update HTTPRoute", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: basic-http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /ip
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		route = fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: basic-http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "update HTTPRoute")

		time.Sleep(6 * time.Second)

		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusNotFound)
	})

	ginkgo.It("Delete HTTPRoute", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: basic-http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /ip
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)

		assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromString(route), "delete HTTPRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "Checking number of routes")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusNotFound)
	})
})

var _ = ginkgo.Describe("suite-gateway: HTTPRoute with filter", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("HTTPRoute with RequestHeaderModifier", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		time.Sleep(time.Second * 15)
		httproute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /headers
    filters:
    - type: RequestHeaderModifier
      requestHeaderModifier:
        add:
        - name: X-Api-Version
          value: v1
        - name: X-api-key
          value: api-value
        set:
        - name: X-Auth
          value: filter
        remove:
        - Remove-header
        - Host
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httproute), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/headers").
			WithHeader("Host", "httpbin.org").
			WithHeader("Remove-Header", "remove").
			WithHeader("X-Auth", "ingress").
			Expect().
			Status(http.StatusOK).
			Body().
			Contains(`"X-Api-Version": "v1"`).
			Contains(`"X-Api-Key": "api-value"`).
			Contains(`"X-Auth": "filter"`).
			NotContains(`"Remove-Header"`)
	})

	ginkgo.It("HTTPRoute with RequestRidrect", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()

		httproute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /headers
    filters:
    - type: RequestRedirect
      requestRedirect:
        scheme: https
        port: 9443
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httproute), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")

		_ = s.NewAPISIXClient().GET("/headers").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusFound).
			Header("Location").Equal("https://httpbin.org:9443/headers")

		httproute2 := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: http-route2
spec:
  hostnames: ["httpbin.com"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /ip
    filters:
    - type: RequestRedirect
      requestRedirect:
        hostname: httpbin.org
        statusCode: 301
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httproute2), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "Checking number of routes")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.com").
			Expect().
			Status(http.StatusMovedPermanently).
			Header("Location").Equal("http://httpbin.org/ip")
	})

	ginkgo.It("HTTPRoute with RequestMirror", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()

		echo := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo
spec:
  selector:
    matchLabels:
      app: echo
  replicas: 1
  template:
    metadata:
      labels:
        app: echo
    spec:
      containers:
      - name: echo
        image: localhost:5000/echo-server:dev
        ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: echo-service
spec:
  selector:
    app: echo
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(s.FormatRegistry(echo)), "creating echo server")

		httproute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: http-route
spec:
  hostnames: ["httpbin.org"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /headers
    filters:
    - type: RequestMirror
      requestMirror:
        backendRef:
          name: echo-service
          port: 80
    backendRefs:
    - name: %s
      port: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httproute), "creating HTTPRoute")
		time.Sleep(time.Second * 6)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")

		_ = s.NewAPISIXClient().GET("/headers").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)

		echoLogs := s.GetDeploymentLogs("echo")
		assert.Contains(ginkgo.GinkgoT(), echoLogs, "GET /headers")
	})
})
