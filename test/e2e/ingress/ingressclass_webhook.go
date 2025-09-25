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

package ingress

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test IngressClass Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "ingressclass-webhook-test",
		EnableWebhook: true,
	})

	Context("IngressClass Validation", func() {
		It("should warn when referenced GatewayProxy does not exist on create and update", func() {
			By("creating IngressClass referencing a missing GatewayProxy")
			missingName := "missing-proxy"
			icYAML := `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-with-missing
spec:
  controller: "%s"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "%s"
    namespace: "%s"
    scope: "Namespace"
`

			output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(icYAML, s.GetControllerName(), missingName, s.Namespace()))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), missingName)))

			time.Sleep(2 * time.Second)

			By("updating IngressClass to reference another missing GatewayProxy")
			missingName2 := "missing-proxy-2"
			output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(icYAML, s.GetControllerName(), missingName2, s.Namespace()))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), missingName2)))

			By("create GatewayProxy")
			err = s.CreateResourceFromString(s.GetGatewayProxySpec())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("updating IngressClass to reference an existing GatewayProxy")
			existingName := "apisix-proxy-config"
			output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(icYAML, s.GetControllerName(), existingName, s.Namespace()))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced GatewayProxy '%s/%s' not found.", s.Namespace(), existingName)))

			By("deleting IngressClass")
			err = s.DeleteResource("IngressClass", "apisix-with-missing")
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
