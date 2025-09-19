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

	"github.com/stretchr/testify/assert"
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
			},
		},
	}

	warnings, err := validator.ValidateCreate(context.TODO(), obj)
	assert.NoError(t, err)
	assert.Len(t, warnings, 2)

	// Check that warnings contain the expected unsupported annotations
	warningsStr := strings.Join(warnings, " ")
	assert.Contains(t, warningsStr, "k8s.apisix.apache.org/use-regex")
	assert.Contains(t, warningsStr, "k8s.apisix.apache.org/enable-websocket")
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
	assert.NoError(t, err)
	assert.Empty(t, warnings)
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
	assert.NoError(t, err)
	assert.Len(t, warnings, 2)

	// Check that warnings contain the expected unsupported annotations
	warningsStr := strings.Join(warnings, " ")
	assert.Contains(t, warningsStr, "k8s.apisix.apache.org/enable-cors")
	assert.Contains(t, warningsStr, "k8s.apisix.apache.org/cors-allow-origin")
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
	assert.NoError(t, err)
	assert.Empty(t, warnings)
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
	assert.NoError(t, err)
	assert.Empty(t, warnings)
}
