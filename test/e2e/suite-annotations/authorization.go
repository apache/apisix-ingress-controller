// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

var _ = ginkgo.Describe("suite-annotations: authorization annotations", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("enable keyAuth in ingress networking/v1", func() {
			err := s.ApisixConsumerKeyAuthCreated("foo", "bar")
			assert.Nil(ginkgo.GinkgoT(), err, "creating keyAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/auth-type: "keyAuth"
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
			err = s.CreateResourceFromString(ing)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
			time.Sleep(5 * time.Second)

			msg401 := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing API key found in request")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("apikey", "bar").
				Expect().
				Status(http.StatusOK)
		})

		ginkgo.It("enable keyAuth in ingress networking/v1beta1", func() {
			err := s.ApisixConsumerKeyAuthCreated("foo", "bar")
			assert.Nil(ginkgo.GinkgoT(), err, "creating keyAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/auth-type: "keyAuth"
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
			err = s.CreateResourceFromString(ing)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
			time.Sleep(5 * time.Second)

			msg401 := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing API key found in request")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("apikey", "bar").
				Expect().
				Status(http.StatusOK)
		})

		ginkgo.It("enable keyAuth in ingress extensions/v1beta1", func() {
			err := s.ApisixConsumerKeyAuthCreated("foo", "bar")
			assert.Nil(ginkgo.GinkgoT(), err, "creating keyAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/auth-type: "keyAuth"
  name: ingress-extensions-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
			err = s.CreateResourceFromString(ing)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
			time.Sleep(5 * time.Second)

			msg401 := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing API key found in request")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("apikey", "bar").
				Expect().
				Status(http.StatusOK)
		})

		ginkgo.It("enable basicAuth in ingress networking/v1", func() {
			err := s.ApisixConsumerBasicAuthCreated("jack1", "jack1-username", "jack1-password")
			assert.Nil(ginkgo.GinkgoT(), err, "creating keyAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/auth-type: "basicAuth"
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPort[0])
			err = s.CreateResourceFromString(ing)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
			time.Sleep(5 * time.Second)

			msg401 := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing authorization in request")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazEtdXNlcm5hbWU6amFjazEtcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusOK)
		})

		ginkgo.It("enable basicAuth in ingress networking/v1beta1", func() {
			err := s.ApisixConsumerBasicAuthCreated("jack1", "jack1-username", "jack1-password")
			assert.Nil(ginkgo.GinkgoT(), err, "creating keyAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/auth-type: "basicAuth"
  name: ingress-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
			err = s.CreateResourceFromString(ing)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
			time.Sleep(5 * time.Second)

			msg401 := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing authorization in request")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazEtdXNlcm5hbWU6amFjazEtcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusOK)
		})

		ginkgo.It("enable basicAuth in ingress networking/v1beta1", func() {
			err := s.ApisixConsumerBasicAuthCreated("jack1", "jack1-username", "jack1-password")
			assert.Nil(ginkgo.GinkgoT(), err, "creating keyAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/auth-type: "basicAuth"
  name: ingress-extensions-v1beta1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          serviceName: %s
          servicePort: %d
`, backendSvc, backendPort[0])
			err = s.CreateResourceFromString(ing)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
			time.Sleep(5 * time.Second)

			msg401 := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing authorization in request")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazEtdXNlcm5hbWU6amFjazEtcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusOK)
		})
	}

	ginkgo.Describe("suite-annotations: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-annotations: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
