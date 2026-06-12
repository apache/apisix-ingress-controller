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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Gateway Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "gateway-webhook-test",
		EnableWebhook: true,
	})

	It("should warn when referenced GatewayProxy does not exist on create and update", func() {
		By("creating GatewayClass with controller name")
		err := s.CreateResourceFromString(s.GetGatewayClassYaml())
		Expect(err).ShouldNot(HaveOccurred())

		By("creating Gateway referencing a missing GatewayProxy")
		missingName := "missing-proxy"
		gwYAML := `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
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

		Eventually(func(g Gomega) {
			output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gwYAML, s.Namespace(), s.Namespace(), missingName))
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), missingName)))
		}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())

		By("updating Gateway to reference another missing GatewayProxy")
		missingName2 := "missing-proxy-2"
		Eventually(func(g Gomega) {
			output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gwYAML, s.Namespace(), s.Namespace(), missingName2))
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), missingName2)))
		}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())

		By("create GatewayProxy")
		err = s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("updating Gateway to reference an existing GatewayProxy")
		existingName := "apisix-proxy-config"
		Eventually(func(g Gomega) {
			output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gwYAML, s.Namespace(), s.Namespace(), existingName))
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), existingName)))
		}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())

		By("delete Gateway")
		err = s.DeleteResource("Gateway", s.Namespace())
		Expect(err).ShouldNot(HaveOccurred())

		By("delete GatewayClass")
		err = s.DeleteResource("GatewayClass", s.Namespace())
		Expect(err).ShouldNot(HaveOccurred())
	})
})
