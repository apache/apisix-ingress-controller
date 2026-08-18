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
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// A bare publishService name must resolve against the GatewayProxy's
// namespace, matching the Gateway API path: the publish Service lives next to
// the GatewayProxy, not in the namespace of whichever Ingress is being
// reconciled.
func TestIngressUpdateStatusBarePublishServiceUsesGatewayProxyNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	gatewayProxy := v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "infra", Name: "proxy"},
		Spec: v1alpha1.GatewayProxySpec{
			PublishService: "apisix-lb",
		},
	}
	publishSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: "infra", Name: "apisix-lb"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{{IP: "203.0.113.20"}},
			},
		},
	}
	ingressClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
	}
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "ing"},
	}

	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(publishSvc, ingressClass, ingress).
		Build()

	updater := &recordingUpdater{}
	r := &IngressReconciler{Client: cli, Log: logr.Discard(), Updater: updater}

	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.GatewayProxies[utils.NamespacedNameKind(ingressClass)] = gatewayProxy

	err := r.updateStatus(context.Background(), tctx, ingress, ingressClass)
	require.NoError(t, err,
		"a bare name must resolve in the GatewayProxy's namespace, not the Ingress's")

	require.Len(t, updater.updates, 1)
	mutated, ok := updater.updates[0].Mutator.Mutate(ingress).(*networkingv1.Ingress)
	require.True(t, ok)
	assert.Equal(t, []networkingv1.IngressLoadBalancerIngress{{IP: "203.0.113.20"}},
		mutated.Status.LoadBalancer.Ingress)
}
