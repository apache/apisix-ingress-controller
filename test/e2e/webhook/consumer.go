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
})
