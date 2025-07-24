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

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

const gatewayProxyYaml = `
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

const ingressClassYaml = `
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

var _ = Describe("Test GlobalRule", Label("apisix.apache.org", "v2", "apisixglobalrule"), func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "apisix.apache.org/apisix-ingress-controller",
	})

	var ingressYaml = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-ingress
spec:
  ingressClassName: apisix
  rules:
  - host: globalrule.example.com
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

	Context("ApisixGlobalRule Basic Operations", func() {
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(ingressClassYaml, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress")
			err = s.CreateResourceFromString(ingressYaml)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")
			time.Sleep(5 * time.Second)

			By("verify Ingress works")
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("globalrule.example.com").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).
				Should(Equal(http.StatusOK))
		})

		It("Test GlobalRule with response-rewrite plugin", func() {
			globalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-response-rewrite
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Global-Rule: "test-response-rewrite"
        X-Global-Test: "enabled"
`

			By("create ApisixGlobalRule with response-rewrite plugin")
			err := s.CreateResourceFromString(globalRuleYaml)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule")

			By("verify ApisixGlobalRule status condition")
			time.Sleep(5 * time.Second)
			gryaml, err := s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-response-rewrite")
			Expect(err).NotTo(HaveOccurred(), "getting ApisixGlobalRule yaml")
			Expect(gryaml).To(ContainSubstring(`status: "True"`))
			Expect(gryaml).To(ContainSubstring("message: The global rule has been accepted and synced to APISIX"))

			By("verify global rule is applied - response should have custom headers")
			resp := s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEqual("test-response-rewrite")
			resp.Header("X-Global-Test").IsEqual("enabled")

			By("delete ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-response-rewrite")
			Expect(err).NotTo(HaveOccurred(), "deleting ApisixGlobalRule")
			time.Sleep(5 * time.Second)

			By("verify global rule is removed - response should not have custom headers")
			resp = s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEmpty()
			resp.Header("X-Global-Test").IsEmpty()
		})

		It("Test GlobalRule update", func() {
			globalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-update
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Update-Test: "version1"
`

			updatedGlobalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-update
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Update-Test: "version2"
        X-New-Header: "added"
`

			By("create initial ApisixGlobalRule")
			err := s.CreateResourceFromString(globalRuleYaml)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule")

			By("verify initial ApisixGlobalRule status condition")
			time.Sleep(5 * time.Second)
			gryaml, err := s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-update")
			Expect(err).NotTo(HaveOccurred(), "getting ApisixGlobalRule yaml")
			Expect(gryaml).To(ContainSubstring(`status: "True"`))
			Expect(gryaml).To(ContainSubstring("message: The global rule has been accepted and synced to APISIX"))

			By("verify initial configuration")
			resp := s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			resp.Header("X-Update-Test").IsEqual("version1")
			resp.Header("X-New-Header").IsEmpty()

			By("update ApisixGlobalRule")
			err = s.CreateResourceFromString(updatedGlobalRuleYaml)
			Expect(err).NotTo(HaveOccurred(), "updating ApisixGlobalRule")

			By("verify updated ApisixGlobalRule status condition")
			time.Sleep(5 * time.Second)
			gryaml, err = s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-update")
			Expect(err).NotTo(HaveOccurred(), "getting updated ApisixGlobalRule yaml")
			Expect(gryaml).To(ContainSubstring(`status: "True"`))
			Expect(gryaml).To(ContainSubstring("message: The global rule has been accepted and synced to APISIX"))
			Expect(gryaml).To(ContainSubstring("observedGeneration: 2"))

			By("verify updated configuration")
			resp = s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			resp.Header("X-Update-Test").IsEqual("version2")
			resp.Header("X-New-Header").IsEqual("added")

			By("delete ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-update")
			Expect(err).NotTo(HaveOccurred(), "deleting ApisixGlobalRule")
		})

		It("Test multiple GlobalRules with different plugins", func() {
			proxyRewriteGlobalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-proxy-rewrite
spec:
  ingressClassName: apisix
  plugins:
  - name: proxy-rewrite
    enable: true
    config:
      headers:
        add:
          X-Global-Proxy: "test"
`

			responseRewriteGlobalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-response-rewrite-multi
spec:
  ingressClassName: apisix
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Global-Multi: "test-multi-rule"
        X-Response-Type: "rewrite"
`

			By("create ApisixGlobalRule with proxy-rewrite plugin")
			err := s.CreateResourceFromString(proxyRewriteGlobalRuleYaml)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule with proxy-rewrite")

			By("create ApisixGlobalRule with response-rewrite plugin")
			err = s.CreateResourceFromString(responseRewriteGlobalRuleYaml)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule with response-rewrite")

			By("verify both ApisixGlobalRule status conditions")
			time.Sleep(5 * time.Second)

			proxyRewriteYaml, err := s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-proxy-rewrite")
			Expect(err).NotTo(HaveOccurred(), "getting proxy-rewrite ApisixGlobalRule yaml")
			Expect(proxyRewriteYaml).To(ContainSubstring(`status: "True"`))
			Expect(proxyRewriteYaml).To(ContainSubstring("message: The global rule has been accepted and synced to APISIX"))

			responseRewriteYaml, err := s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-response-rewrite-multi")
			Expect(err).NotTo(HaveOccurred(), "getting response-rewrite ApisixGlobalRule yaml")
			Expect(responseRewriteYaml).To(ContainSubstring(`status: "True"`))
			Expect(responseRewriteYaml).To(ContainSubstring("message: The global rule has been accepted and synced to APISIX"))

			By("verify both global rules are applied on GET request")
			getResp := s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			getResp.Header("X-Global-Multi").IsEqual("test-multi-rule")
			getResp.Header("X-Response-Type").IsEqual("rewrite")
			getResp.Body().Contains(`"X-Global-Proxy": "test"`)

			By("delete proxy-rewrite ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-proxy-rewrite")
			Expect(err).NotTo(HaveOccurred(), "deleting proxy-rewrite ApisixGlobalRule")
			time.Sleep(5 * time.Second)

			By("verify only response-rewrite global rule remains - proxy-rewrite headers should be removed")
			getRespAfterProxyDelete := s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			getRespAfterProxyDelete.Header("X-Global-Multi").IsEqual("test-multi-rule")
			getRespAfterProxyDelete.Header("X-Response-Type").IsEqual("rewrite")
			getRespAfterProxyDelete.Body().NotContains(`"X-Global-Proxy": "test"`)

			By("delete response-rewrite ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-response-rewrite-multi")
			Expect(err).NotTo(HaveOccurred(), "deleting response-rewrite ApisixGlobalRule")
			time.Sleep(5 * time.Second)

			By("verify all global rules are removed")
			finalResp := s.NewAPISIXClient().
				GET("/get").
				WithHost("globalrule.example.com").
				Expect().
				Status(http.StatusOK)
			finalResp.Header("X-Global-Multi").IsEmpty()
			finalResp.Header("X-Response-Type").IsEmpty()
			finalResp.Body().NotContains(`"X-Global-Proxy": "test"`)
		})
	})
})
