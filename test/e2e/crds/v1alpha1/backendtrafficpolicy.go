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

package v1alpha1

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test BackendTrafficPolicy base on HTTPRoute", Label("apisix.apache.org", "v1alpha1", "backendtrafficpolicy"), func() {
	var (
		s   = scaffold.NewDefaultScaffold()
		err error
	)

	var defaultHTTPRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - "httpbin.org"
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    - path:
        type: Exact
        value: /headers
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
	var gatewayBeforeEach = func() {
		By("create GatewayProxy")
		err = s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create GatewayClass")
		err = s.CreateResourceFromString(s.GetGatewayClassYaml())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")

		By("create Gateway")
		err = s.CreateResourceFromString(s.GetGatewayYaml())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")

		By("create HTTPRoute")
		s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, fmt.Sprintf(defaultHTTPRoute, s.Namespace(), s.Namespace()))
	}

	Context("Rewrite Upstream Host", func() {
		var createUpstreamHost = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.example.com
`

		var updateUpstreamHost = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.update.example.com
`

		BeforeEach(gatewayBeforeEach)

		It("should rewrite upstream host", func() {
			s.ResourceApplied("BackendTrafficPolicy", "httpbin", createUpstreamHost, 1)
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"Host": "httpbin.org",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyContains(
						"httpbin.example.com",
					),
				},
			})

			s.ResourceApplied("BackendTrafficPolicy", "httpbin", updateUpstreamHost, 2)
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"Host": "httpbin.org",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyContains(
						"httpbin.update.example.com",
					),
				},
			})

			err := s.DeleteResourceFromString(createUpstreamHost)
			Expect(err).NotTo(HaveOccurred(), "deleting BackendTrafficPolicy")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"Host": "httpbin.org",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyNotContains(
						"httpbin.update.example.com",
						"httpbin.example.com",
					),
				},
			})
		})
	})

	Context("Section Name", func() {
		// httpbin-service-e2e-test exposes two named ports backed by the same pod:
		// "http" (80) and "http-v2" (8080). Routing one host to each port lets us
		// assert that a sectionName-scoped policy only attaches to the matching port.
		var routeToHTTPV2Port = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin-v2
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - "httpbin-v2.org"
  rules:
  - matches:
    - path:
        type: Exact
        value: /headers
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 8080
`

		var policyTmpl = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
    sectionName: %s
  passHost: rewrite
  upstreamHost: %s
`

		BeforeEach(func() {
			gatewayBeforeEach()
			By("create HTTPRoute routing to the http-v2 (8080) port")
			s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-v2"}, fmt.Sprintf(routeToHTTPV2Port, s.Namespace(), s.Namespace()))
		})

		// assertScopedToPort applies a policy targeting the given port name and
		// asserts it attaches to exactly that port: the host routed to the matched
		// port gets the rewritten upstreamHost, while the host routed to the other
		// port keeps the original host. matchedHost routes to the port named by
		// sectionName; otherHost routes to the other port.
		assertScopedToPort := func(sectionName, rewriteHost, matchedHost, otherHost string) {
			s.ResourceApplied("BackendTrafficPolicy", "httpbin", fmt.Sprintf(policyTmpl, sectionName, rewriteHost), 1)

			// Drive traffic to both hosts so both per-port upstreams are registered.
			for _, host := range []string{matchedHost, otherHost} {
				s.RequestAssert(&scaffold.RequestAssert{
					Method:  "GET",
					Path:    "/headers",
					Host:    host,
					Headers: map[string]string{"Host": host},
					Checks:  []scaffold.ResponseCheckFunc{scaffold.WithExpectedStatus(200)},
				})
			}

			// The Service is split into two per-port upstreams (http/80 and
			// http-v2/8080) with identical nodes. The sectionName-scoped policy must
			// attach to exactly one of them; under name-only matching it would attach
			// to both (the whole Service), which this assertion rejects.
			Eventually(func(g Gomega) {
				ups, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
				g.Expect(err).ToNot(HaveOccurred(), "listing upstreams")
				g.Expect(ups).To(HaveLen(2), "two per-port upstreams should exist")

				var rewritten []*adctypes.Upstream
				for _, u := range ups {
					if u.PassHost == "rewrite" && u.UpstreamHost == rewriteHost {
						rewritten = append(rewritten, u)
					}
				}
				g.Expect(rewritten).To(HaveLen(1), "policy must attach to exactly the sectionName-matched port, not the whole Service")
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())

			By("the matched port reflects the rewritten host")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/headers",
				Host:    matchedHost,
				Headers: map[string]string{"Host": matchedHost},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyContains(rewriteHost),
				},
			})

			By("the other port keeps the original host")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/headers",
				Host:    otherHost,
				Headers: map[string]string{"Host": otherHost},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedBodyNotContains(rewriteHost),
				},
			})
		}

		It("attaches the policy to the http-v2 (8080) port when sectionName is http-v2", func() {
			assertScopedToPort("http-v2", "httpbin.section-v2.example.com", "httpbin-v2.org", "httpbin.org")
		})

		It("attaches the policy to the http (80) port when sectionName is http", func() {
			assertScopedToPort("http", "httpbin.section-http.example.com", "httpbin.org", "httpbin-v2.org")
		})
	})

	Context("Health Check", func() {
		var policyWithActiveHealthCheck = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  healthCheck:
    active:
      type: http
      httpPath: /get
      healthy:
        httpCodes: [200]
        interval: 1s
      unhealthy:
        httpCodes: [500]
        httpFailures: 2
        interval: 1s
`

		var policyWithActiveAndPassiveHealthCheck = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  healthCheck:
    active:
      type: http
      httpPath: /get
      healthy:
        httpCodes: [200]
        interval: 1s
      unhealthy:
        httpCodes: [500]
        httpFailures: 2
        interval: 1s
    passive:
      type: http
      healthy:
        httpCodes: [200]
      unhealthy:
        httpCodes: [502, 503]
        httpFailures: 3
`

		BeforeEach(gatewayBeforeEach)

		It("should configure active health check on upstream", func() {
			s.ResourceApplied("BackendTrafficPolicy", "httpbin", policyWithActiveHealthCheck, 1)

			// Trigger some traffic so APISIX registers the upstream
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
				},
			})
			Eventually(func(g Gomega) {
				ups, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
				g.Expect(err).ToNot(HaveOccurred(), "listing upstreams")
				g.Expect(ups).NotTo(BeEmpty(), "upstreams should not be empty")

				var target *adctypes.Upstream
				for _, u := range ups {
					if u.Checks != nil {
						target = u
						break
					}
				}
				g.Expect(target).NotTo(BeNil(), "upstream with health check should exist")
				g.Expect(target.Checks.Active).NotTo(BeNil(), "active health check should be configured")
				g.Expect(target.Checks.Active.HTTPPath).To(Equal("/get"), "active health check http path")
				g.Expect(target.Checks.Active.Healthy.Interval).To(Equal(1), "active healthy interval")
				g.Expect(target.Checks.Active.Healthy.HTTPStatuses).To(Equal([]int{200}), "active healthy http codes")
				g.Expect(target.Checks.Active.Unhealthy.Interval).To(Equal(1), "active unhealthy interval")
				g.Expect(target.Checks.Active.Unhealthy.HTTPFailures).To(Equal(2), "active unhealthy http failures")
				g.Expect(target.Checks.Active.Unhealthy.HTTPStatuses).To(Equal([]int{500}), "active unhealthy http codes")
				g.Expect(target.Checks.Passive).To(BeNil(), "passive health check should not be configured")
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should configure active and passive health checks on upstream", func() {
			s.ResourceApplied("BackendTrafficPolicy", "httpbin", policyWithActiveAndPassiveHealthCheck, 1)

			// Trigger some traffic so APISIX registers the upstream
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
				},
			})
			Eventually(func(g Gomega) {
				ups, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
				g.Expect(err).ToNot(HaveOccurred(), "listing upstreams")
				g.Expect(ups).NotTo(BeEmpty(), "upstreams should not be empty")

				var target *adctypes.Upstream
				for _, u := range ups {
					if u.Checks != nil && u.Checks.Passive != nil {
						target = u
						break
					}
				}
				g.Expect(target).NotTo(BeNil(), "upstream with active and passive health check should exist")

				// Verify active health check
				g.Expect(target.Checks.Active).NotTo(BeNil(), "active health check should be configured")
				g.Expect(target.Checks.Active.HTTPPath).To(Equal("/get"), "active health check http path")
				g.Expect(target.Checks.Active.Healthy.HTTPStatuses).To(Equal([]int{200}), "active healthy http codes")
				g.Expect(target.Checks.Active.Unhealthy.HTTPFailures).To(Equal(2), "active unhealthy http failures")

				// Verify passive health check
				g.Expect(target.Checks.Passive.Healthy.HTTPStatuses).To(Equal([]int{200}), "passive healthy http codes")
				g.Expect(target.Checks.Passive.Unhealthy.HTTPStatuses).To(Equal([]int{502, 503}), "passive unhealthy http codes")
				g.Expect(target.Checks.Passive.Unhealthy.HTTPFailures).To(Equal(3), "passive unhealthy http failures")
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should remove health check when policy is deleted", func() {
			s.ResourceApplied("BackendTrafficPolicy", "httpbin", policyWithActiveHealthCheck, 1)

			// Trigger traffic to establish upstream
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
				},
			})
			// Verify health check is present on the target upstream
			Eventually(func(g Gomega) {
				ups, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
				g.Expect(err).ToNot(HaveOccurred())
				hasHealthCheck := false
				for _, u := range ups {
					if u.Checks != nil {
						hasHealthCheck = true
						break
					}
				}
				g.Expect(hasHealthCheck).To(BeTrue(), "upstream should have health check before policy deletion")
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())

			// Delete the policy
			err = s.DeleteResourceFromString(policyWithActiveHealthCheck)
			Expect(err).NotTo(HaveOccurred(), "deleting BackendTrafficPolicy")

			// Verify health check is removed from the target upstream
			Eventually(func(g Gomega) {
				ups, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(ups).NotTo(BeEmpty(), "upstreams should still exist after policy deletion")
				for _, u := range ups {
					g.Expect(u.Checks).To(BeNil(), "upstream should not have health check after policy deletion")
				}
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})
	})
})

