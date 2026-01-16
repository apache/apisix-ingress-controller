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
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

// helper to apply an HTTPRoute and run a list of request assertions
func applyHTTPRouteAndAssert(s *scaffold.Scaffold, route string, asserts []scaffold.RequestAssert) {
	s.ResourceApplied("HTTPRoute", "httpbin", route, 1)
	for i := range asserts {
		s.RequestAssert(&asserts[i])
	}
}

var _ = Describe("Test HTTPRoute", Label("networking.k8s.io", "httproute"), func() {
	s := scaffold.NewDefaultScaffold()

	var gatewayClassYaml = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
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
      name: apisix-proxy-config
`

	var beforeEachHTTP = func() {
		By("create GatewayProxy")
		Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create GatewayClass")
		Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred(), "creating GatewayClass")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("GatewayClass", s.Namespace())
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"),
			),
			"check GatewayClass condition",
		)

		By("create Gateway")
		Expect(s.CreateResourceFromString(s.GetGatewayYaml())).NotTo(HaveOccurred(), "creating Gateway")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("Gateway", s.Namespace())
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"),
			),
			"check Gateway condition status",
		)
	}

	var beforeEachHTTPS = func() {
		By("create GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		secretName := _secretName
		createSecret(s, secretName)

		By("create GatewayClass")
		Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred(), "creating GatewayClass")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("GatewayClass", s.Namespace())
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"),
			),
			"check GatewayClass condition",
		)

		By("create Gateway")
		err = s.CreateResourceFromString(fmt.Sprintf(defaultGatewayHTTPS, s.Namespace(), s.Namespace()))
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("Gateway", s.Namespace())
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"),
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
		It("HTTPRoute with multiple hostnames", func() {
			route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example
  - httpbin2.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`, s.Namespace())

			asserts := []scaffold.RequestAssert{
				{
					Method:   "GET",
					Path:     "/get",
					Host:     "httpbin.example",
					Check:    scaffold.WithExpectedStatus(http.StatusOK),
					Timeout:  30 * time.Second,
					Interval: 2 * time.Second,
				},
				{
					Method:   "GET",
					Path:     "/get",
					Host:     "httpbin2.example",
					Check:    scaffold.WithExpectedStatus(http.StatusOK),
					Timeout:  30 * time.Second,
					Interval: 2 * time.Second,
				},
				{
					Method:   "GET",
					Path:     "/get",
					Host:     "httpbin3.example",
					Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
					Timeout:  30 * time.Second,
					Interval: 2 * time.Second,
				},
			}

			applyHTTPRouteAndAssert(s, route, asserts)
		})

		It("HTTPRoute with multiple matches in one rule", func() {
			route := fmt.Sprintf(`
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
    - path:
        type: Exact
        value: /ip
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`, s.Namespace())

			asserts := []scaffold.RequestAssert{
				{
					Method:   "GET",
					Path:     "/get",
					Host:     "httpbin.example",
					Check:    scaffold.WithExpectedStatus(http.StatusOK),
					Timeout:  30 * time.Second,
					Interval: 2 * time.Second,
				},
				{
					Method:   "GET",
					Path:     "/ip",
					Host:     "httpbin.example",
					Check:    scaffold.WithExpectedStatus(http.StatusOK),
					Timeout:  30 * time.Second,
					Interval: 2 * time.Second,
				},
				{
					Method:   "GET",
					Path:     "/status",
					Host:     "httpbin.example",
					Check:    scaffold.WithExpectedStatus(http.StatusNotFound),
					Timeout:  30 * time.Second,
					Interval: 2 * time.Second,
				},
			}

			applyHTTPRouteAndAssert(s, route, asserts)
		})

		It("Service Endpoints changed", func() {
			gatewayName := s.Namespace()
			By("create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "httpbin", fmt.Sprintf(exactRouteByGet, gatewayName), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})

			By("scale httpbin deployment to 0")
			err := s.ScaleHTTPBIN(0)
			Expect(err).NotTo(HaveOccurred(), "scaling httpbin deployment to 0")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusServiceUnavailable),
			})

			By("scale httpbin deployment to 1")
			err = s.ScaleHTTPBIN(1)
			Expect(err).NotTo(HaveOccurred(), "scaling httpbin deployment to 1")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
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

		var respHeaderModifyWithAdd = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: add
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example.resp-header-modify.add
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
          value: "resp-add"
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var respHeaderModifyWithSet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: set
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example.resp-header-modify.set
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: ResponseHeaderModifier
      responseHeaderModifier:
        set:
        - name: X-Resp-Set
          value: "resp-set"
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var respHeaderModifyWithRemove = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: remove
spec:
  parentRefs:
  - name: %s
  hostnames:
  - httpbin.example.resp-header-modify.remove
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
    filters:
    - type: ResponseHeaderModifier
      responseHeaderModifier:
        remove:
        - Server
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

		var corsTestService = `
apiVersion: v1
kind: Service
metadata:
  name: cors-test-service
spec:
  selector:
    app: cors-test
  ports:
  - port: 80
    targetPort: 5678
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cors-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cors-test
  template:
    metadata:
      labels:
        app: cors-test
    spec:
      containers:
      - name: cors-test
        image: hashicorp/http-echo
        args: ["-text=hello", "-listen=:5678"]
        ports:
        - containerPort: 5678
`

		var corsFilter = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: http-route-cors
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - cors-test.example
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    filters:
    - type: CORS
      cors:
        allowOrigins:
        - http://example.com
        allowMethods: 
        - GET
        - POST
        - PUT
        - DELETE
        allowHeaders: 
        - "Origin"
        exposeHeaders: 
        - "Origin"
        allowCredentials: true
    backendRefs:
    - name: cors-test-service
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
			s.ResourceApplied("HTTPRoute", "add", fmt.Sprintf(respHeaderModifyWithAdd, s.Namespace()), 1)
			s.ResourceApplied("HTTPRoute", "set", fmt.Sprintf(respHeaderModifyWithSet, s.Namespace()), 1)
			s.ResourceApplied("HTTPRoute", "remove", fmt.Sprintf(respHeaderModifyWithRemove, s.Namespace()), 1)

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
					}),
					scaffold.WithExpectedNotHeader("Server"),
					scaffold.WithExpectedBodyNotContains(`"X-Resp-Add": "add"`, `"X-Resp-Set": "set"`, `"Server"`),
				},
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.example.resp-header-modify.add",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeader("X-Resp-Add", "resp-add"),
					scaffold.WithExpectedBodyNotContains(`"X-Resp-Add": "resp-add"`),
				},
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.example.resp-header-modify.set",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeader("X-Resp-Set", "resp-set"),
					scaffold.WithExpectedBodyNotContains(`"Server"`),
				},
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.example.resp-header-modify.remove",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedNotHeader("Server"),
					scaffold.WithExpectedBodyNotContains(`"Server"`),
				},
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

		It("HTTPRoute CORS Filter", func() {
			By("create test service and deployment")
			Expect(s.CreateResourceFromStringWithNamespace(corsTestService, s.Namespace())).
				NotTo(HaveOccurred(), "creating CORS test service")

			By("create HTTPRoute with CORS filter")
			s.ResourceApplied("HTTPRoute", "http-route-cors", fmt.Sprintf(corsFilter, s.Namespace(), s.Namespace()), 1)
			By("test simple GET request with CORS headers from allowed origin")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/",
				Host:   "cors-test.example",
				Headers: map[string]string{
					"Origin": "http://example.com",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains("hello"),
					scaffold.WithExpectedHeaders(map[string]string{
						"Access-Control-Allow-Origin":      "http://example.com",
						"Access-Control-Allow-Methods":     "GET,POST,PUT,DELETE",
						"Access-Control-Allow-Headers":     "Origin",
						"Access-Control-Expose-Headers":    "Origin",
						"Access-Control-Allow-Credentials": "true",
					}),
				},
				Timeout:  time.Second * 30,
				Interval: time.Second * 2,
			})

			By("test simple GET request with CORS headers from disallowed origin")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/",
				Host:   "cors-test.example",
				Headers: map[string]string{
					"Origin": "http://disallowed.com",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains("hello"),
					scaffold.WithExpectedNotHeader("Access-Control-Allow-Origin"),
				},
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
				Replicas:  ptr.To(int32(2)),
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
  name: apisix-proxy-config
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
			updatedProxy := fmt.Sprintf(updatedGatewayProxy, s.Deployer.GetAdminEndpoint(resources.DataplaneService), resources.AdminAPIKey)
			err = s.CreateResourceFromString(updatedProxy)
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

	Context("Test Service With AppProtocol", func() {
		var (
			httproute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: nginx
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
    - name: nginx
      port: 443
 `
			httprouteWithWSS = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: nginx-wss
spec:
  parentRefs:
  - name: %s
  hostnames:
  - api6.com
  rules:
  - matches:
    - path:
        type: Exact
        value: /ws
    backendRefs:
    - name: nginx
      port: 8443
 `
		)

		BeforeEach(func() {
			beforeEachHTTPS()
			s.DeployNginx(framework.NginxOptions{
				Namespace: s.Namespace(),
				Replicas:  ptr.To(int32(1)),
			})
		})
		It("HTTPS backend", func() {
			s.ResourceApplied("HTTPRoute", "nginx", fmt.Sprintf(httproute, s.Namespace()), 1)
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "api6.com",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
		})

		It("WSS backend", func() {
			s.ResourceApplied("HTTPRoute", "nginx-wss", fmt.Sprintf(httprouteWithWSS, s.Namespace()), 1)

			By("verify wss connection")
			hostname := "api6.com"
			conn := s.NewWebsocketClient(&tls.Config{
				InsecureSkipVerify: true,
				ServerName:         hostname,
			}, "/ws", http.Header{"Host": []string{hostname}})

			defer func() {
				_ = conn.Close()
			}()

			By("send and receive message through WebSocket")
			testMessage := "hello, this is APISIX"
			err := conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
			Expect(err).ShouldNot(HaveOccurred(), "writing WebSocket message")

			// Then our echo
			_, msg, err := conn.ReadMessage()
			Expect(err).ShouldNot(HaveOccurred(), "reading echo message")
			Expect(string(msg)).To(Equal(testMessage), "message content verification")
		})
	})

	Context("HTTPRoute with sectionName targeting different listeners", func() {
		// Uses port 9080 (HTTP) and port 9081 (HTTP)
		// Both ports are already exposed by the APISIX service
		// Uses in-cluster curl to test server_port vars correctly

		var multiListenerGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
    - name: http-main
      protocol: HTTP
      port: 9080
    - name: http-alt
      protocol: HTTP
      port: 9081
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

		var routeForMainListener = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: route-main
spec:
  parentRefs:
  - name: %s
    sectionName: http-main
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var routeForAltListener = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: route-alt
spec:
  parentRefs:
  - name: %s
    sectionName: http-alt
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var routeNoSectionName = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: route-no-section
spec:
  parentRefs:
  - name: %s
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var routeInvalidSectionName = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: route-invalid-section
spec:
  parentRefs:
  - name: %s
    sectionName: non-existent-listener
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		var routeMultiParentRef = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: route-multi-parent
spec:
  parentRefs:
  - name: %s
    sectionName: http-main
  - name: %s
    sectionName: http-alt
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		// Get the APISIX service name from the deployer
		getApisixServiceName := func() string {
			// The APISIX service is named "apisix" (from framework.ProviderType)
			return "apisix"
		}

		// Run curl from within the cluster to the specified port
		curlInCluster := func(port int, path string) (int, string, error) {
			url := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s",
				getApisixServiceName(), s.Namespace(), port, path)

			// Note: curlimages/curl image already has curl as entrypoint, so we don't pass "curl" again
			output, err := s.RunCurlFromK8s("-s", "-o", "/dev/null", "-w", "%{http_code}", url)
			if err != nil {
				return 0, "", err
			}
			statusCode := 0
			fmt.Sscanf(output, "%d", &statusCode)
			return statusCode, output, nil
		}

		BeforeEach(func() {
			By("create GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred())

			By("create GatewayClass")
			Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred())

			s.RetryAssertion(func() string {
				yaml, _ := s.GetResourceYaml("GatewayClass", s.Namespace())
				return yaml
			}).Should(ContainSubstring(`status: "True"`))
		})

		It("routes traffic to correct backend based on sectionName (using server_port vars)", func() {
			gatewayName := s.Namespace()

			By("create Gateway with two listeners on different ports")
			gateway := fmt.Sprintf(multiListenerGateway, gatewayName, s.Namespace())
			Expect(s.CreateResourceFromString(gateway)).NotTo(HaveOccurred())

			s.RetryAssertion(func() string {
				yaml, _ := s.GetResourceYaml("Gateway", gatewayName)
				return yaml
			}).Should(ContainSubstring(`status: "True"`))

			By("create HTTPRoute targeting http-main listener (port 9080)")
			routeMain := fmt.Sprintf(routeForMainListener, gatewayName)
			s.ResourceApplied("HTTPRoute", "route-main", routeMain, 1)

			By("create HTTPRoute targeting http-alt listener (port 9100)")
			routeAlt := fmt.Sprintf(routeForAltListener, gatewayName)
			s.ResourceApplied("HTTPRoute", "route-alt", routeAlt, 1)

			By("wait for routes to be synced")
			time.Sleep(5 * time.Second)

			By("verify route-main is accessible on port 9080 (via in-cluster curl)")
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9080, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK),
				"route should be accessible on port 9080")

			By("verify route-alt is accessible on port 9081 (via in-cluster curl)")
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9081, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK),
				"route should be accessible on port 9081")

			By("delete route-main and verify route-alt still works")
			err := s.DeleteResourceFromString(routeMain)
			Expect(err).NotTo(HaveOccurred())

			// Port 9080 should now return 404 (route deleted)
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9080, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound),
				"route should return 404 on port 9080 after deletion")

			// Port 9081 should still return 200
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9081, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK),
				"route should still return 200 on port 9081")
		})

		It("should match all listeners when sectionName is omitted", func() {
			gatewayName := s.Namespace()

			By("create Gateway with two listeners")
			gateway := fmt.Sprintf(multiListenerGateway, gatewayName, s.Namespace())
			Expect(s.CreateResourceFromString(gateway)).NotTo(HaveOccurred())

			s.RetryAssertion(func() string {
				yaml, _ := s.GetResourceYaml("Gateway", gatewayName)
				return yaml
			}).Should(ContainSubstring(`status: "True"`))

			By("create HTTPRoute WITHOUT sectionName")
			route := fmt.Sprintf(routeNoSectionName, gatewayName)
			s.ResourceApplied("HTTPRoute", "route-no-section", route, 1)

			By("wait for route sync")
			time.Sleep(5 * time.Second)

			By("verify route is accessible on port 9080")
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9080, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK),
				"route should be accessible on port 9080")

			By("verify route is accessible on port 9081")
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9081, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK),
				"route should be accessible on port 9081")
		})

		It("should not route traffic when sectionName references non-existent listener", func() {
			gatewayName := s.Namespace()

			By("create Gateway with two listeners")
			gateway := fmt.Sprintf(multiListenerGateway, gatewayName, s.Namespace())
			Expect(s.CreateResourceFromString(gateway)).NotTo(HaveOccurred())

			s.RetryAssertion(func() string {
				yaml, _ := s.GetResourceYaml("Gateway", gatewayName)
				return yaml
			}).Should(ContainSubstring(`status: "True"`))

			By("create HTTPRoute with invalid sectionName")
			route := fmt.Sprintf(routeInvalidSectionName, gatewayName)
			Expect(s.CreateResourceFromString(route)).NotTo(HaveOccurred())

			By("wait for reconciliation")
			time.Sleep(5 * time.Second)

			By("verify route is NOT accessible on any port (no matching listener)")
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9080, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound),
				"route should not be accessible when sectionName is invalid")
		})

		It("should route to multiple listeners via multiple parentRefs with sectionName", func() {
			gatewayName := s.Namespace()

			By("create Gateway with two listeners")
			gateway := fmt.Sprintf(multiListenerGateway, gatewayName, s.Namespace())
			Expect(s.CreateResourceFromString(gateway)).NotTo(HaveOccurred())

			s.RetryAssertion(func() string {
				yaml, _ := s.GetResourceYaml("Gateway", gatewayName)
				return yaml
			}).Should(ContainSubstring(`status: "True"`))

			By("create HTTPRoute with multiple parentRefs targeting different listeners")
			route := fmt.Sprintf(routeMultiParentRef, gatewayName, gatewayName)
			s.ResourceApplied("HTTPRoute", "route-multi-parent", route, 1)

			By("wait for route sync")
			time.Sleep(5 * time.Second)

			By("verify route is accessible on port 9080")
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9080, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK),
				"route should be accessible on port 9080")

			By("verify route is accessible on port 9081")
			Eventually(func() (int, error) {
				statusCode, _, err := curlInCluster(9081, "/get")
				return statusCode, err
			}).WithTimeout(30*time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK),
				"route should be accessible on port 9081")
		})
	})
})
