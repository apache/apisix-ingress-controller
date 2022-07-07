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

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
)

func _createAPC(s *scaffold.Scaffold) {
	apc := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
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
	err := s.CreateResourceFromString(apc)
	assert.Nil(ginkgo.GinkgoT(), err)
	err = s.EnsureNumApisixPluginConfigCreated(1)
	assert.Nil(ginkgo.GinkgoT(), err, "Checking number of ApisixPluginConfig")
	time.Sleep(time.Second * 3)
}

func _assert(s *scaffold.Scaffold, ing string) {
	assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))

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
}

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1 with ApisixPluginConfig", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta3",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("networking/v1", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		_createAPC(s)
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-v1
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/plugin-config-name: echo-and-cors-apc
  name: ingress-v1
spec:
  rules:
  - host: httpbin.org
    http:
      paths:
      - path: /ip
        pathType: ImplementationSpecific
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendPorts[0])
		_assert(s, ing)
	})
})

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1beta1 with ApisixPluginConfig", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta3",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("networking/v1beta1", func() {
		_createAPC(s)

		backendSvc, backendPorts := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: ingress-v1beta1
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/plugin-config-name: echo-and-cors-apc
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
`, backendSvc, backendPorts[0])
		_assert(s, ing)
	})
})

var _ = ginkgo.Describe("suite-annotations: annotations.extensions/v1beta1 with ApisixPluginConfig", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta3",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("extensions/v1beta1", func() {
		_createAPC(s)

		backendSvc, backendPorts := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ingress-ext-v1beta1
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/plugin-config-name: echo-and-cors-apc
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
`, backendSvc, backendPorts[0])
		_assert(s, ing)
	})
})
