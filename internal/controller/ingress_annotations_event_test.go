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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

func ingressWithAnnotations(anno map[string]string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ingress", Namespace: "default", Annotations: anno},
	}
}

func TestWarnOnInvalidAnnotations(t *testing.T) {
	recorder := record.NewFakeRecorder(10)
	r := &IngressReconciler{Log: logr.Discard(), EventRecorder: recorder}

	r.warnOnInvalidAnnotations(ingressWithAnnotations(map[string]string{
		annotations.AnnotationsEnableCsrf: "true",
	}))

	select {
	case got := <-recorder.Events:
		assert.Contains(t, got, "Warning")
		assert.Contains(t, got, "InvalidAnnotations")
		assert.Contains(t, got, "csrf")
	default:
		t.Fatal("expected a Warning event for the unparsable annotation")
	}
}

func TestWarnOnInvalidAnnotationsStaysQuietWhenValid(t *testing.T) {
	recorder := record.NewFakeRecorder(10)
	r := &IngressReconciler{Log: logr.Discard(), EventRecorder: recorder}

	r.warnOnInvalidAnnotations(ingressWithAnnotations(map[string]string{
		annotations.AnnotationsEnableCsrf: "true",
		annotations.AnnotationsCsrfKey:    "secret",
	}))
	r.warnOnInvalidAnnotations(ingressWithAnnotations(nil))

	select {
	case got := <-recorder.Events:
		t.Fatalf("expected no event, got %q", got)
	default:
	}
}

// The reconciler is constructed without a recorder in several tests and in any
// caller that does not go through SetupWithManager.
func TestWarnOnInvalidAnnotationsWithoutRecorder(t *testing.T) {
	r := &IngressReconciler{Log: logr.Discard()}

	assert.NotPanics(t, func() {
		r.warnOnInvalidAnnotations(ingressWithAnnotations(map[string]string{
			annotations.AnnotationsEnableCsrf: "true",
		}))
	})
}
