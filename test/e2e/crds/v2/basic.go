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
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("APISIX Standalone Basic Tests", Label("apisix.apache.org", "v2", "basic"), func() {
	var (
		s       = scaffold.NewDefaultScaffold()
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	Context("APISIX HTTP Proxy", func() {
		It("should handle basic HTTP requests", func() {
			httpClient := s.NewAPISIXClient()
			Expect(httpClient).NotTo(BeNil())

			// Test basic connectivity
			httpClient.GET("/anything").
				Expect().
				Status(404).Body().Contains("404 Route Not Found")
		})

		It("should handle basic HTTP requests with additional gateway", func() {
			additionalGatewayID, _, err := s.Deployer.CreateAdditionalGateway("additional-gw")
			Expect(err).NotTo(HaveOccurred())

			httpClient, err := s.NewAPISIXClientForGateway(additionalGatewayID)
			Expect(err).NotTo(HaveOccurred())
			Expect(httpClient).NotTo(BeNil())

			httpClient.GET("/anything").
				Expect().
				Status(404).Body().Contains("404 Route Not Found")
		})

	})

	Context("IngressClass Annotations", func() {
		It("Basic tests", func() {
			const ingressClassYaml = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: %s
  annotations:
    apisix.apache.org/parameters-namespace: %s
spec:
  controller: %s
  parameters:
    apiGroup: apisix.apache.org
    kind: GatewayProxy
    name: apisix-proxy-config
`

			By("create GatewayProxy")

			err := s.CreateResourceFromString(s.GetGatewayProxySpec())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			ingressClass := fmt.Sprintf(ingressClassYaml, s.Namespace(), s.Namespace(), s.GetControllerName())
			err = s.CreateResourceFromString(ingressClass)
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - %s
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			request := func(path string) int {
				return s.NewAPISIXClient().GET(path).WithHost("httpbin").Expect().Raw().StatusCode
			}

			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute,
				fmt.Sprintf(apisixRouteSpec, s.Namespace(), "/get"))

			By("verify ApisixRoute works")
			Eventually(request).WithArguments("/get").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("update ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute,
				fmt.Sprintf(apisixRouteSpec, s.Namespace(), "/headers"))
			Eventually(request).WithArguments("/get").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/headers",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})

			By("delete ApisixRoute")
			err = s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			Eventually(request).WithArguments("/headers").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})
	})
})
