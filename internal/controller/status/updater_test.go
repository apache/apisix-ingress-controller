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

package status

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
)

func gatewayProxyWithCondition(condition metav1.Condition) *v1alpha1.GatewayProxy {
	return &v1alpha1.GatewayProxy{
		Status: v1alpha1.GatewayProxyStatus{
			Conditions: []metav1.Condition{condition},
		},
	}
}

func TestStatusEqualHasAGatewayProxyCase(t *testing.T) {
	condition := metav1.Condition{
		Type:               "DataPlaneAvailable",
		Status:             metav1.ConditionTrue,
		Reason:             "DataPlaneAvailable",
		LastTransitionTime: metav1.Now(),
	}
	a := gatewayProxyWithCondition(condition)
	b := gatewayProxyWithCondition(condition)

	if !statusEqual(a, b) {
		t.Fatal("two GatewayProxy objects with an identical condition must compare equal, not fall through to the default case")
	}
}

func TestStatusEqualIgnoresLastTransitionTimeForGatewayProxy(t *testing.T) {
	a := gatewayProxyWithCondition(metav1.Condition{
		Type:               "DataPlaneAvailable",
		Status:             metav1.ConditionTrue,
		Reason:             "DataPlaneAvailable",
		LastTransitionTime: metav1.NewTime(metav1.Now().Add(-1)),
	})
	b := gatewayProxyWithCondition(metav1.Condition{
		Type:               "DataPlaneAvailable",
		Status:             metav1.ConditionTrue,
		Reason:             "DataPlaneAvailable",
		LastTransitionTime: metav1.Now(),
	})

	if !statusEqual(a, b, cmpIgnoreLastTT) {
		t.Fatal("a fresh LastTransitionTime alone must not make an otherwise-unchanged GatewayProxy condition compare unequal")
	}
}

func TestStatusEqualDetectsAChangedGatewayProxyCondition(t *testing.T) {
	a := gatewayProxyWithCondition(metav1.Condition{Type: "DataPlaneAvailable", Status: metav1.ConditionTrue, Reason: "DataPlaneAvailable"})
	b := gatewayProxyWithCondition(metav1.Condition{Type: "DataPlaneAvailable", Status: metav1.ConditionFalse, Reason: "DataPlaneInstanceUnavailable"})

	if statusEqual(a, b, cmpIgnoreLastTT) {
		t.Fatal("a real condition change must still compare unequal")
	}
}

// reapplySameGatewayProxyCondition mutates a fresh GatewayProxy fixture through
// UpdateHandler.updateStatus twice with a mutator that always writes the same
// condition values (only LastTransitionTime differs between calls, as a real
// mutator's `metav1.Now()` would) and returns the object's resourceVersion after
// each call.
func reapplySameGatewayProxyCondition(t *testing.T) (first, second string) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	gatewayProxy := &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "proxy"},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(gatewayProxy).
		WithStatusSubresource(gatewayProxy).
		Build()

	u := NewStatusUpdateHandler(logr.Discard(), cli)
	mutate := func(obj client.Object) client.Object {
		cp := obj.(*v1alpha1.GatewayProxy).DeepCopy()
		cp.Status.Conditions = []metav1.Condition{{
			Type:               "DataPlaneAvailable",
			Status:             metav1.ConditionFalse,
			Reason:             "DataPlaneInstanceUnavailable",
			Message:            "unreachable",
			LastTransitionTime: metav1.Now(),
		}}
		return cp
	}
	nnk := k8stypes.NamespacedName{Namespace: "default", Name: "proxy"}
	ctx := context.Background()

	require.NoError(t, u.updateStatus(ctx, Update{
		NamespacedName: nnk,
		Resource:       &v1alpha1.GatewayProxy{},
		Mutator:        MutatorFunc(mutate),
	}))
	var afterFirst v1alpha1.GatewayProxy
	require.NoError(t, cli.Get(ctx, nnk, &afterFirst))

	require.NoError(t, u.updateStatus(ctx, Update{
		NamespacedName: nnk,
		Resource:       &v1alpha1.GatewayProxy{},
		Mutator:        MutatorFunc(mutate),
	}))
	var afterSecond v1alpha1.GatewayProxy
	require.NoError(t, cli.Get(ctx, nnk, &afterSecond))

	return afterFirst.ResourceVersion, afterSecond.ResourceVersion
}

func TestUpdateStatusSkipsTheWriteWhenTheConditionDidNotActuallyChange(t *testing.T) {
	first, second := reapplySameGatewayProxyCondition(t)
	if first != second {
		t.Fatalf("reapplying an unchanged condition must not write again: resourceVersion went from %q to %q", first, second)
	}
}
