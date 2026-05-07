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

var _ = Describe("Test Consumer Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "consumer-webhook-test",
		EnableWebhook: true,
	})

	BeforeEach(func() {
		By("creating GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("creating GatewayClass")
		err = s.CreateResourceFromString(s.GetGatewayClassYaml())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(2 * time.Second)

		By("creating Gateway")
		err = s.CreateResourceFromString(s.GetGatewayYaml())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(5 * time.Second)
	})

	It("should warn on missing secret references", func() {
		missingSecret := "missing-consumer-secret"
		consumerName := "webhook-consumer"
		gatewayName := s.Namespace()
		consumerYAML := `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: %s
spec:
  gatewayRef:
    name: %s
  credentials:
  - type: jwt-auth
    secretRef:
      name: %s
`

		output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(consumerYAML, consumerName, gatewayName, missingSecret))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), missingSecret)))

		By("creating referenced secret")
		secretYAML := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
stringData:
  token: %s
`, missingSecret, s.AdminKey())
		err = s.CreateResourceFromString(secretYAML)
		Expect(err).NotTo(HaveOccurred(), "creating consumer secret")

		time.Sleep(2 * time.Second)

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(consumerYAML, consumerName, gatewayName, missingSecret))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), missingSecret)))
	})

	It("should reject invalid plugin config during ADC validation", func() {
		gatewayName := s.Namespace()

		firstConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: webhook-consumer-a
spec:
  gatewayRef:
    name: %s
  credentials:
  - type: key-auth
    name: key-auth-a
    config:
      key: consumer-a-key
`, gatewayName)

		By("creating the first Consumer with valid key-auth config")
		err := s.CreateResourceFromString(firstConsumer)
		Expect(err).NotTo(HaveOccurred(), "creating first Consumer")

		invalidConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: webhook-consumer-b
spec:
  gatewayRef:
    name: %s
  plugins:
  - name: jwt-auth
    config:
      key: consumer-b-key
      algorithm: INVALID_ALGO
`, gatewayName)

		By("creating Consumer with an invalid jwt-auth algorithm in plugins")
		err = s.CreateResourceFromString(invalidConsumer)
		expectAdmissionDenied(s, "consumer", "webhook-consumer-b", err)

		correctedConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: webhook-consumer-b
spec:
  gatewayRef:
    name: %s
  plugins:
  - name: jwt-auth
    config:
      key: consumer-b-key
      algorithm: HS256
      secret: consumer-b-secret
`, gatewayName)

		By("creating corrected Consumer with a valid algorithm")
		err = s.CreateResourceFromString(correctedConsumer)
		Expect(err).NotTo(HaveOccurred(), "creating corrected Consumer")
	})

	It("should reject consumer update that fails ADC validation", func() {
		gatewayName := s.Namespace()
		consumerName := "webhook-consumer-update"

		validConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: %s
spec:
  gatewayRef:
    name: %s
  credentials:
  - type: key-auth
    name: key-auth-update
    config:
      key: update-consumer-key
`, consumerName, gatewayName)

		By("creating valid Consumer")
		err := s.CreateResourceFromString(validConsumer)
		Expect(err).NotTo(HaveOccurred(), "creating initial valid Consumer")

		invalidConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: %s
spec:
  gatewayRef:
    name: %s
  plugins:
  - name: jwt-auth
    config:
      key: update-consumer-jwt-key
      algorithm: INVALID_ALGO
`, consumerName, gatewayName)

		By("updating Consumer with an invalid jwt-auth algorithm in plugins")
		err = s.CreateResourceFromString(invalidConsumer)
		expectUpdateDenied(err)

		correctedConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: %s
spec:
  gatewayRef:
    name: %s
  plugins:
  - name: jwt-auth
    config:
      key: update-consumer-jwt-key
      algorithm: HS256
      secret: update-consumer-secret
`, consumerName, gatewayName)

		By("updating Consumer with a valid algorithm")
		err = s.CreateResourceFromString(correctedConsumer)
		Expect(err).NotTo(HaveOccurred(), "updating Consumer with corrected config")
	})
})
