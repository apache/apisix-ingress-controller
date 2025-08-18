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
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ApisixRoute", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: fmt.Sprintf("apisix.apache.org/apisix-ingress-controller-%d", time.Now().Unix()),
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	BeforeEach(func() {
		By("create GatewayProxy")
		gatewayProxy := s.GetGatewayProxyYaml()
		err := s.CreateResourceFromString(gatewayProxy)
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	Context("Test ApisixRoute", func() {

		It("Basic tests", func() {
			const apisixRouteSpec = `
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
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace(), "/get"))

			By("verify ApisixRoute works")
			Eventually(request).WithArguments("/get").WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("update ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace(), "/headers"))
			Eventually(request).WithArguments("/get").WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
			s.NewAPISIXClient().GET("/headers").WithHost("httpbin").Expect().Status(http.StatusOK)

			By("delete ApisixRoute")
			err := s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			Eventually(request).WithArguments("/headers").WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			By("request /metrics endpoint from controller")

			// Get the metrics service endpoint
			metricsURL := s.GetMetricsEndpoint()

			By("verify metrics content")
			resp, err := http.Get(metricsURL)
			Expect(err).ShouldNot(HaveOccurred(), "request metrics endpoint")
			defer func() {
				_ = resp.Body.Close()
			}()

			Expect(resp.StatusCode).Should(Equal(http.StatusOK))

			body, err := io.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred(), "read metrics response")

			bodyStr := string(body)

			// Verify prometheus format
			Expect(resp.Header.Get("Content-Type")).Should(ContainSubstring("text/plain; version=0.0.4; charset=utf-8"))

			// Verify specific metrics from metrics.go exist
			Expect(bodyStr).Should(ContainSubstring("apisix_ingress_adc_sync_duration_seconds"))
			Expect(bodyStr).Should(ContainSubstring("apisix_ingress_adc_sync_total"))
			Expect(bodyStr).Should(ContainSubstring("apisix_ingress_status_update_queue_length"))
			Expect(bodyStr).Should(ContainSubstring("apisix_ingress_file_io_duration_seconds"))

			// Log metrics for debugging
			fmt.Printf("Metrics endpoint response:\n%s\n", bodyStr)
		})

		It("Test plugins in ApisixRoute", func() {
			const apisixRouteSpecPart0 = `
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
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			const apisixRouteSpecPart1 = ` 
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          X-Global-Rule: "test-response-rewrite"
          X-Global-Test: "enabled"
`
			By("apply ApisixRoute without plugins")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpecPart0, s.Namespace(), s.Namespace()))

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("apply ApisixRoute with plugins")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpecPart0, s.Namespace(), s.Namespace())+apisixRouteSpecPart1)
			time.Sleep(5 * time.Second)

			By("verify plugin works")
			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEqual("test-response-rewrite")
			resp.Header("X-Global-Test").IsEqual("enabled")

			By("remove plugin")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpecPart0, s.Namespace(), s.Namespace()))
			time.Sleep(5 * time.Second)

			By("verify no plugin works")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})

			resp = s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEmpty()
			resp.Header("X-Global-Test").IsEmpty()
		})

		It("Test ApisixRoute match by vars", func() {
			const apisixRouteSpec = `
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
      paths:
      - /*
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: bar
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").
					WithHeader("X-Foo", "bar").
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusNotFound)
		})

		It("Test ApisixRoute filterFunc", func() {
			const apisixRouteSpec = `
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
      paths:
      - /*
      filter_func: |
        function(vars)
          local core = require ('apisix.core')
          local body, err = core.request.get_body()
          if not body then
              return false
          end
          local data, err = core.json.decode(body)
          if not data then
              return false
          end
          if data['foo'] == 'bar' then
              return true
          end
          return false
        end
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").
					WithJSON(map[string]string{"foo": "bar"}).
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusNotFound)
		})

		It("Test ApisixRoute service not found", func() {
			const apisixRouteSpec = `
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
      - %s
    backends:
    - serviceName: service-not-found
      servicePort: 80
`
			request := func(path string) int {
				return s.NewAPISIXClient().GET(path).WithHost("httpbin").Expect().Raw().StatusCode
			}

			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace(), "/get"))

			Eventually(request).WithArguments("/get").WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusServiceUnavailable))
		})

		It("Test ApisixRoute resolveGranularity", func() {
			const apisixRouteSpec = `
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
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
      resolveGranularity: service
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          set:
            "X-Upstream-IP": "$upstream_addr"
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("assert that the request is proxied to the Service ClusterIP")
			service, err := s.GetServiceByName("httpbin-service-e2e-test")
			Expect(err).ShouldNot(HaveOccurred(), "get service")
			clusterIP := net.JoinHostPort(service.Spec.ClusterIP, "80")
			s.NewAPISIXClient().GET("/get").Expect().Header("X-Upstream-IP").IsEqual(clusterIP)
		})

		It("Test ApisixRoute subset", func() {
			const apisixRouteSpec = `
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
      subset: test-subset
`
			const apisixUpstreamSpec0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-service-e2e-test
  namespace: %s
spec:
  ingressClassName: %s
  subsets:
  - name: test-subset
    labels:
      unknown-key: unknown-value
`
			const apisixUpstreamSpec1 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-service-e2e-test
  namespace: %s
spec:
  ingressClassName: %s
  subsets:
  - name: test-subset
    labels:
      app: httpbin-deployment-e2e-test
`
			request := func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("httpbin").Expect().Raw().StatusCode
			}
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apisixRoute, fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))
			Eventually(request).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			// no pod matches the subset label "unknown-key: unknown-value" so there will be no node in the upstream,
			// to request the route will get http.StatusServiceUnavailable
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-service-e2e-test"},
				new(apiv2.ApisixUpstream), fmt.Sprintf(apisixUpstreamSpec0, s.Namespace(), s.Namespace()))
			Eventually(request).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusServiceUnavailable))

			// the pod matches the subset label "app: httpbin-deployment-e2e-test",
			// to request the route will be OK
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-service-e2e-test"},
				new(apiv2.ApisixUpstream), fmt.Sprintf(apisixUpstreamSpec1, s.Namespace(), s.Namespace()))
			Eventually(request).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("Multiple ApisixRoute with same prefix name", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - %s
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			for _, id := range []string{"11111", "1111", "111", "11", "1"} {
				name := fmt.Sprintf("route-%s", id)
				host := fmt.Sprintf("httpbin-%s", id)
				applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: name},
					&apisixRoute, fmt.Sprintf(apisixRouteSpec, name, s.Namespace(), s.Namespace(), host))
			}

			By("verify ApisixRoute works")
			for _, id := range []string{"1", "11", "111", "1111", "11111"} {
				host := fmt.Sprintf("httpbin-%s", id)
				Eventually(func() int {
					return s.NewAPISIXClient().GET("/get").WithHost(host).Expect().Raw().StatusCode
				}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(Equal(http.StatusOK))
			}
		})
	})

	Context("Test ApisixRoute reference ApisixUpstream", func() {
		It("Test reference ApisixUpstream", func() {
			const apisixRouteSpec = `
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
      paths:
      - /*
    upstreams:
    - name: default-upstream
`
			const apisixUpstreamSpec0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: default-upstream
  namespace: %s
spec:
  ingressClassName: %s
  externalNodes:
  - type: Service
    name: httpbin-service-e2e-test
`
			const apisixUpstreamSpec1 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: default-upstream
  namespace: %s
spec:
  ingressClassName: %s
  externalNodes:
  - type: Service
    name: alias-httpbin-service-e2e-test
`
			const serviceSpec = `
apiVersion: v1
kind: Service
metadata:
  name: alias-httpbin-service-e2e-test
spec:
  type: ExternalName
  externalName: httpbin-service-e2e-test
`
			By("create Service, ApisixUpstream and ApisixRoute")
			err := s.CreateResourceFromStringWithNamespace(serviceSpec, s.Namespace())
			Expect(err).ShouldNot(HaveOccurred(), "apply service")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default-upstream"},
				new(apiv2.ApisixUpstream), fmt.Sprintf(apisixUpstreamSpec0, s.Namespace(), s.Namespace()))

			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				new(apiv2.ApisixRoute), fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))

			By("verify that the ApisixUpstream reference a Service which is not ExternalName should not request OK")
			request := func(path string) int {
				return s.NewAPISIXClient().GET(path).WithHost("httpbin").Expect().Raw().StatusCode
			}
			Eventually(request).WithArguments("/get").WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusServiceUnavailable))

			By("verify that ApisixUpstream reference a Service which is ExternalName should request OK")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default-upstream"},
				new(apiv2.ApisixUpstream), fmt.Sprintf(apisixUpstreamSpec1, s.Namespace(), s.Namespace()))
			Eventually(request).WithArguments("/get").WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(Equal(http.StatusOK))
		})

		It("Test a Mix of Backends and Upstreams", func() {
			// apisixUpstreamSpec is an ApisixUpstream reference to the Service httpbin-service-e2e-test
			const apisixUpstreamSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: default-upstream
  namespace: %s
spec:
  ingressClassName: %s
  externalNodes:
  - type: Domain
    name: httpbin-service-e2e-test
  passHost: node
`
			// apisixRouteSpec is an ApisixUpstream uses a backend and reference an upstream.
			// It contains a plugin response-rewrite that lets us know what upstream the gateway forwards the request to.
			const apisixRouteSpec = `
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
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    upstreams:
    - name: default-upstream
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          set:
            "X-Upstream-Host": "$upstream_addr"
`
			By("apply ApisixUpstream")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default-upstream"},
				new(apiv2.ApisixUpstream), fmt.Sprintf(apisixUpstreamSpec, s.Namespace(), s.Namespace()))

			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				new(apiv2.ApisixRoute), fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))

			By("verify ApisixRoute works")
			request := func(path string) int {
				return s.NewAPISIXClient().GET(path).Expect().Raw().StatusCode
			}
			Eventually(request).WithArguments("/get").WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("verify the backends and the upstreams work commonly")
			// .backends -> Service httpbin-service-e2e-test -> Endpoint httpbin-service-e2e-test, so the $upstream_addr value we get is the Endpoint IP.
			// .upstreams -> Service httpbin-service-e2e-test, so the $upstream_addr value we get is the Service ClusterIP.
			var upstreamAddrs = make(map[string]struct{})
			for range 10 {
				upstreamAddr := s.NewAPISIXClient().GET("/get").Expect().Raw().Header.Get("X-Upstream-Host")
				upstreamAddrs[upstreamAddr] = struct{}{}
			}

			endpoints, err := s.GetServiceEndpoints(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-service-e2e-test"})
			Expect(err).ShouldNot(HaveOccurred(), "get endpoints")
			Expect(endpoints).Should(HaveLen(1))
			endpoint := net.JoinHostPort(endpoints[0], "80")

			service, err := s.GetServiceByName("httpbin-service-e2e-test")
			Expect(err).ShouldNot(HaveOccurred(), "get service")
			clusterIP := net.JoinHostPort(service.Spec.ClusterIP, "80")

			Expect(upstreamAddrs).Should(HaveLen(2))
			Eventually(upstreamAddrs).Should(HaveKey(endpoint))
			Eventually(upstreamAddrs).Should(HaveKey(clusterIP))
		})

		It("Test backend implicit reference to apisixupstream", func() {
			var err error

			const apisixRouteSpec = `
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
    - name: response-rewrite
      enable: true
      config:
        headers:
          set:
            "X-Upstream-Host": "$upstream_host"

`
			const apisixUpstreamSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-service-e2e-test
  namespace: %s
spec:
  ingressClassName: %s
  passHost: rewrite
  upstreamHost: hello.httpbin.org
  loadbalancer:
    type: "chash"
    hashOn: "vars"
    key: "server_name"
`
			expectUpstreamHostIs := func(expectedUpstreamHost string) func(ctx context.Context) (bool, error) {
				return func(ctx context.Context) (done bool, err error) {
					resp := s.NewAPISIXClient().GET("/get").WithHost("httpbin").Expect().Raw()
					return resp.StatusCode == http.StatusOK && resp.Header.Get("X-Upstream-Host") == expectedUpstreamHost, nil
				}
			}

			By("apply apisixroute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				new(apiv2.ApisixRoute), fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))

			By("verify ApisixRoute works")
			// expect upstream host is "httpbin"
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, expectUpstreamHostIs("httpbin"))
			Expect(err).ShouldNot(HaveOccurred(), "verify ApisixRoute works")

			By("apply apisixupstream")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-service-e2e-test"},
				new(apiv2.ApisixUpstream), fmt.Sprintf(apisixUpstreamSpec, s.Namespace(), s.Namespace()))

			By("verify backend implicit reference to apisixupstream works")
			// expect upstream host is "hello.httpbin.org" which is rewritten by the apisixupstream
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, expectUpstreamHostIs("hello.httpbin.org"))
			Expect(err).ShouldNot(HaveOccurred(), "check apisixupstream is referenced")
		})
	})

	Context("Test ApisixRoute Traffic Split", func() {
		It("2:1 traffic split test", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: default
 namespace: %s
spec:
 ingressClassName: %s
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /get
   backends:
   - serviceName: httpbin-service-e2e-test
     servicePort: 80
     weight: 10
   - serviceName: %s
     servicePort: 9180
     weight: 5
`
			By("apply ApisixRoute with traffic split")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				new(apiv2.ApisixRoute),
				fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace(), s.Deployer.GetAdminServiceName()))
			verifyRequest := func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("httpbin.org").Expect().Raw().StatusCode
			}
			By("send requests to verify traffic split")
			var (
				successCount int
				failCount    int
			)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			for range 90 {
				code := verifyRequest()
				if code == http.StatusOK {
					successCount++
				} else {
					failCount++
				}
			}

			By("verify traffic distribution ratio")
			ratio := float64(successCount) / float64(failCount)
			expectedRatio := 10.0 / 5.0 // 2:1 ratio
			deviation := math.Abs(ratio - expectedRatio)
			Expect(deviation).Should(BeNumerically("<", 0.5),
				"traffic distribution deviation too large (got %.2f, expected %.2f)", ratio, expectedRatio)
		})

		It("zero-weight test", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: default
 namespace: %s
spec:
 ingressClassName: %s
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /get
   backends:
   - serviceName: httpbin-service-e2e-test
     servicePort: 80
     weight: 10
   - serviceName: %s
     servicePort: 9180
     weight: 0
`
			By("apply ApisixRoute with zero-weight backend")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				new(apiv2.ApisixRoute),
				fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace(), s.Deployer.GetAdminServiceName()))
			verifyRequest := func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("httpbin.org").Expect().Raw().StatusCode
			}

			By("wait for route to be ready")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/get",
				Host:    "httpbin.org",
				Check:   scaffold.WithExpectedStatus(http.StatusOK),
				Timeout: 10 * time.Second,
			})
			By("send requests to verify zero-weight behavior")
			for range 30 {
				code := verifyRequest()
				Expect(code).Should(Equal(200))
			}
		})
		It("valid backend is set even if other backend is invalid", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: default
 namespace: %s
spec:
 ingressClassName: %s
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /get
   backends:
   - serviceName: httpbin-service-e2e-test
     servicePort: 80
     weight: 10
   - serviceName: invalid-service
     servicePort: 9180
     weight: 5
`
			By("apply ApisixRoute with traffic split")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				new(apiv2.ApisixRoute), fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))
			verifyRequest := func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("httpbin.org").Expect().Raw().StatusCode
			}

			By("wait for route to be ready")
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/get",
				Host:    "httpbin.org",
				Check:   scaffold.WithExpectedStatus(http.StatusOK),
				Timeout: 10 * time.Second,
			})
			By("send requests to verify all requests routed to valid upstream")
			for range 30 {
				code := verifyRequest()
				Expect(code).Should(Equal(200))
			}
		})
	})

	Context("Test ApisixRoute sync during startup", func() {
		const route = `
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
      - /get
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`

		const route2 = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: route2
  namespace: %s
spec:
  ingressClassName: %s-nonexistent
  http:
  - name: rule0
    match:
      hosts:
      - httpbin2
      paths:
      - /get
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
		const route3 = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: route3
spec:
  http:
  - name: rule0
    match:
      hosts:
      - httpbin3
      paths:
      - /get
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
		It("Should sync ApisixRoute during startup", func() {
			By("apply ApisixRoute")
			Expect(s.CreateResourceFromStringWithNamespace(fmt.Sprintf(route2, s.Namespace(), s.Namespace()), s.Namespace())).
				ShouldNot(HaveOccurred(), "apply ApisixRoute with nonexistent ingressClassName")
			Expect(s.CreateResourceFromStringWithNamespace(route3, s.Namespace())).ShouldNot(HaveOccurred(), "apply ApisixRoute without ingressClassName")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"},
				&apiv2.ApisixRoute{}, fmt.Sprintf(route, s.Namespace(), s.Namespace()))

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin2",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin3",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})

			By("restart controller and dataplane")
			s.Deployer.ScaleIngress(0)
			s.Deployer.ScaleDataplane(0)
			s.Deployer.ScaleDataplane(1)
			s.Deployer.ScaleIngress(1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin2",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin3",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})
		})
	})

	Context("Test ApisixRoute WebSocket Support", func() {
		It("basic websocket functionality", func() {
			const websocketServerResources = `
apiVersion: v1
kind: Pod
metadata:
  name: websocket-server
  labels:
    app: websocket-server
spec:
  containers:
  - name: websocket-server
    image: jmalloc/echo-server:latest
    ports:
    - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: websocket-server-service
spec:
  selector:
    app: websocket-server
  ports:
    - name: ws
      port: 8080
      protocol: TCP
      targetPort: 8080
`
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: websocket-route
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /echo
    websocket: true
    backends:
    - serviceName: websocket-server-service
      servicePort: 8080
`

			const apisixRouteSpec2 = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: websocket-route
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /echo
    backends:
    - serviceName: websocket-server-service
      servicePort: 8080
`

			By("create WebSocket server resources")
			err := s.CreateResourceFromStringWithNamespace(websocketServerResources, s.Namespace())
			Expect(err).ShouldNot(HaveOccurred(), "creating WebSocket server resources")

			By("create ApisixRoute without WebSocker")
			var apisixRouteWithoutWS apiv2.ApisixRoute
			applier.MustApplyAPIv2(
				types.NamespacedName{Namespace: s.Namespace(), Name: "websocket-route"},
				&apisixRouteWithoutWS,
				fmt.Sprintf(apisixRouteSpec2, s.Namespace(), s.Namespace()),
			)
			time.Sleep(12 * time.Second)

			By("verify WebSocket connection fails without WebSocket enabled")
			u := url.URL{
				Scheme: "ws",
				Host:   s.ApisixHTTPEndpoint(),
				Path:   "/echo",
			}
			headers := http.Header{"Host": []string{"httpbin.org"}}
			_, resp, _ := websocket.DefaultDialer.Dial(u.String(), headers)
			// should receive 200 instead of 101
			Expect(resp.StatusCode).Should(Equal(http.StatusOK))
			By("apply ApisixRoute for WebSocket")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(
				types.NamespacedName{Namespace: s.Namespace(), Name: "websocket-route"},
				&apisixRoute,
				fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()),
			)
			By("wait for WebSocket server to be ready")
			time.Sleep(10 * time.Second)
			By("verify WebSocket connection")
			u = url.URL{
				Scheme: "ws",
				Host:   s.ApisixHTTPEndpoint(),
				Path:   "/echo",
			}
			headers = http.Header{"Host": []string{"httpbin.org"}}

			conn, resp, err := websocket.DefaultDialer.Dial(u.String(), headers)
			Expect(err).ShouldNot(HaveOccurred(), "WebSocket handshake")
			Expect(resp.StatusCode).Should(Equal(http.StatusSwitchingProtocols))

			defer func() {
				_ = conn.Close()
			}()

			By("send and receive message through WebSocket")
			testMessage := "hello, this is APISIX"
			err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
			Expect(err).ShouldNot(HaveOccurred(), "writing WebSocket message")

			// The echo server sends an identification message first
			_, _, err = conn.ReadMessage()
			Expect(err).ShouldNot(HaveOccurred(), "reading identification message")

			// Then our echo
			_, msg, err := conn.ReadMessage()
			Expect(err).ShouldNot(HaveOccurred(), "reading echo message")
			Expect(string(msg)).To(Equal(testMessage), "message content verification")
		})
	})

	Context("Test ApisixRoute with External Services", func() {
		createExternalService := func(externalName string, externalServiceName string) {
			By(fmt.Sprintf("create ExternalName service: %s -> %s", externalServiceName, externalName))
			svcSpec := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  type: ExternalName
  externalName: %s
`, externalServiceName, externalName)
			err := s.CreateResourceFromStringWithNamespace(svcSpec, s.Namespace())
			Expect(err).ShouldNot(HaveOccurred(), "creating ExternalName service")
		}

		createApisixUpstream := func(externalType apiv2.ApisixUpstreamExternalType, name string, upstreamName string) {
			By(fmt.Sprintf("create ApisixUpstream: type=%s, name=%s", externalType, name))
			upstreamSpec := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
  namespace: %s
spec:
  externalNodes:
  - type: %s
    name: %s
`, upstreamName, s.Namespace(), externalType, name)
			var upstream apiv2.ApisixUpstream
			applier.MustApplyAPIv2(
				types.NamespacedName{Namespace: s.Namespace(), Name: upstreamName},
				&upstream,
				upstreamSpec,
			)
		}

		createApisixRoute := func(routeName string, upstreamName string) {
			By("create ApisixRoute referencing ApisixUpstream")
			routeSpec := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /ip
    upstreams:
    - name: %s
`, routeName, s.Namespace(), s.Namespace(), upstreamName)
			var route apiv2.ApisixRoute
			applier.MustApplyAPIv2(
				types.NamespacedName{Namespace: s.Namespace(), Name: routeName},
				&route,
				routeSpec,
			)
		}

		createApisixRouteWithHostRewrite := func(routeName string, host string, upstreamName string) {
			By("create ApisixRoute with host rewrite")
			routeSpec := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /ip
    upstreams:
    - name: %s
    plugins:
    - name: proxy-rewrite
      enable: true
      config:
        host: %s
`, routeName, s.Namespace(), s.Namespace(), upstreamName, host)
			var route apiv2.ApisixRoute
			applier.MustApplyAPIv2(
				types.NamespacedName{Namespace: s.Namespace(), Name: routeName},
				&route,
				routeSpec,
			)
		}

		verifyAccess := func() {
			By("verify access to external service")
			request := func() int {
				return s.NewAPISIXClient().GET("/ip").
					WithHost("httpbin.org").
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).
				Should(Equal(http.StatusOK))
		}

		It("access third-party service directly", func() {
			upstreamName := s.Namespace()
			routeName := s.Namespace()
			createApisixUpstream(apiv2.ExternalTypeDomain, "httpbin.org", upstreamName)
			createApisixRoute(routeName, upstreamName)
			verifyAccess()
		})

		It("access third-party service with host rewrite", func() {
			upstreamName := s.Namespace()
			routeName := s.Namespace()
			createApisixUpstream(apiv2.ExternalTypeDomain, "httpbin.org", upstreamName)
			createApisixRouteWithHostRewrite(routeName, "httpbin.org", upstreamName)
			verifyAccess()
		})

		It("access external domain via ExternalName service", func() {
			externalServiceName := s.Namespace()
			upstreamName := s.Namespace()
			routeName := s.Namespace()
			createExternalService("httpbin.org", externalServiceName)
			createApisixUpstream(apiv2.ExternalTypeService, externalServiceName, upstreamName)
			createApisixRoute(routeName, upstreamName)
			verifyAccess()
		})

		It("access in-cluster service via ExternalName", func() {
			By("create temporary httpbin service")

			By("get FQDN of temporary service")
			fqdn := fmt.Sprintf("%s.%s.svc.cluster.local", "httpbin-service-e2e-test", s.Namespace())

			By("setup external service and route")
			externalServiceName := s.Namespace()
			upstreamName := s.Namespace()
			routeName := s.Namespace()
			createExternalService(fqdn, externalServiceName)
			createApisixUpstream(apiv2.ExternalTypeService, externalServiceName, upstreamName)
			createApisixRoute(routeName, upstreamName)
			verifyAccess()
		})

		Context("complex scenarios", func() {
			It("multiple external services in one upstream", func() {
				upstreamName := s.Namespace()
				routeName := s.Namespace()
				By("create ApisixUpstream with multiple external nodes")
				upstreamSpec := `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
  namespace: %s
spec:
  externalNodes:
  - type: Domain
    name: httpbin.org
  - type: Domain
    name: postman-echo.com
`
				var upstream apiv2.ApisixUpstream
				applier.MustApplyAPIv2(
					types.NamespacedName{Namespace: s.Namespace(), Name: upstreamName},
					&upstream,
					fmt.Sprintf(upstreamSpec, upstreamName, s.Namespace()),
				)

				createApisixRoute(routeName, upstreamName)

				By("verify access to multiple services")
				time.Sleep(7 * time.Second)
				hasEtag := false   // postman-echo.com
				hasNoEtag := false // httpbin.org
				for range 20 {
					headers := s.NewAPISIXClient().GET("/ip").
						WithHeader("Host", "httpbin.org").
						WithHeader("X-Foo", "bar").
						Expect().
						Headers().Raw()
					if _, ok := headers["Etag"]; ok {
						hasEtag = true
					} else {
						hasNoEtag = true
					}
					if hasEtag && hasNoEtag {
						break
					}
				}
				assert.True(GinkgoT(), hasEtag && hasNoEtag, "both httpbin and postman should be accessed at least once")
			})

			It("should be able to use backends and upstreams together", func() {
				upstreamName := s.Namespace()
				routeName := s.Namespace()
				upstreamSpec := `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  externalNodes:
  - type: Domain
    name: postman-echo.com
`
				var upstream apiv2.ApisixUpstream
				applier.MustApplyAPIv2(
					types.NamespacedName{Namespace: s.Namespace(), Name: upstreamName},
					&upstream,
					fmt.Sprintf(upstreamSpec, upstreamName),
				)
				By("create ApisixRoute with both backends and upstreams")
				routeSpec := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /ip
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
      resolveGranularity: service
    upstreams:
    - name: %s
`, routeName, s.Namespace(), s.Namespace(), upstreamName)
				var route apiv2.ApisixRoute
				applier.MustApplyAPIv2(
					types.NamespacedName{Namespace: s.Namespace(), Name: routeName},
					&route,
					routeSpec,
				)
				By("verify access to multiple services")
				time.Sleep(7 * time.Second)
				hasEtag := false   // postman-echo.com
				hasNoEtag := false // httpbin.org
				for range 20 {
					headers := s.NewAPISIXClient().GET("/ip").
						WithHeader("Host", "httpbin.org").
						WithHeader("X-Foo", "bar").
						Expect().
						Headers().Raw()
					if _, ok := headers["Etag"]; ok {
						hasEtag = true
					} else {
						hasNoEtag = true
					}
					if hasEtag && hasNoEtag {
						break
					}
				}
				assert.True(GinkgoT(), hasEtag && hasNoEtag, "both httpbin and postman should be accessed at least once")
			})
		})
	})
})
