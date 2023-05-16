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
package chore

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var (
	_routeConfig = `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
      - /*
    backends:
    - serviceName: %s
      servicePort: %d
`
	_httpServiceConfig = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-service-e2e-test
spec:
  selector:
    app: httpbin-deployment-e2e-test
  ports:
    - name: %s
      port: %d
      protocol: TCP
      targetPort: %d
  type: ClusterIP
`
	_ingressV1Config = `
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
              name: %s
`
	_ingressV1beta1Config = `
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
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
`
	_ingressExtensionsV1beta1Config = `
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
`
)

var _ = ginkgo.Describe("suite-chore: Consistency between APISIX and the CRDs resource of the IngressController", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("ApisixRoute and APISIX of route and upstream", func() {
			httpService := fmt.Sprintf(_httpServiceConfig, "port1", 9080, 9080)
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpService))

			ar := fmt.Sprintf(_routeConfig, "httpbin-service-e2e-test", 9080)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

			upstreams, err := s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), upstreams, 1)
			assert.Contains(ginkgo.GinkgoT(), upstreams[0].Name, "httpbin-service-e2e-test_9080")
			// The correct httpbin pod port is 80
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusBadGateway)

			httpService = fmt.Sprintf(_httpServiceConfig, "port1", 80, 80)
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpService))

			ar = fmt.Sprintf(_routeConfig, "httpbin-service-e2e-test", 80)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(6 * time.Second)

			routes, err := s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), routes, 1)
			upstreams, err = s.ListApisixUpstreams()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), upstreams, 1)
			assert.Contains(ginkgo.GinkgoT(), upstreams[0].Name, "httpbin-service-e2e-test_80")

			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		})
	}

	ginkgo.Describe("suite-chore: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold())
	})
})

var _ = ginkgo.Describe("suite-chore: Consistency between APISIX and the Ingress resource of the IngressController", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("Ingress v1 and APISIX of route and upstream", func() {
		httpService := fmt.Sprintf(_httpServiceConfig, "port1", 9080, 9080)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpService))

		ing := fmt.Sprintf(_ingressV1Config, "httpbin-service-e2e-test", "port1")
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))

		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

		upstreams, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), upstreams, 1)
		assert.Contains(ginkgo.GinkgoT(), upstreams[0].Name, "httpbin-service-e2e-test_9080")
		// The correct httpbin pod port is 80
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusBadGateway)

		httpService = fmt.Sprintf(_httpServiceConfig, "port2", 80, 80)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpService))

		ing = fmt.Sprintf(_ingressV1Config, "httpbin-service-e2e-test", "port2")
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))

		time.Sleep(6 * time.Second)

		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 1)
		upstreams, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), upstreams, 1)
		assert.Contains(ginkgo.GinkgoT(), upstreams[0].Name, "httpbin-service-e2e-test_80")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("Ingress V1beta1 and APISIX of route and upstream", func() {
		httpService := fmt.Sprintf(_httpServiceConfig, "port1", 9080, 9080)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpService))

		ing := fmt.Sprintf(_ingressV1beta1Config, "httpbin-service-e2e-test", 9080)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))

		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

		upstreams, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), upstreams, 1)
		assert.Contains(ginkgo.GinkgoT(), upstreams[0].Name, "httpbin-service-e2e-test_9080")
		// The correct httpbin pod port is 80
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusBadGateway)

		httpService = fmt.Sprintf(_httpServiceConfig, "port2", 80, 80)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpService))

		ing = fmt.Sprintf(_ingressV1beta1Config, "httpbin-service-e2e-test", 80)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))

		time.Sleep(6 * time.Second)

		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 1)
		upstreams, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), upstreams, 1)
		assert.Contains(ginkgo.GinkgoT(), upstreams[0].Name, "httpbin-service-e2e-test_80")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})

	ginkgo.It("Ingress extensionsV1beta1 and APISIX of route and upstream", func() {
		httpService := fmt.Sprintf(_httpServiceConfig, "port1", 9080, 9080)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpService))

		ing := fmt.Sprintf(_ingressExtensionsV1beta1Config, "httpbin-service-e2e-test", 9080)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))

		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

		upstreams, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), upstreams, 1)
		assert.Contains(ginkgo.GinkgoT(), upstreams[0].Name, "httpbin-service-e2e-test_9080")
		// The correct httpbin pod port is 80
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusBadGateway)

		httpService = fmt.Sprintf(_httpServiceConfig, "port2", 80, 80)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(httpService))

		ing = fmt.Sprintf(_ingressExtensionsV1beta1Config, "httpbin-service-e2e-test", 80)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))

		time.Sleep(6 * time.Second)

		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 1)
		upstreams, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), upstreams, 1)
		assert.Contains(ginkgo.GinkgoT(), upstreams[0].Name, "httpbin-service-e2e-test_80")

		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
	})
})

var _ = ginkgo.Describe("suite-chore: apisix route labels sync", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.JustBeforeEach(func() {
			labels := map[string]string{"key": "value", "foo": "bar"}
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
 labels:
   key: value
   foo: bar
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")

			routes, _ := s.ListApisixRoutes()
			assert.Len(ginkgo.GinkgoT(), routes, 1)
			// check if labels exists
			for _, route := range routes {
				eq := reflect.DeepEqual(route.Metadata.Labels, labels)
				assert.True(ginkgo.GinkgoT(), eq)
			}
		})
	}

	ginkgo.Describe("suite-chore: scaffold v2", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "sync",
			IngressAPISIXReplicas: 1,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
		}))
	})
})

var _ = ginkgo.Describe("suite-chore: apisix plugin config labels sync", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.JustBeforeEach(func() {
			labels := map[string]string{"key": "value", "foo": "bar"}
			apc := `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: my-echo
 labels:
   key: value
   foo: bar
spec:
 plugins:
 - name: echo
   enable: true
   config:
    body: "my-echo"
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apc))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixPluginConfigCreated(1), "Checking number of plugin configs")

			pluginConfigs, _ := s.ListApisixPluginConfig()
			assert.Len(ginkgo.GinkgoT(), pluginConfigs, 1)
			// check if labels exists
			for _, pluginConfig := range pluginConfigs {
				eq := reflect.DeepEqual(pluginConfig.Metadata.Labels, labels)
				assert.True(ginkgo.GinkgoT(), eq)
			}
		})
	}

	ginkgo.Describe("suite-chore: scaffold v2", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "sync",
			IngressAPISIXReplicas: 1,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
		}))
	})
})

