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

package types

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetEffectiveIngressClassName(t *testing.T) {
	tests := []struct {
		name    string
		ingress *networkingv1.Ingress
		want    string
	}{
		{
			name: "spec-class",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					IngressClassName: ptr.To("spec-class"),
				},
			},
			want: "spec-class",
		},
		{
			name: "annotation-class",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "annotation-class",
					},
				},
			},
			want: "annotation-class",
		},
		{
			name: "spec-class-and-annotation-class",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					IngressClassName: ptr.To("spec-class"),
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						IngressClassNameAnnotation: "annotation-class",
					},
				},
			},
			want: "spec-class",
		},
		{
			name:    "empty-ingress",
			ingress: &networkingv1.Ingress{},
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEffectiveIngressClassName(tt.ingress)
			if got != tt.want {
				t.Errorf("GetEffectiveIngressClassName() = %v, want %v", got, tt.want)
			}
		})
	}
}
