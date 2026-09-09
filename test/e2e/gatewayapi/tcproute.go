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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("TCPRoute E2E Test", Label("networking.k8s.io", "tcproute"), func() {
	s := scaffold.NewDefaultScaffold()

	// Shared by every TCPRoute context so the listener port cannot drift between them.
	var tcpGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: tcp
    protocol: TCP
    # Must equal APISIX's physical stream_proxy TCP port: the e2e controller runs
    # with listener_port_match_mode=auto, so sectionName targeting injects
    # server_port from this listener; it must match the port connections arrive on.
    port: 9100
    allowedRoutes:
      kinds:
      - kind: TCPRoute
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

	Context("TCPRoute Base", func() {
		var tcpRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: TCPRoute
metadata:
  name: tcp-app-1
spec:
  parentRefs:
  - name: %s
    sectionName: tcp
  rules:
  - backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		BeforeEach(func() {
			// Create GatewayProxy
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).
				NotTo(HaveOccurred(), "creating GatewayProxy")

			// Create GatewayClass
			Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).
				NotTo(HaveOccurred(), "creating GatewayClass")

			// Create Gateway with TCP listener
			Expect(s.CreateResourceFromString(fmt.Sprintf(tcpGateway, s.Namespace(), s.Namespace()))).
				NotTo(HaveOccurred(), "creating Gateway")
		})

		It("should route TCP traffic to backend service", func() {
			By("creating TCPRoute")
			Expect(s.CreateResourceFromString(fmt.Sprintf(tcpRoute, s.Namespace()))).
				NotTo(HaveOccurred(), "creating TCPRoute")

			// Verify TCPRoute status becomes programmed
			routeYaml, _ := s.GetResourceYaml("TCPRoute", "tcp-app-1")
			s.ResourceApplied("TCPRoute", "tcp-app-1", routeYaml, 1)

			By("verifying TCPRoute is functional")
			s.HTTPOverTCPConnectAssert(true, time.Minute*3) // should be able to connect
			By("sending TCP traffic to verify routing")
			s.RequestAssert(&scaffold.RequestAssert{
				Client:   s.NewAPISIXClientOnTCPPort(),
				Method:   "GET",
				Path:     "/get",
				Check:    scaffold.WithExpectedStatus(200),
				Timeout:  time.Second * 60,
				Interval: time.Second * 2,
			})

			By("deleting TCPRoute")
			Expect(s.DeleteResource("TCPRoute", "tcp-app-1")).
				NotTo(HaveOccurred(), "deleting TCPRoute")

			s.HTTPOverTCPConnectAssert(false, time.Minute*3)
		})
	})

	Context("TCPRoute With BackendTrafficPolicy", func() {
		var tcpRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: TCPRoute
metadata:
  name: tcp-tls-upstream
spec:
  parentRefs:
  - name: %s
    sectionName: tcp
  rules:
  - backendRefs:
    - name: nginx
      port: 443
`

		var backendTrafficPolicy = `
apiVersion: apisix.apache.org/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: nginx-tls
spec:
  targetRefs:
  - name: nginx
    kind: Service
    group: ""
  scheme: tls
`

		BeforeEach(func() {
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred(), "creating GatewayClass")
			Expect(s.CreateResourceFromString(fmt.Sprintf(tcpGateway, s.Namespace(), s.Namespace()))).
				NotTo(HaveOccurred(), "creating Gateway")
			s.DeployNginx(framework.NginxOptions{
				Namespace: s.Namespace(),
				Replicas:  ptr.To(int32(1)),
			})
		})

		It("BackendTrafficPolicy scheme tls connects to the upstream over TLS", func() {
			By("creating BackendTrafficPolicy with scheme: tls")
			Expect(s.CreateResourceFromString(backendTrafficPolicy)).NotTo(HaveOccurred(), "creating BackendTrafficPolicy")

			By("creating TCPRoute to the TLS port of nginx")
			s.ResourceApplied("TCPRoute", "tcp-tls-upstream", fmt.Sprintf(tcpRoute, s.Namespace()), 1)

			// The client speaks plain HTTP over TCP; without scheme: tls nginx would
			// reject the request on its TLS port instead of answering with 200.
			s.RequestAssert(&scaffold.RequestAssert{
				Client: s.NewAPISIXClientOnTCPPort(),
				Method: "GET",
				Path:   "/",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(200),
					scaffold.WithExpectedHeader("X-Port", "443"),
					scaffold.WithExpectedBodyContains("Hello, World!"),
				},
				Timeout:  time.Minute * 3,
				Interval: time.Second * 2,
			})
		})
	})

	Context("TCPRoute With L4RoutePolicy", func() {
		var tcpRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: TCPRoute
metadata:
  name: tcp-l4policy
spec:
  parentRefs:
  - name: %s
    sectionName: tcp
  rules:
  - backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		// ip-restriction with blacklist covering all IPv4 addresses blocks all TCP connections.
		var l4RoutePolicyBlockAll = `
apiVersion: apisix.apache.org/v1alpha1
kind: L4RoutePolicy
metadata:
  name: tcp-block-all
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: TCPRoute
    name: tcp-l4policy
  plugins:
  - name: ip-restriction
    config:
      blacklist:
      - "0.0.0.0/0"
`

		var l4RoutePolicyMissingSecret = `
apiVersion: apisix.apache.org/v1alpha1
kind: L4RoutePolicy
metadata:
  name: tcp-block-all
spec:
  targetRefs:
  - group: gateway.networking.k8s.io
    kind: TCPRoute
    name: tcp-l4policy
  plugins:
  - name: ip-restriction
    secretRef:
      name: no-such-secret
    config:
      blacklist:
      - "0.0.0.0/0"
`

		BeforeEach(func() {
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred(), "creating GatewayClass")
			Expect(s.CreateResourceFromString(fmt.Sprintf(tcpGateway, s.Namespace(), s.Namespace()))).
				NotTo(HaveOccurred(), "creating Gateway")
		})

		It("L4RoutePolicy blocks traffic via ip-restriction plugin", func() {
			By("creating TCPRoute")
			s.ResourceApplied("TCPRoute", "tcp-l4policy", fmt.Sprintf(tcpRoute, s.Namespace()), 1)

			By("verifying TCP traffic works before applying L4RoutePolicy")
			s.HTTPOverTCPConnectAssert(true, time.Minute*3)

			By("applying L4RoutePolicy with ip-restriction blacklist")
			s.ApplyL4RoutePolicy(
				types.NamespacedName{Name: s.Namespace()},
				types.NamespacedName{Namespace: s.Namespace(), Name: "tcp-block-all"},
				l4RoutePolicyBlockAll,
				metav1.Condition{
					Type:   string(gatewayv1.PolicyConditionAccepted),
					Status: metav1.ConditionTrue,
				},
			)

			By("verifying TCP traffic is blocked by the L4RoutePolicy")
			s.HTTPOverTCPConnectAssert(false, time.Minute*3)

			By("deleting L4RoutePolicy")
			Expect(s.DeleteResource("L4RoutePolicy", "tcp-block-all")).NotTo(HaveOccurred(), "deleting L4RoutePolicy")

			By("verifying TCP traffic recovers after L4RoutePolicy deletion")
			s.HTTPOverTCPConnectAssert(true, time.Minute*3)
		})

		It("L4RoutePolicy with a missing plugin Secret is rejected", func() {
			By("creating TCPRoute")
			s.ResourceApplied("TCPRoute", "tcp-l4policy", fmt.Sprintf(tcpRoute, s.Namespace()), 1)
			s.HTTPOverTCPConnectAssert(true, time.Minute*3)

			By("applying an L4RoutePolicy whose plugin references a Secret that does not exist")
			s.ApplyL4RoutePolicy(
				types.NamespacedName{Name: s.Namespace()},
				types.NamespacedName{Namespace: s.Namespace(), Name: "tcp-block-all"},
				l4RoutePolicyMissingSecret,
				metav1.Condition{
					Type:   string(gatewayv1.PolicyConditionAccepted),
					Status: metav1.ConditionFalse,
					Reason: string(gatewayv1.PolicyReasonInvalid),
				},
			)

			By("verifying the policy plugins are not attached")
			s.HTTPOverTCPConnectAssert(true, time.Minute*3)
		})
	})
})
