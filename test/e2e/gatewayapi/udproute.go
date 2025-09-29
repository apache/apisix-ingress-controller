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

var _ = Describe("UDPRoute E2E Test", Label("networking.k8s.io", "udproute"), func() {
	s := scaffold.NewDefaultScaffold()
	Context("UDPRoute Base", func() {

		var udpGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: udp
    protocol: UDP
    port: 80
    allowedRoutes:
      kinds:
      - kind: UDPRoute
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

		var udpRoute = `
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: UDPRoute
metadata:
  name: udp-app-1
spec:
  parentRefs:
  - name: %s
    sectionName: udp
  rules:
  - backendRefs:
    - name: %s
      port: %d
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

			// Create Gateway with UDP listener
			gatewayName := s.Namespace()
			Expect(s.CreateResourceFromString(fmt.Sprintf(udpGateway, gatewayName, gatewayClassName))).
				NotTo(HaveOccurred(), "creating Gateway")

			gwyaml, _ := s.GetResourceYaml("Gateway", gatewayName)
			s.ResourceApplied("Gateway", gatewayName, gwyaml, 1)
		})

		It("should route UDP traffic to backend service", func() {
			dnsSvc := s.NewCoreDNSService()
			gatewayName := s.Namespace()
			By("creating UDPRoute")
			Expect(s.CreateResourceFromString(fmt.Sprintf(udpRoute, gatewayName, dnsSvc.Name, dnsSvc.Spec.Ports[0].Port))).
				NotTo(HaveOccurred(), "creating UDPRoute")

			// Verify UDPRoute status becomes programmed
			routeYaml, _ := s.GetResourceYaml("UDPRoute", "udp-app-1")
			s.ResourceApplied("UDPRoute", "udp-app-1", routeYaml, 1)

			svc := s.GetDataplaneService()

			// test dns query
			output, err := s.RunDigDNSClientFromK8s(fmt.Sprintf("@%s", svc.Name), "-p", "9200", "github.com")
			Expect(err).NotTo(HaveOccurred(), "dig github.com via apisix udp proxy")
			Expect(output).To(ContainSubstring("ADDITIONAL SECTION"))

			time.Sleep(3 * time.Second)
			output = s.GetDeploymentLogs(scaffold.CoreDNSDeployment)
			Expect(output).To(ContainSubstring("github.com. udp"))
		})
	})
})
