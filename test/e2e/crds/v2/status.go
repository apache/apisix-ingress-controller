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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test CRD Status", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	var (
		s       = scaffold.NewDefaultScaffold()
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	Context("Test ApisixRoute Sync Status", func() {
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := s.GetGatewayProxyWithServiceYaml()
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
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
		const arWithInvalidIngressClass = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: ar-with-invalid-ingressclass
spec:
  ingressClassName: ar-with-invalid-ingressclass
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
		It("unknown plugin", func() {
			if os.Getenv("PROVIDER_TYPE") == "apisix-standalone" {
				Skip("apisix standalone does not validate unknown plugins")
			}
			By("apply ApisixRoute with valid plugin")
			err := s.CreateResourceFromString(fmt.Sprintf(arWithInvalidPlugin, s.Namespace(), s.Namespace()))
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
			arYaml := fmt.Sprintf(ar, s.Namespace(), s.Namespace())
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, arYaml)

			By("check route in APISIX")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/get",
				Headers: map[string]string{"Host": "httpbin"},
				Check:   scaffold.WithExpectedStatus(200),
			})

			By("get yaml from service")
			serviceYaml, err := s.GetOutputFromString("svc", framework.ProviderType, "-o", "yaml")
			Expect(err).NotTo(HaveOccurred(), "getting service yaml")
			By("update service to type ExternalName with invalid host")
			var k8sservice corev1.Service
			err = yaml.Unmarshal([]byte(serviceYaml), &k8sservice)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling service")
			oldSpec := k8sservice.Spec
			k8sservice.Spec = corev1.ServiceSpec{
				Type:         corev1.ServiceTypeExternalName,
				ExternalName: "invalid.host",
			}
			newServiceYaml, err := yaml.Marshal(k8sservice)
			Expect(err).NotTo(HaveOccurred(), "marshalling service")
			err = s.CreateResourceFromString(string(newServiceYaml))
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("check ApisixRoute status")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml")
				return output
			}).WithTimeout(60 * time.Second).
				Should(
					And(
						ContainSubstring(`status: "False"`),
						ContainSubstring(`reason: SyncFailed`),
					),
				)

			By("update service to original spec")
			serviceYaml, err = s.GetOutputFromString("svc", framework.ProviderType, "-o", "yaml")
			Expect(err).NotTo(HaveOccurred(), "getting service yaml")
			err = yaml.Unmarshal([]byte(serviceYaml), &k8sservice)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling service")
			k8sservice.Spec = oldSpec
			newServiceYaml, err = yaml.Marshal(k8sservice)
			Expect(err).NotTo(HaveOccurred(), "marshalling service")
			err = s.CreateResourceFromString(string(newServiceYaml))
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("check ApisixRoute status after scaling up")
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("ar", "default", "-o", "yaml")
				return output
			}).WithTimeout(60 * time.Second).
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

		It("IngressClass not found", func() {
			err := s.CreateResourceFromString(arWithInvalidIngressClass)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute with invalid IngressClass")

			for range 10 {
				output, err := s.GetOutputFromString("ar", "ar-with-invalid-ingressclass", "-o", "yaml")
				Expect(err).NotTo(HaveOccurred(), "getting ApisixRoute output")
				Expect(output).ShouldNot(
					Or(
						ContainSubstring(`status: "True"`),
						ContainSubstring(`status: "False"`),
					),
				)
				time.Sleep(1 * time.Second)
			}
		})
	})
})