var _ = Describe("Test BackendTrafficPolicy base on Ingress", Label("apisix.apache.org", "v1alpha1", "backendtrafficpolicy"), func() {
	s := scaffold.NewDefaultScaffold()

	var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-default
  annotations:
    ingressclass.kubernetes.io/is-default-class: "true"
spec:
  controller: %s
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: %s
    scope: "Namespace"
`

	var defaultIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress-default
spec:
  rules:
  - host: httpbin.org
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
	var beforeEach = func() {
		By("create GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create IngressClass with GatewayProxy reference")
		err = s.CreateResourceFromString(fmt.Sprintf(defaultIngressClass, s.GetControllerName(), s.Namespace()))
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass with GatewayProxy")

		By("create Ingress with GatewayProxy IngressClass")
		err = s.CreateResourceFromString(defaultIngress)
		Expect(err).NotTo(HaveOccurred(), "creating Ingress with GatewayProxy IngressClass")
	}

	// Tests concerning the default ingress class need to be run serially
	Context("Rewrite Upstream Host", Serial, func() {
		var createUpstreamHost = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
  namespace: %s
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.example.com
`

		var updateUpstreamHost = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: httpbin
  namespace: %s
spec:
  targetRefs:
  - name: httpbin-service-e2e-test
    kind: Service
    group: ""
  passHost: rewrite
  upstreamHost: httpbin.update.example.com
`

		BeforeEach(beforeEach)
		It("should rewrite upstream host", func() {
			reqAssert := &scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"Host": "httpbin.org",
				},
			}

			s.ResourceApplied("BackendTrafficPolicy", "httpbin", fmt.Sprintf(createUpstreamHost, s.Namespace()), 1)
			s.RequestAssert(reqAssert.SetChecks(
				scaffold.WithExpectedStatus(200),
				scaffold.WithExpectedBodyContains("httpbin.example.com"),
			))

			s.ResourceApplied("BackendTrafficPolicy", "httpbin", fmt.Sprintf(updateUpstreamHost, s.Namespace()), 2)
			s.RequestAssert(reqAssert.SetChecks(
				scaffold.WithExpectedStatus(200),
				scaffold.WithExpectedBodyContains("httpbin.update.example.com"),
			))

			err := s.DeleteResourceFromString(fmt.Sprintf(createUpstreamHost, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "deleting BackendTrafficPolicy")

			s.RequestAssert(reqAssert.SetChecks(
				scaffold.WithExpectedStatus(200),
				scaffold.WithExpectedBodyNotContains(
					"httpbin.update.example.com",
					"httpbin.example.com",
				),
			))
		})
	})
})
