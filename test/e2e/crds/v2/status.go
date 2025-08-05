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

package v2

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test CRD Status", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: fmt.Sprintf("apisix.apache.org/apisix-ingress-controller-%d", time.Now().Unix()),
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	Context("Test ApisixRoute Sync Status", func() {
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := getGatewayProxyYaml(s.Namespace(), s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(getIngressClassYaml(s.Namespace(), s.GetControllerName(), s.Namespace()), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})
		const ar = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
		const arWithInvalidPlugin = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    plugins:
    - name: non-existent-plugin
      enable: true
`
		It("unknown plugin", func() {
			if os.Getenv("PROVIDER_TYPE") == "apisix-standalone" {
				Skip("apisix standalone does not validate unknown plugins")
			}
			By("apply ApisixRoute with valid plugin")
			err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(arWithInvalidPlugin, s.Namespace(), s.Namespace()), s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute with valid plugin")

			By("check ApisixRoute status")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml", "-n", s.Namespace())
				return output
			}).Should(
				And(
					ContainSubstring(`status: "False"`),
					ContainSubstring(`reason: SyncFailed`),
					ContainSubstring(`unknown plugin [non-existent-plugin]`),
				),
			)

			By("Update ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, fmt.Sprintf(ar, s.Namespace(), s.Namespace()))

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(200),
			})
		})

		It("dataplane unavailable", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, fmt.Sprintf(ar, s.Namespace(), s.Namespace()))

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/get",
				Headers: map[string]string{"Host": "httpbin"},
				Check:   scaffold.WithExpectedStatus(200),
			})

			s.Deployer.ScaleDataplane(0)

			By("check ApisixRoute status")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml", "-n", s.Namespace())
				return output
			}).WithTimeout(80 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "False"`),
						ContainSubstring(`reason: SyncFailed`),
					),
				)

			s.Deployer.ScaleDataplane(1)

			By("check ApisixRoute status after scaling up")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml", "-n", s.Namespace())
				return output
			}).WithTimeout(80 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "True"`),
						ContainSubstring(`reason: Accepted`),
					),
				)

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(200),
			})
		})

		It("update the same status only once", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, fmt.Sprintf(ar, s.Namespace(), s.Namespace()))

			output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml", "-n", s.Namespace())

			var route apiv2.ApisixRoute
			err := yaml.Unmarshal([]byte(output), &route)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling ApisixRoute")

			Expect(route.Status.Conditions).Should(HaveLen(1), "should have one condition")

			s.Deployer.ScaleIngress(0)
			s.Deployer.ScaleIngress(1)

			output, _ = s.GetOutputFromString("ar", "default", "-o", "yaml", "-n", s.Namespace())

			var route2 apiv2.ApisixRoute
			err = yaml.Unmarshal([]byte(output), &route2)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling ApisixRoute")

			Expect(route2.Status.Conditions).Should(HaveLen(1), "should have one condition")
			Expect(route2.Status.Conditions[0].LastTransitionTime).To(Equal(route.Status.Conditions[0].LastTransitionTime),
				"should not update the same status condition again")
		})
	})

	Context("Test HTTPRoute Sync Status", func() {
		const httproute = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - "httpbin"
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		const gatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

		const defaultGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
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
      name: apisix-proxy-config
`
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := getGatewayProxyYaml(s.Namespace(), s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create GatewayClass")
			gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClass, gatewayClassName, s.GetControllerName()), "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("create Gateway")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGateway, gatewayClassName), s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(5 * time.Second)

			By("check Gateway condition")
			gwyaml, err := s.GetResourceYaml("Gateway", "apisix")
			Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
			Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
			Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"), "checking Gateway condition message")
		})
		AfterEach(func() {
			_ = s.DeleteResource("Gateway", "apisix")
		})

		It("dataplane unavailable", func() {
			By("Create HTTPRoute")
			err := s.CreateResourceFromStringWithNamespace(httproute, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute")

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(200),
			})

			s.Deployer.ScaleDataplane(0)

			By("check ApisixRoute status")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("httproute", "httpbin", "-o", "yaml")
				return output
			}).WithTimeout(80 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "False"`),
						ContainSubstring(`reason: SyncFailed`),
					),
				)

			s.Deployer.ScaleDataplane(1)

			By("check ApisixRoute status after scaling up")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("httproute", "httpbin", "-o", "yaml")
				return output
			}).WithTimeout(80 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "True"`),
						ContainSubstring(`reason: Accepted`),
					),
				)

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(200),
			})
		})
	})
})
