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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GatewayProxy", Label("apisix.apache.org", "v1alpha1", "gatewayproxy"), func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: fmt.Sprintf("apisix.apache.org/apisix-ingress-controller-%d", time.Now().Unix()),
	})

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
  name: %s
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
      name: %s
`

	var gatewayProxyWithEnabledPlugin = `
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
  plugins:
  - name: response-rewrite
    enabled: false
    config: 
      headers:
        X-Proxy-Test: "disabled"
`

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
		err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithEnabledPlugin, s.Namespace(), s.Deployer.GetAdminEndpoint(), s.AdminKey()))
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy with enabled plugin")
		time.Sleep(5 * time.Second)

		By("Create Gateway with GatewayProxy")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayWithProxy, s.Namespace(), gatewayClassName, s.Namespace()), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway with GatewayProxy")
		time.Sleep(5 * time.Second)

		By("check Gateway condition")
		gwyaml, err := s.GetResourceYaml("Gateway", s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
		Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
		Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"), "checking Gateway condition message")
	})

	Context("Test Gateway with enabled GatewayProxy plugin", func() {
		It("Should apply plugin configuration when enabled", func() {
			By("Create HTTPRoute for Gateway with GatewayProxy")
			s.ResourceApplied("HTTPRoute", "test-route", fmt.Sprintf(httpRouteForTest, s.Namespace()), 1)

			By("Check if the plugin is applied")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedHeader("X-Proxy-Test", "enabled"),
				},
			})

			By("Update GatewayProxy with disabled plugin")
			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithDisabledPlugin, s.Namespace(), s.Deployer.GetAdminEndpoint(), s.AdminKey()))
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy with disabled plugin")

			By("Create HTTPRoute for Gateway with GatewayProxy")
			s.ResourceApplied("HTTPRoute", "test-route", fmt.Sprintf(httpRouteForTest, s.Namespace()), 1)

			By("Check if the plugin is not applied")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedHeader("X-Proxy-Test", ""),
				},
			})

			By("should fail to apply GatewayProxy with invalid endpoint")
			var gatewayProxyWithInvalidEndpoint = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
        - "http://invalid-endpoint:9180"
        - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

			By("Update GatewayProxy with invalid endpoint")
			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithInvalidEndpoint, s.Namespace(), s.Deployer.GetAdminEndpoint(), s.AdminKey()))
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy with enabled plugin")

			By("Create HTTPRoute")
			s.ResourceApplied("HTTPRoute", "test-route", fmt.Sprintf(httpRouteForTest, s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedHeader("X-Proxy-Test", ""),
				},
			})
		})
	})

	Context("Test GatewayProxy Provider Validation", func() {
		var (
			gatewayProxyWithInvalidProviderType = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: "InvalidType"
`
			gatewayProxyWithMissingControlPlane = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: "ControlPlane"
`
			gatewayProxyWithValidProvider = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
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
		It("Should reject invalid provider type", func() {
			By("Create GatewayProxy with invalid provider type")
			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithInvalidProviderType, s.Namespace()))
			Expect(err).To(HaveOccurred(), "creating GatewayProxy with invalid provider type")
			Expect(err.Error()).To(ContainSubstring("Invalid value"))
		})

		It("Should reject missing controlPlane configuration", func() {
			By("Create GatewayProxy with missing controlPlane")
			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithMissingControlPlane, s.Namespace()))
			Expect(err).To(HaveOccurred(), "creating GatewayProxy with missing controlPlane")
			Expect(err.Error()).To(ContainSubstring("controlPlane must be specified when type is ControlPlane"))
		})

		It("Should accept valid provider configuration", func() {
			By("Create GatewayProxy with valid provider")
			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyWithValidProvider, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy with valid provider")

			s.RetryAssertion(func() string {
				gpYaml, _ := s.GetResourceYaml("GatewayProxy", s.Namespace())
				return gpYaml
			}).Should(ContainSubstring(`"type":"ControlPlane"`), "checking GatewayProxy is applied")
		})
	})
})
