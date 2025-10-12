// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test TLSRoute", Label("networking.k8s.io", "tlsroute"), func() {
	s := scaffold.NewDefaultScaffold()

	Context("TLSRoute Base", func() {
		var (
			host       = "api6.com"
			secretName = _secretName
			tlsGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: tls-gateway
spec:
  gatewayClassName: %s
  listeners:
    - name: https
      protocol: TLS
      port: 443
      hostname: api6.com
      tls:
        certificateRefs:
        - kind: Secret
          group: ""
          name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`
			tlsRoute = `
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TLSRoute
metadata:
  name: tls-route
spec:
  parentRefs:
  - name: tls-gateway
  hostnames: ["api6.com"]
  rules:
  - backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		)
		BeforeEach(func() {
			createSecret(s, secretName)
			By("create GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

			By("create GatewayClass")
			Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred(), "creating GatewayClass")

			// Create Gateway with TCP listener
			By("create Gateway")
			Expect(s.CreateResourceFromString(fmt.Sprintf(tlsGateway, s.Namespace(), secretName))).NotTo(HaveOccurred(), "creating Gateway")
		})
		It("Basic", func() {
			s.ResourceApplied("TLSRoute", "tls-route", tlsRoute, 1)

			client := s.NewAPISIXClientWithTLSProxy(host)
			s.RequestAssert(&scaffold.RequestAssert{
				Client: client,
				Method: http.MethodGet,
				Path:   "/ip",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Client: client,
				Method: http.MethodGet,
				Path:   "/notfound",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})

			Expect(s.DeleteResourceFromString(tlsRoute)).NotTo(HaveOccurred(), "deleting TLSRoute")

			s.RetryAssertion(func() string {
				var errMsg string
				reporter := &scaffold.ErrorReporter{}
				_ = client.GET("/ip").WithReporter(reporter).Expect()
				if reporter.Err() != nil {
					errMsg = reporter.Err().Error()
				}
				return errMsg
			}).Should(ContainSubstring("EOF"), "should get EOF after deleting TLSRoute")
		})
	})
})
