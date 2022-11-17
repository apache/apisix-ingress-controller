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
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-gateway: Route Attachment", func() {
	s := scaffold.NewDefaultScaffold()

	gatewayClassName := "test-gateway-class"
	gatewayName := "test-gateway"
	// create Gateway resource with AllowedRoute
	ginkgo.JustBeforeEach(func() {
		gatewayClass := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: apisix.apache.org/gateway-controller
`, gatewayClassName)
		gateway := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: Gateway
metadata:
  namespace: %s
  name: %s
spec:
  gatewayClassName: %s
  listeners:
  - protocol: HTTP
    port: 80
    name: same-namespace-route
    allowedRoutes:
      namespaces:
        from: Same
  - protocol: HTTP
    port: 80
    name: cross-namespace-route
    allowedRoutes:
      namespaces:
        from: Selector
        selector:
          matchLabels:
            cross-ns-access: "true"
  - protocol: TCP
    port: 80
    name: tcp-route-only
    allowedRoutes:
      kinds:
      - kind: TCPRoute
  - protocol: HTTP
    hostname: "*.apisix.com"
    port: 80
    name: hostname-match-route
`, s.Namespace(), gatewayName, gatewayClassName)

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(gatewayClass), "creating GatewayClass")
		time.Sleep(time.Second * 5)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(gateway), "creating Gateway")
	})

	ginkgo.It("Gateway cross-namespace routing not allow", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		crossName := fmt.Sprintf("no-label-cross-ns-%d", time.Now().Unix())
		crossNamespace := fmt.Sprintf(`
 apiVersion: v1
 kind: Namespace
 metadata:
  name: %s
`, crossName)

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(crossNamespace), "createing no label namespace")
		time.Sleep(time.Second * 5)
		defer func() {
			s.DeleteResourceFromString(crossNamespace)
		}()

		httpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: HTTPRoute
metadata:
  name: cross-namespace-route
  namespace: %s
spec:
  parentRefs:
  - name: %s
    namespace: %s
    sectionName: same-namespace-route
  rules:
  - matches:
    - path:
        value: /get
    backendRefs:
    - name: %s
      port: %d
`, crossName, gatewayName, s.Namespace(), backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(httpRoute, crossName), "createing cross-namespace http route")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "Checking number of routes")
	})

	ginkgo.It("Gateway cross-namespace routing with selector", func() {
		crossName := fmt.Sprintf("label-cross-ns-%d", time.Now().Unix())
		crossNamespace := fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    cross-ns-access: "true"
    apisix.ingress.watch: %s
`, crossName, s.Namespace())
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(crossNamespace), "createing cross namespace")
		// setup http bin service
		httpService, err := s.NewHTTPBINWithNamespace(crossName)
		assert.Nil(ginkgo.GinkgoT(), err, "create cross namespace httpbin")
		assert.Nil(ginkgo.GinkgoT(), s.WaitAllHTTPBINPodsAvailable(), "wait cross namespace httpbin")
		defer func() {
			s.DeleteResourceFromString(crossNamespace)
		}()

		httpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: HTTPRoute
metadata:
  name: cross-namespace-route-selector
  namespace: %s
spec:
  parentRefs:
  - name: %s
    namespace: %s
    sectionName: cross-namespace-route
  rules:
  - matches:
    - path:
        value: /get
    backendRefs:
    - name: %s
      port: %d
`, crossName, gatewayName, s.Namespace(), httpService.Name, httpService.Spec.Ports[0].Port)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(httpRoute, crossName), "createing cross-namespace http route")
		time.Sleep(time.Second * 3)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
	})

	ginkgo.It("Gateway same namespace routing", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		httpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: HTTPRoute
metadata:
  name: same-namespace-route
  namespace: %s
spec:
  parentRefs:
  - name: %s
    namespace: %s
    sectionName: same-namespace-route
  hostnames: ["gate.apisix.com"]
  rules:
  - matches:
    - path:
        value: /get
    backendRefs:
    - name: %s
      port: %d
`, s.Namespace(), gatewayName, s.Namespace(), backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpRoute), "createing same-namespace http route")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
	})

	ginkgo.It("Gateway ParentRef GroupKind limit", func() {
		// HTTPRoute can't attach to tcp-route-only Gateway Listener
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		httpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: HTTPRoute
metadata:
  name: attach-to-tcp-listener
  namespace: %s
spec:
  parentRefs:
  - name: %s
    namespace: %s
    sectionName: tcp-route-only
  rules:
  - matches:
    - path:
        value: /get
    backendRefs:
    - name: %s
      port: %d
`, s.Namespace(), gatewayName, s.Namespace(), backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpRoute), "createing http route")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "Checking number of routes")

		tcpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: tcp-route
spec:
  parentRefs:
  - name: %s
    sectionName: tcp-route-only
  rules:
  - backendRefs:
    - name: %s
      port: %d
`, gatewayName, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(tcpRoute), "createing tcp route")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(1), "Checking number of routes")
	})

	ginkgo.It("Gateway Listener hostname match", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		// hostname mismatch case
		mismatchHttpRoute := fmt.Sprintf(`
        apiVersion: gateway.networking.k8s.io/v1alpha2
        kind: HTTPRoute
        metadata:
          name: mismatch-route
          namespace: %s
        spec:
          parentRefs:
          - name: %s
            namespace: %s
            sectionName: hostname-match-route
          hostnames: ["gateway.sample.com"]
          rules:
          - matches:
            - path:
                value: /get
            backendRefs:
            - name: %s
              port: %d
        `, s.Namespace(), gatewayName, s.Namespace(), backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(mismatchHttpRoute), "createing http route")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "Checking number of routes")

		// hostname match case
		matchHttpRoute := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: HTTPRoute
metadata:
  name: match-route
  namespace: %s
spec:
  parentRefs:
  - name: %s
    namespace: %s
    sectionName: hostname-match-route
  hostnames: ["gateway.apisix.com"]
  rules:
  - matches:
    - path:
        value: /get
    backendRefs:
    - name: %s
      port: %d
`, s.Namespace(), gatewayName, s.Namespace(), backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(matchHttpRoute), "createing http route")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
	})
})
