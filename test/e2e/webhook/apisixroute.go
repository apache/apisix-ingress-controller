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

var _ = Describe("Test ApisixRoute Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "apisixroute-webhook-test",
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

	It("should warn on missing service references", func() { //nolint:dupl
		missingService := "missing-backend"
		routeName := "webhook-apisixroute"
		routeYAML := `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule-webhook
    match:
      hosts:
      - webhook.example.com
      paths:
      - /webhook
    backends:
    - serviceName: %s
      servicePort: 80
`

		output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(routeYAML, routeName, s.Namespace(), s.Namespace(), missingService))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", s.Namespace(), missingService)))

		By("creating referenced Service")
		serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: placeholder
  ports:
  - name: http
    port: 80
    targetPort: 80
  type: ClusterIP
`, missingService)
		err = s.CreateResourceFromString(serviceYAML)
		Expect(err).NotTo(HaveOccurred(), "creating backend service placeholder")

		time.Sleep(2 * time.Second)

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(routeYAML, routeName, s.Namespace(), s.Namespace(), missingService))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", s.Namespace(), missingService)))
	})

	It("should reject routes that fail ADC validation", func() {
		backendService := "webhook-route-backend"
		routeName := "webhook-apisixroute-invalid"

		By("creating referenced Service")
		serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: placeholder
  ports:
  - name: http
    port: 80
    targetPort: 80
  type: ClusterIP
`, backendService)
		err := s.CreateResourceFromString(serviceYAML)
		Expect(err).NotTo(HaveOccurred(), "creating backend service")

		invalidRouteYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule-invalid
    match:
      hosts:
      - webhook.example.com
      paths:
      - /invalid
    backends:
    - serviceName: %s
      servicePort: 80
      resolveGranularity: service
    plugins:
    - name: response-rewrite
      enable: true
      config:
        status_code: "500"
`, routeName, s.Namespace(), s.Namespace(), backendService)

		By("creating ApisixRoute with invalid plugin config")
		err = s.CreateResourceFromString(invalidRouteYAML)
		expectAdmissionDenied(s, "apisixroute", routeName, err)

		validRouteYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule-valid
    match:
      hosts:
      - webhook.example.com
      paths:
      - /valid
    backends:
    - serviceName: %s
      servicePort: 80
      resolveGranularity: service
`, routeName, s.Namespace(), s.Namespace(), backendService)

		By("creating corrected ApisixRoute")
		err = s.CreateResourceFromString(validRouteYAML)
		Expect(err).NotTo(HaveOccurred(), "creating corrected ApisixRoute")
	})

	It("should reject route update that fails ADC validation", func() {
		backendService := "webhook-route-update-backend"
		routeName := "webhook-apisixroute-update"

		By("creating referenced Service")
		serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: placeholder
  ports:
  - name: http
    port: 80
    targetPort: 80
  type: ClusterIP
`, backendService)
		err := s.CreateResourceFromString(serviceYAML)
		Expect(err).NotTo(HaveOccurred(), "creating backend service")

		validRouteYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule-update
    match:
      hosts:
      - webhook-update.example.com
      paths:
      - /update
    backends:
    - serviceName: %s
      servicePort: 80
      resolveGranularity: service
`, routeName, s.Namespace(), s.Namespace(), backendService)

		By("creating valid ApisixRoute")
		err = s.CreateResourceFromString(validRouteYAML)
		Expect(err).NotTo(HaveOccurred(), "creating initial valid ApisixRoute")

		invalidRouteYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule-update
    match:
      hosts:
      - webhook-update.example.com
      paths:
      - /update
    backends:
    - serviceName: %s
      servicePort: 80
      resolveGranularity: service
    plugins:
    - name: response-rewrite
      enable: true
      config:
        status_code: "500"
`, routeName, s.Namespace(), s.Namespace(), backendService)

		By("updating ApisixRoute with invalid plugin config")
		err = s.CreateResourceFromString(invalidRouteYAML)
		expectUpdateDenied(err)

		By("updating ApisixRoute with corrected config")
		err = s.CreateResourceFromString(validRouteYAML)
		Expect(err).NotTo(HaveOccurred(), "updating ApisixRoute with corrected config")
	})
})
