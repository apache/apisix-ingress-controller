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

package apisix

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test CRD Status", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	assertion := func(actualOrCtx any, args ...any) AsyncAssertion {
		return Eventually(actualOrCtx).WithArguments(args...).WithTimeout(30 * time.Second).ProbeEvery(time.Second)
	}

	Context("Test ApisixRoute Sync Status", func() {
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(ingressClassYaml, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})
		const ar = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
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
spec:
  ingressClassName: apisix
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

		getRequest := func(path string) func() int {
			return func() int {
				return s.NewAPISIXClient().GET(path).WithHost("httpbin").Expect().Raw().StatusCode
			}
		}

		It("unknown plugin", func() {
			if os.Getenv("PROVIDER_TYPE") == "apisix-standalone" {
				Skip("apisix standalone does not validate unknown plugins")
			}
			By("apply ApisixRoute with valid plugin")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, arWithInvalidPlugin)

			By("check ApisixRoute status")
			assertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml")
				return output
			}).Should(
				And(
					ContainSubstring(`status: "False"`),
					ContainSubstring(`reason: SyncFailed`),
					ContainSubstring(`unknown plugin [non-existent-plugin]`),
				),
			)

			By("Update ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, ar)

			By("check ApisixRoute status")
			assertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml")
				return output
			}).Should(
				And(
					ContainSubstring(`status: "True"`),
					ContainSubstring(`reason: Accepted`),
				),
			)

			By("check route in APISIX")
			assertion(getRequest("/get")).Should(Equal(200), "should be able to access the route")
		})

		It("dataplane unavailable", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, ar)

			By("check ApisixRoute status")
			assertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml")
				return output
			}).Should(
				And(
					ContainSubstring(`status: "True"`),
					ContainSubstring(`reason: Accepted`),
				),
			)

			By("check route in APISIX")
			assertion(getRequest("/get")).Should(Equal(200), "should be able to access the route")

			s.Deployer.ScaleDataplane(0)

			By("check ApisixRoute status")
			assertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml")
				return output
			}).Should(
				And(
					ContainSubstring(`status: "False"`),
					ContainSubstring(`reason: SyncFailed`),
				),
			)

			s.Deployer.ScaleDataplane(1)

			By("check ApisixRoute status after scaling up")
			assertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml")
				return output
			}).Should(
				And(
					ContainSubstring(`status: "True"`),
					ContainSubstring(`reason: Accepted`),
				),
			)

			By("check route in APISIX")
			assertion(getRequest("/get")).Should(Equal(200), "should be able to access the route")
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
		const gatewayProxy = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
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
			gatewayProxy := fmt.Sprintf(gatewayProxy, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create GatewayClass")
			gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayClass, gatewayClassName, s.GetControllerName()), "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("create Gateway")
			err = s.CreateResourceFromString(fmt.Sprintf(defaultGateway, gatewayClassName))
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
		getRequest := func(path string) func() int {
			return func() int {
				return s.NewAPISIXClient().GET(path).WithHost("httpbin").Expect().Raw().StatusCode
			}
		}
		var resourceApplied = func(resourType, resourceName, resourceRaw string, observedGeneration int) {
			Expect(s.CreateResourceFromString(resourceRaw)).
				NotTo(HaveOccurred(), fmt.Sprintf("creating %s", resourType))

			Eventually(func() string {
				hryaml, err := s.GetResourceYaml(resourType, resourceName)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("getting %s yaml", resourType))
				return hryaml
			}, "8s", "2s").
				Should(
					SatisfyAll(
						ContainSubstring(`status: "True"`),
						ContainSubstring(fmt.Sprintf("observedGeneration: %d", observedGeneration)),
					),
					fmt.Sprintf("checking %s condition status", resourType),
				)
			time.Sleep(5 * time.Second)
		}

		It("dataplane unavailable", func() {
			By("Create HTTPRoute")
			resourceApplied("HTTPRoute", "httpbin", httproute, 1)

			By("check route in APISIX")
			assertion(getRequest("/get")).Should(Equal(200), "should be able to access the route")

			s.Deployer.ScaleDataplane(0)
			time.Sleep(10 * time.Second)

			By("check ApisixRoute status")
			assertion(func() string {
				output, _ := s.GetOutputFromString("httproute", "httpbin", "-o", "yaml")
				return output
			}).Should(
				And(
					ContainSubstring(`status: "False"`),
					ContainSubstring(`reason: SyncFailed`),
				),
			)

			s.Deployer.ScaleDataplane(1)
			time.Sleep(10 * time.Second)

			By("check ApisixRoute status after scaling up")
			assertion(func() string {
				output, _ := s.GetOutputFromString("httproute", "httpbin", "-o", "yaml")
				return output
			}).Should(
				And(
					ContainSubstring(`status: "True"`),
					ContainSubstring(`reason: Accepted`),
				),
			)

			By("check route in APISIX")
			assertion(getRequest("/get")).Should(Equal(200), "should be able to access the route")
		})
	})
})
