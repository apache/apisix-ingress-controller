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
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

const testConsumerNamespace = "default"

// recordingProvider records the objects passed to Delete and can be told to fail.
type recordingProvider struct {
	deleted   []types.NamespacedName
	deleteErr error
}

func (p *recordingProvider) Register(string, *http.ServeMux) {}

func (p *recordingProvider) Update(context.Context, *provider.TranslateContext, client.Object) error {
	return nil
}

func (p *recordingProvider) Delete(_ context.Context, obj client.Object) error {
	p.deleted = append(p.deleted, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
	return p.deleteErr
}

func (p *recordingProvider) Start(context.Context) error { return nil }

func (p *recordingProvider) NeedLeaderElection() bool { return true }

func newApisixConsumerReconciler(t *testing.T, cli client.Client, p provider.Provider) *ApisixConsumerReconciler {
	t.Helper()

	// A readiness manager with no registered GVKs becomes ready as soon as it is
	// started. Readier.Done blocks on the started channel, so it must be started.
	readier := readiness.NewReadinessManager(cli, logr.Discard())
	require.NoError(t, readier.Start(context.Background()))

	return &ApisixConsumerReconciler{
		Client:   cli,
		Scheme:   cli.Scheme(),
		Log:      logr.Discard(),
		Provider: p,
		Readier:  readier,
	}
}

func apisixConsumerScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, apiv2.AddToScheme(scheme))
	return scheme
}

// A deleted ApisixConsumer must be removed from the provider and then reported
// as reconciled. Returning the NotFound error instead makes controller-runtime
// treat the reconcile as failed and requeue it indefinitely with exponential
// backoff, even though the delete succeeded.
func TestApisixConsumerReconcile_DeletedObjectDoesNotRequeue(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(apisixConsumerScheme(t)).Build()
	p := &recordingProvider{}
	r := newApisixConsumerReconciler(t, cli, p)

	key := types.NamespacedName{Namespace: testConsumerNamespace, Name: "gone"}
	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.Equal(t, []types.NamespacedName{key}, p.deleted)
}

// A provider failure must still surface, so the fix does not swallow real errors.
func TestApisixConsumerReconcile_DeleteErrorIsReturned(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(apisixConsumerScheme(t)).Build()
	p := &recordingProvider{deleteErr: errors.New("provider unavailable")}
	r := newApisixConsumerReconciler(t, cli, p)

	key := types.NamespacedName{Namespace: testConsumerNamespace, Name: "gone"}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider unavailable")
}

// A Get failure that is not NotFound must be returned, and must not be mistaken
// for a deletion.
func TestApisixConsumerReconcile_NonNotFoundGetErrorIsReturned(t *testing.T) {
	cli := fake.NewClientBuilder().
		WithScheme(apisixConsumerScheme(t)).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return k8serrors.NewInternalError(errors.New("boom"))
			},
		}).
		Build()
	p := &recordingProvider{}
	r := newApisixConsumerReconciler(t, cli, p)

	key := types.NamespacedName{Namespace: testConsumerNamespace, Name: "present"}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})

	require.Error(t, err)
	assert.True(t, k8serrors.IsInternalError(err), "want an internal error, got %v", err)
	assert.Empty(t, p.deleted, "a non-NotFound Get error must not trigger a provider delete")
}
