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

package gatewayapi

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Check if controller cache gets synced with correct resources", Label("networking.k8s.io", "basic"), func() {

	var gatewayProxyYaml = `
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

	var defautlGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
  namespace: %s
spec:
  controllerName: %s
`

	var defautlGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
  namespace: %s
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

	var ResourceApplied = func(s *scaffold.Scaffold, resourType, resourceName, ns, resourceRaw string, observedGeneration int) {
		Expect(s.CreateResourceFromStringWithNamespace(resourceRaw, ns)).
			NotTo(HaveOccurred(), fmt.Sprintf("creating %s", resourType))

		Eventually(func() string {
			hryaml, err := s.GetResourceYamlFromNamespace(resourType, resourceName, ns)
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
		time.Sleep(3 * time.Second)
	}
	var beforeEach = func(s *scaffold.Scaffold, gatewayName string) {
		err := s.CreateResourceFromString(fmt.Sprintf(`
kind: Namespace
apiVersion: v1
metadata:
  name: %s
`, gatewayName))
		Expect(err).NotTo(HaveOccurred(), "creating namespace")
		By(fmt.Sprintf("create GatewayClass for controller %s", s.GetControllerName()))

		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
		err = s.CreateResourceFromStringWithNamespace(gatewayProxy, gatewayName)
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defautlGatewayClass, gatewayClassName, gatewayName, s.GetControllerName()), gatewayName)
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(10 * time.Second)

		By("check GatewayClass condition")
		gcyaml, err := s.GetResourceYamlFromNamespace("GatewayClass", gatewayClassName, gatewayName)
		Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
		Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
		Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"), "checking GatewayClass condition message")

		By("create Gateway")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defautlGateway, gatewayName, gatewayName, gatewayClassName), gatewayName)
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(10 * time.Second)

		By("check Gateway condition")
		gwyaml, err := s.GetResourceYamlFromNamespace("Gateway", gatewayName, gatewayName)
		Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
		Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
		Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"), "checking Gateway condition message")
	}

	Context("Create resource with first controller", func() {
		s1 := scaffold.NewScaffold(&scaffold.Options{
			Name:           "gateway1",
			ControllerName: "apisix.apache.org/apisix-ingress-controller-1",
		})
		var route1 = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
  namespace: gateway1
spec:
  parentRefs:
  - name: gateway1
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    filters:
    - type: RequestMirror
      requestMirror:
        backendRef:
          name: echo-service
          port: 80
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
      weight: 50
    - name: nginx
      port: 80
      weight: 50
 `
		BeforeEach(func() {
			beforeEach(s1, "gateway1")
		})
		It("Apply resource ", func() {
			ResourceApplied(s1, "HTTPRoute", "httpbin", "gateway1", route1, 1)
			routes, err := s1.DefaultDataplaneResource().Route().List(s1.Context)
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(HaveLen(1))
			assert.Equal(GinkgoT(), routes[0].Labels["k8s/controller-name"], "apisix.apache.org/apisix-ingress-controller-1")
		})
	})
	Context("Create resource with second controller", func() {
		s2 := scaffold.NewScaffold(&scaffold.Options{
			Name:           "gateway2",
			ControllerName: "apisix.apache.org/apisix-ingress-controller-2",
		})
		var route2 = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin2
  namespace: gateway2
spec:
  parentRefs:
  - name: gateway2
  hostnames:
  - httpbin.example
  rules:
  - matches:
    - path:
        type: Exact
        value: /get
    filters:
    - type: RequestMirror
      requestMirror:
        backendRef:
          name: echo-service
          port: 80
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
      weight: 50
    - name: nginx
      port: 80
      weight: 50
`
		BeforeEach(func() {
			beforeEach(s2, "gateway2")
		})
		It("Apply resource ", func() {
			ResourceApplied(s2, "HTTPRoute", "httpbin2", "gateway2", route2, 1)
			routes, err := s2.DefaultDataplaneResource().Route().List(s2.Context)
			Expect(err).NotTo(HaveOccurred())
			Expect(routes).To(HaveLen(1))
			assert.Equal(GinkgoT(), routes[0].Labels["k8s/controller-name"], "apisix.apache.org/apisix-ingress-controller-2")
		})
	})
})
