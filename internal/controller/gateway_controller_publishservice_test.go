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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
)

type recordingUpdater struct {
	updates []status.Update
}

func (u *recordingUpdater) Update(update status.Update) { u.updates = append(u.updates, update) }

var gatewayPreviousAddrs = func() []gatewayv1.GatewayStatusAddress {
	addrType := gatewayv1.IPAddressType
	return []gatewayv1.GatewayStatusAddress{{Type: &addrType, Value: "203.0.113.10"}}
}()

// newGatewayPublishServiceFixture builds a Gateway with previously published
// addresses, wired to a GatewayProxy with the given publishService.
func newGatewayPublishServiceFixture(
	t *testing.T,
	publishService string,
	interceptorFuncs interceptor.Funcs,
	extraObjects ...client.Object,
) (*GatewayReconciler, *recordingProvider, *recordingUpdater) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	gatewayClass := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController(config.ControllerConfig.ControllerName),
		},
	}
	gatewayProxy := &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "proxy"},
		Spec: v1alpha1.GatewayProxySpec{
			PublishService: publishService,
		},
	}
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "apisix",
			Infrastructure: &gatewayv1.GatewayInfrastructure{
				ParametersRef: &gatewayv1.LocalParametersReference{
					Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
					Kind:  KindGatewayProxy,
					Name:  "proxy",
				},
			},
		},
		Status: gatewayv1.GatewayStatus{Addresses: gatewayPreviousAddrs},
	}

	objects := append([]client.Object{gatewayClass, gatewayProxy, gateway}, extraObjects...)
	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(gateway).
		WithInterceptorFuncs(interceptorFuncs).
		Build()

	prov := &recordingProvider{}
	updater := &recordingUpdater{}
	return &GatewayReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  updater,
	}, prov, updater
}

func reconcileGateway(t *testing.T, r *GatewayReconciler) (ctrl.Result, error) {
	t.Helper()
	return r.Reconcile(context.Background(),
		ctrl.Request{NamespacedName: k8stypes.NamespacedName{Namespace: "default", Name: "gw"}})
}

// mutatedGatewayStatus applies the single recorded status update and returns
// the Gateway status it would have written.
func mutatedGatewayStatus(t *testing.T, updater *recordingUpdater) gatewayv1.GatewayStatus {
	t.Helper()
	require.Len(t, updater.updates, 1, "conditions must still be written")
	mutated, ok := updater.updates[0].Mutator.Mutate(&gatewayv1.Gateway{}).(*gatewayv1.Gateway)
	require.True(t, ok)
	return mutated.Status
}

// A missing publish Service must not block the data plane push or count as a
// reconcile error: it surfaces as Programmed=False/AddressNotAssigned and a
// requeue picks the addresses up once the Service exists.
func TestGatewayReconcilePublishServiceNotFound(t *testing.T) {
	r, prov, updater := newGatewayPublishServiceFixture(t, "missing-svc", interceptor.Funcs{})

	result, err := reconcileGateway(t, r)

	assert.NoError(t, err, "a missing publish Service must not count as a reconcile error")
	assert.Equal(t, publishServiceRetryInterval, result.RequeueAfter,
		"must poll for the Service until a Service watch makes this event-driven")
	assert.Equal(t, 1, prov.updated, "Provider.Update must run even when the publish Service cannot be resolved")

	gotStatus := mutatedGatewayStatus(t, updater)
	programmed := meta.FindStatusCondition(gotStatus.Conditions, string(gatewayv1.GatewayConditionProgrammed))
	require.NotNil(t, programmed)
	assert.Equal(t, metav1.ConditionFalse, programmed.Status)
	assert.Equal(t, string(gatewayv1.GatewayReasonAddressNotAssigned), programmed.Reason)
	assert.Contains(t, programmed.Message, "missing-svc")
	assert.True(t, meta.IsStatusConditionTrue(gotStatus.Conditions, string(gatewayv1.GatewayConditionAccepted)),
		"an unresolvable publish Service must not flip Accepted to False")
	assert.Equal(t, gatewayPreviousAddrs, gotStatus.Addresses,
		"previously published addresses must survive a resolve failure")
}

// An invalid publishService format is handled like NotFound: surfaced on the
// Programmed condition, not the reconcile error.
func TestGatewayReconcilePublishServiceBadFormat(t *testing.T) {
	r, prov, updater := newGatewayPublishServiceFixture(t, "a/b/c", interceptor.Funcs{})

	result, err := reconcileGateway(t, r)

	assert.NoError(t, err, "an invalid publishService value must not count as a reconcile error")
	assert.Equal(t, publishServiceRetryInterval, result.RequeueAfter)
	assert.Equal(t, 1, prov.updated)

	gotStatus := mutatedGatewayStatus(t, updater)
	programmed := meta.FindStatusCondition(gotStatus.Conditions, string(gatewayv1.GatewayConditionProgrammed))
	require.NotNil(t, programmed)
	assert.Equal(t, metav1.ConditionFalse, programmed.Status)
	assert.Equal(t, string(gatewayv1.GatewayReasonAddressNotAssigned), programmed.Reason)
	assert.Contains(t, programmed.Message, "a/b/c")
}

// A non-NotFound lookup failure is a genuine API failure: returned as a
// reconcile error, without blaming the user's config on Programmed.
func TestGatewayReconcilePublishServiceAPIFailure(t *testing.T) {
	apiDown := apierrors.NewInternalError(context.DeadlineExceeded)
	r, prov, updater := newGatewayPublishServiceFixture(t, "some-svc", interceptor.Funcs{
		Get: func(ctx context.Context, cli client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, isService := obj.(*corev1.Service); isService {
				return apiDown
			}
			return cli.Get(ctx, key, obj, opts...)
		},
	})

	result, err := reconcileGateway(t, r)

	assert.ErrorIs(t, err, apiDown, "an API failure must be returned for backoff retry")
	assert.Equal(t, ctrl.Result{}, result)
	assert.Equal(t, 1, prov.updated, "Provider.Update must run even when the address lookup fails")

	gotStatus := mutatedGatewayStatus(t, updater)
	assert.True(t, meta.IsStatusConditionTrue(gotStatus.Conditions, string(gatewayv1.GatewayConditionProgrammed)),
		"an API failure is not a user configuration problem")
	assert.Equal(t, gatewayPreviousAddrs, gotStatus.Addresses)
}
