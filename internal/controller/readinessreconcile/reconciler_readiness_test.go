// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package readinessreconcile holds tests that verify API v2 reconcilers signal
// the shared readiness manager without requiring envtest (see #2725 / #2729).
package readinessreconcile_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	ctrlapi "github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	apisixtypes "github.com/apache/apisix-ingress-controller/internal/types"
)

type recordingReadinessManager struct {
	mu        sync.Mutex
	doneCalls []recordingReadinessDoneCall
}

type recordingReadinessDoneCall struct {
	obj client.Object
	nn  k8stypes.NamespacedName
}

func (r *recordingReadinessManager) RegisterGVK(_ ...readiness.GVKConfig) {}

func (r *recordingReadinessManager) Start(_ context.Context) error { return nil }

func (r *recordingReadinessManager) IsReady() bool { return true }

func (r *recordingReadinessManager) WaitReady(_ context.Context, _ time.Duration) bool {
	return true
}

func (r *recordingReadinessManager) Done(obj client.Object, nn k8stypes.NamespacedName) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.doneCalls = append(r.doneCalls, recordingReadinessDoneCall{obj: obj, nn: nn})
}

func (r *recordingReadinessManager) lastDone() (schema.GroupVersionKind, k8stypes.NamespacedName, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.doneCalls) == 0 {
		return schema.GroupVersionKind{}, k8stypes.NamespacedName{}, false
	}
	c := r.doneCalls[len(r.doneCalls)-1]
	return apisixtypes.GvkOf(c.obj), c.nn, true
}

type noopStatusUpdater struct{}

func (noopStatusUpdater) Update(_ status.Update) {}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	if err := apiv2.AddToScheme(s); err != nil {
		t.Fatalf("apiv2 AddToScheme: %v", err)
	}
	return s
}

func TestApisixPluginConfigReconciler_CallsReadinessDone(t *testing.T) {
	ctx := context.Background()
	ns := "ns-apc"
	icName := "ic-apc-test"

	scheme := testScheme(t)
	ic := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: icName},
		Spec: networkingv1.IngressClassSpec{
			Controller: "apisix.apache.org/apisix-ingress-controller",
		},
	}
	pc := &apiv2.ApisixPluginConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc1", Namespace: ns},
		Spec: apiv2.ApisixPluginConfigSpec{
			IngressClassName: icName,
			Plugins: []apiv2.ApisixRoutePlugin{
				{Name: "echo", Enable: true, Config: apiextensionsv1.JSON{Raw: []byte(`{}`)}},
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, ic, pc).
		Build()

	readier := &recordingReadinessManager{}
	r := &ctrlapi.ApisixPluginConfigReconciler{
		Client:  cl,
		Scheme:  scheme,
		Log:     logr.Discard(),
		Updater: noopStatusUpdater{},
		Readier: readier,
	}

	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{Namespace: ns, Name: pc.Name},
	})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	gvk, nn, ok := readier.lastDone()
	if !ok {
		t.Fatal("expected ReadinessManager.Done to be called")
	}
	wantGVK := apisixtypes.GvkOf(&apiv2.ApisixPluginConfig{})
	if gvk != wantGVK {
		t.Fatalf("Done object GVK = %v, want %v", gvk, wantGVK)
	}
	wantNN := k8stypes.NamespacedName{Namespace: ns, Name: pc.Name}
	if nn != wantNN {
		t.Fatalf("Done nn = %v, want %v", nn, wantNN)
	}
}

func TestApisixUpstreamReconciler_CallsReadinessDone(t *testing.T) {
	ctx := context.Background()
	ns := "ns-au"
	icName := "ic-au-test"

	scheme := testScheme(t)
	ic := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: icName},
		Spec: networkingv1.IngressClassSpec{
			Controller: "apisix.apache.org/apisix-ingress-controller",
		},
	}
	au := &apiv2.ApisixUpstream{
		ObjectMeta: metav1.ObjectMeta{Name: "au1", Namespace: ns},
		Spec: apiv2.ApisixUpstreamSpec{
			IngressClassName: icName,
			ExternalNodes: []apiv2.ApisixUpstreamExternalNode{
				{Type: apiv2.ExternalTypeDomain, Name: "example.com"},
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, ic, au).
		Build()

	readier := &recordingReadinessManager{}
	r := &ctrlapi.ApisixUpstreamReconciler{
		Client:  cl,
		Scheme:  scheme,
		Log:     logr.Discard(),
		Updater: noopStatusUpdater{},
		Readier: readier,
	}

	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{Namespace: ns, Name: au.Name},
	})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	gvk, nn, ok := readier.lastDone()
	if !ok {
		t.Fatal("expected ReadinessManager.Done to be called")
	}
	wantGVK := apisixtypes.GvkOf(&apiv2.ApisixUpstream{})
	if gvk != wantGVK {
		t.Fatalf("Done object GVK = %v, want %v", gvk, wantGVK)
	}
	wantNN := k8stypes.NamespacedName{Namespace: ns, Name: au.Name}
	if nn != wantNN {
		t.Fatalf("Done nn = %v, want %v", nn, wantNN)
	}
}
