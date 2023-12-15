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

package ingress

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

type headers struct {
	Headers struct {
		Accept    string `json:"Accept"`
		Host      string `json:"Host"`
		UserAgent string `json:"User-Agent"`
	} `json:"headers"`
}

var _ = ginkgo.Describe("suite-ingress-features: namespacing filtering enable", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "enable-namespace-selector",
		IngressAPISIXReplicas: 1,
		ApisixResourceVersion: scaffold.ApisixResourceVersion().Default,
		NamespaceSelectorLabel: map[string]string{
			fmt.Sprintf("namespace-selector-%d", time.Now().Nanosecond()): "watch",
		},
		DisableNamespaceLabel: true,
	})

	ginkgo.Context("with namespace_selector", func() {
		namespace1 := fmt.Sprintf("namespace-selector-1-%d", time.Now().Nanosecond())
		namespace2 := fmt.Sprintf("namespace-selector-2-%d", time.Now().Nanosecond())

		createNamespaceLabel := func(namespace string) {
			k8s.CreateNamespaceWithMetadata(ginkgo.GinkgoT(), &k8s.KubectlOptions{ConfigPath: scaffold.GetKubeconfig()}, metav1.ObjectMeta{Name: namespace, Labels: s.NamespaceSelectorLabel()})
			_, err := s.NewHTTPBINWithNamespace(namespace)
			time.Sleep(6 * time.Second)
			assert.Nil(ginkgo.GinkgoT(), err, "create second httpbin service")
		}

		deleteNamespace := func(namespace string) {
			_ = k8s.DeleteNamespaceE(ginkgo.GinkgoT(), &k8s.KubectlOptions{ConfigPath: scaffold.GetKubeconfig()}, namespace)
		}

		ginkgo.It("resources in other namespaces should be ignored", func() {
			createNamespaceLabel(namespace1)
			defer deleteNamespace(namespace1)

			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			route1 := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: httpbin.com
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route1, namespace1), "creating ingress")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
			time.Sleep(time.Second * 6)
			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
			var placeholder ip
			err := json.Unmarshal([]byte(body), &placeholder)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")

			// Now create another ingress in default namespace.
			route2 := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /headers
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route2, "default"), "creating ingress")
			time.Sleep(6 * time.Second)
			routes, err := s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), routes, 1)
			_ = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)

			route3 := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: local.httpbin.org
    http:
      paths:
      - path: /headers
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route3), "creating ingress")
			time.Sleep(6 * time.Second)
			routes, err = s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), routes, 1)
			_ = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "local.httpbin.org").Expect().Status(http.StatusNotFound)

			// remove route1
			assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromStringWithNamespace(route1, namespace1), "delete ingress")
			time.Sleep(6 * time.Second)

			deleteNamespace(namespace1)
			time.Sleep(6 * time.Second)
			routes, err = s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), routes, 0)

			// restart ingress-controller
			s.RestartIngressControllerDeploy()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), routes, 0)

			createNamespaceLabel(namespace2)
			defer deleteNamespace(namespace2)
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route1, namespace2), "creating ingress")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
			_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)
			_ = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)
			_ = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "local.httpbin.org").Expect().Status(http.StatusNotFound)
		})
	})
})

var _ = ginkgo.Describe("suite-ingress-features: namespacing filtering disable", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                     "disable-namespace-selector",
		IngressAPISIXReplicas:    1,
		ApisixResourceVersion:    scaffold.ApisixResourceVersion().Default,
		DisableNamespaceSelector: true,
	})
	ginkgo.Context("without namespace_selector", func() {
		namespace := "second-httpbin-service-namespace"

		// create another http-bin service in a new namespace.
		ginkgo.BeforeEach(func() {
			k8s.CreateNamespace(ginkgo.GinkgoT(), &k8s.KubectlOptions{
				ConfigPath: scaffold.GetKubeconfig(),
			}, namespace)
			_, err := s.NewHTTPBINWithNamespace(namespace)
			assert.Nil(ginkgo.GinkgoT(), err, "create new httpbin service")
		})

		// clean this tmp namespace when test case is done.
		ginkgo.AfterEach(func() {
			err := k8s.DeleteNamespaceE(ginkgo.GinkgoT(), &k8s.KubectlOptions{ConfigPath: scaffold.GetKubeconfig()}, namespace)
			assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", namespace)
		})

		ginkgo.It("all resources will be watched", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			route := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-httpbin.com
spec:
  ingressClassName: apisix
  rules:
  - host: local.httpbin.org.host.only.734212
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d	  
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating first ingress")
			time.Sleep(3 * time.Second)

			// Now create another ingress in another namespace.
			backendSvc, backendSvcPort = s.DefaultHTTPBackend()
			route = fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-second-httpbin-service-namespace.httpbin.com
spec:
  ingressClassName: apisix
  rules:
  - host: second-httpbin-service-namespace.httpbin.com
    http:
      paths:
      - path: /headers
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route, namespace), "creating second ingress")

			// restart ingress-controller
			s.RestartIngressControllerDeploy()

			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "local.httpbin.org.host.only.734212").Expect().Status(http.StatusOK).Body().Raw()
			var placeholder ip
			err := json.Unmarshal([]byte(body), &placeholder)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")
			assert.NotEqual(ginkgo.GinkgoT(), ip{}, placeholder)
			body = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "second-httpbin-service-namespace.httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
			var headerResponse headers
			err = json.Unmarshal([]byte(body), &headerResponse)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling header")
			assert.NotEqual(ginkgo.GinkgoT(), headers{}, headerResponse)
		})
	})
})

