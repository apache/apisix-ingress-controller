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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
)

func buildIngressClassTestClient(t *testing.T, objs ...runtime.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, apiv2.AddToScheme(scheme))

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&networkingv1.IngressClass{}, indexer.IngressClass, indexer.IngressClassIndexFunc).
		WithRuntimeObjects(objs...).
		Build()
}

// capturingLogger records every log line emitted at the default verbosity;
// V(1) and above are suppressed, mirroring the controller's default log level.
func capturingLogger() (logr.Logger, *[]string) {
	lines := &[]string{}
	log := funcr.New(func(prefix, args string) {
		*lines = append(*lines, args)
	}, funcr.Options{})
	return log, lines
}

func ourIngressClass(isDefault bool) *networkingv1.IngressClass {
	ic := &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
		Spec:       networkingv1.IngressClassSpec{Controller: config.GetControllerName()},
	}
	if isDefault {
		ic.Annotations = map[string]string{defaultIngressClassAnnotation: "true"}
	}
	return ic
}

func foreignIngressClass(name string) *networkingv1.IngressClass {
	return &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       networkingv1.IngressClassSpec{Controller: "example.com/other-controller"},
	}
}

func apisixRouteWithClass(ingressClassName string) *apiv2.ApisixRoute {
	return &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-route", Namespace: "demo-ns"},
		Spec:       apiv2.ApisixRouteSpec{IngressClassName: ingressClassName},
	}
}

func joinedLines(lines *[]string) string {
	return strings.Join(*lines, "\n")
}

// A CR without ingressClassName is silently dropped today when no IngressClass
// of ours is marked default; the predicate must explain the drop in the log.
func TestMatchesIngressClassPredicate_LogsWhenNoDefaultIngressClass(t *testing.T) {
	c := buildIngressClassTestClient(t, ourIngressClass(false))
	log, lines := capturingLogger()
	p := MatchesIngressClassPredicate(c, log)

	admitted := p.Create(event.CreateEvent{Object: apisixRouteWithClass("")})

	require.False(t, admitted)
	require.Len(t, *lines, 1, "exactly one log line must explain the skip")
	require.Contains(t, joinedLines(lines), "demo-ns/demo-route")
	require.Contains(t, joinedLines(lines), "ApisixRoute")
	require.Contains(t, joinedLines(lines), "no spec.ingressClassName")
	require.Contains(t, joinedLines(lines), "marked as default")
}

// Referencing an IngressClass that does not exist is a user typo the log must
// surface.
func TestMatchesIngressClassPredicate_LogsWhenIngressClassNotFound(t *testing.T) {
	c := buildIngressClassTestClient(t, ourIngressClass(false))
	log, lines := capturingLogger()
	p := MatchesIngressClassPredicate(c, log)

	admitted := p.Create(event.CreateEvent{Object: apisixRouteWithClass("no-such-class")})

	require.False(t, admitted)
	require.Len(t, *lines, 1, "exactly one log line must explain the skip")
	require.Contains(t, joinedLines(lines), "demo-ns/demo-route")
	require.Contains(t, joinedLines(lines), "no-such-class")
}

// A CR explicitly bound to another controller's IngressClass belongs to that
// controller; it must stay quiet at the default log level.
func TestMatchesIngressClassPredicate_SilentForForeignIngressClass(t *testing.T) {
	c := buildIngressClassTestClient(t, ourIngressClass(false), foreignIngressClass("other"))
	log, lines := capturingLogger()
	p := MatchesIngressClassPredicate(c, log)

	admitted := p.Create(event.CreateEvent{Object: apisixRouteWithClass("other")})

	require.False(t, admitted)
	require.Empty(t, *lines, "resources of other controllers must not be logged at the default level")
}

// Regression guard: matching resources are admitted without any log noise.
func TestMatchesIngressClassPredicate_AdmitsMatchingClass(t *testing.T) {
	c := buildIngressClassTestClient(t, ourIngressClass(false))
	log, lines := capturingLogger()
	p := MatchesIngressClassPredicate(c, log)

	require.True(t, p.Create(event.CreateEvent{Object: apisixRouteWithClass("apisix")}))
	require.Empty(t, *lines)
}

// Regression guard: the default-IngressClass fallback still admits CRs without
// an explicit ingressClassName, without log noise.
func TestMatchesIngressClassPredicate_AdmitsViaDefaultClass(t *testing.T) {
	c := buildIngressClassTestClient(t, ourIngressClass(true))
	log, lines := capturingLogger()
	p := MatchesIngressClassPredicate(c, log)

	require.True(t, p.Create(event.CreateEvent{Object: apisixRouteWithClass("")}))
	require.Empty(t, *lines)
}

// An update whose old and new objects both miss must produce one line, not two.
func TestMatchesIngressClassPredicate_UpdateLogsOnce(t *testing.T) {
	c := buildIngressClassTestClient(t, ourIngressClass(false))
	log, lines := capturingLogger()
	p := MatchesIngressClassPredicate(c, log)

	admitted := p.Update(event.UpdateEvent{
		ObjectOld: apisixRouteWithClass(""),
		ObjectNew: apisixRouteWithClass(""),
	})

	require.False(t, admitted)
	require.Len(t, *lines, 1, "an update event must log the skip exactly once")
}
