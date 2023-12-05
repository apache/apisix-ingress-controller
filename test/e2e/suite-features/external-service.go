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
	"time"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: external services", func() {
	PhaseCreateExternalService := func(s *scaffold.Scaffold, name, externalName string) {
		extService := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  type: ExternalName
  externalName: %s
`, name, externalName)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(extService))
	}
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

	PhaseCreateApisixRouteWithHostRewrite := func(s *scaffold.Scaffold, name, upstream, rewriteHost string) {
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
    plugins:
    - name: proxy-rewrite
      enable: true
      config:
        host: %s
`, name, upstream, rewriteHost)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
	}

	PhaseCreateApisixUpstream := func(s *scaffold.Scaffold, name string, nodeType v2.ApisixUpstreamExternalType, nodeName string) {
		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  externalNodes:
  - type: %s
    name: %s
`, name, nodeType, nodeName)
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

	PhaseValidateFirstUpstream := func(s *scaffold.Scaffold, length int, node string, port, weight int) string {
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, length, "upstream count")
		upstream := ups[0]
		assert.Len(ginkgo.GinkgoT(), upstream.Nodes, 1)
		assert.Equal(ginkgo.GinkgoT(), node, upstream.Nodes[0].Host)
		assert.Equal(ginkgo.GinkgoT(), port, upstream.Nodes[0].Port)
		assert.Equal(ginkgo.GinkgoT(), weight, upstream.Nodes[0].Weight)

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

	// PhaseValidateRouteAccessCode := func(s *scaffold.Scaffold, upstreamId string, code int) {
	// 	routes, err := s.ListApisixRoutes()
	// 	assert.Nil(ginkgo.GinkgoT(), err)
	// 	assert.Len(ginkgo.GinkgoT(), routes, 1, "route count")
	// 	assert.Equal(ginkgo.GinkgoT(), upstreamId, routes[0].UpstreamId)

	// 	_ = s.NewAPISIXClient().GET("/ip").
	// 		WithHeader("Host", "httpbin.org").
	// 		WithHeader("X-Foo", "bar").
	// 		Expect().
	// 		Status(code)
	// }

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
          image: "localhost:5000/httpbin:dev"
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
	// 1. ApisixRoute refers to ApisixUpstream, ApisixUpstream refers to third-party service
	// 2. ApisixRoute refers to ApisixUpstream, ApisixUpstream refers to ExternalName service
	// 3. ApisixRoute refers to ApisixUpstream, ApisixUpstream refers to multiple third-party or ExternalName services
	// 4. ApisixRoute refers to ApisixUpstream and Backends, ApisixUpstream refers to ExternalName service
	// --- Update Cases ---
	// o 1. ApisixRoute refers to ApisixUpstream, but the ApisixUpstream is created later
	// o 2. ApisixRoute refers to ApisixUpstream, but the ExternalName service is created later
	// o 3. ApisixRoute refers to ApisixUpstream, but the ApisixUpstream is updated and change to another ExternalName service
	// o 4. ApisixRoute refers to ApisixUpstream, the ApisixUpstream doesn't change, but the ExternalName service itself is updated
	// --- Delete Cases ---
	// 1. ApisixRoute is deleted, the generated resources should be removed

	s := scaffold.NewDefaultV2Scaffold()

	ginkgo.Describe("basic function: ", func() {
		ginkgo.It("should be able to access third-party service", func() {
			// -- Data preparation --
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeDomain, "httpbin.org")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			time.Sleep(time.Second * 6)

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, "httpbin.org", 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)
		})
		ginkgo.It("should be able to access third-party service with plugins", func() {
			// -- Data preparation --
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeDomain, "httpbin.org")

			// -- update --
			PhaseCreateApisixRouteWithHostRewrite(s, "httpbin-route", "httpbin-upstream", "httpbin.org")
			time.Sleep(time.Second * 6)

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, "httpbin.org", 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)
		})
		ginkgo.It("should be able to access external domain ExternalName service", func() {
			// -- Data preparation --
			PhaseCreateExternalService(s, "ext-httpbin", "httpbin.org")
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeService, "ext-httpbin")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			time.Sleep(time.Second * 6)

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, "httpbin.org", 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)
		})
		ginkgo.It("should be able to access in-cluster ExternalName service", func() {
			// -- Data preparation --
			fqdn := PhaseCreateHttpbin(s, "httpbin-temp")
			time.Sleep(time.Second * 10)

			// We are only testing the functionality of the external service and do not care which namespace the service is in.
			// The namespace of the external service should be watched.
			PhaseCreateExternalService(s, "ext-httpbin", fqdn)
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeService, "ext-httpbin")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			time.Sleep(time.Second * 6)

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, fqdn, 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)
		})
	})
	ginkgo.Describe("complex usage: ", func() {
		PhaseCreateApisixUpstreamWithMultipleExternalNodes := func(s *scaffold.Scaffold, name string,
			nodeTypeA v2.ApisixUpstreamExternalType, nodeNameA string, nodeTypeB v2.ApisixUpstreamExternalType, nodeNameB string) {
			au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: %s
spec:
  externalNodes:
  - type: %s
    name: %s
  - type: %s
    name: %s
`, name, nodeTypeA, nodeNameA, nodeTypeB, nodeNameB)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(au))
		}

		PhaseCreateApisixRouteWithHostRewriteAndBackend := func(s *scaffold.Scaffold, name, upstream, hostRewrite, serviceName string, servicePort int) {
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
    backends:
    - serviceName: %s
      servicePort: %d
      resolveGranularity: service
    plugins:
    - name: proxy-rewrite
      enable: true
      config:
        host: %s
`, name, upstream, serviceName, servicePort, hostRewrite)

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
		}

		validateHttpbinAndPostmanAreAccessed := func() {
			hasEtag := false   // postman-echo.com
			hasNoEtag := false // httpbin.org
			for i := 0; i < 20; i++ {
				headers := s.NewAPISIXClient().GET("/ip").
					WithHeader("Host", "httpbin.org").
					WithHeader("X-Foo", "bar").
					Expect().
					Status(http.StatusOK).
					Headers().Raw()
				if _, ok := headers["Etag"]; ok {
					hasEtag = true
				} else {
					hasNoEtag = true
				}
				if hasEtag && hasNoEtag {
					break
				}
			}

			assert.True(ginkgo.GinkgoT(), hasEtag && hasNoEtag, "both httpbin and postman should be accessed at least once")
		}

		type validateFactor struct {
			port   int
			weight int
		}
		// Note: expected nodes has unique host
		PhaseValidateMultipleNodes := func(s *scaffold.Scaffold, length int, nodes map[string]*validateFactor) {
			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, 1, "upstream count")

			upstream := ups[0]
			assert.Len(ginkgo.GinkgoT(), upstream.Nodes, length)
			for _, node := range upstream.Nodes {
				host := node.Host
				if factor, ok := nodes[host]; ok {
					assert.Equal(ginkgo.GinkgoT(), factor.port, node.Port)
					assert.Equal(ginkgo.GinkgoT(), factor.weight, node.Weight)
				} else {
					err := fmt.Errorf("host %s appear but it shouldn't", host)
					assert.Nil(ginkgo.GinkgoT(), err)
				}
			}

			routes, err := s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), routes, 1, "route count")
			assert.Equal(ginkgo.GinkgoT(), ups[0].ID, routes[0].UpstreamId)

			validateHttpbinAndPostmanAreAccessed()
		}

		// Note: expected nodes has unique host
		PhaseValidateTrafficSplit := func(s *scaffold.Scaffold, length int, upstreamId string, nodes map[string]*validateFactor) {
			ups, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), ups, length, "upstream count")

			for _, upstream := range ups {
				assert.Len(ginkgo.GinkgoT(), upstream.Nodes, 1)
				host := upstream.Nodes[0].Host
				if factor, ok := nodes[host]; ok {
					assert.Equal(ginkgo.GinkgoT(), factor.port, upstream.Nodes[0].Port)
					assert.Equal(ginkgo.GinkgoT(), factor.weight, upstream.Nodes[0].Weight)
				} else {
					err := fmt.Errorf("host %s appear but it shouldn't", host)
					assert.Nil(ginkgo.GinkgoT(), err)
				}
			}

			routes, err := s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), routes, 1, "route count")
			assert.Equal(ginkgo.GinkgoT(), upstreamId, routes[0].UpstreamId)

			validateHttpbinAndPostmanAreAccessed()
		}

		ginkgo.It("should be able to access multiple external services", func() {
			// -- Data preparation --
			PhaseCreateApisixUpstreamWithMultipleExternalNodes(s, "httpbin-upstream",
				v2.ExternalTypeDomain, "httpbin.org", v2.ExternalTypeDomain, "postman-echo.com")
			PhaseCreateApisixRouteWithHostRewrite(s, "httpbin-route", "httpbin-upstream", "postman-echo.com")
			time.Sleep(time.Second * 6)

			// -- validation --
			PhaseValidateMultipleNodes(s, 2, map[string]*validateFactor{
				"httpbin.org": {
					port:   80,
					weight: translation.DefaultWeight,
				},
				"postman-echo.com": {
					port:   80,
					weight: translation.DefaultWeight,
				},
			})
		})
		ginkgo.It("should be able to use backends and upstreams together", func() {
			// -- Data preparation --
			PhaseCreateHttpbin(s, "httpbin-temp")
			time.Sleep(time.Second * 10)
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeDomain, "postman-echo.com")
			PhaseCreateApisixRouteWithHostRewriteAndBackend(s, "httpbin-route", "httpbin-upstream", "postman-echo.com", "httpbin-temp", 80)
			time.Sleep(time.Second * 6)

			svc, err := s.GetServiceByName("httpbin-temp")
			assert.Nil(ginkgo.GinkgoT(), err, "get httpbin service")
			ip := svc.Spec.ClusterIP

			upName := apisixv1.ComposeUpstreamName(s.Namespace(), "httpbin-temp", "", 80, types.ResolveGranularity.Service)
			upID := id.GenID(upName)

			// -- validation --
			PhaseValidateTrafficSplit(s, 2, upID, map[string]*validateFactor{
				ip: {
					port:   80,
					weight: translation.DefaultWeight,
				},
				"postman-echo.com": {
					port:   80,
					weight: translation.DefaultWeight,
				},
			})
		})
	})
	ginkgo.Describe("update function: ", func() {
		ginkgo.It("should be able to create the ApisixUpstream later", func() {
			// -- Data preparation --
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			time.Sleep(time.Second * 6)
			PhaseValidateNoUpstreams(s)

			// -- Data Update --
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeDomain, "httpbin.org")
			time.Sleep(time.Second * 6)

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, "httpbin.org", 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)
		})
		ginkgo.It("should be able to create the ExternalName service later", func() {
			// -- Data preparation --
			fqdn := PhaseCreateHttpbin(s, "httpbin-temp")
			time.Sleep(time.Second * 10)
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeService, "ext-httpbin")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			time.Sleep(time.Second * 6)
			PhaseValidateNoUpstreams(s)

			// -- Data update --
			PhaseCreateExternalService(s, "ext-httpbin", fqdn)
			time.Sleep(time.Second * 6)
			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, fqdn, 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)
		})
		ginkgo.It("should be able to update the ApisixUpstream later", func() {
			// -- Data preparation --
			fqdn := PhaseCreateHttpbin(s, "httpbin-temp")
			time.Sleep(time.Second * 10)
			PhaseCreateExternalService(s, "ext-httpbin", fqdn)
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeService, "doesnt-exist")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			time.Sleep(time.Second * 6)
			PhaseValidateNoUpstreams(s)

			// -- Data update --
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeService, "ext-httpbin")
			time.Sleep(time.Second * 6)

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, fqdn, 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)
		})
		ginkgo.It("should be able to update the ExternalName service later", func() {
			// -- Data preparation --
			PhaseCreateExternalService(s, "ext-httpbin", "unknown.org")
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeService, "ext-httpbin")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			time.Sleep(time.Second * 6)
			PhaseValidateFirstUpstream(s, 1, "unknown.org", 80, translation.DefaultWeight)

			// -- Data update --
			PhaseCreateExternalService(s, "ext-httpbin", "httpbin.org")
			time.Sleep(time.Second * 6)

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, "httpbin.org", 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)
		})
	})
	ginkgo.Describe("delete function: ", func() {
		ginkgo.It("should be able to delete resources", func() {
			// -- Data preparation --
			PhaseCreateApisixUpstream(s, "httpbin-upstream", v2.ExternalTypeDomain, "httpbin.org")
			PhaseCreateApisixRoute(s, "httpbin-route", "httpbin-upstream")
			time.Sleep(time.Second * 6)

			// -- validation --
			upstreamId := PhaseValidateFirstUpstream(s, 1, "httpbin.org", 80, translation.DefaultWeight)
			PhaseValidateRouteAccess(s, upstreamId)

			// -- delete --
			assert.Nil(ginkgo.GinkgoT(), s.DeleteResource("ar", "httpbin-route"), "delete route")
			assert.Nil(ginkgo.GinkgoT(), s.DeleteResource("au", "httpbin-upstream"), "delete upstream")
			time.Sleep(time.Second * 6)

			// -- validate --
			PhaseValidateNoRoutes(s)
			PhaseValidateNoUpstreams(s)
		})
	})
})
