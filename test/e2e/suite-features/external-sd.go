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

package features

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: external service discovery", func() {

	PhaseCreateApisixRoute := func(s *scaffold.Scaffold, name, upstream string) {
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
        - /*
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: bar
    upstreams:
    - name: %s
`, name, upstream)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
	}

	PhaseCreateApisixUpstream := func(s *scaffold.Scaffold, name, discoveryType, serviceName string) {
		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  discovery:
    type: %s
    serviceName: %s
`, name, discoveryType, fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, s.Namespace()))
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(au))
	}

	PhaseValidateNoUpstreams := func(s *scaffold.Scaffold) {
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 0, "upstream count")
	}

	PhaseValidateNoRoutes := func(s *scaffold.Scaffold) {
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 0, "route count")
	}

	PhaseValidateFirstUpstream := func(s *scaffold.Scaffold, length int, serviceName, discoveryType string) string {
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, length, "upstream count")
		upstream := ups[0]
		assert.Equal(ginkgo.GinkgoT(), serviceName, upstream.ServiceName)
		assert.Equal(ginkgo.GinkgoT(), discoveryType, upstream.DiscoveryType)

		return upstream.ID
	}

	PhaseValidateRouteAccess := func(s *scaffold.Scaffold, upstreamId string) {
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 1, "route count")
		assert.Equal(ginkgo.GinkgoT(), upstreamId, routes[0].UpstreamId)

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "bar").
			Expect().
			Status(http.StatusOK)
	}

	//PhaseValidateRouteAccessCode := func(s *scaffold.Scaffold, upstreamId string, code int) {
	//routes, err := s.ListApisixRoutes()
	//assert.Nil(ginkgo.GinkgoT(), err)
	//assert.Len(ginkgo.GinkgoT(), routes, 1, "route count")
	//assert.Equal(ginkgo.GinkgoT(), upstreamId, routes[0].UpstreamId)

	//_ = s.NewAPISIXClient().GET("/ip").
	//WithHeader("Host", "httpbin.org").
	//WithHeader("X-Foo", "bar").
	//Expect().
	//Status(code)
	//}

	PhaseCreateHttpbin := func(s *scaffold.Scaffold, name string) string {
		_httpbinDeploymentTemplate := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  strategy:
    rollingUpdate:
      maxSurge: 50%%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: %s
    spec:
      terminationGracePeriodSeconds: 0
      containers:
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 2
            periodSeconds: 5
            successThreshold: 1
            tcpSocket:
              port: 80
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 2
            periodSeconds: 5
            successThreshold: 1
            tcpSocket:
              port: 80
            timeoutSeconds: 2
          image: "localhost:5000/kennethreitz/httpbin:dev"
          imagePullPolicy: IfNotPresent
          name: httpbin
          ports:
            - containerPort: 80
              name: "http"
              protocol: "TCP"
`, name, name, name)
		_httpService := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: %s
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
  type: ClusterIP
`, name, name)

		err := s.CreateResourceFromString(s.FormatRegistry(_httpbinDeploymentTemplate))
		assert.Nil(ginkgo.GinkgoT(), err, "create temp httpbin deployment")
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_httpService), "create temp httpbin service")

		return fmt.Sprintf("httpbin-temp.%s.svc.cluster.local", s.Namespace())
	}

	// Cases:
	// --- Basic Function ---
	// 1. ApisixRoute refers to ApisixUpstream, ApisixUpstream refers to service discovery
	// 2. ApisixRoute refers to ApisixUpstream and Backends, ApisixUpstream refers to service discovery
	// --- Update Cases ---
	// o 1. ApisixRoute refers to ApisixUpstream, but the ApisixUpstream is created later
	// --- Delete Cases ---
	// 1. ApisixRoute is deleted, the generated resources should be removed

	opts := &scaffold.Options{
		Name:                  "default",
		IngressAPISIXReplicas: 1,
		ApisixResourceVersion: config.ApisixV2,
	}

	adminVersion := os.Getenv("APISIX_ADMIN_API_VERSION")
	if adminVersion == "v3" {
		opts.APISIXConfigPath = "testdata/apisix-gw-config-v3-with-sd.yaml"
	} else {
		// fallback to v2
		opts.APISIXConfigPath = "testdata/apisix-gw-config-with-sd.yaml"
	}

	s := scaffold.NewScaffold(opts)

	ginkgo.Describe("basic function: ", func() {
		ginkgo.It("should be able to access through service discovery", func() {
			// -- Data preparation --
			fqdn := PhaseCreateHttpbin(s, "httpbin-temp")
			// After creating a Service, a record will be added in DNS.
			// We use it for service discovery
			PhaseCreateApisixUpstream(s, "httpbin-upstream", "dns", "httpbin-temp")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, fqdn, "dns")
			PhaseValidateRouteAccess(s, upstreamId)
		})
	})

	ginkgo.Describe("update function: ", func() {
		ginkgo.It("should be able to create the ApisixUpstream later", func() {
			// -- Data preparation --
			fqdn := PhaseCreateHttpbin(s, "httpbin-temp")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			PhaseValidateNoUpstreams(s)

			// -- Data Update --
			PhaseCreateApisixUpstream(s, "httpbin-upstream", "dns", "httpbin-temp")

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, fqdn, "dns")
			PhaseValidateRouteAccess(s, upstreamId)
		})

		ginkgo.It("should be able to create the target service later", func() {
			// -- Data preparation --
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			PhaseValidateNoUpstreams(s)
			PhaseCreateApisixUpstream(s, "httpbin-upstream", "dns", "httpbin-temp")

			// -- Data Update --
			fqdn := PhaseCreateHttpbin(s, "httpbin-temp")

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, fqdn, "dns")
			PhaseValidateRouteAccess(s, upstreamId)
		})
	})

	ginkgo.Describe("delete function: ", func() {
		ginkgo.It("should be able to delete resources", func() {
			// -- Data preparation --
			fqdn := PhaseCreateHttpbin(s, "httpbin-temp")
			PhaseCreateApisixUpstream(s, "httpbin-upstream", "dns", "httpbin-temp")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, fqdn, "dns")
			PhaseValidateRouteAccess(s, upstreamId)

			// -- delete --
			assert.Nil(ginkgo.GinkgoT(), s.DeleteResource("ar", "httpbin-route"), "delete route")
			assert.Nil(ginkgo.GinkgoT(), s.DeleteResource("au", "httpbin-upstream"), "delete upstream")
			time.Sleep(time.Second * 15)

			// -- validate --
			PhaseValidateNoRoutes(s)
			PhaseValidateNoUpstreams(s)
		})
	})

})
