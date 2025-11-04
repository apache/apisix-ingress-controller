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

			Eventually(func() bool {
				routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
				if err != nil {
					return false
				}
				if len(routes) != 1 {
					return false
				}
				if routes[0].EnableWebsocket == nil || !*routes[0].EnableWebsocket {
					return false
				}
				return true
			}).WithTimeout(30 * time.Second).ProbeEvery(2 * time.Second).Should(BeTrue())
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
			ingressCSRF = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: csrf
  annotations:
    k8s.apisix.apache.org/enable-csrf: "true"
    k8s.apisix.apache.org/csrf-key: "foo-key"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin.example
    http:
      paths:
      - path: /anything
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`
			allowMethods = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: allow-methods
  annotations:
    k8s.apisix.apache.org/http-allow-methods: "GET,POST"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin.example
    http:
      paths:
      - path: /anything
        pathType: Exact
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`

			blockMethods = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: block-methods
  annotations:
    k8s.apisix.apache.org/http-block-methods: "DELETE"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin2.example
    http:
      paths:
      - path: /anything
        pathType: Exact
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`
			ingressKeyAuth = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: key-auth
  annotations:
    k8s.apisix.apache.org/auth-type: "keyAuth"
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
			ingressBasicAuth = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: basic-auth
  annotations:
    k8s.apisix.apache.org/auth-type: "basicAuth"
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

			ingressRewriteTarget = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rewrite-target
  annotations:
    k8s.apisix.apache.org/rewrite-target: "/get"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin.example
    http:
      paths:
      - path: /test
        pathType: Exact
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`

			ingressRewriteTargetRegex = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: rewrite-target-regex
  annotations:
    k8s.apisix.apache.org/rewrite-target-regex: "/sample/(.*)"
    k8s.apisix.apache.org/rewrite-target-regex-template: "/$1"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin-regex.example
    http:
      paths:
      - path: /sample
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`
			responseRewrite = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: response-rewrite
  annotations:
    k8s.apisix.apache.org/enable-response-rewrite: "true"
    k8s.apisix.apache.org/response-rewrite-status-code: "400"
    k8s.apisix.apache.org/response-rewrite-body: "custom response body"
    k8s.apisix.apache.org/response-rewrite-body-base64: "false"
    k8s.apisix.apache.org/response-rewrite-set-header: "X-Custom-Header:custom-value"
    k8s.apisix.apache.org/response-rewrite-add-header: "X-Add-Header:added-value"
    k8s.apisix.apache.org/response-rewrite-remove-header: "Server"
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
			responseRewriteBase64 = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: response-rewrite-base64
  annotations:
    k8s.apisix.apache.org/enable-response-rewrite: "true"
    k8s.apisix.apache.org/response-rewrite-status-code: "400"
    k8s.apisix.apache.org/response-rewrite-body: "Y3VzdG9tIHJlc3BvbnNlIGJvZHk="
    k8s.apisix.apache.org/response-rewrite-body-base64: "true"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin-base64.example
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

			ingressAllowlist = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: allowlist
  annotations:
    k8s.apisix.apache.org/allowlist-source-range: "10.0.5.0/16"
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

			ingressBlocklist = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: blocklist
  annotations:
    k8s.apisix.apache.org/blocklist-source-range: "127.0.0.1"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin-block.example
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
			ingressForwardAuth = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: forward-auth
  annotations:
    k8s.apisix.apache.org/auth-uri: %s
    k8s.apisix.apache.org/auth-request-headers: Authorization
    k8s.apisix.apache.org/auth-upstream-headers: X-User-ID
    k8s.apisix.apache.org/auth-client-headers: Location
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

		It("csrf", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressCSRF, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			time.Sleep(5 * time.Second)

			By("Request without CSRF token should fail")
			msg401 := s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			Expect(msg401).To(ContainSubstring("no csrf token in headers"), "checking error message")

			By("GET request should succeed and return CSRF token in cookie")
			resp := s.NewAPISIXClient().
				GET("/anything").
				WithHeader("Host", "httpbin.example").
				Expect().
				Status(http.StatusOK)
			resp.Header("Set-Cookie").NotEmpty()

			cookie := resp.Cookie("apisix-csrf-token")
			token := cookie.Value().Raw()

			By("POST request with valid CSRF token should succeed")
			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.example").
				WithHeader("apisix-csrf-token", token).
				WithCookie("apisix-csrf-token", token).
				Expect().
				Status(http.StatusOK)

			By("Verify CSRF plugin is configured in the route")
			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Route")
			Expect(routes).To(HaveLen(1), "checking Route length")
			Expect(routes[0].Plugins).To(HaveKey("csrf"), "checking Route plugins")
			jsonBytes, err := json.Marshal(routes[0].Plugins["csrf"])
			Expect(err).NotTo(HaveOccurred(), "marshalling csrf plugin config")
			var csrfConfig map[string]any
			err = json.Unmarshal(jsonBytes, &csrfConfig)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling csrf plugin config")
			Expect(csrfConfig["key"]).To(Equal("foo-key"), "checking csrf key")
		})

		It("plugin-config-name annotation", func() {
			// Create ApisixPluginConfig
			pluginConfig := `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-plugin-config
