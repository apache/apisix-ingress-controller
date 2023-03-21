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
package ingress

import (
	"fmt"
	"net/http"
	"time"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-cluster: ApisixClusterConfig with v2 and v2beta3 ", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("enable prometheus", func() {
			adminSvc, adminPort := s.ApisixAdminServiceAndPort()
			assert.Nil(ginkgo.GinkgoT(), s.NewApisixClusterConfig("default", true, true), "creating ApisixClusterConfig")

			defer func() {
				assert.Nil(ginkgo.GinkgoT(), s.DeleteApisixClusterConfig("default", true, true))
			}()

			// Wait until the ApisixClusterConfig create event was delivered.
			time.Sleep(3 * time.Second)

			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: default
spec:
  http:
  - name: public-api
    match:
      paths:
      - /apisix/prometheus/metrics
    backends:
    - serviceName: %s
      servicePort: %d
    plugins:
    - name: public-api
      enable: true
`, adminSvc, adminPort)

			err := s.CreateVersionedApisixResource(ar)
			assert.Nil(ginkgo.GinkgoT(), err, "creating ApisixRouteConfig")

			time.Sleep(3 * time.Second)

			grs, err := s.ListApisixGlobalRules()
			assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Equal(ginkgo.GinkgoT(), grs[0].ID, id.GenID("default"))
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			_, ok := grs[0].Plugins["prometheus"]
			assert.Equal(ginkgo.GinkgoT(), ok, true)

			resp := s.NewAPISIXClient().GET("/apisix/prometheus/metrics").Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("# HELP apisix_etcd_modify_indexes Etcd modify index for APISIX keys")
			resp.Body().Contains("# HELP apisix_etcd_reachable Config server etcd reachable from APISIX, 0 is unreachable")
			resp.Body().Contains("# HELP apisix_node_info Info of APISIX node")

			time.Sleep(3 * time.Second)

			if s.ApisixResourceVersion() != config.ApisixV2beta3 {
				resp1 := s.NewAPISIXClient().GET("/apisix/prometheus/metrics").Expect()
				resp1.Status(http.StatusOK)
				resp1.Body().Contains("public-api")
			}
		})
	}

	ginkgo.Describe("suite-features: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultV2beta3Scaffold)
	})
	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})

var _ = ginkgo.Describe("suite-cluster: Testin ApisixClusterConfig with IngressClass apisix", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		IngressClass:          "apisix",
	})

	ginkgo.It("ApisiClusterConfig should be ignored", func() {
		// create ApisixConsumer resource with ingressClassName: ignore
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: ignore
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 0)
	})

	ginkgo.It("ApisiClusterConfig should be handled", func() {
		// create ApisixConsumer resource without ingressClassName
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok := agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixConsumer resource with ingressClassName: apisix
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: apisix
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
	})
})

var _ = ginkgo.Describe("suite-cluster: Testing ApisixClusterConfig with IngressClass apisix-and-all", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		IngressClass:          "apisix-and-all",
	})

	ginkgo.It("ApisiClusterConfig should be handled", func() {
		// create ApisixConsumer resource without ingressClassName
		acc := `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok := agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixConsumer resource with ingressClassName: apisix
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: apisix
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		// update ApisixConsumer resource with ingressClassName: watch
		acc = `
apiVersion: apisix.apache.org/v2
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  ingressClassName: watch
  monitoring:
    prometheus:
      enable: true
      prefer_name: true
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(acc, ""))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
	})
})
