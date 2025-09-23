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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ApisixUpstream", Label("apisix.apache.org", "v2", "apisixupstream"), func() {
	var (
		s       = scaffold.NewDefaultScaffold()
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)
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

	Context("Health Check", func() {
		It("active and passive", func() {
			auWithHealthcheck := `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: active
spec:
  ingressClassName: %s
  externalNodes:
  - type: Domain
    name: httpbin-service-e2e-test
  - type: Domain
    name: invalid.httpbin.host
  - type: Domain
    name: invalid1.httpbin.host
  retries: 1
  healthCheck:
    active:
      type: http
      httpPath: /ip
      healthy:
        httpCodes: [200]
        interval: 1s
      unhealthy:
        httpFailures: 2
        interval: 1s
    passive:
      healthy:
        httpCodes: [200]
      unhealthy:
        httpCodes: [502]
`
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "active"},
				&apiv2.ApisixUpstream{}, fmt.Sprintf(auWithHealthcheck, s.Namespace()))

			ar := `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
  ingressClassName: %s
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /*
    upstreams:
    - name: active
`
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-route"},
				&apiv2.ApisixRoute{}, fmt.Sprintf(ar, s.Namespace()))

			By("triggering the health check")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/ip",
				Host:   "httpbin.org",
			})
			time.Sleep(2 * time.Second)

			ups, err := s.Deployer.DefaultDataplaneResource().Upstream().List(context.Background())
			Expect(err).ToNot(HaveOccurred(), "listing upstreams")
			Expect(ups).To(HaveLen(1), "the number of upstreams")
			Expect(ups[0].Nodes).To(HaveLen(3), "the number of upstream nodes")
			Expect(ups[0].Checks).ToNot(BeNil(), "the healthcheck configuration")
			Expect(ups[0].Checks.Active).ToNot(BeNil(), "the active healthcheck configuration")
			Expect(ups[0].Checks.Active.Healthy).ToNot(BeNil(), "the active healthy configuration")
			Expect(ups[0].Checks.Active.Unhealthy).ToNot(BeNil(), "the active unhealthy configuration")
			Expect(ups[0].Checks.Active.Healthy.Interval).To(Equal(1), "the healthy interval")
			Expect(ups[0].Checks.Active.Healthy.HTTPStatuses).To(Equal([]int{200}), "the healthy http status")
			Expect(ups[0].Checks.Active.Unhealthy.Interval).To(Equal(1), "the unhealthy interval")
			Expect(ups[0].Checks.Active.Unhealthy.HTTPFailures).To(Equal(2), "the unhealthy http failures")
			Expect(ups[0].Checks.Passive).ToNot(BeNil(), "the passive healthcheck configuration")
			Expect(ups[0].Checks.Passive.Healthy).ToNot(BeNil(), "the passive healthy configuration")
			Expect(ups[0].Checks.Passive.Unhealthy).ToNot(BeNil(), "the passive unhealthy configuration")
			Expect(ups[0].Checks.Passive.Healthy.HTTPStatuses).To(Equal([]int{200}), "the passive healthy http status")
			Expect(ups[0].Checks.Passive.Unhealthy.HTTPStatuses).To(Equal([]int{502}), "the passive unhealthy http status")

			for range 100 {
				s.NewAPISIXClient().GET("/ip").WithHost("httpbin.org").Expect().Status(200)
			}
		})
	})
})
