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

package v2

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

const gatewayProxyYamlPluginConfig = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
  namespace: default
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

const ingressClassYamlPluginConfig = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "default"
    scope: "Namespace"
`

var _ = Describe("Test ApisixPluginConfig", Label("apisix.apache.org", "v2", "apisixpluginconfig"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	Context("Test ApisixPluginConfig", func() {
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYamlPluginConfig, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(ingressClassYamlPluginConfig, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})

		It("Basic ApisixPluginConfig test", func() {
			const apisixPluginConfigSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-plugin-config
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Plugin-Config: "test-response-rewrite"
        X-Plugin-Test: "enabled"
`

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-route
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    plugin_config_name: test-plugin-config
`

			By("apply ApisixPluginConfig")
			var apisixPluginConfig apiv2.ApisixPluginConfig
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-plugin-config"}, &apisixPluginConfig, apisixPluginConfigSpec)

			By("apply ApisixRoute that references ApisixPluginConfig")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-route"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works with plugin config")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("verify plugin from ApisixPluginConfig works")
			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Plugin-Config").IsEqual("test-response-rewrite")
			resp.Header("X-Plugin-Test").IsEqual("enabled")

			By("delete ApisixRoute")
			err := s.DeleteResource("ApisixRoute", "test-route")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")

			By("delete ApisixPluginConfig")
			err = s.DeleteResource("ApisixPluginConfig", "test-plugin-config")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixPluginConfig")

			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		It("Test ApisixPluginConfig update", func() {
			const apisixPluginConfigSpecV1 = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-plugin-config-update
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Version: "v1"
`

			const apisixPluginConfigSpecV2 = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-plugin-config-update
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Version: "v2"
        X-Updated: "true"
`

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-route-update
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    plugin_config_name: test-plugin-config-update
`

			By("apply initial ApisixPluginConfig")
			var apisixPluginConfig apiv2.ApisixPluginConfig
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-plugin-config-update"}, &apisixPluginConfig, apisixPluginConfigSpecV1)

			By("apply ApisixRoute that references ApisixPluginConfig")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-route-update"}, &apisixRoute, apisixRouteSpec)

			By("verify initial plugin config works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Version").IsEqual("v1")
			resp.Header("X-Updated").IsEmpty()

			By("update ApisixPluginConfig")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-plugin-config-update"}, &apisixPluginConfig, apisixPluginConfigSpecV2)
			time.Sleep(5 * time.Second)

			By("verify updated plugin config works")
			resp = s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Version").IsEqual("v2")
			resp.Header("X-Updated").IsEqual("true")

			By("delete resources")
			err := s.DeleteResource("ApisixRoute", "test-route-update")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			err = s.DeleteResource("ApisixPluginConfig", "test-plugin-config-update")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixPluginConfig")
		})

		It("Test ApisixPluginConfig with disabled plugin", func() {
			const apisixPluginConfigSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-plugin-config-disabled
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: false
    config:
      headers:
        X-Should-Not-Exist: "disabled"
  - name: cors
    enable: true
    config:
      allow_origins: "*"
      allow_methods: "GET,POST"
`

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-route-disabled
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    plugin_config_name: test-plugin-config-disabled
`

			By("apply ApisixPluginConfig with disabled plugin")
			var apisixPluginConfig apiv2.ApisixPluginConfig
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-plugin-config-disabled"}, &apisixPluginConfig, apisixPluginConfigSpec)

			By("apply ApisixRoute that references ApisixPluginConfig")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-route-disabled"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("verify disabled plugin is not applied")
			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Should-Not-Exist").IsEmpty()

			By("verify enabled plugin is applied")
			resp.Header("Access-Control-Allow-Origin").IsEqual("*")

			By("delete resources")
			err := s.DeleteResource("ApisixRoute", "test-route-disabled")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			err = s.DeleteResource("ApisixPluginConfig", "test-plugin-config-disabled")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixPluginConfig")
		})

		It("Test ApisixPluginConfig overridden by route plugins", func() {
			const apisixPluginConfigSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-plugin-config-override
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-From-Config: "plugin-config"
        X-Shared: "from-config"
`

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-route-override
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    plugin_config_name: test-plugin-config-override
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          X-From-Route: "route"
          X-Shared: "from-route"
`

			By("apply ApisixPluginConfig")
			var apisixPluginConfig apiv2.ApisixPluginConfig
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-plugin-config-override"}, &apisixPluginConfig, apisixPluginConfigSpec)

			By("apply ApisixRoute with overriding plugins")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-route-override"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("verify route plugins override plugin config")
			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-From-Config").IsEmpty()
			resp.Header("X-From-Route").IsEqual("route")
			resp.Header("X-Shared").IsEqual("from-route")

			By("delete resources")
			err := s.DeleteResource("ApisixRoute", "test-route-override")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			err = s.DeleteResource("ApisixPluginConfig", "test-plugin-config-override")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixPluginConfig")
		})

		It("Test cross-namespace ApisixPluginConfig reference", func() {
			const crossNamespaceApisixPluginConfigSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: cross-ns-plugin-config
  namespace: default
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Cross-Namespace: "true"
        X-Namespace: "default"
`

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-route-cross-ns
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    plugin_config_name: cross-ns-plugin-config
    plugin_config_namespace: default
`

			By("apply ApisixPluginConfig in default namespace")
			err := s.CreateResourceFromStringWithNamespace(crossNamespaceApisixPluginConfigSpec, "default")
			Expect(err).NotTo(HaveOccurred(), "creating default/cross-ns-plugin-config")
			time.Sleep(5 * time.Second)

			By("apply ApisixRoute in test namespace that references ApisixPluginConfig in default namespace")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-route-cross-ns"}, &apisixRoute, apisixRouteSpec)

			By("verify cross-namespace reference works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Cross-Namespace").IsEqual("true")
			resp.Header("X-Namespace").IsEqual("default")

			By("delete resources")
			err = s.DeleteResource("ApisixRoute", "test-route-cross-ns")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			err = s.DeleteResourceFromStringWithNamespace(crossNamespaceApisixPluginConfigSpec, "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixPluginConfig")
		})

		It("Test ApisixPluginConfig with SecretRef", func() {
			const secretSpec = `
apiVersion: v1
kind: Secret
metadata:
  name: plugin-secret
type: Opaque
data:
  key: dGVzdC1rZXk=
  username: dGVzdC11c2Vy
  password: dGVzdC1wYXNzd29yZA==
`

			const apisixPluginConfigSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-plugin-config-secret
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    secretRef: plugin-secret
    config:
      headers:
        X-Secret-Ref: "true"
`

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-route-secret
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    plugin_config_name: test-plugin-config-secret
`

			By("apply Secret")
			err := s.CreateResourceFromStringWithNamespace(secretSpec, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating Secret")

			By("apply ApisixPluginConfig with SecretRef")
			var apisixPluginConfig apiv2.ApisixPluginConfig
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-plugin-config-secret"}, &apisixPluginConfig, apisixPluginConfigSpec)

			By("apply ApisixRoute that references ApisixPluginConfig")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-route-secret"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works with SecretRef")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Secret-Ref").IsEqual("true")

			By("delete resources")
			err = s.DeleteResource("ApisixRoute", "test-route-secret")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			err = s.DeleteResource("ApisixPluginConfig", "test-plugin-config-secret")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixPluginConfig")
			err = s.DeleteResource("Secret", "plugin-secret")
			Expect(err).ShouldNot(HaveOccurred(), "deleting Secret")
		})
	})
})
