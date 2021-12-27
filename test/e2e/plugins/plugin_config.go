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
package plugins

import (
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"time"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("ApisixPluginConfig", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta3",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("add ApisixPluginConfig from definition", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixPluginConfig
metadata:
 name: %s
spec:
 plugins:
 - before_body: "This is the preface222"
   after_body: "This is the epilogue3333"
   headers:
     X-Foo: v1
     X-Foo2: v2
`, backendSvc)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apc))

		time.Sleep(time.Second * 3)

		ar := fmt.Sprintf(`
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
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: echo
     enable: true
     config:
       before_body: "This is the preface"
       after_body: "This is the epilogue"
       headers:
         X-Foo: v1
         X-Foo2: v2 
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixPluginConfigCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")

		time.Sleep(3 * time.Second)
		pcs, err := s.ListApisixPluginConfig()
		assert.Nil(ginkgo.GinkgoT(), err, nil, "listing pluginConfigs")
		assert.Len(ginkgo.GinkgoT(), pcs, 1)
		assert.Len(ginkgo.GinkgoT(), pcs[0].Plugins, 1)

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Header("X-Foo").Equal("v1")
		resp.Header("X-Foo2").Equal("v2")
		resp.Body().Contains("This is the preface")
		resp.Body().Contains("origin")
		resp.Body().Contains("This is the epilogue")
	})

	ginkgo.It("disable plugin", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		ar := fmt.Sprintf(`
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
       - /status/*
   backends:
   - serviceName: %s
     servicePort: %d
     resolveGranularity: service
   plugins:
   - name: api-breaker
     enable: false
     config:
       break_response_code: 502
       unhealthy:
         http_statuses:
         - 505
         failures: 2
       max_breaker_sec: 3
       healthy:
         http_statuses:
         - 200
         successes: 2
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixPluginConfigCreated(0)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of pluginConfigs")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/status/200").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)

		for i := 0; i < 2; i++ {
			resp = s.NewAPISIXClient().GET("/status/505").WithHeader("Host", "httpbin.org").Expect()
			resp.Status(505)
		}

		// Trigger the api-breaker threshold
		resp = s.NewAPISIXClient().GET("/status/505").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(505)
	})
})
