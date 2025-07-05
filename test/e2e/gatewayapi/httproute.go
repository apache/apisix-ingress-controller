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

	"github.com/gruntwork-io/terratest/modules/retry"
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
  name: apisix-proxy-config
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
		return fmt.Sprintf(gatewayProxyYaml, framework.ProviderType, s.AdminKey())
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
  name: apisix
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
      name: apisix-proxy-config
`
	var defaultGatewayHTTPS = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
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

	var ResourceApplied = func(resourType, resourceName, resourceRaw string, observedGeneration int) {
		Expect(s.CreateResourceFromString(resourceRaw)).
			NotTo(HaveOccurred(), fmt.Sprintf("creating %s", resourType))

		Eventually(func() string {
			hryaml, err := s.GetResourceYaml(resourType, resourceName)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("getting %s yaml", resourType))
			return hryaml
		}, "8s", "2s").
			Should(
				SatisfyAll(
					ContainSubstring(`status: "True"`),
					ContainSubstring(fmt.Sprintf("observedGeneration: %d", observedGeneration)),
				),
				fmt.Sprintf("checking %s condition status", resourType),
			)
		time.Sleep(5 * time.Second)
	}

	var beforeEachHTTP = func() {
		By("create GatewayProxy")
		err := s.CreateResourceFromString(getGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, gatewayClassName, s.GetControllerName()), "")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(5 * time.Second)

		By("check GatewayClass condition")
		gcyaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
		Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
		Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
		Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"), "checking GatewayClass condition message")

		By("create Gateway")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGateway, gatewayClassName), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(5 * time.Second)

		By("check Gateway condition")
		gwyaml, err := s.GetResourceYaml("Gateway", "apisix")
		Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
		Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
		Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"), "checking Gateway condition message")
	}

	var beforeEachHTTPS = func() {
		By("create GatewayProxy")
		err := s.CreateResourceFromString(getGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		secretName := _secretName
		createSecret(s, secretName)
		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, gatewayClassName, s.GetControllerName()), "")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(5 * time.Second)

		By("check GatewayClass condition")
		gcyaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
		Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
		Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
		Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"), "checking GatewayClass condition message")

		By("create Gateway")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGatewayHTTPS, gatewayClassName), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(5 * time.Second)

		By("check Gateway condition")
		gwyaml, err := s.GetResourceYaml("Gateway", "apisix")
		Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
		Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
		Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"), "checking Gateway condition message")
	}
	Context("HTTPRoute with HTTPS Gateway", func() {
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
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
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("access dataplane to check the HTTPRoute")
			s.NewAPISIXHttpsClient("api6.com").
				GET("/get").
				WithHost("api6.com").
				Expect().
				Status(200)
			By("delete HTTPRoute")
			err := s.DeleteResourceFromString(exactRouteByGet)
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute")
			time.Sleep(5 * time.Second)

			s.NewAPISIXHttpsClient("api6.com").
				GET("/get").
				WithHost("api6.com").
				Expect().
				Status(404)
		})
	})

	Context("HTTPRoute with Multiple Gateway", func() {
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
  - name: apisix
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
			additionalGatewayClassName = fmt.Sprintf("apisix-%d", time.Now().Unix())
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, additionalGatewayClassName, s.GetControllerName()), "")
			Expect(err).NotTo(HaveOccurred(), "creating additional GatewayClass")
			time.Sleep(5 * time.Second)
			By("Check additional GatewayClass condition")
			gcyaml, err := s.GetResourceYaml("GatewayClass", additionalGatewayClassName)
			Expect(err).NotTo(HaveOccurred(), "getting additional GatewayClass yaml")
			Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking additional GatewayClass condition status")
			Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"), "checking additional GatewayClass condition message")

			additionalGatewayProxy := fmt.Sprintf(additionalGatewayProxyYaml, s.Deployer.GetAdminEndpoint(resources.DataplaneService), resources.AdminAPIKey)
			err = s.CreateResourceFromStringWithNamespace(additionalGatewayProxy, resources.DataplaneService.Namespace)
			Expect(err).NotTo(HaveOccurred(), "creating additional GatewayProxy")

			By("Create additional Gateway")
			err = s.CreateResourceFromStringWithNamespace(
				fmt.Sprintf(additionalGateway, additionalGatewayClassName),
				additionalSvc.Namespace,
			)
			Expect(err).NotTo(HaveOccurred(), "creating additional Gateway")
			time.Sleep(5 * time.Second)
		})

		It("HTTPRoute should be accessible through both gateways", func() {
			By("Create HTTPRoute referencing both gateways")
			multiGatewayRoute := fmt.Sprintf(multiGatewayHTTPRoute, s.Namespace(), additionalSvc.Namespace)
			ResourceApplied("HTTPRoute", "multi-gateway-route", multiGatewayRoute, 1)

			By("Access through default gateway")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(http.StatusOK)

			By("Access through additional gateway")
			client, err := s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating client for additional gateway")

			client.
				GET("/get").
				WithHost("httpbin-additional.example").
				Expect().
				Status(http.StatusOK)

			By("Delete Additional Gateway")
			err = s.DeleteResourceFromStringWithNamespace(fmt.Sprintf(additionalGateway, additionalGatewayClassName), additionalSvc.Namespace)
			Expect(err).NotTo(HaveOccurred(), "deleting additional Gateway")
			time.Sleep(5 * time.Second)

			By("HTTPRoute should still be accessible through default gateway")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(http.StatusOK)

			By("HTTPRoute should not be accessible through additional gateway")
			client, err = s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating client for additional gateway")

			client.
				GET("/get").
				WithHost("httpbin-additional.example").
				Expect().
				Status(http.StatusNotFound)
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
  externalName: postman-echo.com
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
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
  - name: apisix
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
  - name: apisix
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
spec:
  parentRefs:
  - name: apisix
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
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("access dataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				Expect().
				Status(404)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			By("delete HTTPRoute")
			err := s.DeleteResourceFromString(exactRouteByGet)
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute")
			time.Sleep(5 * time.Second)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(404)
		})

		It("Delete Gateway after apply HTTPRoute", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("access dataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			By("delete Gateway")
			err := s.DeleteResource("Gateway", "apisix")
			Expect(err).NotTo(HaveOccurred(), "deleting Gateway")

			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					Expect().
					Raw().StatusCode
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		It("Proxy External Service", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", httprouteWithExternalName, 1)

			By("checking the external service response")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.external").
				Expect().
				Status(http.StatusMovedPermanently)
		})

		It("Match Port", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", invalidBackendPort, 1)

			serviceResources, err := s.DefaultDataplaneResource().Service().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing services")
			Expect(serviceResources).To(HaveLen(1), "checking service length")

			serviceResource := serviceResources[0]
			nodes := serviceResource.Upstream.Nodes
			Expect(nodes).To(HaveLen(1), "checking nodes length")
			Expect(nodes[0].Port).To(Equal(80))
		})

		It("Delete HTTPRoute during restart", func() {
			By("create HTTPRoute httpbin")
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("create HTTPRoute httpbin2")
			ResourceApplied("HTTPRoute", "httpbin2", exactRouteByGet2, 1)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin2.example").
				Expect().
				Status(200)

			s.Deployer.ScaleIngress(0)

			By("delete HTTPRoute httpbin2")
			err := s.DeleteResource("HTTPRoute", "httpbin2")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute httpbin2")

			s.Deployer.ScaleIngress(1)
			time.Sleep(1 * time.Minute)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin2.example").
				Expect().
				Status(404)
		})
	})

	Context("HTTPRoute Rule Match", func() {
		var exactRouteByGet = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
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
spec:
  parentRefs:
  - name: apisix
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
  - name: apisix
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
  - name: apisix
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
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("access daataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/get/xxx").
				WithHost("httpbin.example").
				Expect().
				Status(404)
		})

		It("HTTPRoute Prefix Match", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", prefixRouteByStatus, 1)

			By("access daataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/status/200").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				GET("/status/201").
				WithHost("httpbin.example").
				Expect().
				Status(201)
		})

		It("HTTPRoute Method Match", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", methodRouteGETAndDELETEByAnything, 1)

			By("access daataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/anything").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				DELETE("/anything").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			s.NewAPISIXClient().
				POST("/anything").
				WithHost("httpbin.example").
				Expect().
				Status(404)
		})

		It("HTTPRoute Vars Match", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", varsRoute, 1)

			By("access dataplane to check the HTTPRoute")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(http.StatusNotFound)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				WithHeader("X-Route-Name", "httpbin").
				Expect().
				Status(http.StatusOK)
		})

		It("HTTPRoutePolicy in effect", func() {
			By("create HTTPRoute")
			s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, varsRoute)
			request := func() int {
				return s.NewAPISIXClient().GET("/get").
					WithHost("httpbin.example").WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("create HTTPRoutePolicy")
			s.ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				httpRoutePolicy,
			)

			By("access dataplane to check the HTTPRoutePolicy")
			Eventually(request).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				WithHeader("X-Route-Name", "httpbin").
				WithHeader("X-HRP-Name", "http-route-policy-0").
				WithQuery("hrp_name", "http-route-policy-0").
				Expect().
				Status(http.StatusOK)

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
				types.NamespacedName{Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				changedHTTPRoutePolicy,
			)

			// use the old vars cannot match any route
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					WithHeader("X-HRP-Name", "http-route-policy-0").
					WithQuery("hrp_name", "http-route-policy-0").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			// use the new vars can match the route
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				WithHeader("X-Route-Name", "httpbin").
				WithHeader("X-HRP-Name", "new-hrp-name").
				Expect().
				Status(http.StatusOK)

			By("delete the HTTPRoutePolicy")
			err := s.DeleteResource("HTTPRoutePolicy", "http-route-policy-0")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoutePolicy")
			Eventually(func() string {
				_, err := s.GetResourceYaml("HTTPRoutePolicy", "http-route-policy-0")
				return err.Error()
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(ContainSubstring(`httproutepolicies.apisix.apache.org "http-route-policy-0" not found`))
			// access the route without additional vars should be OK
			message := retry.DoWithRetry(s.GinkgoT, "", 10, time.Second, func() (string, error) {
				statusCode := s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
				if statusCode != http.StatusOK {
					return "", errors.Errorf("unexpected status code: %v", statusCode)
				}
				return "request OK", nil
			})
			s.Logf(message)
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
			s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, varsRoute)

			By("create HTTPRoutePolices")
			for name, spec := range map[string]string{
				"http-route-policy-0": httpRoutePolicy0,
				"http-route-policy-1": httpRoutePolicy1,
				"http-route-policy-2": httpRoutePolicy2,
			} {
				s.ApplyHTTPRoutePolicy(
					types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					spec,
					metav1.Condition{
						Type: string(v1alpha2.PolicyConditionAccepted),
					},
				)
			}
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1", "http-route-policy-2"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionFalse,
						Reason: string(v1alpha2.PolicyReasonConflicted),
					},
				)
			}

			// assert that conflict policies are not in effect
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("delete HTTPRoutePolicies")
			err := s.DeleteResource("HTTPRoutePolicy", "http-route-policy-2")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoutePolicy %s", "http-route-policy-2")
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionTrue,
						Reason: string(v1alpha2.PolicyReasonAccepted),
					},
				)
			}
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			By("update HTTPRoutePolicy")
			err = s.CreateResourceFromString(httpRoutePolicy1Priority20)
			Expect(err).NotTo(HaveOccurred(), "update HTTPRoutePolicy's priority to 20")
			framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
				types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-1"},
				metav1.Condition{
					Type: string(v1alpha2.PolicyConditionAccepted),
				},
			)
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1"} {
				framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 10*time.Second,
					types.NamespacedName{Namespace: s.Namespace(), Name: "apisix"},
					types.NamespacedName{Namespace: s.Namespace(), Name: name},
					metav1.Condition{
						Type:   string(v1alpha2.PolicyConditionAccepted),
						Status: metav1.ConditionFalse,
						Reason: string(v1alpha2.PolicyReasonConflicted),
					},
				)
			}
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("HTTPRoutePolicy status changes on HTTPRoute deleting", func() {
			By("create HTTPRoute")
			s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, varsRoute)

			By("create HTTPRoutePolicy")
			s.ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				httpRoutePolicy,
			)

			By("access dataplane to check the HTTPRoutePolicy")
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				WithHeader("X-Route-Name", "httpbin").
				WithHeader("X-HRP-Name", "http-route-policy-0").
				WithQuery("hrp_name", "http-route-policy-0").
				Expect().
				Status(http.StatusOK)

			By("delete the HTTPRoute, assert the HTTPRoutePolicy's status will be changed")
			err := s.DeleteResource("HTTPRoute", "httpbin")
			Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoute")
			message := retry.DoWithRetry(s.GinkgoT, "request the deleted route", 10, time.Second, func() (string, error) {
				statusCode := s.NewAPISIXClient().
					GET("/get").
					WithHost("httpbin.example").
					WithHeader("X-Route-Name", "httpbin").
					WithHeader("X-HRP-Name", "http-route-policy-0").
					WithQuery("hrp_name", "http-route-policy-0").
					Expect().Raw().StatusCode
				if statusCode != http.StatusNotFound {
					return "", errors.Errorf("unexpected status code: %v", statusCode)
				}
				return "the route is deleted", nil
			})
			s.Logf(message)

			err = framework.PollUntilHTTPRoutePolicyHaveStatus(s.K8sClient, 8*time.Second, types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
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
spec:
  parentRefs:
  - name: apisix
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
spec:
  parentRefs:
  - name: apisix
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
spec:
  parentRefs:
  - name: apisix
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
spec:
  parentRefs:
  - name: apisix
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
spec:
  parentRefs:
  - name: apisix
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
spec:
  parentRefs:
  - name: apisix
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
spec:
  parentRefs:
  - name: apisix
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
			ResourceApplied("HTTPRoute", "httpbin", reqHeaderModifyByHeaders, 1)

			By("access daataplane to check the HTTPRoute")
			respExp := s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.example").
				WithHeader("X-Req-Add", "test").
				WithHeader("X-Req-Removed", "test").
				WithHeader("X-Req-Set", "test").
				Expect()

			respExp.Status(200)
			respExp.Body().
				Contains(`"X-Req-Add": "test,add"`).
				Contains(`"X-Req-Set": "set"`).
				NotContains(`"X-Req-Removed": "remove"`)

		})

		It("HTTPRoute ResponseHeaderModifier", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", respHeaderModifyByHeaders, 1)

			By("access daataplane to check the HTTPRoute")
			respExp := s.NewAPISIXClient().
				GET("/headers").
				WithHost("httpbin.example").
				Expect()

			respExp.Status(200)
			respExp.Header("X-Resp-Add").IsEqual("add")
			respExp.Header("X-Resp-Set").IsEqual("set")
			respExp.Header("Server").IsEmpty()
			respExp.Body().
				NotContains(`"X-Resp-Add": "add"`).
				NotContains(`"X-Resp-Set": "set"`).
				NotContains(`"Server"`)
		})

		It("HTTPRoute RequestRedirect", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", httpsRedirectByHeaders, 1)

			s.NewAPISIXClient().GET("/headers").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusFound).
				Header("Location").IsEqual("https://httpbin.example:9443/headers")

			By("update HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", hostnameRedirectByHeaders, 2)

			s.NewAPISIXClient().GET("/headers").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusMovedPermanently).
				Header("Location").IsEqual("http://httpbin.org/headers")
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
spec:
  parentRefs:
  - name: apisix
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
			ResourceApplied("HTTPRoute", "httpbin", echoRoute, 1)

			time.Sleep(time.Second * 6)

			_ = s.NewAPISIXClient().GET("/headers").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusOK)

			echoLogs := s.GetDeploymentLogs("echo")
			Expect(echoLogs).To(ContainSubstring("GET /headers"))
		})

		It("HTTPRoute URLRewrite with ReplaceFullPath And Hostname", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", replaceFullPathAndHost, 1)

			By("/replace/201 should be rewritten to /headers")
			s.NewAPISIXClient().GET("/replace/201").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusOK).
				Body().
				Contains("replace.example.org")

			By("/replace/500 should be rewritten to /headers")
			s.NewAPISIXClient().GET("/replace/500").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusOK).
				Body().
				Contains("replace.example.org")
		})

		It("HTTPRoute URLRewrite with ReplacePrefixMatch", func() {
			By("create HTTPRoute")
			ResourceApplied("HTTPRoute", "httpbin", replacePrefixMatch, 1)

			By("/replace/201 should be rewritten to /status/201")
			s.NewAPISIXClient().GET("/replace/201").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusCreated)

			By("/replace/500 should be rewritten to /status/500")
			s.NewAPISIXClient().GET("/replace/500").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusInternalServerError)
		})

		It("HTTPRoute ExtensionRef", func() {
			By("create HTTPRoute")
			err := s.CreateResourceFromString(echoPlugin)
			Expect(err).NotTo(HaveOccurred(), "creating PluginConfig")
			ResourceApplied("HTTPRoute", "httpbin", extensionRefEchoPlugin, 1)

			s.NewAPISIXClient().GET("/get").
				WithHeader("Host", "httpbin.example").
				Expect().
				Body().
				Contains("Hello, World!!")

			err = s.CreateResourceFromString(echoPluginUpdated)
			Expect(err).NotTo(HaveOccurred(), "updating PluginConfig")
			time.Sleep(5 * time.Second)

			s.NewAPISIXClient().GET("/get").
				WithHeader("Host", "httpbin.example").
				Expect().
				Body().
				Contains("Updated")
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
  - name: apisix
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
  - name: apisix
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
			ResourceApplied("HTTPRoute", "httpbin", sameWeiht, 1)

			var (
				hitNginxCnt   = 0
				hitHttpbinCnt = 0
			)
			for i := 0; i < 100; i++ {
				body := s.NewAPISIXClient().GET("/get").
					WithHeader("Host", "httpbin.example").
					Expect().
					Status(http.StatusOK).
					Body().Raw()

				if strings.Contains(body, "Hello") {
					hitNginxCnt++
				} else {
					hitHttpbinCnt++
				}
			}
			Expect(hitNginxCnt - hitHttpbinCnt).To(BeNumerically("~", 0, 2))

			ResourceApplied("HTTPRoute", "httpbin", oneWeiht, 2)

			hitNginxCnt = 0
			hitHttpbinCnt = 0
			for i := 0; i < 100; i++ {
				body := s.NewAPISIXClient().GET("/get").
					WithHeader("Host", "httpbin.example").
					Expect().
					Status(http.StatusOK).
					Body().Raw()

				if strings.Contains(body, "Hello") {
					hitNginxCnt++
				} else {
					hitHttpbinCnt++
				}
			}
			Expect(hitHttpbinCnt - hitNginxCnt).To(Equal(100))
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
  - name: apisix
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
			ResourceApplied("HTTPRoute", "httpbin", exactRouteByGet, 1)

			By("verify HTTPRoute works")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)

			By("create additional gateway group to get new admin key")
			var err error
			additionalGatewayGroupID, _, err = s.Deployer.CreateAdditionalGateway("gateway-proxy-update")
			Expect(err).NotTo(HaveOccurred(), "creating additional gateway group")

			resources, exists := s.GetAdditionalGateway(additionalGatewayGroupID)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")

			client, err := s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating APISIX client for additional gateway group")

			By("HTTPRoute not found for additional gateway group")
			client.
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(404)

			By("update GatewayProxy with new admin key")
			updatedProxy := fmt.Sprintf(updatedGatewayProxy, s.Deployer.GetAdminEndpoint(resources.DataplaneService), resources.AdminAPIKey)
			err = s.CreateResourceFromString(updatedProxy)
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("verify HTTPRoute works for additional gateway group")
			client.
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(200)
		})
	})

	/*
		Context("HTTPRoute Status Updated", func() {
		})

		Context("HTTPRoute ParentRefs With Multiple Gateway", func() {
		})


		Context("HTTPRoute BackendRefs Discovery", func() {
		})
	*/
})
