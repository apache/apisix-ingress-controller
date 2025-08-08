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

package v1alpha1

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test BackendTrafficPolicy base on HTTPRoute", Label("apisix.apache.org", "v1alpha1", "backendtrafficpolicy"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: fmt.Sprintf("apisix.apache.org/apisix-ingress-controller-%d", time.Now().Unix()),
		})
		err error
	)

	var defaultGatewayProxy = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

	var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

	var defaultGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: %s
`

	var defaultHTTPRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - "httpbin.org"
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    - path:
        type: Exact
        value: /headers
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
	Context("Rewrite Upstream Host", func() {
		var createUpstreamHost = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.example.com
`

		var updateUpstreamHost = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.update.example.com
`

		BeforeEach(func() {
			gatewayName := s.Namespace()
			By("create GatewayProxy")
			gatewayProxyName := gatewayName
			err = s.CreateResourceFromString(fmt.Sprintf(defaultGatewayProxy, gatewayProxyName, s.Deployer.GetAdminEndpoint(), s.AdminKey()))
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(time.Second)

			By("create GatewayClass")
			gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
			err = s.CreateResourceFromString(fmt.Sprintf(defaultGatewayClass, gatewayClassName, s.GetControllerName()))
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(time.Second)

			By("create Gateway")
			err = s.CreateResourceFromString(fmt.Sprintf(defaultGateway, gatewayName, gatewayClassName, gatewayProxyName))
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(time.Second)

			By("create HTTPRoute")
			s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, fmt.Sprintf(defaultHTTPRoute, gatewayName, s.Namespace()))
		})
		It("should rewrite upstream host", func() {
			s.ResourceApplied("BackendTrafficPolicy", "httpbin", createUpstreamHost, 1)
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"Host": "httpbin.org",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyContains(
						"httpbin.example.com",
					),
				},
			})

			s.ResourceApplied("BackendTrafficPolicy", "httpbin", updateUpstreamHost, 2)
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"Host": "httpbin.org",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyContains(
						"httpbin.update.example.com",
					),
				},
			})

			err := s.DeleteResourceFromString(createUpstreamHost)
			Expect(err).NotTo(HaveOccurred(), "deleting BackendTrafficPolicy")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"Host": "httpbin.org",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyNotContains(
						"httpbin.update.example.com",
						"httpbin.example.com",
					),
				},
			})
		})
	})
})

var _ = Describe("Test BackendTrafficPolicy base on Ingress", Label("apisix.apache.org", "v1alpha1", "backendtrafficpolicy"), func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: fmt.Sprintf("apisix.apache.org/apisix-ingress-controller-%d", time.Now().Unix()),
	})

	var defaultGatewayProxy = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`
	var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-default
  annotations:
    ingressclass.kubernetes.io/is-default-class: "true"
spec:
  controller: "%s"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "%s"
    namespace: "%s"
    scope: "Namespace"
`

	var defaultIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress-default
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`
	var beforeEach = func() {
		By("create GatewayProxy")
		gatewayProxyName := s.Namespace()
		gatewayProxy := fmt.Sprintf(defaultGatewayProxy, gatewayProxyName, s.Deployer.GetAdminEndpoint(), s.AdminKey())
		err := s.CreateResourceFromString(gatewayProxy)
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create IngressClass with GatewayProxy reference")
		err = s.CreateResourceFromString(fmt.Sprintf(defaultIngressClass, s.GetControllerName(), gatewayProxyName, s.Namespace()))
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass with GatewayProxy")

		By("create Ingress with GatewayProxy IngressClass")
		err = s.CreateResourceFromString(defaultIngress)
		Expect(err).NotTo(HaveOccurred(), "creating Ingress with GatewayProxy IngressClass")
	}

	// Tests concerning the default ingress class need to be run serially
	Context("Rewrite Upstream Host", Serial, func() {
		var createUpstreamHost = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
  namespace: %s
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.example.com
`

		var updateUpstreamHost = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
  namespace: %s
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.update.example.com
`

		BeforeEach(beforeEach)
		It("should rewrite upstream host", func() {
			reqAssert := &scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"Host": "httpbin.org",
				},
			}

			s.ResourceApplied("BackendTrafficPolicy", "httpbin", fmt.Sprintf(createUpstreamHost, s.Namespace()), 1)
			s.RequestAssert(reqAssert.SetChecks(
				scaffold.WithExpectedStatus(200),
				scaffold.WithExpectedBodyContains("httpbin.example.com"),
			))

			s.ResourceApplied("BackendTrafficPolicy", "httpbin", fmt.Sprintf(updateUpstreamHost, s.Namespace()), 2)
			s.RequestAssert(reqAssert.SetChecks(
				scaffold.WithExpectedStatus(200),
				scaffold.WithExpectedBodyContains("httpbin.update.example.com"),
			))

			err := s.DeleteResourceFromString(fmt.Sprintf(createUpstreamHost, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "deleting BackendTrafficPolicy")

			s.RequestAssert(reqAssert.SetChecks(
				scaffold.WithExpectedStatus(200),
				scaffold.WithExpectedBodyNotContains(
					"httpbin.update.example.com",
					"httpbin.example.com",
				),
			))
		})
	})
})
