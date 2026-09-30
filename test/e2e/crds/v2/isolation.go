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
	// A consistent-hash load balancer needs a key to hash on. The data plane checks that
	// in code rather than in its schema, so the configuration gets past ADC's schema
	// check and is rejected only by the data plane itself.
	const rejectedHashKey = ""
	const acceptedHashKey = "    key: remote_addr"
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
		// The rejected route is the only one this ApisixRoute has, so none of it is served.
		expectStatus("ar", "rejected",
			ContainSubstring(`status: "False"`),
			ContainSubstring(`reason: SyncFailed`),
			ContainSubstring(`limit-count`),
		)

		By("fix the rejected ApisixRoute")
		apply(fmt.Sprintf(routes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace(), acceptedPlugin))

		By("both ApisixRoutes are served")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
		expectStatus("ar", "rejected", ContainSubstring(`reason: Accepted`))
	})

	It("isolates a route whose upstream configuration is rejected", func() {
		// The rejected route needs a backend of its own: routes reuses
		// httpbin-service-e2e-test for both routes, and an ApisixUpstream named after that
		// service would apply to the valid route too.
		const aliasBackend = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-alias
  namespace: %s
spec:
  selector:
    app: httpbin-deployment-e2e-test
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 80
  type: ClusterIP
---
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-alias
  namespace: %s
spec:
  ingressClassName: %s
  loadbalancer:
    type: chash
    hashOn: vars
%s
`
		const upstreamRoutes = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: valid-upstream
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - valid-upstream.example.com
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
---
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: rejected-upstream
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - rejected-upstream.example.com
      paths:
      - /*
    backends:
    - serviceName: httpbin-alias
      servicePort: 80
`
		By("apply a valid and a rejected ApisixRoute, the rejected one using a rejected upstream configuration on its own backend")
		apply(fmt.Sprintf(aliasBackend, s.Namespace(), s.Namespace(), s.Namespace(), rejectedHashKey))
		apply(fmt.Sprintf(upstreamRoutes, s.Namespace(), s.Namespace(), s.Namespace(), s.Namespace()))

		By("the valid ApisixRoute stays served and the rejected one reports why it is not")
		expectServed("valid-upstream.example.com")
		expectStatus("ar", "rejected-upstream",
			ContainSubstring(`status: "False"`),
			ContainSubstring(`reason: SyncFailed`),
		)

		By("fix the upstream configuration")
		apply(fmt.Sprintf(aliasBackend, s.Namespace(), s.Namespace(), s.Namespace(), acceptedHashKey))

		By("both ApisixRoutes are served")
		expectServed("valid-upstream.example.com")
		expectServed("rejected-upstream.example.com")
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
  loadbalancer:
    type: chash
    hashOn: vars
%s
  externalNodes:
  - type: Domain
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
		apply(fmt.Sprintf(namedUpstream, s.Namespace(), s.Namespace(), rejectedHashKey, s.Namespace(), s.Namespace()))

		By("the valid rule is served and the ApisixRoute reports the dropped one")
		expectServed("valid.example.com")
		expectStatus("ar", "named-upstream",
			ContainSubstring(`status: "False"`),
			ContainSubstring(`reason: SyncFailed`),
		)

		By("fix the ApisixUpstream")
		apply(fmt.Sprintf(namedUpstream, s.Namespace(), s.Namespace(), acceptedHashKey, s.Namespace(), s.Namespace()))

		By("both rules are served")
		expectServed("valid.example.com")
		expectServed("rejected.example.com")
	})
})