spec:
  ingressClassName: %s
  plugins:
  - name: echo
    enable: true
    config:
      body: "hello from plugin config"
`
			Expect(s.CreateResourceFromString(fmt.Sprintf(pluginConfig, s.Namespace()))).ShouldNot(HaveOccurred(), "creating ApisixPluginConfig")

			// Create Ingress with plugin-config-name annotation
			ingressWithPluginConfig := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: plugin-config-test
  annotations:
    k8s.apisix.apache.org/plugin-config-name: "test-plugin-config"
spec:
  ingressClassName: %s
  rules:
  - host: plugin-config.example
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
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressWithPluginConfig, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "plugin-config.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusOK),
					scaffold.WithExpectedBodyContains("hello from plugin config"),
				},
			})

			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Route")
			Expect(routes).ToNot(BeEmpty(), "checking Route length")

			Expect(routes).To(HaveLen(1), "checking Route length")
			Expect(routes[0].Plugins).To(HaveKey("echo"), "checking Route has echo plugin from PluginConfig")

			// Verify plugin config content
			jsonBytes, err := json.Marshal(routes[0].Plugins["echo"])
			Expect(err).NotTo(HaveOccurred(), "marshalling echo plugin config")
			var echoConfig map[string]any
			err = json.Unmarshal(jsonBytes, &echoConfig)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling echo plugin config")
			Expect(echoConfig["body"]).To(Equal("hello from plugin config"), "checking echo plugin body")
		})
		It("methods", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(allowMethods, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")
			Expect(s.CreateResourceFromString(fmt.Sprintf(blockMethods, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			tets := []*scaffold.RequestAssert{
				{
					Method: "GET",
					Path:   "/anything",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "POST",
					Path:   "/anything",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "PUT",
					Path:   "/anything",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusMethodNotAllowed),
				},
				{
					Method: "PATCH",
					Path:   "/anything",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusMethodNotAllowed),
				},
				{
					Method: "DELETE",
					Path:   "/anything",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusMethodNotAllowed),
				},
				{
					Method: "GET",
					Path:   "/anything",
					Host:   "httpbin2.example",
					Check:  scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "POST",
					Path:   "/anything",
					Host:   "httpbin2.example",
					Check:  scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "PUT",
					Path:   "/anything",
					Host:   "httpbin2.example",
					Check:  scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "PATCH",
					Path:   "/anything",
					Host:   "httpbin2.example",
					Check:  scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "DELETE",
					Path:   "/anything",
					Host:   "httpbin2.example",
					Check:  scaffold.WithExpectedStatus(http.StatusMethodNotAllowed),
				},
			}

			for _, test := range tets {
				s.RequestAssert(test)
			}
		})
		It("authentication", func() {
			var (
				keyAuth = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: key
spec:
  ingressClassName: %s
  authParameter:
    keyAuth:
      value:
        key: test-key
`
				basicAuth = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: basic
spec:
  ingressClassName: %s
  authParameter:
    basicAuth:
      value:
        username: test-user
        password: test-password
