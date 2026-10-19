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

var _ = Describe("Test managed resource isolation", Label("networking.k8s.io", "ingress"), func() {
	if framework.ProviderType != framework.ProviderTypeAPISIX {
		return
	}
	s := scaffold.NewDefaultScaffold()
	const gatewayProxyYaml = `
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

	const ingressClassYaml = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: %s
spec:
  controller: "%s"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "%s"
    scope: "Namespace"
`

	const ingressYaml = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin
spec:
  ingressClassName: %s
  rules:
  - host: managed-resources.example
    http:
      paths:
      - path: %s
        pathType: Exact
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`

	It("preserves external routes while reconciling and deleting an IngressClass", func() {
		admin := s.Deployer.AdminAPIClient()
		const externalID = "external-route"
		admin.PUT("/apisix/admin/services/external-service").
			WithHeader("X-API-KEY", s.AdminKey()).
			WithJSON(map[string]any{
				"upstream": map[string]any{"type": "roundrobin", "nodes": map[string]int{"127.0.0.1:1980": 1}},
			}).Expect().Status(http.StatusCreated)
		admin.PUT("/apisix/admin/routes/"+externalID).
			WithHeader("X-API-KEY", s.AdminKey()).
			WithJSON(map[string]any{
				"uri":        "/external",
				"service_id": "external-service",
				"plugins":    map[string]any{"redirect": map[string]any{"uri": "/preserved"}},
			}).Expect().Status(http.StatusCreated)

		proxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
		class := fmt.Sprintf(ingressClassYaml, s.Namespace(), s.GetControllerName(), s.Namespace())
		ingress := fmt.Sprintf(ingressYaml, s.Namespace(), "/get")
		Expect(s.CreateResourceFromString(proxy)).To(Succeed())
		Expect(s.CreateResourceFromStringWithNamespace(class, "")).To(Succeed())
		Expect(s.CreateResourceFromString(ingress)).To(Succeed())
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET", Path: "/get", Host: "managed-resources.example",
			Check: scaffold.WithExpectedStatus(http.StatusOK),
		})
		assertExternal := func() {
			admin.GET("/apisix/admin/routes/"+externalID).
				WithHeader("X-API-KEY", s.AdminKey()).Expect().Status(http.StatusOK).
				JSON().Object().Value("value").Object().Value("uri").IsEqual("/external")
		}
		assertExternal()

		By("update the managed route without deleting the external route")
		Expect(s.CreateResourceFromString(fmt.Sprintf(ingressYaml, s.Namespace(), "/anything"))).To(Succeed())
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET", Path: "/anything", Host: "managed-resources.example",
			Check: scaffold.WithExpectedStatus(http.StatusOK),
		})
		assertExternal()

		By("delete the IngressClass and wait for managed routes to disappear")
		Expect(s.DeleteResourceFromStringWithNamespace(class, "")).To(Succeed())
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET", Path: "/anything", Host: "managed-resources.example",
			Check: scaffold.WithExpectedStatus(http.StatusNotFound),
		})
		assertExternal()
	})
})
