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

package gatewayapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test HTTPRoute", Label("networking.k8s.io", "httproute"), func() {
	s := scaffold.NewDefaultScaffold()

	var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      service:
        name: %s
        port: 9180
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`
	getGatewayProxySpec := func() string {
		return fmt.Sprintf(gatewayProxyYaml, s.Namespace(), framework.ProviderType, s.AdminKey())
	}

	var gatewayClassYaml = `
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
	var defaultGatewayHTTPS = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTPS
      port: 443
      hostname: api6.com
      tls:
        certificateRefs:
        - kind: Secret
          group: ""
          name: test-apisix-tls
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: %s
`

	var beforeEachHTTP = func() {
		Expect(s.CreateResourceFromStringWithNamespace(getGatewayProxySpec(), s.Namespace())).
			NotTo(HaveOccurred(), "creating GatewayProxy")

		gatewayClassName := s.Namespace()
		Expect(s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, gatewayClassName, s.GetControllerName()), "")).
			NotTo(HaveOccurred(), "creating GatewayClass")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("GatewayClass", gatewayClassName)
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"),
			),
			"check GatewayClass condition",
		)
		gatewayName := s.Namespace()
		Expect(s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGateway, gatewayName, gatewayClassName, s.Namespace()), s.Namespace())).
			NotTo(HaveOccurred(), "creating Gateway")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("Gateway", gatewayName)
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controlle"),
			),
			"check Gateway condition status",
		)
	}

	var beforeEachHTTPS = func() {
		By("create GatewayProxy")
		err := s.CreateResourceFromStringWithNamespace(getGatewayProxySpec(), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		secretName := _secretName
		createSecret(s, secretName)

		By("create GatewayClass")
		gatewayClassName := s.Namespace()
		Expect(s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, gatewayClassName, s.GetControllerName()), "")).
			NotTo(HaveOccurred(), "creating GatewayClass")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("GatewayClass", gatewayClassName)
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"),
			),
			"check GatewayClass condition",
		)

		By("create Gateway")
		gatewayName := s.Namespace()
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGatewayHTTPS, gatewayName, gatewayClassName, s.Namespace()), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("Gateway", gatewayName)
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controlle"),
			),
			"check Gateway condition status",
		)
	}

	Context("HTTPRoute with HTTPS Gateway", func() {
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - api6.com
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		BeforeEach(beforeEachHTTPS)

		It("Create/Updtea/Delete HTTPRoute", func() {
			By("create HTTPRoute")
			gatewayName := s.Namespace()
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(exactRouteByGet, gatewayName), 1)

			By("access dataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "api6.com",
				Check:    scaffold.WithExpectedStatus(200),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("delete HTTPRoute")
			err := s.DeleteResourceFromString(fmt.Sprintf(exactRouteByGet, gatewayName))
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "api6.com",
				Check:  scaffold.WithExpectedStatus(404),
			})
		})
	})

	Context("HTTPRoute with Multiple Gateway", Serial, func() {
		var additionalGatewayGroupID string
		var additionalSvc *corev1.Service
		var additionalGatewayClassName string

		var additionalGatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: additional-proxy-config
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

		var additionalGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: additional-gateway
spec:
  gatewayClassName: %s
  listeners:
    - name: http-additional
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: additional-proxy-config
`

		// HTTPRoute that references both gateways
		var multiGatewayHTTPRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: multi-gateway-route