var _ = ginkgo.Describe("suite-chore: apisix upstream labels sync", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.JustBeforeEach(func() {
			labels := map[string]string{"key": "value", "foo": "bar"}
			au := `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: foo
  labels:
    foo: bar
	key: value
spec:
  timeout:
    read: 10s
    send: 10s
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(au))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			upstreams, _ := s.ListApisixUpstreams()
			assert.Len(ginkgo.GinkgoT(), upstreams, 1)
			// check if labels exists
			for _, route := range upstreams {
				eq := reflect.DeepEqual(route.Metadata.Labels, labels)
				assert.True(ginkgo.GinkgoT(), eq)
			}
		})
	}

	ginkgo.Describe("suite-chore: scaffold v2", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "sync",
			IngressAPISIXReplicas: 1,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
		}))
	})
})

var _ = ginkgo.Describe("suite-chore: apisix consumer labels sync", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.JustBeforeEach(func() {
			labels := map[string]string{"key": "value", "foo": "bar"}
			ac := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: foo
  labels:
    key: value
	foo: bar
spec:
  authParameter:
    jwtAuth:
      secretRef:
        name: jwt
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixConsumersCreated(1), "Checking number of consumers")

			consumers, _ := s.ListApisixConsumers()
			assert.Len(ginkgo.GinkgoT(), consumers, 1)
			// check if labels exists
			for _, route := range consumers {
				eq := reflect.DeepEqual(route.Labels, labels)
				assert.True(ginkgo.GinkgoT(), eq)
			}
		})
	}

	ginkgo.Describe("suite-chore: scaffold v2", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "sync",
			IngressAPISIXReplicas: 1,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
		}))
	})
})
