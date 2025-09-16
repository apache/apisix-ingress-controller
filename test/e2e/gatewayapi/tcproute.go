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

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TCPRoute E2E Test", func() {
	s := scaffold.NewDefaultScaffold()

	var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
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
		return fmt.Sprintf(gatewayProxyYaml, s.Namespace(), framework.ProviderType, s.AdminKey())
	}

	var gatewayClassYaml = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`
	Context("TCPRoute Base", func() {
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
    port: 80
    allowedRoutes:
      kinds:
      - kind: TCPRoute
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: %s
`

		var tcpRoute = `
apiVersion: gateway.networking.k8s.io/v1alpha2
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
			Expect(s.CreateResourceFromStringWithNamespace(getGatewayProxySpec(), s.Namespace())).
				NotTo(HaveOccurred(), "creating GatewayProxy")

			// Create GatewayClass
			gatewayClassName := s.Namespace()
			Expect(s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClassYaml, gatewayClassName, s.GetControllerName()), "")).
				NotTo(HaveOccurred(), "creating GatewayClass")

			s.RetryAssertion(func() string {
				gcyaml, _ := s.GetResourceYaml("GatewayClass", gatewayClassName)
				return gcyaml
			}).Should(
				And(
					ContainSubstring(`status: "True"`),
					ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"),
				),
				"check GatewayClass condition",
			)

			// Create Gateway with TCP listener
			gatewayName := s.Namespace()
			Expect(s.CreateResourceFromStringWithNamespace(fmt.Sprintf(tcpGateway, gatewayName, gatewayClassName, s.Namespace()), s.Namespace())).
				NotTo(HaveOccurred(), "creating Gateway")

			s.RetryAssertion(func() string {
				gwyaml, _ := s.GetResourceYaml("Gateway", gatewayName)
				return gwyaml
			}).Should(
				And(
					ContainSubstring(`status: "True"`),
					ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controlle"),
				),
				"check Gateway condition status",
			)
		})

		It("should route TCP traffic to backend service", func() {
			gatewayName := s.Namespace()
			By("creating TCPRoute")
			Expect(s.CreateResourceFromString(fmt.Sprintf(tcpRoute, gatewayName))).
				NotTo(HaveOccurred(), "creating TCPRoute")

			// Verify TCPRoute status becomes programmed
			s.RetryAssertion(func() string {
				routeYaml, _ := s.GetResourceYaml("TCPRoute", "tcp-app-1")
				return routeYaml
			}).Should(
				ContainSubstring(`status: "True"`),
				"check TCPRoute status",
			)

			By("verifying TCPRoute is functional")
			s.HTTPOverTCPConnectAssert(true, time.Second*10) // should be able to connect
			By("sending TCP traffic to verify routing")
			s.RequestAssert(&scaffold.RequestAssert{
				Client:   s.NewAPISIXClientOnTCPPort(),
				Method:   "GET",
				Path:     "/get",
				Check:    scaffold.WithExpectedStatus(200),
				Timeout:  time.Minute * 30,
				Interval: time.Second * 2,
			})

			By("deleting TCPRoute")
			Expect(s.DeleteResource("TCPRoute", "tcp-app-1")).
				NotTo(HaveOccurred(), "deleting TCPRoute")

			s.HTTPOverTCPConnectAssert(false, time.Second*10)
		})
	})
})
