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
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// recordingProvider counts data plane pushes so tests can assert whether a
// reconcile reached Provider.Update.
type recordingProvider struct {
	updated int
}

func (p *recordingProvider) Update(context.Context, *provider.TranslateContext, client.Object) error {
	p.updated++
	return nil
}
func (p *recordingProvider) Delete(context.Context, client.Object) error { return nil }
func (p *recordingProvider) Start(context.Context) error                 { return nil }
func (p *recordingProvider) NeedLeaderElection() bool                    { return false }
func (p *recordingProvider) Register(string, *http.ServeMux)             {}

type recordingUpdater struct {
	updates []status.Update
}

func (u *recordingUpdater) Update(update status.Update) { u.updates = append(u.updates, update) }

// A publishService that cannot be resolved (typo, or the Service not created
// yet) is a status-only problem: the reconcile must still push the Gateway to
// the data plane and keep any previously published addresses, returning the
// error only so the resolution retries with backoff.
func TestGatewayReconcilePublishServiceResolveErrorDoesNotBlockDataPlane(t *testing.T) {
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
			PublishService: "missing-svc",
		},
	}
	addrType := gatewayv1.IPAddressType
	previousAddrs := []gatewayv1.GatewayStatusAddress{{Type: &addrType, Value: "203.0.113.10"}}
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
		Status: gatewayv1.GatewayStatus{Addresses: previousAddrs},
	}

	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(gatewayClass, gatewayProxy, gateway).
		WithStatusSubresource(gateway).
		Build()

	prov := &recordingProvider{}
	updater := &recordingUpdater{}
	r := &GatewayReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  updater,
	}

	result, err := r.Reconcile(context.Background(),
		ctrl.Request{NamespacedName: k8stypes.NamespacedName{Namespace: "default", Name: "gw"}})

	assert.Error(t, err, "the resolve failure must be returned so it retries with backoff")
	assert.Equal(t, ctrl.Result{}, result)
	assert.Equal(t, 1, prov.updated, "Provider.Update must run even when the publish Service cannot be resolved")

	require.Len(t, updater.updates, 1, "conditions must still be written")
	mutated, ok := updater.updates[0].Mutator.Mutate(gateway).(*gatewayv1.Gateway)
	require.True(t, ok)
	assert.Equal(t, previousAddrs, mutated.Status.Addresses,
		"previously published addresses must survive a transient resolve failure")
	assert.True(t, meta.IsStatusConditionTrue(mutated.Status.Conditions, string(gatewayv1.GatewayConditionAccepted)),
		"an unresolvable publish Service must not flip Accepted to False")
}
