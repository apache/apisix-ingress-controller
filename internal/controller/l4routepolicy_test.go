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

package controller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestProcessL4RoutePolicy_InvalidPluginConfigSetsRejectedStatus(t *testing.T) {
	policy := &v1alpha1.L4RoutePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  "default",
			Name:       "tcp-policy",
			Generation: 3,
		},
		Spec: v1alpha1.L4RoutePolicySpec{
			TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{{
				LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
					Group: gatewayv1.GroupName,
					Kind:  "TCPRoute",
					Name:  "tcp-route",
				},
			}},
			Plugins: []v1alpha1.Plugin{{
				Name:   "ip-restriction",
				Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
			}},
		},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(policy).
		WithIndex(&v1alpha1.L4RoutePolicy{}, indexer.PolicyTargetRefs, indexer.L4RoutePolicyIndexFunc).
		Build()
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.RouteParentRefs = []gatewayv1.ParentReference{{Name: "gateway"}}

	ProcessL4RoutePolicy(cli, logr.Discard(), tctx, "default", "tcp-route", "TCPRoute")

	key := k8stypes.NamespacedName{Namespace: "default", Name: "tcp-policy"}
	require.NotNil(t, tctx.L4RoutePolicies[key], "the policy must reach translation so rendering stops the update")
	require.Len(t, tctx.StatusUpdaters, 1)
	mutated := tctx.StatusUpdaters[0].Mutator.Mutate(&v1alpha1.L4RoutePolicy{}).(*v1alpha1.L4RoutePolicy)
	require.Len(t, mutated.Status.Ancestors, 1)
	require.Len(t, mutated.Status.Ancestors[0].Conditions, 1)
	condition := mutated.Status.Ancestors[0].Conditions[0]
	assert.Equal(t, string(gatewayv1.PolicyConditionAccepted), condition.Type)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	assert.Equal(t, string(gatewayv1.PolicyReasonInvalid), condition.Reason)
	assert.Equal(t, int64(3), condition.ObservedGeneration)
	assert.Equal(t, `plugin "ip-restriction" has invalid configuration`, condition.Message)
}
