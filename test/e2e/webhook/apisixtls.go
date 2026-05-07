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

		By("creating referenced TLS secrets with valid certificate material")
		serverCert, serverKey := s.GenerateCert(GinkgoT(), []string{"webhook.example.com"})
		err = s.NewKubeTlsSecret(serverSecret, serverCert.String(), serverKey.String())
		Expect(err).NotTo(HaveOccurred(), "creating server TLS secret")

		caCert, _, _, _, _ := s.GenerateMACert(GinkgoT(), []string{"webhook.example.com"})
		err = s.NewClientCASecret(clientSecret, caCert.String(), "")
		Expect(err).NotTo(HaveOccurred(), "creating client CA secret")

		time.Sleep(2 * time.Second)

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(tlsYAML, tlsName, s.Namespace(), s.Namespace(), serverSecret, s.Namespace(), clientSecret, s.Namespace()))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), serverSecret)))
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), clientSecret)))
	})

	It("should reject invalid TLS material during ADC validation", func() {
		serverSecret := "invalid-server-tls"
		tlsName := "webhook-apisixtls-invalid"
		host := "invalid-webhook.example.com"

		By("creating a referenced TLS secret with invalid certificate data")
		invalidServerSecretYAML := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: kubernetes.io/tls
stringData:
  tls.crt: not-a-cert
  tls.key: not-a-key
`, serverSecret, s.Namespace())
		err := s.CreateResourceFromString(invalidServerSecretYAML)
		Expect(err).NotTo(HaveOccurred(), "creating invalid server TLS secret")

		tlsYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, tlsName, s.Namespace(), s.Namespace(), host, serverSecret, s.Namespace())

		By("creating ApisixTls backed by invalid certificate material")
		err = s.CreateResourceFromString(tlsYAML)
		expectAdmissionDenied(s, "apisixtls", tlsName, err)

		By("replacing the secret with valid certificate material")
		err = s.DeleteResource("Secret", serverSecret)
		Expect(err).NotTo(HaveOccurred(), "deleting invalid server TLS secret")

		serverCert, serverKey := s.GenerateCert(GinkgoT(), []string{host})
		err = s.NewKubeTlsSecret(serverSecret, serverCert.String(), serverKey.String())
		Expect(err).NotTo(HaveOccurred(), "creating valid server TLS secret")

		// Wait for the webhook cache to reflect the recreated Secret before submitting ApisixTls.
		time.Sleep(2 * time.Second)

		By("creating corrected ApisixTls")
		err = s.CreateResourceFromString(tlsYAML)
		Expect(err).NotTo(HaveOccurred(), "creating corrected ApisixTls")
	})

	It("should reject TLS update with invalid certificate material", func() {
		validSecret := "update-valid-tls"
		invalidSecret := "update-invalid-tls"
		tlsName := "webhook-apisixtls-update"
		host := "update-webhook.example.com"

		By("creating a valid TLS secret")
		serverCert, serverKey := s.GenerateCert(GinkgoT(), []string{host})
		err := s.NewKubeTlsSecret(validSecret, serverCert.String(), serverKey.String())
		Expect(err).NotTo(HaveOccurred(), "creating valid server TLS secret")

		By("creating an invalid TLS secret with bad certificate material")
		invalidSecretYAML := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: kubernetes.io/tls
stringData:
  tls.crt: not-a-cert
  tls.key: not-a-key
`, invalidSecret, s.Namespace())
		err = s.CreateResourceFromString(invalidSecretYAML)
		Expect(err).NotTo(HaveOccurred(), "creating invalid server TLS secret")

		validTLSYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, tlsName, s.Namespace(), s.Namespace(), host, validSecret, s.Namespace())

		By("creating valid ApisixTls")
		err = s.CreateResourceFromString(validTLSYAML)
		Expect(err).NotTo(HaveOccurred(), "creating initial valid ApisixTls")

		invalidTLSYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, tlsName, s.Namespace(), s.Namespace(), host, invalidSecret, s.Namespace())

		By("updating ApisixTls to reference the invalid certificate secret")
		err = s.CreateResourceFromString(invalidTLSYAML)
		expectUpdateDenied(err)

		By("updating ApisixTls back to the valid certificate secret")
		err = s.CreateResourceFromString(validTLSYAML)
		Expect(err).NotTo(HaveOccurred(), "updating ApisixTls with valid certificate")
	})
})
