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

package translator

import (
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
)

func makeL4RoutePolicy(namespace, name, targetKind, targetName string, plugins []v1alpha1.Plugin) *v1alpha1.L4RoutePolicy {
	return &v1alpha1.L4RoutePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.L4RoutePolicySpec{
			TargetRefs: []gatewayv1alpha2.LocalPolicyTargetReferenceWithSectionName{
				{
					LocalPolicyTargetReference: gatewayv1alpha2.LocalPolicyTargetReference{
						Group: gatewayv1alpha2.GroupName,
						Kind:  gatewayv1alpha2.Kind(targetKind),
						Name:  gatewayv1alpha2.ObjectName(targetName),
					},
				},
			},
			Plugins: plugins,
		},
	}
}

func mustJSON(v any) apiextensionsv1.JSON {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return apiextensionsv1.JSON{Raw: b}
}

func TestAttachL4RoutePolicyPlugins_AttachesMatchingPolicy(t *testing.T) {
	tr := NewTranslator(logr.Discard(), "")

	policy := makeL4RoutePolicy("default", "my-policy", "TCPRoute", "my-tcp-route", []v1alpha1.Plugin{
		{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 100, "burst": 50})},
		{Name: "ip-restriction", Config: mustJSON(map[string]any{"whitelist": []string{"10.0.0.0/8"}})},
	})

	policies := map[k8stypes.NamespacedName]*v1alpha1.L4RoutePolicy{
		{Namespace: "default", Name: "my-policy"}: policy,
	}

	plugins := adctypes.Plugins{}
	tr.AttachL4RoutePolicyPlugins(policies, "default", "my-tcp-route", "TCPRoute", plugins)

	assert.Len(t, plugins, 2)
	assert.Contains(t, plugins, "limit-conn")
	assert.Contains(t, plugins, "ip-restriction")

	cfg := plugins["limit-conn"].(map[string]any)
	assert.EqualValues(t, 100, cfg["conn"])
}

func TestAttachL4RoutePolicyPlugins_NoMatchOnKind(t *testing.T) {
	tr := NewTranslator(logr.Discard(), "")

	policy := makeL4RoutePolicy("default", "udp-policy", "UDPRoute", "my-udp-route", []v1alpha1.Plugin{
		{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 10})},
	})

	policies := map[k8stypes.NamespacedName]*v1alpha1.L4RoutePolicy{
		{Namespace: "default", Name: "udp-policy"}: policy,
	}

	plugins := adctypes.Plugins{}
	// Looking for TCPRoute, but policy targets UDPRoute — should not match.
	tr.AttachL4RoutePolicyPlugins(policies, "default", "my-udp-route", "TCPRoute", plugins)

	assert.Empty(t, plugins)
}

func TestAttachL4RoutePolicyPlugins_NoMatchOnNamespace(t *testing.T) {
	tr := NewTranslator(logr.Discard(), "")

	policy := makeL4RoutePolicy("other-ns", "my-policy", "TCPRoute", "my-tcp-route", []v1alpha1.Plugin{
		{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 10})},
	})

	policies := map[k8stypes.NamespacedName]*v1alpha1.L4RoutePolicy{
		{Namespace: "other-ns", Name: "my-policy"}: policy,
	}

	plugins := adctypes.Plugins{}
	// Route is in "default" namespace, policy is in "other-ns" — should not match.
	tr.AttachL4RoutePolicyPlugins(policies, "default", "my-tcp-route", "TCPRoute", plugins)

	assert.Empty(t, plugins)
}

func TestAttachL4RoutePolicyPlugins_EmptyPlugins(t *testing.T) {
	tr := NewTranslator(logr.Discard(), "")

	policy := makeL4RoutePolicy("default", "empty-policy", "TCPRoute", "my-tcp-route", nil)

	policies := map[k8stypes.NamespacedName]*v1alpha1.L4RoutePolicy{
		{Namespace: "default", Name: "empty-policy"}: policy,
	}

	plugins := adctypes.Plugins{}
	tr.AttachL4RoutePolicyPlugins(policies, "default", "my-tcp-route", "TCPRoute", plugins)

	assert.Empty(t, plugins)
}

func TestAttachL4RoutePolicyPlugins_EmptyPolicies(t *testing.T) {
	tr := NewTranslator(logr.Discard(), "")
	plugins := adctypes.Plugins{}
	tr.AttachL4RoutePolicyPlugins(nil, "default", "my-tcp-route", "TCPRoute", plugins)
	assert.Empty(t, plugins)
}
