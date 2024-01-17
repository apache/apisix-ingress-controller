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
package plugins

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-other: ApisixPluginConfig", func() {
	s := scaffold.NewDefaultV2Scaffold()
	ginkgo.It("add crd from definition", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: echo-and-cors-apc
spec:
 plugins:
 - name: echo
   enable: true
   config:
    before_body: "This is the preface"
    after_body: "This is the epilogue"
    headers:
     X-Foo: v1
     X-Foo2: v2
 - name: cors
   enable: true
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		err := s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")

		time.Sleep(time.Second * 3)

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
      weight: 10
    plugin_config_name: echo-and-cors-apc
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		time.Sleep(3 * time.Second)
		pcs, err := s.ListApisixPluginConfig()
		assert.Nil(ginkgo.GinkgoT(), err, nil, "listing pluginConfigs")
		assert.Len(ginkgo.GinkgoT(), pcs, 1)
		assert.Len(ginkgo.GinkgoT(), pcs[0].Plugins, 2)

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Header("X-Foo").Equal("v1")
		resp.Header("X-Foo2").Equal("v2")
		resp.Header("Access-Control-Allow-Origin").Equal("*")
		resp.Header("Access-Control-Allow-Methods").Equal("*")
		resp.Header("Access-Control-Allow-Headers").Equal("*")
		resp.Header("Access-Control-Expose-Headers").Equal("*")
		resp.Header("Access-Control-Max-Age").Equal("5")
		resp.Body().Contains("This is the preface")
		resp.Body().Contains("origin")
		resp.Body().Contains("This is the epilogue")
	})

	ginkgo.It("ApisixPluginConfig replace body", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: test-apc-1
spec:
 plugins:
 - name: echo
   enable: true
   config:
    body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Equal("my custom body")
	})

	ginkgo.It("disable plugin", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: test-apc-1
spec:
 plugins:
 - name: echo
   enable: false
   config:
    body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		time.Sleep(6 * time.Second)
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Contains("origin")
		resp.Body().NotContains("my custom body")
	})

	ginkgo.It("enable plugin and then delete it", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: test-apc-1
spec:
 plugins:
 - name: echo
   enable: true
   config:
    body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugin_config_name: test-apc-1 
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Equal("my custom body")

		apc = fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: test-apc-1
spec:
 plugins:
 - name: echo
   enable: false
   config:
    body: "my custom body"
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().NotContains("my custom body")
		resp.Body().Contains("origin")
	})

	ginkgo.It("empty config", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: test-apc-1
spec:
 plugins:
 - name: cors
   enable: true
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Header("Access-Control-Allow-Origin").Equal("*")
		resp.Header("Access-Control-Allow-Methods").Equal("*")
		resp.Header("Access-Control-Allow-Headers").Equal("*")
		resp.Header("Access-Control-Expose-Headers").Equal("*")
		resp.Header("Access-Control-Max-Age").Equal("5")
		resp.Body().Contains("origin")
	})

	ginkgo.It("finer granularity config", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  plugins:
    - name: cors
      enable: true
      config:
        allow_origins: http://foo.bar.org
        allow_methods: "GET,POST"
        max_age: 3600
        expose_headers: x-foo,x-baz
        allow_headers: x-from-ingress
        allow_credential: true
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("Origin", "http://foo.bar.org").
			Expect()
		resp.Status(http.StatusOK)
		resp.Header("Access-Control-Allow-Origin").Equal("http://foo.bar.org")
		resp.Header("Access-Control-Allow-Methods").Equal("GET,POST")
		resp.Header("Access-Control-Allow-Headers").Equal("x-from-ingress")
		resp.Header("Access-Control-Expose-Headers").Equal("x-foo,x-baz")
		resp.Header("Access-Control-Max-Age").Equal("3600")
		resp.Header("Access-Control-Allow-Credentials").Equal("true")
		resp.Body().Contains("origin")

		resp = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("Origin", "http://foo.bar2.org").
			Expect()
		resp.Header("Access-Control-Allow-Methods").Empty()
		resp.Header("Access-Control-Allow-Headers").Empty()
		resp.Header("Access-Control-Expose-Headers").Empty()
		resp.Header("Access-Control-Max-Age").Empty()
		// httpbin set it by itself.
		// resp.Header("Access-Control-Allow-Credentials").Empty()
		resp.Body().Contains("origin")
	})

	ginkgo.It("disable plugin", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  plugins:
    - name: cors
      enable: false
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		// httpbin sets this header by itself.
		// resp.Header("Access-Control-Allow-Origin").Empty()
		resp.Header("Access-Control-Allow-Methods").Empty()
		resp.Header("Access-Control-Allow-Headers").Empty()
		resp.Header("Access-Control-Expose-Headers").Empty()
		resp.Header("Access-Control-Max-Age").Empty()
		resp.Body().Contains("origin")
	})

	ginkgo.It("enable plugin and then delete it", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  plugins:
  - name: cors
    enable: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugin_config_name: test-apc-1
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)

		resp.Header("Access-Control-Allow-Origin").Equal("*")
		resp.Header("Access-Control-Allow-Methods").Equal("*")
		resp.Header("Access-Control-Allow-Headers").Equal("*")
		resp.Header("Access-Control-Expose-Headers").Equal("*")
		resp.Header("Access-Control-Max-Age").Equal("5")
		resp.Body().Contains("origin")

		apc = `
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
  name: test-apc-1
spec:
  plugins:
  - name: cors
    enable: false
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		// EnsureNumApisixRoutesCreated cannot be used to ensure update Correctness.
		time.Sleep(6 * time.Second)

		resp = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)

		// httpbin sets this header by itself.
		// resp.Header("Access-Control-Allow-Origin").Empty()
		resp.Header("Access-Control-Allow-Methods").Empty()
		resp.Header("Access-Control-Allow-Headers").Empty()
		resp.Header("Access-Control-Expose-Headers").Empty()
		resp.Header("Access-Control-Max-Age").Empty()
		resp.Body().Contains("origin")
	})
	ginkgo.It("applies plugin config for route with upstream", func() {
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: httpbin-plugins
spec:
 plugins:
 - name: proxy-rewrite
   enable: true
   config:
     regex_uri:
     - ^/httpbin/(.*)
     - /$1			
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apc))

		au := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
 name: httpbin-upstream
spec:
 externalNodes:
 - type: Domain
   name: httpbin.org
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(au))

		ar := `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: httpbin-route-rule
   match:
    hosts:
    - httpbin.org
    paths:
    - /httpbin/*
    methods:
    - GET
   upstreams:
   - name: httpbin-upstream
   plugin_config_name: httpbin-plugins
`

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		time.Sleep(6 * time.Second)
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/httpbin/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
	})
})

var _ = ginkgo.Describe("suite-plugins-other: ApisixPluginConfig cross namespace", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		NamespaceSelectorLabel: map[string][]string{
			"apisix.ingress.watch": {"test"},
		},
	})
	ginkgo.It("ApisixPluginConfig cross namespace", func() {
		testns := `
apiVersion: v1
kind: Namespace
metadata:
  name: test
  labels:
    apisix.ingress.watch: test
`
		err := s.CreateResourceFromString(testns)
		assert.Nil(ginkgo.GinkgoT(), err, "Creating test namespace")
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixPluginConfig
metadata:
 name: echo-and-cors-apc
 namespace: test
spec:
 plugins:
 - name: echo
   enable: true
   config:
    before_body: "This is the preface"
    after_body: "This is the epilogue"
    headers:
     X-Foo: v1
     X-Foo2: v2
 - name: cors
   enable: true
`)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(apc, "test"))

		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")

		time.Sleep(time.Second * 3)

		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
      weight: 10
    plugin_config_name: echo-and-cors-apc
    plugin_config_namespace: test
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		time.Sleep(3 * time.Second)
		pcs, err := s.ListApisixPluginConfig()
		assert.Nil(ginkgo.GinkgoT(), err, nil, "listing pluginConfigs")
		assert.Len(ginkgo.GinkgoT(), pcs, 1)
		assert.Len(ginkgo.GinkgoT(), pcs[0].Plugins, 2)

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Header("X-Foo").Equal("v1")
		resp.Header("X-Foo2").Equal("v2")
		resp.Header("Access-Control-Allow-Origin").Equal("*")
		resp.Header("Access-Control-Allow-Methods").Equal("*")
		resp.Header("Access-Control-Allow-Headers").Equal("*")
		resp.Header("Access-Control-Expose-Headers").Equal("*")
		resp.Header("Access-Control-Max-Age").Equal("5")
		resp.Body().Contains("This is the preface")
		resp.Body().Contains("origin")
		resp.Body().Contains("This is the epilogue")
	})
})
