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
package apisix

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestApisixUpstreamCRD_Convert(t *testing.T) {
	assert := assert.New(t)

	// get yaml from string
	var crd configv1.ApisixUpstream
	bytes := []byte(upstreamYaml)
	if err := yaml.Unmarshal(bytes, &crd); err != nil {
		assert.Error(err)
	} else {
		au3 := &ApisixUpstreamBuilder{CRD: &crd, Ep: &EndpointRequestTest{}} // mock endpoints
		// convert
		if upstreams, err := au3.Convert(); err != nil {
			assert.Error(err)
		} else {
			// equals or deepCompare
			upstreamExpect := buildExpectUpstream()
			//upstreamsExpect := []*v1.Upstream{upstreamExpect}
			b := equals(upstreams[0], upstreamExpect)
			//b := reflect.DeepEqual(upstreams, []*v1.Upstream{upstreamExpect})
			if !b {
				assert.True(b, "convert upstream not expected")
				assert.Error(fmt.Errorf("convert upstream not expect"))
			}
			t.Log("[upstream convert] ok")
		}
	}
}

func equals(s, d *v1.Upstream) bool {
	if s.Name != d.Name || s.FullName != d.FullName || s.Group != d.Group {
		return false
	}

	if s.FromKind != d.FromKind || s.Type != d.Type || s.Key != d.Key || s.HashOn != d.HashOn {
		return false
	}

	return true
}

// mock BuildEps
type EndpointRequestTest struct{}

func (epr *EndpointRequestTest) BuildEps(ns, name string, port int) []v1.Node {
	nodes := make([]v1.Node, 0)
	return nodes
}

func buildExpectUpstream() *v1.Upstream {
	fullName := "cloud_httpserver_8080"
	LBType := "chash"
	HashOn := "header"
	Key := "hello_key"
	fromKind := "ApisixUpstream"
	group := ""
	upstreamExpect := &v1.Upstream{
		Group:           group,
		ResourceVersion: group,
		FullName:        fullName,
		Name:            fullName,
		Type:            LBType,
		HashOn:          HashOn,
		Key:             Key,
		FromKind:        fromKind,
	}
	return upstreamExpect
}

var upstreamYaml = `
kind: ApisixUpstream
apiVersion: apisix.apache.org/v1
metadata:
  name: httpserver
  namespace: cloud
spec:
  ports:
  - loadbalancer:
      hashOn: header
      key: hello_key
      type: chash
    port: 8080
`
