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
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
)

const servicePortTestNamespace = "default"

var servicePortRouteKey = k8stypes.NamespacedName{Namespace: servicePortTestNamespace, Name: "route"}

// newServicePortFixture wires an ApisixRoute whose single backend names port
// against a Service exposing servicePorts.
func newServicePortFixture(
	t *testing.T,
	port intstr.IntOrString,
	servicePorts []corev1.ServicePort,
) (*ApisixRouteReconciler, *recordingProvider, *recordingUpdater) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, apiv2.AddToScheme(scheme))

	ingressClass := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
		Spec:       networkingv1.IngressClassSpec{Controller: config.GetControllerName()},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: servicePortTestNamespace, Name: "backend"},
		Spec:       corev1.ServiceSpec{ClusterIP: "10.0.0.1", Ports: servicePorts},
	}
	route := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: servicePortTestNamespace, Name: "route"},
		Spec: apiv2.ApisixRouteSpec{
			IngressClassName: "apisix",
			HTTP: []apiv2.ApisixRouteHTTP{{
				Name:     "rule",
				Match:    apiv2.ApisixRouteHTTPMatch{Hosts: []string{"crd.test"}, Paths: []string{"/*"}},
				Backends: []apiv2.ApisixRouteHTTPBackend{{ServiceName: "backend", ServicePort: port}},
			}},
		},
	}

	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects([]client.Object{ingressClass, service, route}...).
		WithStatusSubresource(route).
		Build()

	readier := readiness.NewReadinessManager(cli, logr.Discard())
	require.NoError(t, readier.Start(context.Background()))

	prov := &recordingProvider{}
	updater := &recordingUpdater{}
	return &ApisixRouteReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  updater,
		Readier:  readier,
	}, prov, updater
}

// acceptedCondition applies the recorded status update and returns the Accepted
// condition it would have written.
func acceptedCondition(t *testing.T, updater *recordingUpdater) metav1.Condition {
	t.Helper()
	require.Len(t, updater.updates, 1, "the reconcile must report a status")
	mutated, ok := updater.updates[0].Mutator.Mutate(&apiv2.ApisixRoute{}).(*apiv2.ApisixRoute)
	require.True(t, ok)
	require.Len(t, mutated.Status.Conditions, 1)
	return mutated.Status.Conditions[0]
}

// An empty servicePort is compared against Service port names, so it silently
// matches a single-port Service that omits its port name. Nothing in the CRD
// schema rejects it, so validateHTTPBackend is the only thing standing between
// this value and a published route; the admission webhook runs the same check.
func TestApisixRouteReconcile_EmptyServicePortIsRejected(t *testing.T) {
	for name, ports := range map[string][]corev1.ServicePort{
		"named port":   {{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080)}},
		"unnamed port": {{Port: 80, TargetPort: intstr.FromInt32(8080)}},
	} {
		t.Run(name, func(t *testing.T) {
			r, prov, updater := newServicePortFixture(t, intstr.FromString(""), ports)

			_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: servicePortRouteKey})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "servicePort must not be empty")
			assert.Zero(t, prov.updated, "a route with an unresolvable port must not be published")

			cond := acceptedCondition(t, updater)
			assert.Equal(t, metav1.ConditionFalse, cond.Status)
			assert.Contains(t, cond.Message, "servicePort must not be empty")
		})
	}
}

// A Service that exists but has no such port used to be reported as accepted and
// published with no upstream node, which answers 503. The message also has to name
// the port, not claim the Service is missing.
func TestApisixRouteReconcile_UnknownServicePortIsRejected(t *testing.T) {
	r, prov, updater := newServicePortFixture(t, intstr.FromString("https"),
		[]corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080)}})

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: servicePortRouteKey})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "service port not found")
	assert.NotContains(t, err.Error(), "service not found",
		"the Service resolves; only the port does not")
	assert.Zero(t, prov.updated)

	cond := acceptedCondition(t, updater)
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, string(apiv2.ConditionReasonInvalidSpec), cond.Reason)
}

// A port that does resolve must still be published.
func TestApisixRouteReconcile_ResolvableServicePortIsPublished(t *testing.T) {
	for name, port := range map[string]intstr.IntOrString{
		"by number": intstr.FromInt32(80),
		"by name":   intstr.FromString("http"),
	} {
		t.Run(name, func(t *testing.T) {
			r, prov, updater := newServicePortFixture(t, port,
				[]corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080)}})

			_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: servicePortRouteKey})

			require.NoError(t, err)
			assert.Equal(t, 1, prov.updated)
			assert.Equal(t, metav1.ConditionTrue, acceptedCondition(t, updater).Status)
		})
	}
}
