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
	gomegatypes "github.com/onsi/gomega/types"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

// Every case applies a valid and a rejected resource together, so the first sync carries
// both: the sync converges when the rejected one is excluded and the valid one is served.
// Fixing the rejected one then has to bring it in too.
//
// What makes a resource rejected is always something the data plane checks for every
// backend: a value outside the schema's range, or a known plugin configured in a way its
// own check_schema refuses. An unknown plugin name is not used, since apisix-standalone
// accepts those.
var _ = Describe("Test bad resource isolation", Label("apisix.apache.org", "v2", "isolation"), func() {
	s := scaffold.NewDefaultScaffold()

	// rejectedPlugin is a plugin every backend loads, configured with a count the plugin
	// itself refuses (it has to be greater than 0).
	const rejectedPlugin = `
    plugins:
    - name: limit-count
      enable: true
      config:
        count: 0
        time_window: 60
        rejected_code: 503
        key: remote_addr
`
	const acceptedPlugin = `
    plugins:
    - name: limit-count
      enable: true
      config:
        count: 100
        time_window: 60
        rejected_code: 503
        key: remote_addr
`
	// routes is one ApisixRoute serving valid.example.com and one serving
	// rejected.example.com, whose plugin configuration is filled in per case.
	const routes = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: valid
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - valid.example.com
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: rejected
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - rejected.example.com
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
%s
`

	expectServed := func(host string) {
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET",
			Path:   "/get",
			Host:   host,
			Check:  scaffold.WithExpectedStatus(http.StatusOK),
		})
	}
	expectStatus := func(resource, name string, matchers ...gomegatypes.GomegaMatcher) {
		s.RetryAssertion(func() string {
			output, _ := s.GetOutputFromString(resource, name, "-o", "yaml", "-n", s.Namespace())
			return output
		}).Should(And(matchers...))
	}
	apply := func(yaml string) {
		Expect(s.CreateResourceFromString(yaml)).NotTo(HaveOccurred(), "applying resources")
	}

	BeforeEach(func() {
		By("create GatewayProxy")
		Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create IngressClass")
		Expect(s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")).NotTo(HaveOccurred(), "creating IngressClass")
	})

	It("isolates a rejected route", func() {
		By("apply a valid and a rejected ApisixRoute")
		apply(fmt.Sprintf(routes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace(), rejectedPlugin))

		By("the valid ApisixRoute is served and the rejected one reports why it is not")
		expectServed("valid.example.com")
		expectStatus("ar", "rejected",
			ContainSubstring(`status: "False"`),
			ContainSubstring(`reason: SyncFailed`),
			ContainSubstring(`failed to check the configuration of plugin limit-count`),
		)

		By("fix the rejected ApisixRoute")
		apply(fmt.Sprintf(routes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace(), acceptedPlugin))

		By("both ApisixRoutes are served")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
		expectStatus("ar", "rejected", ContainSubstring(`reason: Accepted`))
	})

	It("isolates a rejected rule of an otherwise valid route", func() {
		const partialRoute = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: partial
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: served
    match:
      hosts:
      - valid.example.com
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
  - name: rejected
    match:
      hosts:
      - rejected.example.com
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
%s
`
		By("apply an ApisixRoute with one valid and one rejected rule")
		apply(fmt.Sprintf(partialRoute, s.Namespace(), s.Namespace(), rejectedPlugin))

		By("the valid rule is served and the ApisixRoute reports the dropped one")
		expectServed("valid.example.com")
		expectStatus("ar", "partial",
			ContainSubstring(`type: PartiallyInvalid`),
			ContainSubstring(`failed to check the configuration of plugin limit-count`),
		)

		By("fix the rejected rule")
		apply(fmt.Sprintf(partialRoute, s.Namespace(), s.Namespace(), acceptedPlugin))

		By("both rules are served")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
		expectStatus("ar", "partial", Not(ContainSubstring(`type: PartiallyInvalid`)))
	})

	It("isolates a route whose upstream configuration is rejected", func() {
		// retries has no lower bound in the CRD and the data plane requires it to be at
		// least 0, so this reaches the data plane and is rejected on schema grounds.
		const upstream = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-service-e2e-test
  namespace: %s
spec:
  ingressClassName: %s
  retries: %d
`
		By("apply a valid and a rejected ApisixRoute, the rejected one using a rejected upstream configuration")
		apply(fmt.Sprintf(routes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace(), ""))
		apply(fmt.Sprintf(upstream, s.Namespace(), s.Namespace(), -1))

		By("the rejected ApisixRoute reports why it is not served")
		expectStatus("ar", "rejected",
			ContainSubstring(`status: "False"`),
			ContainSubstring(`reason: SyncFailed`),
		)

		By("fix the upstream configuration")
		apply(fmt.Sprintf(upstream, s.Namespace(), s.Namespace(), 1))

		By("both ApisixRoutes are served")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
	})

	It("isolates the rule of a rejected named upstream", func() {
		// A named upstream is referenced by its service's traffic-split, so dropping it
		// alone would leave that reference dangling: the whole rule goes instead.
		const namedUpstream = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: external
  namespace: %s
spec:
  ingressClassName: %s
  retries: %d
  externalNodes:
  - type: Service
    name: httpbin-service-e2e-test
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: named-upstream
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: served
    match:
      hosts:
      - valid.example.com
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
  - name: rejected
    match:
      hosts:
      - rejected.example.com
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    upstreams:
    - name: external
`
		By("apply an ApisixRoute whose second rule references a rejected ApisixUpstream")
		apply(fmt.Sprintf(namedUpstream, s.Namespace(), s.Namespace(), -1, s.Namespace(), s.Namespace()))

		By("the valid rule is served and the ApisixRoute reports the dropped one")
		expectServed("valid.example.com")
		expectStatus("ar", "named-upstream", ContainSubstring(`type: PartiallyInvalid`))

		By("fix the ApisixUpstream")
		apply(fmt.Sprintf(namedUpstream, s.Namespace(), s.Namespace(), 1, s.Namespace(), s.Namespace()))

		By("both rules are served")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
	})

	It("isolates a rejected certificate", func() {
		const tls = `
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: rejected
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - ssl.example.com
  secret:
    name: rejected-cert
    namespace: %s
`
		cert, key := s.GenerateCert(GinkgoT(), []string{"ssl.example.com"})
		By("apply a valid ApisixRoute and an ApisixTls whose private key the data plane cannot parse")
		// The certificate itself is valid, so the controller accepts it; only the data
		// plane rejects the key.
		Expect(s.NewKubeTlsSecret("rejected-cert", cert.String(), "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----")).
			NotTo(HaveOccurred(), "creating Secret")
		apply(fmt.Sprintf(routes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace(), ""))
		apply(fmt.Sprintf(tls, s.Namespace(), s.Namespace(), s.Namespace()))

		By("the ApisixRoutes are served and the ApisixTls reports why it is not")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
		expectStatus("apisixtls", "rejected",
			ContainSubstring(`status: "False"`),
			ContainSubstring(`reason: SyncFailed`),
		)

		By("fix the private key")
		Expect(s.NewKubeTlsSecret("rejected-cert", cert.String(), key.String())).NotTo(HaveOccurred(), "updating Secret")

		By("the ApisixTls is accepted")
		expectStatus("apisixtls", "rejected", ContainSubstring(`reason: Accepted`))
	})

	It("isolates a rejected global rule", func() {
		const globalRule = `
apiVersion: apisix.apache.org/v2
kind: ApisixGlobalRule
metadata:
  name: rejected
  namespace: %s
spec:
  ingressClassName: %s
  plugins:
  - name: limit-count
    enable: true
    config:
      count: %d
      time_window: 60
      rejected_code: 503
      key: remote_addr
`
		By("apply a valid ApisixRoute and a rejected ApisixGlobalRule")
		apply(fmt.Sprintf(routes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace(), ""))
		apply(fmt.Sprintf(globalRule, s.Namespace(), s.Namespace(), 0))

		By("the ApisixRoutes are served and the ApisixGlobalRule reports why it is not")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
		expectStatus("apisixglobalrule", "rejected",
			ContainSubstring(`status: "False"`),
			ContainSubstring(`reason: SyncFailed`),
		)

		By("fix the ApisixGlobalRule")
		apply(fmt.Sprintf(globalRule, s.Namespace(), s.Namespace(), 100))

		By("the ApisixGlobalRule is accepted and the routes keep being served")
		expectStatus("apisixglobalrule", "rejected", ContainSubstring(`reason: Accepted`))
		expectServed("valid.example.com")
	})

	It("isolates a rejected GatewayProxy plugin", func() {
		gatewayProxyWithPlugin := func(count int) string {
			return s.GetGatewayProxySpec() + fmt.Sprintf(`  plugins:
  - name: limit-count
    enabled: true
    config:
      count: %d
      time_window: 60
      rejected_code: 503
      key: remote_addr
`, count)
		}
		By("apply a valid ApisixRoute and a GatewayProxy carrying a rejected plugin")
		apply(fmt.Sprintf(routes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace(), ""))
		apply(gatewayProxyWithPlugin(0))

		By("the ApisixRoutes are served and the GatewayProxy reports the dropped plugin")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
		expectStatus("gatewayproxy", "apisix-proxy-config",
			ContainSubstring(`type: PluginsProgrammed`),
			ContainSubstring(`reason: Invalid`),
			ContainSubstring(`limit-count`),
		)

		By("fix the GatewayProxy plugin")
		apply(gatewayProxyWithPlugin(100))

		By("the GatewayProxy reports its plugins programmed")
		expectStatus("gatewayproxy", "apisix-proxy-config",
			ContainSubstring(`type: PluginsProgrammed`),
			ContainSubstring(`reason: Programmed`),
		)
		expectServed("valid.example.com")
	})

	It("isolates rejected GatewayProxy plugin metadata", func() {
		gatewayProxyWithMetadata := func(logFormat string) string {
			return s.GetGatewayProxySpec() + fmt.Sprintf(`  pluginMetadata:
    http-logger:
      log_format: %s
`, logFormat)
		}
		By("apply a valid ApisixRoute and a GatewayProxy carrying rejected plugin metadata")
		apply(fmt.Sprintf(routes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace(), ""))
		apply(gatewayProxyWithMetadata(`"not an object"`))

		By("the ApisixRoutes are served and the GatewayProxy reports the dropped metadata")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
		expectStatus("gatewayproxy", "apisix-proxy-config",
			ContainSubstring(`type: PluginsProgrammed`),
			ContainSubstring(`reason: Invalid`),
			ContainSubstring(`http-logger`),
		)

		By("fix the plugin metadata")
		apply(gatewayProxyWithMetadata(`{"host": "$host"}`))

		By("the GatewayProxy reports its plugins programmed")
		expectStatus("gatewayproxy", "apisix-proxy-config",
			ContainSubstring(`type: PluginsProgrammed`),
			ContainSubstring(`reason: Programmed`),
		)
		expectServed("valid.example.com")
	})
})
