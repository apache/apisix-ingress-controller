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

package translation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestTranslateApisixUpstreamExternalNodesDomainType(t *testing.T) {
	tr := &translator{}
	defaultPort := 80
	defaultWeight := 80
	specifiedPort := 8080
	testCases := map[*v2.ApisixUpstream][]apisixv1.UpstreamNode{
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name:   "domain.foobar.com",
					Type:   v2.ExternalTypeDomain,
					Weight: &defaultWeight,
					Port:   &defaultPort,
				}},
			},
		}: {{
			Host:   "domain.foobar.com",
			Port:   defaultPort,
			Weight: defaultWeight,
		}},
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name:   "domain.foobar.com",
					Type:   v2.ExternalTypeDomain,
					Weight: &defaultWeight,
					Port:   &specifiedPort,
				}},
			},
		}: {{
			Host:   "domain.foobar.com",
			Port:   specifiedPort,
			Weight: defaultWeight,
		}},
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name:   "domain.foobar.com",
					Type:   v2.ExternalTypeDomain,
					Weight: &defaultWeight,
				}},
			},
		}: {{
			Host:   "domain.foobar.com",
			Port:   defaultPort,
			Weight: defaultWeight,
		}},
	}
	for k, v := range testCases {
		result, _ := tr.TranslateApisixUpstreamExternalNodes(k)
		assert.Equal(t, v, result)
	}
}

func TestTranslateApisixUpstreamExternalNodesDomainTypeError(t *testing.T) {
	tr := &translator{}
	testCases := []v2.ApisixUpstream{
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name: "https://domain.foobar.com",
					Type: v2.ExternalTypeDomain,
				}},
			}},
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name: "grpc://domain.foobar.com",
					Type: v2.ExternalTypeDomain,
				}},
			}},
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name: "-domain.foobar.com",
					Type: v2.ExternalTypeDomain,
				}},
			}},
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name: "-123.foobar.com",
					Type: v2.ExternalTypeDomain,
				}},
			}},
		{
			Spec: &v2.ApisixUpstreamSpec{
				ExternalNodes: []v2.ApisixUpstreamExternalNode{{
					Name: "-123.FOOBAR.com",
					Type: v2.ExternalTypeDomain,
				}},
			}},
	}

	for _, k := range testCases {
		_, err := tr.TranslateApisixUpstreamExternalNodes(&k)
		assert.Error(t, err)
	}
}
