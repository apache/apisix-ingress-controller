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

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ApisixConsumer Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "apisixconsumer-webhook-test",
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

	It("should reject missing authentication secrets", func() {
		missingSecret := "missing-basic-secret"
		consumerName := "webhook-apisixconsumer"
		consumerYAML := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  authParameter:
    basicAuth:
      secretRef:
        name: %s
`

		output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(consumerYAML, consumerName, s.Namespace(), s.Namespace(), missingSecret))
		expectAdmissionDenied(s, "apisixconsumer", consumerName, err, fmt.Sprintf("%s/%s", s.Namespace(), missingSecret))
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), missingSecret)))

		By("creating referenced secret")
		secretYAML := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
stringData:
  username: demo
  password: demo
`, missingSecret)
		err = s.CreateResourceFromString(secretYAML)
		Expect(err).NotTo(HaveOccurred(), "creating basic auth secret")

		time.Sleep(2 * time.Second)

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(consumerYAML, consumerName, s.Namespace(), s.Namespace(), missingSecret))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), missingSecret)))
	})

	It("should reject invalid plugin config during ADC validation", func() {
		if framework.ProviderType != framework.ProviderTypeAPISIXStandalone {
			Skip("ADC validation requires apisix-standalone backend")
		}

		firstConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: webhook-apisixconsumer-a
  namespace: %s
spec:
  ingressClassName: %s
  authParameter:
    jwtAuth:
      value:
        key: consumer-a-key
        algorithm: HS256
`, s.Namespace(), s.Namespace())

		By("creating the first ApisixConsumer with valid jwt-auth config")
		err := s.CreateResourceFromString(firstConsumer)
		Expect(err).NotTo(HaveOccurred(), "creating first ApisixConsumer")

		invalidConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: webhook-apisixconsumer-b
  namespace: %s
spec:
  ingressClassName: %s
  authParameter:
    jwtAuth:
      value:
        key: consumer-b-key
        algorithm: INVALID_ALGO
`, s.Namespace(), s.Namespace())

		By("creating ApisixConsumer with an invalid jwt-auth algorithm")
		err = s.CreateResourceFromString(invalidConsumer)
		expectAdmissionDenied(s, "apisixconsumer", "webhook-apisixconsumer-b", err)

		correctedConsumer := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: webhook-apisixconsumer-b
  namespace: %s
spec:
  ingressClassName: %s
  authParameter:
    jwtAuth:
      value:
        key: consumer-b-key
        algorithm: HS256
`, s.Namespace(), s.Namespace())

		By("creating corrected ApisixConsumer with a valid algorithm")
		err = s.CreateResourceFromString(correctedConsumer)
		Expect(err).NotTo(HaveOccurred(), "creating corrected ApisixConsumer")
	})
})
