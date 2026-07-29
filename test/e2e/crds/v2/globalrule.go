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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GlobalRule", Label("apisix.apache.org", "v2", "apisixglobalrule"), func() {
	s := scaffold.NewDefaultScaffold()

	// globalRuleAccepted polls until the named ApisixGlobalRule reports the
	// accepted/synced status, replacing fixed sleeps before status checks.
	globalRuleAccepted := func(name string, extraSubstrings ...string) {
		matchers := []OmegaMatcher{
			ContainSubstring(`status: "True"`),
			ContainSubstring("message: The global rule has been accepted and synced to APISIX"),
		}
		for _, sub := range extraSubstrings {
			matchers = append(matchers, ContainSubstring(sub))
		}
		s.RetryAssertion(func() (string, error) {
			return s.GetResourceYaml("ApisixGlobalRule", name)
		}).Should(And(matchers...))
	}

	var ingressYaml = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-ingress
spec:
  ingressClassName: %s
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
			err := s.CreateResourceFromString(s.GetGatewayProxySpec())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")

			By("create Ingress")
			err = s.CreateResourceFromString(fmt.Sprintf(ingressYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")

			By("verify Ingress works")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
		})

		It("Test GlobalRule with response-rewrite plugin", func() {
			globalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-response-rewrite
spec:
  ingressClassName: %s
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Global-Rule: "test-response-rewrite"
        X-Global-Test: "enabled"
`

			By("create ApisixGlobalRule with response-rewrite plugin")
			err := s.CreateResourceFromString(fmt.Sprintf(globalRuleYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule")

			By("verify ApisixGlobalRule status condition")
			globalRuleAccepted("test-global-rule-response-rewrite")

			By("verify global rule is applied - response should have custom headers")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeader("X-Global-Rule", "test-response-rewrite"),
					scaffold.WithExpectedHeader("X-Global-Test", "enabled"),
				},
			})

			By("delete ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-response-rewrite")
			Expect(err).NotTo(HaveOccurred(), "deleting ApisixGlobalRule")

			By("verify global rule is removed - response should not have custom headers")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedNotHeader("X-Global-Rule"),
					scaffold.WithExpectedNotHeader("X-Global-Test"),
				},
			})
		})

		It("Test GlobalRule update", func() {
			globalRuleYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-update
spec:
  ingressClassName: %s
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
  ingressClassName: %s
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Update-Test: "version2"
        X-New-Header: "added"
`

			By("create initial ApisixGlobalRule")
			err := s.CreateResourceFromString(fmt.Sprintf(globalRuleYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule")

			By("verify initial ApisixGlobalRule status condition")
			globalRuleAccepted("test-global-rule-update")

			By("verify initial configuration")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeader("X-Update-Test", "version1"),
					scaffold.WithExpectedNotHeader("X-New-Header"),
				},
			})

			By("update ApisixGlobalRule")
			err = s.CreateResourceFromString(fmt.Sprintf(updatedGlobalRuleYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "updating ApisixGlobalRule")

			By("verify updated ApisixGlobalRule status condition")
			globalRuleAccepted("test-global-rule-update", "observedGeneration: 2")

			By("verify updated configuration")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeader("X-Update-Test", "version2"),
					scaffold.WithExpectedHeader("X-New-Header", "added"),
				},
			})

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
  ingressClassName: %s
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
  ingressClassName: %s
  plugins:
  - name: response-rewrite
    enable: true
    config:
      headers:
        X-Global-Multi: "test-multi-rule"
        X-Response-Type: "rewrite"
`

			By("create ApisixGlobalRule with proxy-rewrite plugin")
			err := s.CreateResourceFromString(fmt.Sprintf(proxyRewriteGlobalRuleYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule with proxy-rewrite")

			By("create ApisixGlobalRule with response-rewrite plugin")
			err = s.CreateResourceFromString(fmt.Sprintf(responseRewriteGlobalRuleYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule with response-rewrite")

			By("verify both ApisixGlobalRule status conditions")
			globalRuleAccepted("test-global-rule-proxy-rewrite")
			globalRuleAccepted("test-global-rule-response-rewrite-multi")

			By("verify both global rules are applied on GET request")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeader("X-Global-Multi", "test-multi-rule"),
					scaffold.WithExpectedHeader("X-Response-Type", "rewrite"),
					scaffold.WithExpectedBodyContains(`"X-Global-Proxy": "test"`),
				},
			})

			By("delete proxy-rewrite ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-proxy-rewrite")
			Expect(err).NotTo(HaveOccurred(), "deleting proxy-rewrite ApisixGlobalRule")

			By("verify only response-rewrite global rule remains - proxy-rewrite headers should be removed")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeader("X-Global-Multi", "test-multi-rule"),
					scaffold.WithExpectedHeader("X-Response-Type", "rewrite"),
					scaffold.WithExpectedBodyNotContains(`"X-Global-Proxy": "test"`),
				},
			})

			By("delete response-rewrite ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-response-rewrite-multi")
			Expect(err).NotTo(HaveOccurred(), "deleting response-rewrite ApisixGlobalRule")

			By("verify all global rules are removed")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedNotHeader("X-Global-Multi"),
					scaffold.WithExpectedNotHeader("X-Response-Type"),
					scaffold.WithExpectedBodyNotContains(`"X-Global-Proxy": "test"`),
				},
			})
		})

		It("Test GlobalRule with plugin using secretRef", func() {
			secretYaml := `
apiVersion: v1
kind: Secret
metadata:
  name: echo-secret
  namespace: %s
type: Opaque
stringData:
  body: "GlobalRule with secret test"
`

			globalRuleWithSecretYaml := `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: test-global-rule-with-secret
spec:
  ingressClassName: %s
  plugins:
  - name: echo
    enable: true
    secretRef: echo-secret
`

			By("create Secret for GlobalRule")
			err := s.CreateResourceFromString(fmt.Sprintf(secretYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating Secret for GlobalRule")

			By("create ApisixGlobalRule with plugin secretRef")
			err = s.CreateResourceFromString(fmt.Sprintf(globalRuleWithSecretYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixGlobalRule with secretRef")

			By("verify ApisixGlobalRule status condition")
			globalRuleAccepted("test-global-rule-with-secret")

			By("verify global rule with secret is applied")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains("GlobalRule with secret test"),
				},
			})

			By("update Secret")
			updatedSecretYaml := `
apiVersion: v1
kind: Secret
metadata:
  name: echo-secret
  namespace: %s
type: Opaque
stringData:
  body: "GlobalRule with secret test updated"
`
			err = s.CreateResourceFromString(fmt.Sprintf(updatedSecretYaml, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "updating Secret")

			By("verify global rule with updated secret")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "globalrule.example.com",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains("GlobalRule with secret test updated"),
				},
			})

			By("delete Secret")
			err = s.DeleteResource("Secret", "echo-secret")
			Expect(err).NotTo(HaveOccurred(), "deleting Secret")

			By("verify ApisixGlobalRule status shows error after secret deletion")
			s.RetryAssertion(func() (string, error) {
				return s.GetResourceYaml("ApisixGlobalRule", "test-global-rule-with-secret")
			}).Should(And(
				ContainSubstring(`status: "False"`),
				ContainSubstring("failed to get Secret"),
			))

			By("delete ApisixGlobalRule")
			err = s.DeleteResource("ApisixGlobalRule", "test-global-rule-with-secret")
			Expect(err).NotTo(HaveOccurred(), "deleting ApisixGlobalRule")
		})
	})
})
