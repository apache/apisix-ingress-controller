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
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins: serverless plugin", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta3",
	}
	s := scaffold.NewScaffold(opts)

	ginkgo.JustBeforeEach(func() {
		json := `{
			"uri":"/auth",
			"plugins":{
				"serverless-pre-function":{
					"phase":"rewrite",
					"functions":[
						"return function (conf, ctx)\n    local core = require(\"apisix.core\");\n    local authorization = core.request.header(ctx, \"Authorization\");\n    if authorization == \"123\" then\n        core.response.exit(200);\n    elseif authorization == \"321\" then\n        core.response.set_header(\"X-User-ID\", \"i-am-user\");\n        core.response.exit(200);\n    else core.response.set_header(\"Location\", \"http://example.com/auth\");\n        core.response.exit(403);\n    end\nend"
					]
				}
			}
		}`
		assert.Nil(ginkgo.GinkgoT(), s.CreateApisixRouteByChever("serverless", []byte(json)), "create serverless route")
	})

	ginkgo.JustAfterEach(func() {
		assert.Nil(ginkgo.GinkgoT(), s.DeleteApisixRouteByChever("serverless"), "clean up serverless route")
	})

	ginkgo.It("enable in ingress networking/v1", func() {
		backendSvc, backendPort := s.DefaultHTTPBackend()
		ing := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/auth-uri: %s
    k8s.apisix.apache.org/auth-request-headers: Authorization
    k8s.apisix.apache.org/auth-upstream-headers: X-User-ID
    k8s.apisix.apache.org/auth-client-headers: Location
  name: ingress-v1
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
`, "http://127.0.0.1:9080/serverless", backendSvc, backendPort[0])
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)

		resp := s.NewAPISIXClient().GET("/serverless").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)

	})
})
