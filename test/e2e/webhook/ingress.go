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

package webhook

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Ingress Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "webhook-test",
		EnableWebhook: true,
	})

	BeforeEach(func() {
		By("create GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	Context("Ingress Validation", func() {
		It("should warn about unsupported annotations on create", func() {

			By("creating Ingress with unsupported annotations")
			ingressYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-webhook-unsupported
  namespace: %s
  annotations:
    k8s.apisix.apache.org/use-regex: "true"
    k8s.apisix.apache.org/enable-websocket: "true"
spec:
  ingressClassName: %s
  rules:
  - host: webhook-test.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`, s.Namespace(), s.Namespace())

			output, err := s.CreateResourceFromStringAndGetOutput(ingressYAML)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).To(ContainSubstring(`Warning: Annotation 'k8s.apisix.apache.org/enable-websocket' is not supported`))
			Expect(output).To(ContainSubstring(`Warning: Annotation 'k8s.apisix.apache.org/use-regex' is not supported`))

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "webhook-test.example.com",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
		})

		It("should warn about unsupported annotations on update", func() {
			By("creating Ingress without unsupported annotations")
			initialIngressYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-webhook-update
  namespace: %s
spec:
  ingressClassName: %s
  rules:
  - host: webhook-test-update.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`, s.Namespace(), s.Namespace())

			output, err := s.CreateResourceFromStringAndGetOutput(initialIngressYAML)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).ShouldNot(ContainSubstring(`Warning`))

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "webhook-test-update.example.com",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})

			By("updating Ingress with unsupported annotations")
			updatedIngressYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-webhook-update
  namespace: %s
  annotations:
    k8s.apisix.apache.org/enable-cors: "true"
spec:
  ingressClassName: %s
  rules:
  - host: webhook-test-update.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`, s.Namespace(), s.Namespace())

			output, err = s.CreateResourceFromStringAndGetOutput(updatedIngressYAML)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output).To(ContainSubstring(`Warning: Annotation 'k8s.apisix.apache.org/enable-cors' is not supported`))

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "webhook-test-update.example.com",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
		})
	})
})
