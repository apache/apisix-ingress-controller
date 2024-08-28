// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package annotations

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-annotations: rewrite annotations", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("enable in ingress networking/v1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/rewrite-target: "/ip"
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /sample
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/sample").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)
	})

	ginkgo.It("enable in ingress networking/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/rewrite-target: "/ip"
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /sample
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/sample").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)
	})
})

var _ = ginkgo.Describe("suite-annotations: rewrite regex annotations", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("enable in ingress networking/v1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/rewrite-target-regex: "/sample/(.*)"
    k8s.apisix.apache.org/rewrite-target-regex-template: "/$1"
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /sample
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/sample/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/sample/get").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("enable in ingress networking/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/rewrite-target-regex: "/sample/(.*)"
    k8s.apisix.apache.org/rewrite-target-regex-template: "/$1"
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /sample
        pathType: Prefix
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		_ = s.NewAPISIXClient().GET("/sample/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		_ = s.NewAPISIXClient().GET("/sample/get").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})
})

var _ = ginkgo.FDescribe("suite-annotations: rewrite header annotations", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("enable in ingress networking/v1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/rewrite-target-regex: "/sample/(.*)"
    k8s.apisix.apache.org/rewrite-target-regex-template: "/$1"
    k8s.apisix.apache.org/rewrite-add-header: "X-Api-Version:v1,X-Api-Engine:Apisix"
    k8s.apisix.apache.org/rewrite-set-header: "X-Api-Custom:extended"
    k8s.apisix.apache.org/rewrite-remove-header: "X-Test"
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /sample
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		resp := s.NewAPISIXClient().GET("/sample/get").WithHeader("Host", "httpbin.org").WithHeader("X-Api-Custom", "Basic").WithHeader("X-Test", "Test").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Contains("\"X-Api-Version\": \"v1\"")
		resp.Body().Contains("\"X-Api-Engine\": \"Apisix\"")
		resp.Body().Contains("\"X-Api-Custom\": \"extended\"")
		resp.Body().NotContains("\"X-Test\"")
	})
	ginkgo.It("enable in ingress networking/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/rewrite-target-regex: "/sample/(.*)"
    k8s.apisix.apache.org/rewrite-target-regex-template: "/$1"
    k8s.apisix.apache.org/rewrite-add-header: "X-Api-Version:v1,X-Api-Engine:Apisix"
    k8s.apisix.apache.org/rewrite-set-header: "X-Api-Custom:extended"
    k8s.apisix.apache.org/rewrite-remove-header: "X-Test"
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /sample
        pathType: Prefix
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		resp := s.NewAPISIXClient().GET("/sample/get").WithHeader("Host", "httpbin.org").WithHeader("X-Api-Custom", "Basic").WithHeader("X-Test", "Test").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Contains("\"X-Api-Version\": \"v1\"")
		resp.Body().Contains("\"X-Api-Engine\": \"Apisix\"")
		resp.Body().Contains("\"X-Api-Custom\": \"extended\"")
		resp.Body().NotContains("\"X-Test\"")
	})
})
