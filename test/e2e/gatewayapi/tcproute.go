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

var _ = Describe("TCPRoute E2E Test", Label("networking.k8s.io", "tcproute"), func() {
	s := scaffold.NewDefaultScaffold()
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
      name: apisix-proxy-config
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
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).
				NotTo(HaveOccurred(), "creating GatewayProxy")

			// Create GatewayClass
			gatewayClassName := s.Namespace()
			Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).
				NotTo(HaveOccurred(), "creating GatewayClass")
			gcyaml, _ := s.GetResourceYaml("GatewayClass", gatewayClassName)
			s.ResourceApplied("GatewayClass", gatewayClassName, gcyaml, 1)

			// Create Gateway with TCP listener
			gatewayName := s.Namespace()
			Expect(s.CreateResourceFromString(fmt.Sprintf(tcpGateway, gatewayName, gatewayClassName))).
				NotTo(HaveOccurred(), "creating Gateway")

			gwyaml, _ := s.GetResourceYaml("Gateway", gatewayName)
			s.ResourceApplied("Gateway", gatewayName, gwyaml, 1)
		})

		It("should route TCP traffic to backend service", func() {
			gatewayName := s.Namespace()
			By("creating TCPRoute")
			Expect(s.CreateResourceFromString(fmt.Sprintf(tcpRoute, gatewayName))).
				NotTo(HaveOccurred(), "creating TCPRoute")

			// Verify TCPRoute status becomes programmed
			routeYaml, _ := s.GetResourceYaml("TCPRoute", "tcp-app-1")
			s.ResourceApplied("TCPRoute", "tcp-app-1", routeYaml, 1)

			By("verifying TCPRoute is functional")
			s.HTTPOverTCPConnectAssert(true, time.Minute*5) // should be able to connect
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

			s.HTTPOverTCPConnectAssert(false, time.Minute*5)
		})
	})
})