`
			)
			Expect(s.CreateResourceFromString(fmt.Sprintf(keyAuth, s.Namespace()))).ShouldNot(HaveOccurred(), "creating ApisixConsumer for keyAuth")
			Expect(s.CreateResourceFromString(fmt.Sprintf(basicAuth, s.Namespace()))).ShouldNot(HaveOccurred(), "creating ApisixConsumer for basicAuth")
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressKeyAuth, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressBasicAuth, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			tests := []*scaffold.RequestAssert{
				{
					Method: "GET",
					Path:   "/get",
					Host:   "httpbin.example",
					BasicAuth: &scaffold.BasicAuth{
						Username: "test-user",
						Password: "test-password",
					},
					Check: scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "GET",
					Path:   "/get",
					Host:   "httpbin.example",
					BasicAuth: &scaffold.BasicAuth{
						Username: "invalid-user",
						Password: "invalid-password",
					},
					Check: scaffold.WithExpectedStatus(http.StatusUnauthorized),
				},
				{
					Method: "GET",
					Path:   "/ip",
					Host:   "httpbin.example",
					Headers: map[string]string{
						"apikey": "test-key",
					},
					Check: scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "GET",
					Path:   "/ip",
					Host:   "httpbin.example",
					Headers: map[string]string{
						"apikey": "invalid-key",
					},
					Check: scaffold.WithExpectedStatus(http.StatusUnauthorized),
				},
			}
			for _, test := range tests {
				s.RequestAssert(test)
			}
		})

		It("proxy-rewrite with rewrite-target", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressRewriteTarget, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/test",
				Host:    "httpbin.example",
				Timeout: 60 * time.Second,
				Check:   scaffold.WithExpectedStatus(http.StatusOK),
			})

			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Route")
			Expect(routes).ToNot(BeEmpty(), "checking Route length")
			Expect(routes[0].Plugins).To(HaveKey("proxy-rewrite"), "checking Route has proxy-rewrite plugin")

			jsonBytes, err := json.Marshal(routes[0].Plugins["proxy-rewrite"])
			Expect(err).NotTo(HaveOccurred(), "marshalling proxy-rewrite plugin config")
			var rewriteConfig map[string]any
			err = json.Unmarshal(jsonBytes, &rewriteConfig)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling proxy-rewrite plugin config")
			Expect(rewriteConfig["uri"]).To(Equal("/get"), "checking proxy-rewrite uri")
		})

		It("proxy-rewrite with regex", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressRewriteTargetRegex, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/sample/get",
				Host:    "httpbin-regex.example",
				Timeout: 60 * time.Second,
				Check:   scaffold.WithExpectedStatus(http.StatusOK),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/sample/anything",
				Host:   "httpbin-regex.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})

			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Route")
			Expect(routes).ToNot(BeEmpty(), "checking Route length")
			Expect(routes[0].Plugins).To(HaveKey("proxy-rewrite"), "checking Route has proxy-rewrite plugin")

			jsonBytes, err := json.Marshal(routes[0].Plugins["proxy-rewrite"])
			Expect(err).NotTo(HaveOccurred(), "marshalling proxy-rewrite plugin config")
			var rewriteConfig map[string]any
			err = json.Unmarshal(jsonBytes, &rewriteConfig)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling proxy-rewrite plugin config")

			regexUri, ok := rewriteConfig["regex_uri"].([]any)
			Expect(ok).To(BeTrue(), "checking regex_uri is array")
			Expect(regexUri).To(HaveLen(2), "checking regex_uri length")
			Expect(regexUri[0]).To(Equal("/sample/(.*)"), "checking regex pattern")
			Expect(regexUri[1]).To(Equal("/$1"), "checking regex template")
		})

		It("response-rewrite", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(responseRewrite, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusBadRequest),
					scaffold.WithExpectedBodyContains("custom response body"),
					scaffold.WithExpectedHeader("X-Custom-Header", "custom-value"),
					scaffold.WithExpectedHeader("X-Add-Header", "added-value"),
				},
			})

			By("Verify response-rewrite plugin is configured in the route")
			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Route")
			Expect(routes).To(HaveLen(1), "checking Route length")
			Expect(routes[0].Plugins).To(HaveKey("response-rewrite"), "checking Route plugins")

			jsonBytes, err := json.Marshal(routes[0].Plugins["response-rewrite"])
			Expect(err).NotTo(HaveOccurred(), "marshalling response-rewrite plugin config")
			var rewriteConfig map[string]any
			err = json.Unmarshal(jsonBytes, &rewriteConfig)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling response-rewrite plugin config")
			Expect(rewriteConfig["status_code"]).To(Equal(float64(400)), "checking status code")
			Expect(rewriteConfig["body"]).To(Equal("custom response body"), "checking body")
		})

		It("response-rewrite with base64", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(responseRewriteBase64, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin-base64.example",
				Checks: []scaffold.ResponseCheckFunc{
					scaffold.WithExpectedStatus(http.StatusBadRequest),
					scaffold.WithExpectedBodyContains("custom response body"),
				},
			})
			By("Verify response-rewrite plugin is configured in the route")
			routes, err := s.DefaultDataplaneResource().Route().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Route")
			Expect(routes).To(HaveLen(1), "checking Route length")
			Expect(routes[0].Plugins).To(HaveKey("response-rewrite"), "checking Route plugins")

			jsonBytes, err := json.Marshal(routes[0].Plugins["response-rewrite"])
			Expect(err).NotTo(HaveOccurred(), "marshalling response-rewrite plugin config")
			var rewriteConfig map[string]any
			err = json.Unmarshal(jsonBytes, &rewriteConfig)
			Expect(err).NotTo(HaveOccurred(), "unmarshalling response-rewrite plugin config")
			Expect(rewriteConfig["status_code"]).To(Equal(float64(400)), "checking status code")
			Expect(rewriteConfig["body_base64"]).To(BeTrue(), "checking body_base64")
		})

		It("ip-restriction", func() {
			By("Test allowlist - create ingress with IP allowlist")
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressAllowlist, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress with allowlist")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/ip",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusForbidden),
			})

			By("Test blocklist - create ingress with IP blocklist")
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressBlocklist, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress with blocklist")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/ip",
				Host:   "httpbin-block.example",
				Check:  scaffold.WithExpectedStatus(http.StatusForbidden),
			})
		})
		It("forward-auth", func() {
			s.DeployNginx(framework.NginxOptions{
				Namespace: s.Namespace(),
				Replicas:  ptr.To(int32(1)),
			})

			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressForwardAuth, "http://nginx/auth", s.Namespace()))).
				ShouldNot(HaveOccurred(), "creating ApisixConsumer for forwardAuth")

			tests := []*scaffold.RequestAssert{
				{
					Method: "GET",
					Path:   "/get",
					Host:   "httpbin.example",
					Headers: map[string]string{
						"Authorization": "123",
					},
					Checks: []scaffold.ResponseCheckFunc{
						scaffold.WithExpectedStatus(http.StatusOK),
						scaffold.WithExpectedBodyContains(`"X-User-Id": "user-123"`),
					},
				},
				{
					Method: "GET",
					Path:   "/get",
					Host:   "httpbin.example",
					Headers: map[string]string{
						"Authorization": "456",
					},
					Checks: []scaffold.ResponseCheckFunc{
						scaffold.WithExpectedStatus(http.StatusUnauthorized),
						scaffold.WithExpectedHeader("Location", "http://example.com/auth"),
					},
				},
			}
			for _, test := range tests {
				s.RequestAssert(test)
			}
		})
	})

	Context("Service Namespace", func() {
		var (
			ns  string
			svc = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: httpbin-service-e2e-test.%s.svc
`
			ingressSvcNamespace = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: retries
  annotations:
    k8s.apisix.apache.org/svc-namespace: %s
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
            name: httpbin-external-domain
            port:
              number: 80
