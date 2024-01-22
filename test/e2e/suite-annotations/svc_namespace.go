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

	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-annotations: svc-namespace annotations reference service in the same namespace", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("networking/v1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/svc-namespace: "%s"
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
`, s.Namespace(), backendSvc, backendPort[0])
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

	ginkgo.It("networking/v1beta1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/svc-namespace: "%s"
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
`, s.Namespace(), backendSvc, backendPort[0])

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
})

var _ = ginkgo.Describe("suite-annotations: svc-namespace annotations cross-namespace reference service", func() {
	s := scaffold.NewDefaultScaffold()

	createNamespace := func(namespace string, watch string) {
		k8s.CreateNamespaceWithMetadata(ginkgo.GinkgoT(),
			&k8s.KubectlOptions{ConfigPath: scaffold.GetKubeconfig()},
			metav1.ObjectMeta{Name: namespace, Labels: map[string]string{
				"apisix.ingress.watch": watch,
			}})
	}

	deleteNamespace := func(namespace string) {
		_ = k8s.DeleteNamespaceE(ginkgo.GinkgoT(), &k8s.KubectlOptions{ConfigPath: scaffold.GetKubeconfig()}, namespace)
	}

	ginkgo.It("networking/v1beta1", func() {

		newNs := fmt.Sprintf("second-svc-namespace-%d", time.Now().Nanosecond())
		oldNs := s.Namespace()
		createNamespace(newNs, oldNs)
		defer deleteNamespace(newNs)

		backendSvc, backendPort := s.DefaultHTTPBackend()
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

		err := s.CreateResourceFromStringWithNamespace(ing, newNs)
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

	ginkgo.It("networking/v1", func() {
		newNs := fmt.Sprintf("second-svc-namespace-%d", time.Now().Nanosecond())
		oldNs := s.Namespace()
		createNamespace(newNs, oldNs)
		defer deleteNamespace(newNs)

		backendSvc, backendPort := s.DefaultHTTPBackend()
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
		err := s.CreateResourceFromStringWithNamespace(ing, newNs)
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
})
