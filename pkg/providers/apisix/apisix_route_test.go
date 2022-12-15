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

package apisix

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
)

func TestIsApisixRouteEffective(t *testing.T) {
	type tc struct {
		ac     *apisixRouteController
		ar     kube.ApisixRoute
		result bool
	}
	tcs := []tc{{
		ac: &apisixRouteController{

			apisixCommon: &apisixCommon{
				Common: &providertypes.Common{
					Config: &config.Config{
						Kubernetes: config.KubernetesConfig{
							IngressClass: "barfoo",
						},
					},
				},
			},
		},
		ar:     genApisixRoute(&configv2.ApisixRoute{Spec: configv2.ApisixRouteSpec{IngressClass: "barfoo"}}),
		result: true,
	},
		{
			ac: &apisixRouteController{

				apisixCommon: &apisixCommon{
					Common: &providertypes.Common{
						Config: &config.Config{
							Kubernetes: config.KubernetesConfig{
								IngressClass: "barfoo",
							},
						},
					},
				},
			},
			ar:     genApisixRoute(&configv2.ApisixRoute{Spec: configv2.ApisixRouteSpec{IngressClass: "foobar"}}),
			result: false,
		},
		{
			ac: &apisixRouteController{

				apisixCommon: &apisixCommon{
					Common: &providertypes.Common{
						Config: &config.Config{
							Kubernetes: config.KubernetesConfig{
								IngressClass: "*",
							},
						},
					},
				},
			},
			ar:     genApisixRoute(&configv2.ApisixRoute{Spec: configv2.ApisixRouteSpec{IngressClass: "foobar"}}),
			result: true,
		},
		{
			ac: &apisixRouteController{

				apisixCommon: &apisixCommon{
					Common: &providertypes.Common{
						Config: &config.Config{
							Kubernetes: config.KubernetesConfig{
								IngressClass: "*",
							},
						},
					},
				},
			},
			ar:     genApisixRoute(&configv2beta3.ApisixRoute{}),
			result: true,
		},
		{
			ac: &apisixRouteController{

				apisixCommon: &apisixCommon{
					Common: &providertypes.Common{
						Config: &config.Config{
							Kubernetes: config.KubernetesConfig{
								IngressClass: "barfoo",
							},
						},
					},
				},
			},
			ar:     genApisixRoute(&configv2beta3.ApisixRoute{}),
			result: true,
		},
	}
	for _, item := range tcs {
		assert.Equal(t, item.result, item.ac.isApisixRouteEffective(item.ar))
	}
}
func genApisixRoute(obj interface{}) kube.ApisixRoute {
	ret, _ := kube.NewApisixRoute(obj)
	return ret
}