var _ = ginkgo.Describe("suite-ingress-features: namespacing un-label", func() {
	labelName, labelValue := fmt.Sprintf("namespace-selector-%d", time.Now().Nanosecond()), "watch"
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "un-label",
		IngressAPISIXReplicas: 1,
		ApisixResourceVersion: scaffold.ApisixResourceVersion().Default,
		NamespaceSelectorLabel: map[string]string{
			labelName: labelValue,
		},
		DisableNamespaceLabel: true,
	})
	namespace1 := fmt.Sprintf("un-label-%d", time.Now().Nanosecond())

	ginkgo.It("un-label", func() {
		client := s.GetKubernetesClient()

		ns := fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s
  labels:
    %s: %s
`, namespace1, labelName, labelValue)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(ns, namespace1), "creating namespace")
		//defer s.DeleteResourceFromStringWithNamespace(ns, namespace1)
		_, err := s.NewHTTPBINWithNamespace(namespace1)
		assert.Nil(ginkgo.GinkgoT(), err, "create httpbin service in", namespace1)

		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		route1 := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: httpbin.com
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route1, namespace1), "creating ingress")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
		time.Sleep(time.Second * 6)
		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()

		assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromStringWithNamespace(route1, namespace1), "deleting ingress")

		time.Sleep(6 * time.Second)

		// un-label
		_, err = client.CoreV1().Namespaces().Update(
			context.Background(),
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   namespace1,
				Labels: map[string]string{},
			}},
			metav1.UpdateOptions{},
		)
		assert.Nil(ginkgo.GinkgoT(), err, "unlabel the namespace")
		time.Sleep(6 * time.Second)
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 0)

		route2 := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /headers
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route2), "creating ingress")
		time.Sleep(6 * time.Second)
		routes, err = s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 0)
		_ = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusNotFound)

		route3 := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: local.httpbin.org
    http:
      paths:
      - path: /headers
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route3, "default"), "creating ingress")
		time.Sleep(6 * time.Second)
		routes, err = s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 0)
		_ = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "local.httpbin.org").Expect().Status(http.StatusNotFound)

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route1, namespace1), "creating ingress")
		time.Sleep(time.Second * 6)
		routes, err = s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 0)
		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound)
	})
})

var _ = ginkgo.Describe("suite-ingress-features: namespacing from no-label to label", func() {
	labelName, labelValue := fmt.Sprintf("namespace-selector-%d", time.Now().Nanosecond()), "watch"
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "from-no-label-to-label",
		IngressAPISIXReplicas: 1,
		ApisixResourceVersion: scaffold.ApisixResourceVersion().Default,
		NamespaceSelectorLabel: map[string]string{
			labelName: labelValue,
		},
		DisableNamespaceLabel: true,
	})
	namespace1 := fmt.Sprintf("namespace-selector-%d", time.Now().Nanosecond())

	ginkgo.It("from no-label to label", func() {
		client := s.GetKubernetesClient()
		ns := fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s
`, namespace1)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(ns, namespace1), "creating namespace")
		//defer s.DeleteResourceFromStringWithNamespace(ns, namespace1)
		_, err := s.NewHTTPBINWithNamespace(namespace1)
		assert.Nil(ginkgo.GinkgoT(), err, "create httpbin service in", namespace1)

		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		route1 := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: httpbin.com
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route1, namespace1), "creating ingress")
		time.Sleep(time.Second * 6)
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 0)
		_ = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound)

		// label
		_, err = client.CoreV1().Namespaces().Update(
			context.Background(),
			&v1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: namespace1,
				Labels: map[string]string{
					labelName: labelValue,
				},
			}},
			metav1.UpdateOptions{},
		)
		assert.Nil(ginkgo.GinkgoT(), err, "label the namespace")
		time.Sleep(6 * time.Second)
		routes, err = s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 1)
		body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
		var placeholder ip
		err = json.Unmarshal([]byte(body), &placeholder)
		assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")
	})
})
