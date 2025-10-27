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

package ingress

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Ingress With Annotations", Label("networking.k8s.io", "ingress"), func() {
	s := scaffold.NewDefaultScaffold()

	Context("Upstream", func() {
		var (
			ingressRetries = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: retries
  annotations:
    k8s.apisix.apache.org/upstream-retries: "3"
spec:
  ingressClassName: %s
  rules:
  - host: nginx.example
    http:
      paths:
      - path: /get
        pathType: Exact
        backend:
          service:
            name: nginx
            port:
              number: 80
`
			ingressSchemeHTTPS = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: https-backend
  annotations:
    k8s.apisix.apache.org/upstream-scheme: https
spec:
  ingressClassName: %s
  rules:
  - host: nginx.example
    http:
      paths:
      - path: /get
        pathType: Exact
        backend:
          service:
            name: nginx
            port:
              number: 7443
`

			ingressTimeouts = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: timeouts
  annotations:
    k8s.apisix.apache.org/upstream-read-timeout: "2s"
    k8s.apisix.apache.org/upstream-send-timeout: "3s"
    k8s.apisix.apache.org/upstream-connect-timeout: "4s"
spec:
  ingressClassName: %s
  rules:
  - host: nginx.example
    http:
      paths:
      - path: /delay
        pathType: Exact
        backend:
          service:
            name: nginx
            port:
              number: 443
`

			ingressCORS = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cors
  annotations:
    k8s.apisix.apache.org/enable-cors: "true"
    k8s.apisix.apache.org/cors-allow-origin: "https://allowed.example"
    k8s.apisix.apache.org/cors-allow-methods: "GET,POST"
    k8s.apisix.apache.org/cors-allow-headers: "Origin,Authorization"
spec:
  ingressClassName: %s
  rules:
  - host: cors.example
    http:
      paths:
      - path: /get
        pathType: Exact
        backend:
          service:
            name: nginx
            port:
              number: 80
`

			ingressWebSocket = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: websocket
  annotations:
    k8s.apisix.apache.org/enable-websocket: "true"
spec:
  ingressClassName: %s
  rules:
  - host: nginx.example
    http:
      paths:
      - path: /ws
        pathType: Exact
        backend:
          service:
            name: nginx
            port:
              number: 80
`
		)
		BeforeEach(func() {
			s.DeployNginx(framework.NginxOptions{
				Namespace: s.Namespace(),
				Replicas:  ptr.To(int32(1)),
			})
			By("create GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

			By("create IngressClass")
			err := s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})
		It("retries", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressRetries, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "nginx.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			upstreams, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Upstream")
			Expect(upstreams).To(HaveLen(1), "checking Upstream length")
			Expect(upstreams[0].Retries).To(Equal(ptr.To(int64(3))), "checking Upstream retries")
		})
		It("scheme", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressSchemeHTTPS, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "nginx.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			upstreams, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Upstream")
			Expect(upstreams).To(HaveLen(1), "checking Upstream length")
			Expect(upstreams[0].Scheme).To(Equal("https"), "checking Upstream scheme")
		})
		It("timeouts", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressTimeouts, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/delay",
				Host:   "nginx.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})

			_ = s.NewAPISIXClient().GET("/delay").WithQuery("delay", "10").
				WithHost("nginx.example").Expect().Status(http.StatusGatewayTimeout)

			_ = s.NewAPISIXClient().GET("/delay").WithHost("nginx.example").Expect().Status(http.StatusOK)

			upstreams, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Upstream")
			Expect(upstreams).To(HaveLen(1), "checking Upstream length")
			Expect(upstreams[0].Timeout).ToNot(BeNil(), "checking Upstream timeout")
			Expect(upstreams[0].Timeout.Read).To(Equal(2), "checking Upstream read timeout")
			Expect(upstreams[0].Timeout.Send).To(Equal(3), "checking Upstream send timeout")
			Expect(upstreams[0].Timeout.Connect).To(Equal(4), "checking Upstream connect timeout")
		})

		It("cors annotations", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressCORS, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "cors.example",
				Headers: map[string]string{
					"Origin": "https://allowed.example",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedHeaders(map[string]string{
						"Access-Control-Allow-Origin":  "https://allowed.example",
						"Access-Control-Allow-Methods": "GET,POST",
						"Access-Control-Allow-Headers": "Origin,Authorization",
					}),
				},
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "cors.example",
				Headers: map[string]string{
					"Origin": "https://blocked.example",
				},
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedNotHeader("Access-Control-Allow-Origin"),
				},
			})

			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Service")
			Expect(routes).To(HaveLen(1), "checking Route length")
			Expect(routes[0].Plugins).To(HaveKey("cors"), "checking Route plugins")
			jsonBytes, err := json.Marshal(routes[0].Plugins["cors"])
			Expect(err).NotTo(HaveOccurred(), "marshalling cors plugin config")
			var corsConfig map[string]any
			err = json.Unmarshal(jsonBytes, &corsConfig)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling cors plugin config")
			Expect(corsConfig["allow_origins"]).To(Equal("https://allowed.example"), "checking cors allow origins")
			Expect(corsConfig["allow_methods"]).To(Equal("GET,POST"), "checking cors allow methods")
			Expect(corsConfig["allow_headers"]).To(Equal("Origin,Authorization"), "checking cors allow headers")
		})

		It("websocket", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressWebSocket, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Route")
			Expect(routes).To(HaveLen(1), "checking Route length")
			Expect(routes[0].EnableWebsocket).To(Equal(ptr.To(true)), "checking Route EnableWebsocket")
		})
	})

	Context("Plugins", func() {
		var (
			tohttps = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tohttps
  annotations:
    k8s.apisix.apache.org/http-to-https: "true"
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
			redirect = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: redirect
  annotations:
    k8s.apisix.apache.org/http-redirect: "/anything$uri"
    k8s.apisix.apache.org/http-redirect-code: "308"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin.example
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`
		)
		BeforeEach(func() {
			By("create GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

			By("create IngressClass")
			err := s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})
		It("redirect", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(tohttps, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")
			Expect(s.CreateResourceFromString(fmt.Sprintf(redirect, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusMovedPermanently),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/ip",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusPermanentRedirect),
			})

			_ = s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.example").
				Expect().
				Status(http.StatusMovedPermanently).
				Header("Location").IsEqual("https://httpbin.example:9443/get")

			_ = s.NewAPISIXClient().
				GET("/ip").
				WithHost("httpbin.example").
				Expect().
				Status(http.StatusPermanentRedirect).
				Header("Location").IsEqual("/anything/ip")
		})
	})
})