spec:
  parentRefs:
  - name: %s
    namespace: %s
  - name: additional-gateway
    namespace: %s
  hostnames:
  - httpbin.example
  - httpbin-additional.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		BeforeEach(func() {
			beforeEachHTTP()

			By("Create additional gateway group")
			var err error
			additionalGatewayGroupID, additionalSvc, err = s.Deployer.CreateAdditionalGateway("multi-gw")
			Expect(err).NotTo(HaveOccurred(), "creating additional gateway group")

			By("Create additional GatewayProxy")
			// Get admin key for the additional gateway group
			resources, exists := s.GetAdditionalGateway(additionalGatewayGroupID)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")

			By("Create additional GatewayClass")
			additionalGatewayClassName = fmt.Sprintf("additional-gatewayclass-%d", time.Now().Nanosecond())
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, additionalGatewayClassName, s.GetControllerName()), "")
			Expect(err).NotTo(HaveOccurred(), "creating additional GatewayClass")

			By("Check additional GatewayClass condition")
			s.RetryAssertion(func() string {
				gcyaml, _ := s.GetResourceYaml("GatewayClass", additionalGatewayClassName)
				return gcyaml
			}).Should(
				And(
					ContainSubstring(`status: "True"`),
					ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"),
				),
			)

			additionalGatewayProxy := fmt.Sprintf(additionalGatewayProxyYaml, s.Deployer.GetAdminEndpoint(resources.DataplaneService), resources.AdminAPIKey)
			err = s.CreateResourceFromStringWithNamespace(additionalGatewayProxy, resources.DataplaneService.Namespace)
			Expect(err).NotTo(HaveOccurred(), "creating additional GatewayProxy")

			By("Create additional Gateway")
			err = s.CreateResourceFromStringWithNamespace(
				fmt.Sprintf(additionalGateway, additionalGatewayClassName),
				additionalSvc.Namespace,
			)
			Expect(err).NotTo(HaveOccurred(), "creating additional Gateway")
		})

		It("HTTPRoute should be accessible through both gateways", func() {
			By("Create HTTPRoute referencing both gateways")
			multiGatewayRoute := fmt.Sprintf(multiGatewayHTTPRoute, s.Namespace(), s.Namespace(), additionalSvc.Namespace)
			s.ResourceApplied("HTTPRoute", "multi-gateway-route", multiGatewayRoute, 1)

			By("Access through default gateway")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("Access through additional gateway")
			client, err := s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating client for additional gateway")

			s.RequestAssert(&scaffold.RequestAssert{
				Client:   client,
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin-additional.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("Delete Additional Gateway")
			err = s.DeleteResourceFromStringWithNamespace(fmt.Sprintf(additionalGateway, additionalGatewayClassName), additionalSvc.Namespace)
			Expect(err).NotTo(HaveOccurred(), "deleting additional Gateway")

			By("HTTPRoute should still be accessible through default gateway")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("HTTPRoute should not be accessible through additional gateway")
			client, err = s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating client for additional gateway")

			s.RequestAssert(&scaffold.RequestAssert{
				Client: client,
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin-additional.example",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})
		})
	})

	Context("HTTPRoute Base", func() {
		var httprouteWithExternalName = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: httpbin-service-e2e-test
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.external
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-external-domain
      port: 80
`
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		var exactRouteByGet2 = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin2
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin2.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		var invalidBackendPort = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-multiple-port
  namespace: %s
spec:
  selector:
    app: httpbin-deployment-e2e-test
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
    - name: invalid
      port: 10031
      protocol: TCP
      targetPort: 10031
    - name: http2
      port: 8080
      protocol: TCP
      targetPort: 80
  type: ClusterIP
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-multiple-port
      port: 80
`

		BeforeEach(beforeEachHTTP)

		It("Create/Update/Delete HTTPRoute", func() {
			gatewayName := s.Namespace()
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(exactRouteByGet, gatewayName), 1)

			By("access dataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			Expect(s.DeleteResourceFromString(fmt.Sprintf(exactRouteByGet, gatewayName))).
				NotTo(HaveOccurred(), "deleting HTTPRoute")

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("Delete Gateway after apply HTTPRoute", func() {
			gatewayName := s.Namespace()
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(exactRouteByGet, gatewayName), 1)

			By("access dataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			Expect(s.DeleteResource("Gateway", gatewayName)).
				NotTo(HaveOccurred(), "deleting Gateway")

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("Proxy External Service", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(httprouteWithExternalName, s.Namespace(), s.Namespace()), 1)

			By("checking the external service response")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.external",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("Match Port", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(invalidBackendPort, s.Namespace(), s.Namespace(), s.Namespace()), 1)

			s.RetryAssertion(func() error {
				serviceResources, err := s.DefaultDataplaneResource().Service().List(context.Background())
				if err != nil {
					return errors.Wrap(err, "listing services")
				}
				if len(serviceResources) != 1 {
					return fmt.Errorf("expected 1 service, got %d", len(serviceResources))
				}

				serviceResource := serviceResources[0]
				nodes := serviceResource.Upstream.Nodes
				if len(nodes) != 1 {
					return fmt.Errorf("expected 1 node, got %d", len(nodes))
				}
				if nodes[0].Port != 80 {
					return fmt.Errorf("expected node port 80, got %d", nodes[0].Port)
				}
				return nil
			}).Should(Succeed(), "checking service port")
		})

		It("Delete HTTPRoute during restart", func() {
			By("create HTTPRoute httpbin")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(exactRouteByGet, s.Namespace()), 1)

			By("create HTTPRoute httpbin2")
			s.ResourceApplied("HTTPRoute", "httpbin2", fmt.Sprintf(exactRouteByGet2, s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin2.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.Deployer.ScaleIngress(0)

			Expect(s.DeleteResource("HTTPRoute", "httpbin2")).
				NotTo(HaveOccurred(), "deleting HTTPRoute httpbin2")

			s.Deployer.ScaleIngress(1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/get",
				Host:    "httpbin.example",
				Timeout: 1 * time.Minute,
				Check:   scaffold.WithExpectedStatus(http.StatusOK),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin2.example",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})
	})

	Context("HTTPRoute Rule Match", func() {
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		var varsRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
      headers:
        - type: Exact
          name: X-Route-Name
          value: httpbin
    # name: get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		const httpRoutePolicy = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
    # sectionName: get
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin-1
    sectionName: get
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
  - - arg_hrp_name
    - ==
    - http-route-policy-0
`

		var prefixRouteByStatus = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: PathPrefix
        value: /status
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var methodRouteGETAndDELETEByAnything = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /anything
      method: GET
    - path:
        type: Exact
        value: /anything
      method: DELETE
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		BeforeEach(beforeEachHTTP)

		It("HTTPRoute Exact Match", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(exactRouteByGet, s.Namespace(), s.Namespace()), 1)

			By("access daataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get/xxx",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoute Prefix Match", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(prefixRouteByStatus, s.Namespace()), 1)

			By("access daataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/status/200",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/status/201",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusCreated),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoute Method Match", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(methodRouteGETAndDELETEByAnything, s.Namespace()), 1)

			By("access daataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/anything",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "DELETE",
				Path:     "/anything",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "POST",
				Path:     "/anything",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoute Vars Match", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(varsRoute, s.Namespace(), s.Namespace()), 1)

			By("access dataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoutePolicy in effect", func() {
			By("create HTTPRoute")
			s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, fmt.Sprintf(varsRoute, s.Namespace(), s.Namespace()))
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("create HTTPRoutePolicy")
			s.ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: s.Namespace()},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				httpRoutePolicy,
			)

			By("access dataplane to check the HTTPRoutePolicy")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Query: map[string]any{
					"hrp_name": "http-route-policy-0",
				},
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
					"X-HRP-Name":   "http-route-policy-0",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("update HTTPRoutePolicy")
			const changedHTTPRoutePolicy = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
    # sectionName: get
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - new-hrp-name
`
			s.ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: s.Namespace()},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				changedHTTPRoutePolicy,
			)

			// use the old vars cannot match any route
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Query: map[string]any{
					"hrp_name": "http-route-policy-0",
				},
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
					"X-HRP-Name":   "http-route-policy-0",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			// use the new vars can match the route
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
					"X-HRP-Name":   "new-hrp-name",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("delete the HTTPRoutePolicy")
			err := s.DeleteResource("HTTPRoutePolicy", "http-route-policy-0")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoutePolicy")
			Eventually(func() string {
				_, err := s.GetResourceYaml("HTTPRoutePolicy", "http-route-policy-0")
				return err.Error()
			}).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(ContainSubstring(`httproutepolicies.apisix.apache.org "http-route-policy-0" not found`))
			// access the route without additional vars should be OK
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoutePolicy conflicts", func() {
			const httpRoutePolicy0 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
			const httpRoutePolicy1 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-1
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
			const httpRoutePolicy1Priority20 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-1
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  priority: 20
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
			const httpRoutePolicy2 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-2
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: httpbin-1
  priority: 30
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
			By("create HTTPRoute")
			s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, fmt.Sprintf(varsRoute, s.Namespace(), s.Namespace()))

			By("create HTTPRoutePolices")
			for name, spec := range map[string]string{
				"http-route-policy-0": httpRoutePolicy0,
				"http-route-policy-1": httpRoutePolicy1,
				"http-route-policy-2": httpRoutePolicy2,
			} {
				s.ApplyHTTPRoutePolicy(
					types.NamespacedName{Namespace: s.Namespace(), Name: s.Namespace()},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					spec,
					metav1.Condition{
						Type: string(v1alpha2.PolicyConditionAccepted),
					},
				)
			}
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1", "http-route-policy-2"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: s.Namespace()},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionFalse,
						Reason: string(v1alpha2.PolicyReasonConflicted),
					},
				)
			}

			// assert that conflict policies are not in effect
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("delete HTTPRoutePolicies")
			err := s.DeleteResource("HTTPRoutePolicy", "http-route-policy-2")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoutePolicy %s", "http-route-policy-2")
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: s.Namespace()},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionTrue,
						Reason: string(v1alpha2.PolicyReasonAccepted),
					},
				)
			}
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("update HTTPRoutePolicy")
			err = s.CreateResourceFromStringWithNamespace(httpRoutePolicy1Priority20, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "update HTTPRoutePolicy's priority to 20")
			framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
				types.NamespacedName{Namespace: s.Namespace(), Name: s.Namespace()},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-1"},
				metav1.Condition{
					Type: string(v1alpha2.PolicyConditionAccepted),
				},
			)
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: s.Namespace()},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionFalse,
						Reason: string(v1alpha2.PolicyReasonConflicted),
					},
				)
			}

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoutePolicy status changes on HTTPRoute deleting", func() {
			By("create HTTPRoute")
			s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, fmt.Sprintf(varsRoute, s.Namespace(), s.Namespace()))

			By("create HTTPRoutePolicy")
			s.ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: s.Namespace()},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				httpRoutePolicy,
			)

			By("access dataplane to check the HTTPRoutePolicy")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Query: map[string]any{
					"hrp_name": "http-route-policy-0",
				},
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
					"X-HRP-Name":   "http-route-policy-0",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("delete the HTTPRoute, assert the HTTPRoutePolicy's status will be changed")
			Expect(s.DeleteResource("HTTPRoute", "httpbin")).
				NotTo(HaveOccurred(), "deleting HTTPRoute")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Query: map[string]any{
					"hrp_name": "http-route-policy-0",
				},
				Headers: map[string]string{
					"X-Route-Name": "httpbin",
					"X-HRP-Name":   "http-route-policy-0",
				},
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			err := framework.PollUntilHTTPRoutePolicyHaveStatus(s.K8sClient, 8*time.Second, types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				func(hrp *v1alpha1.HTTPRoutePolicy) bool {
					return len(hrp.Status.Ancestors) == 0
				},
			)
			Expect(err).NotTo(HaveOccurred(), "HTPRoutePolicy.Status should has no ancestor")
		})
	})

	Context("HTTPRoute Filters", func() {
		var reqHeaderModifyByHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: RequestHeaderModifier
      requestHeaderModifier:
        add:
        - name: X-Req-Add
          value: "add"
        set:
        - name: X-Req-Set
          value: "set"
        remove:
        - X-Req-Removed
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var respHeaderModifyByHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: ResponseHeaderModifier
      responseHeaderModifier:
        add:
        - name: X-Resp-Add
          value: "add"
        set:
        - name: X-Resp-Set
          value: "set"
        remove:
        - Server
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var httpsRedirectByHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: RequestRedirect
      requestRedirect:
        scheme: https
        port: 9443
`

		var hostnameRedirectByHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: RequestRedirect
      requestRedirect:
        hostname: httpbin.org
        statusCode: 301
`

		var replacePrefixMatch = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: PathPrefix
        value: /replace
    filters:
    - type: URLRewrite
      urlRewrite:
        path:
          type: ReplacePrefixMatch
          replacePrefixMatch: /status
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var replaceFullPathAndHost = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: PathPrefix
        value: /replace
    filters:
    - type: URLRewrite
      urlRewrite:
        hostname: replace.example.org
        path:
          type: ReplaceFullPath
          replaceFullPath: /headers
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var echoPlugin = `
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: example-plugin-config
spec:
  plugins:
  - name: echo
    config:
      body: "Hello, World!!"
`
		var echoPluginUpdated = `
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: example-plugin-config
spec:
  plugins:
  - name: echo
    config:
      body: "Updated"
`
		var extensionRefEchoPlugin = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    filters:
    - type: ExtensionRef
      extensionRef:
        group: apisix.apache.org
        kind: PluginConfig
        name: example-plugin-config
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		BeforeEach(beforeEachHTTP)

		It("HTTPRoute RequestHeaderModifier", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(reqHeaderModifyByHeaders, s.Namespace(), s.Namespace()), 1)

			By("access daataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.example",
				Headers: map[string]string{
					"X-Req-Add":     "test",
					"X-Req-Removed": "test",
					"X-Req-Set":     "test",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains(`"X-Req-Add": "test,add"`, `"X-Req-Set": "set"`),
					scaffold.WithExpectedBodyNotContains(`"X-Req-Removed": "remove"`),
				},
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoute ResponseHeaderModifier", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(respHeaderModifyByHeaders, s.Namespace(), s.Namespace()), 1)

			By("access daataplane to check the HTTPRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeaders(map[string]string{
						"X-Resp-Add": "add",
						"X-Resp-Set": "set",
						"Server":     "",
					}),
					scaffold.WithExpectedBodyNotContains(`"X-Resp-Add": "add"`, `"X-Resp-Set": "set"`, `"Server"`),
				},
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoute RequestRedirect", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(httpsRedirectByHeaders, s.Namespace(), s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusFound),
					scaffold.WithExpectedHeader("Location", "https://httpbin.example:9443/headers"),
				},
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("update HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(hostnameRedirectByHeaders, s.Namespace(), s.Namespace()), 2)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusMovedPermanently),
					scaffold.WithExpectedHeader("Location", "http://httpbin.org/headers"),
				},
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoute RequestMirror", func() {
			echoRoute := `
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
        image: jmalloc/echo-server:latest
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
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /headers
    filters:
    - type: RequestMirror
      requestMirror:
        backendRef:
          name: echo-service
          port: 80
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(echoRoute, s.Namespace(), s.Namespace()), 1)

			s.RetryAssertion(func() string {
				resp := s.NewAPISIXClient().GET("/headers").WithHost("httpbin.example").Expect().Raw()
				if resp.StatusCode != http.StatusOK {
					return fmt.Sprintf("expected status OK, got %d", resp.StatusCode)
				}
				return s.GetDeploymentLogs("echo")
			}).WithTimeout(2 * time.Minute).Should(ContainSubstring("GET /headers"))
		})

		It("HTTPRoute URLRewrite with ReplaceFullPath And Hostname", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(replaceFullPathAndHost, s.Namespace(), s.Namespace()), 1)

			By("/replace/201 should be rewritten to /headers")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/replace/201",
				Host:   "httpbin.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains("replace.example.org"),
				},
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("/replace/500 should be rewritten to /headers")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/replace/500",
				Host:   "httpbin.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains("replace.example.org"),
				},
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoute URLRewrite with ReplacePrefixMatch", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(replacePrefixMatch, s.Namespace(), s.Namespace()), 1)

			By("/replace/201 should be rewritten to /status/201")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/replace/201",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusCreated),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("/replace/500 should be rewritten to /status/500")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/replace/500",
				Host:   "httpbin.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusInternalServerError),
				},
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})

		It("HTTPRoute ExtensionRef", func() {
			By("create HTTPRoute")
			Expect(s.CreateResourceFromStringWithNamespace(echoPlugin, s.Namespace())).
				NotTo(HaveOccurred(), "creating PluginConfig")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(extensionRefEchoPlugin, s.Namespace(), s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedBodyContains("Hello, World!!"),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			Expect(s.CreateResourceFromStringWithNamespace(echoPluginUpdated, s.Namespace())).
				NotTo(HaveOccurred(), "updating PluginConfig")

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedBodyContains("Updated"),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})
	})

	Context("HTTPRoute Multiple Backend", func() {
		var sameWeiht = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
      weight: 50
    - name: nginx
      port: 80
      weight: 50
 `
		var oneWeiht = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
      weight: 100
    - name: nginx
      port: 80
      weight: 0
 `

		BeforeEach(func() {
			beforeEachHTTP()
			s.DeployNginx(framework.NginxOptions{
				Namespace: s.Namespace(),
			})
		})
		It("HTTPRoute Canary", func() {
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(sameWeiht, s.Namespace()), 1)
			time.Sleep(5 * time.Second)

			s.RetryAssertion(func() int {
				var (
					hitNginxCnt   = 0
					hitHttpbinCnt = 0
				)
				for range 20 {
					resp := s.NewAPISIXClient().GET("/get").
						WithHeader("Host", "httpbin.example").
						Expect()
					body := resp.Body().Raw()
					status := resp.Raw().StatusCode
					if status != http.StatusOK {
						return -100
					}

					if strings.Contains(body, "Hello") {
						hitNginxCnt++
					} else {
						hitHttpbinCnt++
					}
				}
				return hitNginxCnt - hitHttpbinCnt
			}).WithTimeout(2 * time.Minute).Should(BeNumerically("~", 0, 2))

			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(oneWeiht, s.Namespace()), 2)

			s.RetryAssertion(func() int {
				var (
					hitNginxCnt   = 0
					hitHttpbinCnt = 0
				)
				for range 20 {
					resp := s.NewAPISIXClient().GET("/get").
						WithHeader("Host", "httpbin.example").
						Expect()
					body := resp.Body().Raw()
					status := resp.Raw().StatusCode
					if status != http.StatusOK {
						return -100
					}

					if strings.Contains(body, "Hello") {
						hitNginxCnt++
					} else {
						hitHttpbinCnt++
					}
				}
				return hitHttpbinCnt - hitNginxCnt
			}).WithTimeout(2 * time.Minute).Should(Equal(20))
		})
	})

	Context("HTTPRoute with GatewayProxy Update", func() {
		var additionalGatewayGroupID string

		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var updatedGatewayProxy = `
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

		BeforeEach(beforeEachHTTP)

		It("Should sync HTTPRoute when GatewayProxy is updated", func() {
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(exactRouteByGet, s.Namespace()), 1)

			By("verify HTTPRoute works")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("create additional gateway group to get new admin key")
			var err error
			additionalGatewayGroupID, _, err = s.Deployer.CreateAdditionalGateway("gateway-proxy-update")
			Expect(err).NotTo(HaveOccurred(), "creating additional gateway group")

			resources, exists := s.GetAdditionalGateway(additionalGatewayGroupID)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")

			client, err := s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating APISIX client for additional gateway group")

			By("HTTPRoute not found for additional gateway group")
			s.RequestAssert(&scaffold.RequestAssert{
				Client:   client,
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("update GatewayProxy with new admin key")
			updatedProxy := fmt.Sprintf(updatedGatewayProxy, s.Namespace(), s.Deployer.GetAdminEndpoint(resources.DataplaneService), resources.AdminAPIKey)
			err = s.CreateResourceFromStringWithNamespace(updatedProxy, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy")

			By("verify HTTPRoute works for additional gateway group")
			s.RequestAssert(&scaffold.RequestAssert{
				Client:   client,
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin.example",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})
		})
	})

	Context("Test HTTPRoute Load Balancing", func() {
		BeforeEach(beforeEachHTTP)
		It("Test load balancing with ExternalName services", func() {
			const servicesSpec = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: httpbin-service-e2e-test
---
apiVersion: v1
kind: Service
metadata:
  name: mockapi7-external-domain
spec:
  type: ExternalName
  externalName: mock.api7.ai
---
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: passhost-node
spec:
  targetRefs:
  - name: httpbin-external-domain
    kind: Service
    group: ""
  - name: mockapi7-external-domain
    kind: Service
    group: ""
  passHost: node
  scheme: http
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: lb-route
spec:
  parentRefs:
  - name: %s
  rules:
  - matches:
    - path:
        type: Exact
        value: /headers
    backendRefs:
    - name: httpbin-external-domain
      port: 80
      weight: 1
    - name: mockapi7-external-domain
      port: 80
      weight: 1
`

			By("apply services and HTTPRoute")
			err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(servicesSpec, s.Namespace()), s.Namespace())
			Expect(err).ShouldNot(HaveOccurred(), "apply services and HTTPRoute")
			time.Sleep(10 * time.Second)

			By("verify load balancing works")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/headers",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Timeout:  1 * time.Minute,
				Interval: 2 * time.Second,
			})
			// Test multiple requests to verify load balancing
			upstreamHosts := make(map[string]int)
			totalRequests := 20

			for range totalRequests {
				statusCode := s.NewAPISIXClient().GET("/headers").Expect().Raw().StatusCode
				Expect(statusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusMovedPermanently)))

				switch statusCode {
				case http.StatusOK:
					upstreamHosts["httpbin-service-e2e-test"]++
				case http.StatusMovedPermanently:
					upstreamHosts["mock.api7.ai"]++
				}
				time.Sleep(100 * time.Millisecond) // Small delay between requests
			}

			By("verify both upstreams received requests")
			Expect(upstreamHosts).Should(HaveLen(2))

			for host, count := range upstreamHosts {
				Expect(count).Should(BeNumerically(">", 0), fmt.Sprintf("upstream %s should receive requests", host))
			}
		})
	})

	Context("Test HTTPRoute sync during startup", func() {
		BeforeEach(beforeEachHTTP)
		var route = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var route2 = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin2
spec:
  parentRefs:
  - name: apisix-nonexistent
  hostnames:
  - httpbin2
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		It("Should sync ApisixRoute during startup", func() {
			By("apply ApisixRoute")
			Expect(s.CreateResourceFromStringWithNamespace(route2, s.Namespace())).ShouldNot(HaveOccurred(), "applying HTTPRoute with non-existent parent")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(route, s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Interval: time.Second * 2,
				Timeout:  30 * time.Second,
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin2",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Interval: time.Second * 2,
				Timeout:  30 * time.Second,
			})

			time.Sleep(8 * time.Second)
			By("restart controller and dataplane")
			s.Deployer.ScaleIngress(0)
			s.Deployer.ScaleDataplane(0)
			s.Deployer.ScaleDataplane(1)
			s.Deployer.ScaleIngress(1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin",
				Check:    scaffold.WithExpectedStatus(http.StatusOK),
				Interval: time.Second * 2,
				Timeout:  30 * time.Second,
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method:   "GET",
				Path:     "/get",
				Host:     "httpbin2",
				Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
				Interval: time.Second * 2,
				Timeout:  30 * time.Second,
			})
		})

	})
})
