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
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GatewayProxy", func() {
	s := scaffold.NewDefaultScaffold()

	var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

	var gatewayWithProxy = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
spec:
  gatewayClassName: %s
  listeners:
    - name: http
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

	var gatewayProxyWithEnabledPlugin = `
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
  plugins:
  - name: response-rewrite
    enabled: true
    config: 
      headers:
        X-Proxy-Test: "enabled"
`

	var gatewayProxyWithDisabledPlugin = `
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
  plugins:
  - name: response-rewrite
    enabled: false
    config: 
      headers:
        X-Proxy-Test: "disabled"
`
	var (
		gatewayProxyWithPluginMetadata0 = `
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
  plugins:
  - name: error-page
    enabled: true
    config: {}
  pluginMetadata:
    error-page: {
      "enable": true,
      "error_404": {
          "body": "404 from plugin metadata",
          "content-type": "text/plain"
      }
    }
`
		gatewayProxyWithPluginMetadata1 = `
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
  plugins:
  - name: error-page
    enabled: true
    config: {}
  pluginMetadata:
    error-page: {
      "enable": false,
      "error_404": {
          "body": "404 from plugin metadata",
          "content-type": "text/plain"
      }
    }
`
	)

	var httpRouteForTest = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: test-route
spec:
  parentRefs:
  - name: %s
  hostnames:
  - example.com
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

	var resourceApplied = func(resourceType, resourceName, resourceRaw string, observedGeneration int) {
		Expect(s.CreateResourceFromString(resourceRaw)).
			NotTo(HaveOccurred(), fmt.Sprintf("creating %s", resourceType))

		Eventually(func() string {
			hryaml, err := s.GetResourceYaml(resourceType, resourceName)
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("getting %s yaml", resourceType))
			return hryaml
		}).WithTimeout(8*time.Second).ProbeEvery(2*time.Second).
			Should(
				SatisfyAll(
					ContainSubstring(`status: "True"`),
					ContainSubstring(fmt.Sprintf("observedGeneration: %d", observedGeneration)),
				),
				fmt.Sprintf("checking %s condition status", resourceType),
			)
		time.Sleep(3 * time.Second)
	}

	var (
		gatewayClassName string
	)

	BeforeEach(func() {
		By("Create GatewayClass")
		gatewayClassName = fmt.Sprintf("apisix-%d", time.Now().Unix())
		err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGatewayClass, gatewayClassName, s.GetControllerName()), "")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(5 * time.Second)

		By("Check GatewayClass condition")
		gcYaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
		Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
		Expect(gcYaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
		Expect(gcYaml).To(ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"), "checking GatewayClass condition message")

		By("Create GatewayProxy with enabled plugin")
		err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithEnabledPlugin, s.Deployer.GetAdminEndpoint(), s.AdminKey()))
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy with enabled plugin")
		time.Sleep(5 * time.Second)

		By("Create Gateway with GatewayProxy")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayWithProxy, gatewayClassName), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway with GatewayProxy")
		time.Sleep(5 * time.Second)

		By("check Gateway condition")
		gwyaml, err := s.GetResourceYaml("Gateway", "apisix")
		Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
		Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
		Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"), "checking Gateway condition message")
	})

	AfterEach(func() {
		By("Clean up resources")
		_ = s.DeleteResourceFromString(fmt.Sprintf(httpRouteForTest, "apisix"))
		_ = s.DeleteResourceFromString(fmt.Sprintf(gatewayWithProxy, gatewayClassName))
		_ = s.DeleteResourceFromString(fmt.Sprintf(gatewayProxyWithEnabledPlugin, s.Deployer.GetAdminEndpoint(), s.AdminKey()))
	})

	Context("Test Gateway with enabled GatewayProxy plugin", func() {
		It("Should apply plugin configuration when enabled", func() {
			By("Create HTTPRoute for Gateway with GatewayProxy")
			resourceApplied("HTTPRoute", "test-route", fmt.Sprintf(httpRouteForTest, "apisix"), 1)

			By("Check if the plugin is applied")
			resp := s.NewAPISIXClient().
				GET("/get").
				WithHost("example.com").
				Expect().
				Status(200)

			resp.Header("X-Proxy-Test").IsEqual("enabled")

			By("Update GatewayProxy with disabled plugin")
			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithDisabledPlugin, s.Deployer.GetAdminEndpoint(), s.AdminKey()))
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy with disabled plugin")
			time.Sleep(5 * time.Second)

			By("Create HTTPRoute for Gateway with GatewayProxy")
			resourceApplied("HTTPRoute", "test-route", fmt.Sprintf(httpRouteForTest, "apisix"), 1)

			By("Check if the plugin is not applied")
			resp = s.NewAPISIXClient().
				GET("/get").
				WithHost("example.com").
				Expect().
				Status(200)

			resp.Header("X-Proxy-Test").IsEmpty()
		})
	})

	Context("Test Gateway with PluginMetadata", func() {
		var (
			err error
		)

		PIt("Should work OK with error-page", func() {
			By("Update GatewayProxy with PluginMetadata")
			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithPluginMetadata0, s.Deployer.GetAdminEndpoint(), s.AdminKey()))
			Expect(err).ShouldNot(HaveOccurred())
			time.Sleep(5 * time.Second)

			By("Create HTTPRoute for Gateway with GatewayProxy")
			resourceApplied("HTTPRoute", "test-route", fmt.Sprintf(httpRouteForTest, "apisix"), 1)

			time.Sleep(5 * time.Second)
			By("Check PluginMetadata working")
			s.NewAPISIXClient().
				GET("/not-found").
				WithHost("example.com").
				Expect().
				Status(http.StatusNotFound).
				Body().Contains("404 from plugin metadata")

			By("Update GatewayProxy with PluginMetadata")
			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithPluginMetadata1, s.Deployer.GetAdminEndpoint(), s.AdminKey()))
			Expect(err).ShouldNot(HaveOccurred())
			time.Sleep(5 * time.Second)

			By("Check PluginMetadata working")
			s.NewAPISIXClient().
				GET("/not-found").
				WithHost("example.com").
				Expect().
				Status(http.StatusNotFound).
				Body().Contains(`{"error_msg":"404 Route Not Found"}`)

			By("Delete GatewayProxy")
			err = s.DeleteResourceFromString(fmt.Sprintf(gatewayProxyWithPluginMetadata0, s.Deployer.GetAdminEndpoint(), s.AdminKey()))
			Expect(err).ShouldNot(HaveOccurred())
			time.Sleep(5 * time.Second)

			By("Check PluginMetadata is not working")
			s.NewAPISIXClient().
				GET("/not-found").
				WithHost("example.com").
				Expect().
				Status(http.StatusNotFound).
				Body().Contains(`{"error_msg":"404 Route Not Found"}`)
		})
	})

	var (
		gatewayProxyWithInvalidProviderType = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: "InvalidType"
`
		gatewayProxyWithMissingControlPlane = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: "ControlPlane"
`
		gatewayProxyWithValidProvider = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: "ControlPlane"
    controlPlane:
      endpoints:
        - "http://localhost:9180"
      auth:
        type: "AdminKey"
        adminKey:
          value: "test-key"
`
	)

	Context("Test GatewayProxy Provider Validation", func() {
		AfterEach(func() {
			By("Clean up GatewayProxy resources")
			_ = s.DeleteResourceFromString(gatewayProxyWithInvalidProviderType)
			_ = s.DeleteResourceFromString(gatewayProxyWithMissingControlPlane)
			_ = s.DeleteResourceFromString(gatewayProxyWithValidProvider)
		})

		It("Should reject invalid provider type", func() {
			By("Create GatewayProxy with invalid provider type")
			err := s.CreateResourceFromString(gatewayProxyWithInvalidProviderType)
			Expect(err).To(HaveOccurred(), "creating GatewayProxy with invalid provider type")
			Expect(err.Error()).To(ContainSubstring("Invalid value"))
		})

		It("Should reject missing controlPlane configuration", func() {
			By("Create GatewayProxy with missing controlPlane")
			err := s.CreateResourceFromString(gatewayProxyWithMissingControlPlane)
			Expect(err).To(HaveOccurred(), "creating GatewayProxy with missing controlPlane")
			Expect(err.Error()).To(ContainSubstring("controlPlane must be specified when type is ControlPlane"))
		})

		It("Should accept valid provider configuration", func() {
			By("Create GatewayProxy with valid provider")
			err := s.CreateResourceFromString(gatewayProxyWithValidProvider)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy with valid provider")

			Eventually(func() string {
				gpYaml, err := s.GetResourceYaml("GatewayProxy", "apisix-proxy-config")
				Expect(err).NotTo(HaveOccurred(), "getting GatewayProxy yaml")
				return gpYaml
			}).WithTimeout(8*time.Second).ProbeEvery(2*time.Second).
				Should(ContainSubstring(`"type":"ControlPlane"`), "checking GatewayProxy is applied")
		})
	})
})
