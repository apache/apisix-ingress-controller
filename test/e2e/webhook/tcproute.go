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

package webhook

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test TCPRoute Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "tcproute-webhook-test",
		EnableWebhook: true,
	})

	const tcpGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: tcp
    protocol: TCP
    port: 9000
    allowedRoutes:
      kinds:
      - kind: TCPRoute
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

	BeforeEach(func() {
		By("creating GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("creating GatewayClass")
		err = s.CreateResourceFromString(s.GetGatewayClassYaml())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(2 * time.Second)

		By("creating Gateway with TCP listener")
		err = s.CreateResourceFromString(fmt.Sprintf(tcpGateway, s.Namespace(), s.Namespace()))
		Expect(err).NotTo(HaveOccurred(), "creating TCP-capable Gateway")
		time.Sleep(5 * time.Second)
	})

	It("should warn on missing backend services", func() {
		missingService := "missing-tcp-backend"
		routeName := "webhook-tcproute"
		gatewayName := s.Namespace()
		routeYAML := `
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: %s
spec:
  parentRefs:
  - name: %s
    sectionName: tcp
  rules:
  - backendRefs:
    - name: %s
      port: 80
`

		output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(routeYAML, routeName, gatewayName, missingService))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", s.Namespace(), missingService)))

		By("creating referenced backend service")
		backendService := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: placeholder
  ports:
  - name: tcp
    port: 80
    targetPort: 80
  type: ClusterIP
`, missingService)
		err = s.CreateResourceFromString(backendService)
		Expect(err).NotTo(HaveOccurred(), "creating tcp backend service")

		time.Sleep(2 * time.Second)

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(routeYAML, routeName, gatewayName, missingService))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", s.Namespace(), missingService)))
	})
})
