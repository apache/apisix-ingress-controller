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
	"time"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress-features: Testing ApisixClusterConfig with IngressClass", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		IngressClass:          "apisix",
	})

	ginkgo.It("ApisixClusterConfig", func() {
		// === should be ignored ===

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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 0)

		assert.Nil(ginkgo.GinkgoT(), s.DeleteApisixClusterConfig("default", true, true), "delete apisix cluster config")

		// === should be handled ===
		// create ApisixConsumer resource without ingressClassName
		acc = `
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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		assert.Nil(ginkgo.GinkgoT(), s.DeleteApisixClusterConfig("default", true, true), "delete apisix cluster config")
	})
})

var _ = ginkgo.Describe("suite-ingress-features: Testing ApisixClusterConfig with IngressClass apisix-and-all", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		Name:                  "ingress-class",
		IngressAPISIXReplicas: 1,
		IngressClass:          "apisix-and-all",
	})

	ginkgo.It("ApisixClusterConfig should be handled", func() {
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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(acc))
		time.Sleep(6 * time.Second)

		agrs, err = s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), agrs, 1)
		assert.Equal(ginkgo.GinkgoT(), agrs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), agrs[0].Plugins, 1)
		_, ok = agrs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		assert.Nil(ginkgo.GinkgoT(), s.DeleteApisixClusterConfig("default", true, true), "delete apisix cluster config")
	})
})
