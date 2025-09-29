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

var _ = Describe("Test ApisixTls Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "apisixtls-webhook-test",
		EnableWebhook: true,
	})

	BeforeEach(func() {
		By("creating GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("creating IngressClass")
		err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	It("should warn on missing TLS secrets", func() {
		serverSecret := "missing-server-tls"
		clientSecret := "missing-client-ca"
		tlsName := "webhook-apisixtls"
		tlsYAML := `
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - webhook.example.com
  secret:
    name: %s
    namespace: %s
  client:
    caSecret:
      name: %s
      namespace: %s
`

		output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(tlsYAML, tlsName, s.Namespace(), s.Namespace(), serverSecret, s.Namespace(), clientSecret, s.Namespace()))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), serverSecret)))
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), clientSecret)))

		By("creating referenced TLS secrets")
		serverSecretYAML := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: kubernetes.io/tls
stringData:
  tls.crt: dummy-cert
  tls.key: dummy-key
`, serverSecret, s.Namespace())
		err = s.CreateResourceFromString(serverSecretYAML)
		Expect(err).NotTo(HaveOccurred(), "creating server TLS secret")

		clientSecretYAML := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  ca.crt: dummy-ca
`, clientSecret, s.Namespace())
		err = s.CreateResourceFromString(clientSecretYAML)
		Expect(err).NotTo(HaveOccurred(), "creating client CA secret")

		time.Sleep(2 * time.Second)

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(tlsYAML, tlsName, s.Namespace(), s.Namespace(), serverSecret, s.Namespace(), clientSecret, s.Namespace()))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), serverSecret)))
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), clientSecret)))
	})
})
