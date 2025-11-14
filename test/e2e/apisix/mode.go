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
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Multi-Mode Deployment", Label("networking.k8s.io", "ingress"), func() {
	s := scaffold.NewDefaultScaffold()

	Context("apisix and apisix-standalone", func() {
		var ns1 string
		var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      mode: %s
      service:
        name: %s
        port: 9180
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

		const ingressClassYaml = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: %s
spec:
  controller: %s
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: %s
    scope: Namespace
`
		var ingressHttpbin = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin
spec:
  ingressClassName: %s
  rules:
  - host: httpbin.example
    http:
      paths:
      - path: /get
        pathType: Exact
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`
		var ingressHttpbin2 = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin2
spec:
  ingressClassName: %s
  rules:
  - host: httpbin2.example
    http:
      paths:
      - path: /get
        pathType: Exact
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`

		It("apisix and apisix-standalone", func() {
			gateway1, svc1, err := s.Deployer.CreateAdditionalGatewayWithOptions("multi-mode-v1", scaffold.DeployDataplaneOptions{
				ProviderType: framework.ProviderTypeAPISIX,
			})
			Expect(err).NotTo(HaveOccurred(), "creating Additional Gateway")

			resources1, exists := s.GetAdditionalGateway(gateway1)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")
			ns1 = resources1.Namespace

			By("create GatewayProxy for Additional Gateway")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayProxyYaml, framework.ProviderTypeAPISIX, svc1.Name, resources1.AdminAPIKey), resources1.Namespace)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy for Additional Gateway")

			By("create IngressClass for Additional Gateway")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(ingressClassYaml, ns1, s.GetControllerName(), resources1.Namespace), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass for Additional Gateway")

			gateway2, svc2, err := s.Deployer.CreateAdditionalGatewayWithOptions("multi-mode-v2", scaffold.DeployDataplaneOptions{
				ProviderType: framework.ProviderTypeAPISIXStandalone,
			})
			Expect(err).NotTo(HaveOccurred(), "creating Additional Gateway")

			resources2, exists := s.GetAdditionalGateway(gateway2)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")
			ns2 := resources2.Namespace

			By("create GatewayProxy for Additional Gateway")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayProxyYaml, framework.ProviderTypeAPISIXStandalone, svc2.Name, resources2.AdminAPIKey), resources2.Namespace)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy for Additional Gateway")

			By("create IngressClass for Additional Gateway")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(ingressClassYaml, ns2, s.GetControllerName(), resources2.Namespace), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass for Additional Gateway")

			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressHttpbin, ns1))).ShouldNot(HaveOccurred(), "creating Ingress in ns1")
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressHttpbin2, ns2))).ShouldNot(HaveOccurred(), "creating Ingress in ns2")

			client1, err := s.NewAPISIXClientForGateway(gateway1)
			Expect(err).NotTo(HaveOccurred(), "creating APISIX client for gateway1")

			client2, err := s.NewAPISIXClientForGateway(gateway2)
			Expect(err).NotTo(HaveOccurred(), "creating APISIX client for gateway2")

			s.RequestAssert(&scaffold.RequestAssert{
				Client: client1,
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Client: client2,
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Client: client1,
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin2.example",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Client: client2,
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin2.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
		})
	})
})
