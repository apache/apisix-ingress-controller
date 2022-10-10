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

var _ = ginkgo.Describe("suite-annotations: svc-namespace annotations", func() {

	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("same namespace in ingress networking/v1", func() {
			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
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
			err := s.CreateResourceFromString(ing)
			if err != nil {
				assert.Fail(ginkgo.GinkgoT(), err.Error(), "creating ingress")
			}

			time.Sleep(5 * time.Second)

			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})

		ginkgo.It("different namespace in ingress networking/v1", func() {
			backendSvc, backendPort := s.DefaultHTTPBackend()
			oldNs := s.Namespace()
			newNs := oldNs + "-new"
			ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/svc-namespace: %s
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
`, oldNs, backendSvc, backendPort[0])
			s.UpdateNamespace(newNs)
			err := s.CreateResourceFromString(ing)
			if err != nil {
				assert.Fail(ginkgo.GinkgoT(), err.Error(), "creating ingress")
			}
			s.UpdateNamespace(oldNs)

			time.Sleep(5 * time.Second)

			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})

		ginkgo.It("same namespace in ingress networking/v1beta1", func() {
			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/svc-namespace: ""
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

			err := s.CreateResourceFromString(ing)
			if err != nil {
				assert.Fail(ginkgo.GinkgoT(), err.Error(), "creating ingress")
			}

			time.Sleep(5 * time.Second)

			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})

		ginkgo.It("different namespace in ingress networking/v1beta1", func() {
			backendSvc, backendPort := s.DefaultHTTPBackend()
			oldNs := s.Namespace()
			newNs := oldNs + "-new"
			ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/svc-namespace: %s
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
`, oldNs, backendSvc, backendPort[0])

			s.UpdateNamespace(newNs)
			err := s.CreateResourceFromString(ing)
			if err != nil {
				assert.Fail(ginkgo.GinkgoT(), err.Error(), "creating ingress")
			}
			s.UpdateNamespace(oldNs)

			time.Sleep(5 * time.Second)

			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})

		ginkgo.It("same namespace in ingress extensions/v1beta1", func() {
			backendSvc, backendPort := s.DefaultHTTPBackend()
			ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix	
  name: ingress-extensions-v1beta1
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
			err := s.CreateResourceFromString(ing)
			if err != nil {
				assert.Fail(ginkgo.GinkgoT(), err.Error(), "creating ingress")
			}
			time.Sleep(5 * time.Second)

			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})

		ginkgo.It("different namespace in ingress extensions/v1beta1", func() {
			backendSvc, backendPort := s.DefaultHTTPBackend()
			oldNs := s.Namespace()
			newNs := oldNs + "-new"
			ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/svc-namespace: %s
  name: ingress-extensions-v1beta1
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
`, oldNs, backendSvc, backendPort[0])
			s.UpdateNamespace(newNs)
			err := s.CreateResourceFromString(ing)
			if err != nil {
				assert.Fail(ginkgo.GinkgoT(), err.Error(), "creating ingress")
			}
			s.UpdateNamespace(oldNs)
			time.Sleep(5 * time.Second)

			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})

	}

	ginkgo.Describe("suite-annotations: scaffold v2beta3", func() {
		s := scaffold.NewDefaultV2beta3Scaffold()
		// k8s.CreateNamespace(ginkgo.GinkgoT(), &k8s.KubectlOptions{
		// 	ConfigPath: scaffold.GetKubeconfig(),
		// }, s.Namespace()+"-new")
		suites(s)
		// k8s.DeleteNamespace(ginkgo.GinkgoT(), &k8s.KubectlOptions{
		// 	ConfigPath: scaffold.GetKubeconfig(),
		// }, s.Namespace()+"-new")
	})

	ginkgo.Describe("suite-annotations: scaffold v2", func() {
		s := scaffold.NewDefaultV2Scaffold()
		// k8s.CreateNamespace(ginkgo.GinkgoT(), &k8s.KubectlOptions{
		// 	ConfigPath: scaffold.GetKubeconfig(),
		// }, s.Namespace()+"-new")
		suites(s)
		// k8s.DeleteNamespace(ginkgo.GinkgoT(), &k8s.KubectlOptions{
		// 	ConfigPath: scaffold.GetKubeconfig(),
		// }, s.Namespace()+"-new")
	})

})
