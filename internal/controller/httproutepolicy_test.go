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
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func malformedVarsPolicy(kind, target string) *v1alpha1.HTTPRoutePolicy {
	group := gatewayv1.GroupName
	if kind == KindIngress {
		group = networkingv1.GroupName
	}
	return &v1alpha1.HTTPRoutePolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "policy"},
		Spec: v1alpha1.HTTPRoutePolicySpec{
			TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{{
				LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
					Group: gatewayv1.Group(group),
					Kind:  gatewayv1.Kind(kind),
					Name:  gatewayv1.ObjectName(target),
				},
			}},
			Vars: []apiextensionsv1.JSON{
				{Raw: []byte(`["remote_addr","==","10.0.0.1"]`)},
				{Raw: []byte(`{"remote_addr":"10.0.0.0/8"}`)},
			},
		},
	}
}

func policyClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().WithScheme(retractPluginConfigScheme(t)).
		WithObjects(objects...).
		WithIndex(&v1alpha1.HTTPRoutePolicy{}, indexer.PolicyTargetRefs, indexer.HTTPRoutePolicyIndexFunc).
		Build()
}

// requireInvalidPolicyStatus applies the recorded status update and checks it rejects the policy.
func requireInvalidPolicyStatus(t *testing.T, policy *v1alpha1.HTTPRoutePolicy, mutated client.Object) {
	t.Helper()
	got, ok := mutated.(*v1alpha1.HTTPRoutePolicy)
	require.True(t, ok)
	require.Len(t, got.Status.Ancestors, 1)
	require.Len(t, got.Status.Ancestors[0].Conditions, 1)
	cond := got.Status.Ancestors[0].Conditions[0]
	assert.Equal(t, string(gatewayv1.PolicyConditionAccepted), cond.Type)
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, string(gatewayv1.PolicyReasonInvalid), cond.Reason)
	assert.Contains(t, cond.Message, "invalid spec.vars[1]")
	assert.Equal(t, policy.Name, got.Name)
}

func TestHTTPRouteProcessHTTPRoutePolicies_MalformedVars(t *testing.T) {
	policy := malformedVarsPolicy(KindHTTPRoute, "route")
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "gw"}}},
			Rules:           []gatewayv1.HTTPRouteRule{{}},
		},
	}
	r := &HTTPRouteReconciler{Client: policyClient(t, policy.DeepCopy()), Log: logr.Discard()}
	tctx := provider.NewDefaultTranslateContext(context.Background())

	require.NoError(t, r.processHTTPRoutePolicies(tctx, route))

	// Kept so translation fails instead of publishing the route without its vars.
	require.Len(t, tctx.HTTPRoutePolicies, 1)
	require.Len(t, tctx.StatusUpdaters, 1)
	requireInvalidPolicyStatus(t, policy, tctx.StatusUpdaters[0].Mutator.Mutate(policy.DeepCopy()))
}

type failingUpdateProvider struct {
	recordingProvider
}

func (p *failingUpdateProvider) Update(context.Context, *provider.TranslateContext, client.Object) error {
	p.updated++
	return errors.New("translation failed")
}

func TestIngressReconcile_ReportsMalformedPolicyVars(t *testing.T) {
	policy := malformedVarsPolicy(KindIngress, "ing")
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ing"},
		Spec:       networkingv1.IngressSpec{IngressClassName: ptrTo("apisix")},
	}
	cli := policyClient(t, retractIngressClass(), ingress, policy.DeepCopy())
	updater := &recordingUpdater{}
	prov := &failingUpdateProvider{}
	r := &IngressReconciler{
		Client:   cli,
		Scheme:   retractPluginConfigScheme(t),
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  updater,
		Readier:  newRetractReadier(t, cli),
	}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(ingress)})

	require.Error(t, err)
	assert.Equal(t, 1, prov.updated)
	var found bool
	for _, u := range updater.updates {
		if _, ok := u.Resource.(*v1alpha1.HTTPRoutePolicy); ok {
			found = true
			requireInvalidPolicyStatus(t, policy, u.Mutator.Mutate(policy.DeepCopy()))
		}
	}
	assert.True(t, found, "policy status must be reported even though the Ingress update failed")
}
