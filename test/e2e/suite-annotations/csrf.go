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

var _ = ginkgo.Describe("suite-annotations: csrf annotations", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("enable csrf in ingress networking/v1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/enable-csrf: "true"
    k8s.apisix.apache.org/csrf-key: "foo-key"
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /*
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing), "creating ingress")

		time.Sleep(5 * time.Second)

		msg401 := s.NewAPISIXClient().
			POST("/anything").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusUnauthorized).
			Body().
			Raw()
		assert.Contains(ginkgo.GinkgoT(), msg401, "no csrf token in headers")

		resp := s.NewAPISIXClient().
			GET("/anything").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)
		resp.Header("Set-Cookie").NotEmpty()

		cookie := resp.Cookie("apisix-csrf-token")
		token := cookie.Value().Raw()

		_ = s.NewAPISIXClient().
			POST("/anything").
			WithHeader("Host", "httpbin.org").
			WithHeader("apisix-csrf-token", token).
			WithCookie("apisix-csrf-token", token).
			Expect().
			Status(http.StatusOK)

	})

	ginkgo.It("enable csrf in ingress networking/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/enable-csrf: "true"
    k8s.apisix.apache.org/csrf-key: "foo-key"
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /*
        pathType: Prefix
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing), "creating ingress")

		time.Sleep(5 * time.Second)

		msg401 := s.NewAPISIXClient().
			POST("/anything").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusUnauthorized).
			Body().
			Raw()
		assert.Contains(ginkgo.GinkgoT(), msg401, "no csrf token in headers")

		resp := s.NewAPISIXClient().
			GET("/anything").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)
		resp.Header("Set-Cookie").NotEmpty()

		cookie := resp.Cookie("apisix-csrf-token")
		token := cookie.Value().Raw()

		_ = s.NewAPISIXClient().
			POST("/anything").
			WithHeader("Host", "httpbin.org").
			WithHeader("apisix-csrf-token", token).
			WithCookie("apisix-csrf-token", token).
			Expect().
			Status(http.StatusOK)
	})
})
