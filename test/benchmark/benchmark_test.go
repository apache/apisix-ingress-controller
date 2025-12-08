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

package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/api7/gopkg/pkg/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var report = &BenchmarkReport{}
var totalRoutes = 2000
var totalConsumers = 2000

var _ = BeforeSuite(func() {
	routes := os.Getenv("BENCHMARK_ROUTES")
	if routes != "" {
		_, err := fmt.Sscanf(routes, "%d", &totalRoutes)
		Expect(err).NotTo(HaveOccurred(), "parsing BENCHMARK_ROUTES")
	}
	consumers := os.Getenv("BENCHMARK_CONSUMERS")
	if consumers != "" {
		_, err := fmt.Sscanf(consumers, "%d", &totalConsumers)
		Expect(err).NotTo(HaveOccurred(), "parsing BENCHMARK_CONSUMERS")
	}
})
var _ = AfterSuite(func() {
	report.PrintTable()
})

const gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      service:
        name: %s
        port: 9180
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

var _ = Describe("Benchmark Test", func() {
	var (
		s                = scaffold.NewDefaultScaffold()
		controlAPIClient scaffold.ControlAPIClient
	)

	BeforeEach(func() {
		By("port-forward to control api service")
		var err error
		controlAPIClient, err = s.ControlAPIClient()
		Expect(err).NotTo(HaveOccurred(), "create control api client")
	})

	Context("Benchmark ApisixRoute", func() {
		const ingressClassYaml = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: "%s"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: %s
    scope: "Namespace"
`

		const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /get
      exprs:
      - subject:
          scope: Header
          name: X-Route-Name
        op: Equal
        value: %s
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
		var apisixRouteSpecHeaders = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /headers
      exprs:
      - subject:
          scope: Header
          name: X-Route-Name
        op: Equal
        value: %s
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`

		var apisixUpstreamSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-service-e2e-test
spec:
  ingressClassName: apisix
  scheme: https
`
		var apisixRouteSpecKeyAuth = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: key-auth
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /get
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    authentication:
      enable: true
      type: keyAuth
`
		var keyAuth = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: %s
spec:
  ingressClassName: apisix
  authParameter:
    keyAuth:
      value:
        key: %s
`

		getRouteName := func(i int) string {
			return fmt.Sprintf("test-route-%04d", i)
		}

		createBatchApisixRoutes := func(number int) string {
			var buf bytes.Buffer
			for i := 0; i < number; i++ {
				name := getRouteName(i)
				fmt.Fprintf(&buf, apisixRouteSpec, name, name)
				buf.WriteString("\n---\n")
			}
			return buf.String()
		}
		getConsumerName := func(i int) string {
			return fmt.Sprintf("consumer-%04d", i)
		}
		createBatchConsumers := func(number int) string {
			var buf bytes.Buffer
			for i := 0; i < number; i++ {
				name := getConsumerName(i)
				fmt.Fprintf(&buf, keyAuth, name, name)
				buf.WriteString("\n---\n")
			}
			return buf.String()
		}

		benchmark := func(scenario string) {
			s.Deployer.ScaleIngress(0)
			By(fmt.Sprintf("prepare %d ApisixRoutes", totalRoutes))
			err := s.CreateResourceFromString(createBatchApisixRoutes(totalRoutes))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoutes")
			s.Deployer.ScaleIngress(1)

			now := time.Now()
			By(fmt.Sprintf("start cale time for applying %d ApisixRoutes to take effect", totalRoutes))
			err = s.EnsureNumService(controlAPIClient, func(actual int) bool { return actual == totalRoutes })
			Expect(err).ShouldNot(HaveOccurred())
			costTime := time.Since(now)
			report.Add(scenario, fmt.Sprintf("Apply %d ApisixRoutes", totalRoutes), costTime)

			By("Test the time required for an ApisixRoute update to take effect")
			name := getRouteName(int(time.Now().Unix()))
			err = s.CreateResourceFromString(fmt.Sprintf(apisixRouteSpecHeaders, name, name))
			Expect(err).NotTo(HaveOccurred())
			now = time.Now()
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/headers").WithHeader("X-Route-Name", name).Expect().Raw().StatusCode
			}).WithTimeout(15 * time.Minute).ProbeEvery(100 * time.Millisecond).Should(Equal(http.StatusOK))
			report.Add(scenario, fmt.Sprintf("Update a single ApisixRoute base on %d ApisixRoutes", totalRoutes), time.Since(now))

			By("Test the time required for a service endpoint change to take effect")
			err = s.ScaleHTTPBIN(2)
			Expect(err).NotTo(HaveOccurred(), "scale httpbin deployment")
			now = time.Now()
			err = s.EnsureNumUpstreamNodes(controlAPIClient, "", 2)
			Expect(err).ShouldNot(HaveOccurred())
			costTime = time.Since(now)
			report.Add(scenario, fmt.Sprintf("Service endpoint change base on %d ApisixRoutes", totalRoutes), costTime)

			By("Test the time required for an ApisixUpstream update to take effect")
			err = s.CreateResourceFromString(apisixUpstreamSpec)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixUpstream")
			now = time.Now()
			err = s.ExpectUpstream(controlAPIClient, "", func(upstream adctypes.Upstream) bool {
				if upstream.Scheme != "https" {
					log.Warnf("expect upstream: [%s] scheme to be https, but got [%s]", upstream.Name, upstream.Scheme)
					return false
				}
				return true
			})
			Expect(err).ShouldNot(HaveOccurred())
			costTime = time.Since(now)
			report.Add(scenario, fmt.Sprintf("Update ApisixUpstream base on %d ApisixRoutes", totalRoutes), costTime)
		}

		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.ProviderType, s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(ingressClassYaml, s.GetControllerName(), s.Namespace()), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})
		It("benchmark ApisixRoute", func() {
			benchmark("ApisixRoute Benchmark")
		})
		It("10 apisix-standalone pod scale benchmark", func() {
			if framework.ProviderType != framework.ProviderTypeAPISIXStandalone {
				Skip("only apisix-standalone support scale benchmark")
			}
			s.Deployer.ScaleDataplane(10)
			benchmark("ApisixRoute Benchmark with 10 apisix-standalone pods")
		})
		It("ApisixRoute With Consumers benchmark", func() {
			s.Deployer.ScaleIngress(0)
			By(fmt.Sprintf("prepare %d ApisixConsumers", totalRoutes))
			err := s.CreateResourceFromString(createBatchConsumers(totalRoutes))
			Expect(err).NotTo(HaveOccurred(), "creating ApisixConsumers")
			err = s.CreateResourceFromString(apisixRouteSpecKeyAuth)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute with KeyAuth")
			s.Deployer.ScaleIngress(1)

			now := time.Now()
			Eventually(func() error {
				consumer, err := s.DefaultDataplaneResource().Consumer().List(context.Background())
				if err != nil {
					return err
				}
				if len(consumer) != totalConsumers {
					return fmt.Errorf("expect %d consumers, but got %d", totalConsumers, len(consumer))
				}
				return nil
			}).WithTimeout(15*time.Minute).ProbeEvery(1*time.Second).ShouldNot(HaveOccurred(), "waiting for all consumers to be synced to APISIX")
			costTime := time.Since(now)
			report.AddResult(TestResult{
				Scenario:         "ApisixRoute With Consumers Benchmark",
				CaseName:         fmt.Sprintf("Apply %d ApisixConsumers and ApisixRoute with KeyAuth", totalConsumers),
				CostTime:         costTime,
				IsRequestGateway: true,
			})
		})
	})

	Context("Benchmark HTTPRoute", func() {
		const httpRouteSpec = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
spec:
  parentRefs:
  - name: %s
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
      headers:
        - type: Exact
          name: X-Route-Name
          value: %s
    # name: get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
		const httpRouteSpecHeaders = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
spec:
  parentRefs:
  - name: %s
  rules:
  - matches: 
    - path:
        type: Exact
        value: /headers
      headers:
        - type: Exact
          name: X-Route-Name
          value: %s
    # name: get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

		createBatchHTTPRoutes := func(number int, parentGateway string) string {
			var buf bytes.Buffer
			for i := 0; i < number; i++ {
				name := getRouteName(i)
				fmt.Fprintf(&buf, httpRouteSpec, name, parentGateway, name)
				buf.WriteString("\n---\n")
			}
			return buf.String()
		}

		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.ProviderType, s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create GatewayClass")
			Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred(), "creating GatewayClass")

			By("create Gateway")
			Expect(s.CreateResourceFromString(s.GetGatewayYaml())).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(5 * time.Second)
		})

		It("benchmark HTTPRoute", func() {
			s.Deployer.ScaleIngress(0)
			By(fmt.Sprintf("prepare %d HTTPRoute", totalRoutes))
			err := s.CreateResourceFromString(createBatchHTTPRoutes(totalRoutes, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute")
			s.Deployer.ScaleIngress(1)

			now := time.Now()
			By(fmt.Sprintf("start cale time for applying %d HTTPRoute to take effect", totalRoutes))
			err = s.EnsureNumService(controlAPIClient, func(actual int) bool { return actual == totalRoutes })
			Expect(err).ShouldNot(HaveOccurred())
			costTime := time.Since(now)
			report.Add("HTTPRoute Benchmark", fmt.Sprintf("Apply %d HTTPRoute", totalRoutes), costTime)

			By("Test the time required for an HTTPRoute update to take effect")
			name := getRouteName(int(time.Now().Unix()))
			err = s.CreateResourceFromString(fmt.Sprintf(httpRouteSpecHeaders, name, s.Namespace(), name))
			Expect(err).NotTo(HaveOccurred())
			now = time.Now()
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/headers").WithHeader("X-Route-Name", name).Expect().Raw().StatusCode
			}).WithTimeout(5 * time.Minute).ProbeEvery(100 * time.Millisecond).Should(Equal(http.StatusOK))
			report.AddResult(TestResult{
				Scenario: "HTTPRoute Benchmark",
				CaseName: fmt.Sprintf("Update a single HTTPRoute base on %d HTTPRoute", totalRoutes),
				CostTime: time.Since(now),
			})

			By("Test the time required for a service endpoint change to take effect")
			err = s.ScaleHTTPBIN(2)
			Expect(err).NotTo(HaveOccurred(), "scale httpbin deployment")
			now = time.Now()
			err = s.EnsureNumUpstreamNodes(controlAPIClient, "", 2)
			Expect(err).ShouldNot(HaveOccurred())
			costTime = time.Since(now)
			report.Add("HTTPRoute Benchmark", fmt.Sprintf("Service endpoint change base on %d HTTPRoute", totalRoutes), costTime)
		})
	})
})

func getRouteName(i int) string {
	return fmt.Sprintf("test-route-%04d", i)
}
