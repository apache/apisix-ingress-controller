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
	"fmt"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("ApisixRoute Testing", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("create and then scale upstream pods to 2 ", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(2), "scaling number of httpbin instances")
		assert.Nil(ginkgo.GinkgoT(), s.WaitAllHTTPBINPodsAvailable(), "waiting for all httpbin pods ready")
		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2, "upstreams nodes not expect")
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
	})

	ginkgo.It("create, update, then remove", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

		// update
		apisixRoute = fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: "barbaz"
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)

		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound)
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").WithHeader("X-Foo", "barbaz").Expect().Status(http.StatusOK)

		// remove
		assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(apisixRoute))
		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups, 0, "upstreams nodes not expect")

		body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound).Body().Raw()
		assert.Contains(ginkgo.GinkgoT(), body, "404 Route Not Found")
	})

	ginkgo.It("create, update, remove k8s service, remove ApisixRoute", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

		// update
		apisixRoute = fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: "barbaz"
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)

		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound)
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").WithHeader("X-Foo", "barbaz").Expect().Status(http.StatusOK)
		// remove k8s service first
		s.DeleteHTTPBINService()
		// remove
		assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(apisixRoute))
		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups, 0, "upstreams nodes not expect")

		body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound).Body().Raw()
		assert.Contains(ginkgo.GinkgoT(), body, "404 Route Not Found")
	})

	ginkgo.It("change route rule name", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "listing routes in APISIX")
		assert.Len(ginkgo.GinkgoT(), routes, 1)

		upstreams, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams in APISIX")
		assert.Len(ginkgo.GinkgoT(), upstreams, 1)

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK)

		apisixRoute = fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1_1
    match:
      hosts:
      - httpbin.com
      paths:
      - /headers
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

		newRoutes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "listing routes in APISIX")
		assert.Len(ginkgo.GinkgoT(), newRoutes, 1)
		newUpstreams, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams in APISIX")
		assert.Len(ginkgo.GinkgoT(), newUpstreams, 1)

		// Upstream doesn't change.
		assert.Equal(ginkgo.GinkgoT(), newUpstreams[0].ID, upstreams[0].ID)
		assert.Equal(ginkgo.GinkgoT(), newUpstreams[0].Name, upstreams[0].Name)

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().
			Status(http.StatusNotFound).
			Body().Contains("404 Route Not Found")

		s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.com").Expect().
			Status(http.StatusOK)
	})

	ginkgo.It("same route rule name between two ApisixRoute objects", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
---
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route-2
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /headers
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0], backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().
			Status(http.StatusOK).
			Body().
			Contains("origin")
		s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.com").Expect().
			Status(http.StatusOK).
			Body().
			Contains("headers").
			Contains("httpbin.com")
	})

	ginkgo.It("route priority", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    priority: 1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
  - name: rule2
    priority: 2
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: barbazbar
    backend:
      serviceName: %s
      servicePort: %d
    plugins:
    - name: request-id
      enable: true
`, backendSvc, backendSvcPort[0], backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

		// Hit rule1
		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Contains("origin")
		resp.Header("X-Request-Id").Empty()

		// Hit rule2
		resp = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").WithHeader("X-Foo", "barbazbar").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Contains("origin")
		resp.Header("X-Request-Id").NotEmpty()
	})

	ginkgo.It("verify route/upstream items", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    priority: 1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "listing routes")
		assert.Len(ginkgo.GinkgoT(), routes, 1)
		name := s.Namespace() + "_" + "httpbin-route" + "_" + "rule1"
		assert.Equal(ginkgo.GinkgoT(), routes[0].Name, name)
		assert.Equal(ginkgo.GinkgoT(), routes[0].Uris, []string{"/ip"})
		assert.Equal(ginkgo.GinkgoT(), routes[0].Hosts, []string{"httpbin.com"})
		assert.Equal(ginkgo.GinkgoT(), routes[0].Desc,
			"Created by apisix-ingress-controller, DO NOT modify it manually")
		assert.Equal(ginkgo.GinkgoT(), routes[0].Labels, map[string]string{
			"managed-by": "apisix-ingress-controller",
		})

		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Desc,
			"Created by apisix-ingress-controller, DO NOT modify it manually")
		assert.Equal(ginkgo.GinkgoT(), ups[0].Labels, map[string]string{
			"managed-by": "apisix-ingress-controller",
		})

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Contains("origin")

		resp = s.NewAPISIXClient().GET("/ip").Expect()
		resp.Status(http.StatusNotFound)
		resp.Body().Contains("404 Route Not Found")
	})

	ginkgo.It("service is referenced by two ApisixRoutes", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		ar1 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route-1
spec:
  http:
  - name: rule1
    priority: 1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		ar2 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route-2
spec:
  http:
  - name: rule1
    priority: 1
    match:
      hosts:
      - httpbin.com
      paths:
      - /status/200
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		err := s.CreateResourceFromString(ar1)
		assert.Nil(ginkgo.GinkgoT(), err)
		err = s.CreateResourceFromString(ar2)
		assert.Nil(ginkgo.GinkgoT(), err)

		time.Sleep(3 * time.Second)

		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "listing routes")
		assert.Len(ginkgo.GinkgoT(), routes, 2)

		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].ID, routes[0].UpstreamId)
		assert.Equal(ginkgo.GinkgoT(), ups[0].ID, routes[1].UpstreamId)

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Contains("origin")

		resp = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.com").Expect()
		resp.Status(http.StatusOK)

		// Delete ar1
		err = s.RemoveResourceByString(ar1)
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(3 * time.Second)

		routes, err = s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "listing routes")
		assert.Len(ginkgo.GinkgoT(), routes, 1)
		name := s.Namespace() + "_" + "httpbin-route-2" + "_" + "rule1"
		assert.Equal(ginkgo.GinkgoT(), routes[0].Name, name)

		// As httpbin service is referenced by ar2, the corresponding upstream still exists.
		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].ID, routes[0].UpstreamId)

		resp = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect()
		resp.Status(http.StatusNotFound)
		resp = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.com").Expect()
		resp.Status(http.StatusOK)

		// Delete ar2
		err = s.RemoveResourceByString(ar2)
		assert.Nil(ginkgo.GinkgoT(), err)
		time.Sleep(3 * time.Second)

		routes, err = s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err, "listing routes")
		assert.Len(ginkgo.GinkgoT(), routes, 0)

		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "listing upstreams")
		assert.Len(ginkgo.GinkgoT(), ups, 0)

		resp = s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.com").Expect()
		resp.Status(http.StatusNotFound)
	})
})
