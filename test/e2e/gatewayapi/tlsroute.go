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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
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
apiVersion: gateway.networking.k8s.io/v1
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
			// TLSRoute is served by the stream subsystem through a port-forward
			// tunnel, which needs the same headroom as the other stream specs
			// (tcproute/udproute); the 30s default is too tight on a loaded runner.
			s.RequestAssert(&scaffold.RequestAssert{
				Client:  client,
				Method:  http.MethodGet,
				Path:    "/ip",
				Check:   scaffold.WithExpectedStatus(http.StatusOK),
				Timeout: time.Minute * 3,
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Client:  client,
				Method:  http.MethodGet,
				Path:    "/notfound",
				Check:   scaffold.WithExpectedStatus(http.StatusNotFound),
				Timeout: time.Minute * 3,
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
			}).WithTimeout(time.Minute*3).
				Should(ContainSubstring("EOF"), "should get EOF after deleting TLSRoute")
		})
	})

	Context("TLSRoute Passthrough", func() {
		// The certificate the e2e nginx serves, and the CA that signed it.
		const backendSNI = "server.example.com"

		var passthroughGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: tls-passthrough-gateway
spec:
  gatewayClassName: %s
  listeners:
    - name: passthrough
      protocol: TLS
      # Must equal APISIX's physical stream_proxy listen configured with
      # tls_passthrough (see the e2e apisix manifest): the controller runs with
      # listener_port_match_mode=auto, so the port reaches the stream route as a
      # server_port match and has to be one the data plane actually accepts on.
      port: 9120
      hostname: server.example.com
      tls:
        mode: Passthrough
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

		var passthroughRoute = `
apiVersion: gateway.networking.k8s.io/v1
kind: TLSRoute
metadata:
  name: tls-passthrough-route
spec:
  parentRefs:
  - name: tls-passthrough-gateway
    sectionName: passthrough
  hostnames: ["server.example.com"]
  rules:
  - backendRefs:
    - name: nginx
      port: 443
`

		BeforeEach(func() {
			By("create GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

			By("create GatewayClass")
			Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred(), "creating GatewayClass")

			By("create Gateway with a Passthrough listener")
			Expect(s.CreateResourceFromString(fmt.Sprintf(passthroughGateway, s.Namespace()))).NotTo(HaveOccurred(), "creating Gateway")

			By("deploy the TLS backend")
			s.DeployNginx(framework.NginxOptions{
				Namespace: s.Namespace(),
				Replicas:  ptr.To(int32(1)),
			})
		})

		It("forwards the stream to the backend that owns the certificate", func() {
			s.ResourceApplied("TLSRoute", "tls-passthrough-route", passthroughRoute, 1)

			// The client verifies the served chain against the backend's own CA.
			// The gateway holds no certificate for this listener - Passthrough
			// takes no certificateRefs - so a chain that validates here can only
			// have come from nginx, which is what passthrough means.
			s.RequestAssert(&scaffold.RequestAssert{
				Client: s.NewAPISIXClientWithTLSPassthrough(backendSNI, []byte(framework.TestCACert)),
				Method: http.MethodGet,
				Path:   "/",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains("Hello, World!"),
				},
				Timeout:  time.Minute * 3,
				Interval: time.Second * 2,
			})
		})
	})
})