`
		)
		BeforeEach(func() {
			ns = s.Namespace() + "-v2"
			s.CreateNamespace(ns)
			err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(svc, s.Namespace()), ns)
			Expect(err).NotTo(HaveOccurred(), "creating Service in custom namespace")

			By("create GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})
		AfterEach(func() {
			s.DeleteNamespace(ns)
		})
		It("svc-namespace", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressSvcNamespace, ns, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
		})
	})

	Context("Route", func() {
		var (
			ingressRegex = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: regex
  annotations:
    k8s.apisix.apache.org/use-regex: "true"
spec:
  ingressClassName: %s
  rules:
  - host: httpbin.example
    http:
      paths:
      - path: /anything/.*/ok
        pathType: ImplementationSpecific
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
		It("regex match", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressRegex, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			tests := []*scaffold.RequestAssert{
				{
					Method: "GET",
					Path:   "/anything/test/ok",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "GET",
					Path:   "/anything/ip/ok",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusOK),
				},
				{
					Method: "GET",
					Path:   "/test/notok",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
				},

				{
					Method: "GET",
					Path:   "/anything",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
				},
				{
					Method: "GET",
					Path:   "/anything/test/notok",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
				},
				{
					Method: "GET",
					Path:   "/anything/ok",
					Host:   "httpbin.example",
					Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
				},
			}
			for _, test := range tests {
				s.RequestAssert(test)
			}
		})
	})
})
