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
package main

import (
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/api7/ingress-controller/pkg/config"
	"github.com/api7/ingress-controller/test/e2e/scaffold"
)

func main() {
	//defer ginkgo.GinkgoRecover()
	// FIXME Remove these lines after we creating the first e2e test case.
	opts := &scaffold.Options{
		Name:               "sample",
		Kubeconfig:         "/Users/alex/.kube/config",
		ETCDImage:          "bitnami/etcd:3.4.14-debian-10-r0",
		IngressAPISIXImage: "viewking/apisix-ingress-controller:dev",
		APISIXImage:        "apache/apisix:latest",
		HTTPBINImage:       "kennethreitz/httpbin",
		IngressAPISIXConfig: &config.Config{
			LogLevel:   "info",
			LogOutput:  "stdout",
			HTTPListen: ":8080",
			APISIX: config.APISIXConfig{
				// We don't use FQDN since we don't know the namespace in advance.
				BaseURL: "https://apisix-service-e2e-test:9180/apisix",
			},
		},
		APISIXConfigPath:        "/Users/alex/Workstation/tokers/apisix-ingress-controller/test/e2e/testdata/apisix-gw-config.yaml",
		APISIXDefaultConfigPath: "/Users/alex/Workstation/tokers/apisix-ingress-controller/test/e2e/testdata/apisix-gw-config-default.yaml",
	}
	s := scaffold.NewScaffold(opts)
	assert.NotNil(ginkgo.GinkgoT(), s, "creating scaffold")
	s.BeforeEach()
	s.AfterEach()
}
