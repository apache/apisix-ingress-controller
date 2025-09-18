// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"context"
	"strings"
	"testing"

	networkingk8siov1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressCustomValidator_ValidateCreate_UnsupportedAnnotations(t *testing.T) {
	validator := IngressCustomValidator{}
	obj := &networkingk8siov1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"k8s.apisix.apache.org/use-regex":        "true",
				"k8s.apisix.apache.org/enable-websocket": "true",
				"nginx.ingress.kubernetes.io/rewrite":    "/new-path",
			},
		},
	}

	warnings, err := validator.ValidateCreate(context.TODO(), obj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "k8s.apisix.apache.org/use-regex") {
		t.Errorf("Expected warning to contain 'k8s.apisix.apache.org/use-regex', got %s", warnings[0])
	}
	if !strings.Contains(warnings[1], "k8s.apisix.apache.org/enable-websocket") {
		t.Errorf("Expected warning to contain 'k8s.apisix.apache.org/enable-websocket', got %s", warnings[1])
	}
}

func TestIngressCustomValidator_ValidateCreate_SupportedAnnotations(t *testing.T) {
	validator := IngressCustomValidator{}
	obj := &networkingk8siov1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"ingressclass.kubernetes.io/is-default-class": "true",
			},
		},
	}

	warnings, err := validator.ValidateCreate(context.TODO(), obj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}
}

func TestIngressCustomValidator_ValidateUpdate_UnsupportedAnnotations(t *testing.T) {
	validator := IngressCustomValidator{}
	oldObj := &networkingk8siov1.Ingress{}
	obj := &networkingk8siov1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"k8s.apisix.apache.org/enable-cors":       "true",
				"k8s.apisix.apache.org/cors-allow-origin": "*",
			},
		},
	}

	warnings, err := validator.ValidateUpdate(context.TODO(), oldObj, obj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "k8s.apisix.apache.org/enable-cors") {
		t.Errorf("Expected warning to contain 'k8s.apisix.apache.org/enable-cors', got %s", warnings[0])
	}
	if !strings.Contains(warnings[1], "k8s.apisix.apache.org/cors-allow-origin") {
		t.Errorf("Expected warning to contain 'k8s.apisix.apache.org/cors-allow-origin', got %s", warnings[1])
	}
}

func TestIngressCustomValidator_ValidateDelete_NoWarnings(t *testing.T) {
	validator := IngressCustomValidator{}
	obj := &networkingk8siov1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"k8s.apisix.apache.org/use-regex": "true",
			},
		},
	}

	warnings, err := validator.ValidateDelete(context.TODO(), obj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}
}

func TestIngressCustomValidator_ValidateCreate_NoAnnotations(t *testing.T) {
	validator := IngressCustomValidator{}
	obj := &networkingk8siov1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
		},
	}

	warnings, err := validator.ValidateCreate(context.TODO(), obj)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}
}
