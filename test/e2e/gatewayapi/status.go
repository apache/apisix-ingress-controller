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

var _ = Describe("Test Gateway API Status", Label("networking.k8s.io", "httproute"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: fmt.Sprintf("apisix.apache.org/apisix-ingress-controller-%d", time.Now().Unix()),
		})
	)

	Context("Test HTTPRoute Sync Status", func() {
		const httproute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: %s
spec:
  parentRefs:
  - name: %s
  hostnames:
  - "httpbin"
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		const gatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

		const defaultGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: %s
`
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := s.GetGatewayProxyYaml()
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create GatewayClass")
			gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
			err = s.CreateResourceFromString(fmt.Sprintf(gatewayClass, gatewayClassName, s.GetControllerName()))
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("create Gateway")
			err = s.CreateResourceFromString(fmt.Sprintf(defaultGateway, s.Namespace(), s.Namespace(), gatewayClassName, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(5 * time.Second)

			By("check Gateway condition")
			gwyaml, err := s.GetResourceYaml("Gateway", s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
			Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
			Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"), "checking Gateway condition message")
		})
		AfterEach(func() {
			_ = s.DeleteResource("Gateway", s.Namespace())
		})

		It("dataplane unavailable", func() {
			By("Create HTTPRoute")
			err := s.CreateResourceFromString(fmt.Sprintf(httproute, s.Namespace(), s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute")

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(200),
			})
			time.Sleep(8 * time.Second)
			s.Deployer.ScaleDataplane(0)

			By("check ApisixRoute status")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("httproute", "httpbin", "-o", "yaml", "-n", s.Namespace())
				return output
			}).WithTimeout(80 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "False"`),
						ContainSubstring(`reason: SyncFailed`),
					),
				)
			time.Sleep(8 * time.Second)
			s.Deployer.ScaleDataplane(1)

			By("check ApisixRoute status after scaling up")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("httproute", "httpbin", "-o", "yaml", "-n", s.Namespace())
				return output
			}).WithTimeout(80 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "True"`),
						ContainSubstring(`reason: Accepted`),
					),
				)

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(200),
			})
		})
	})
})
