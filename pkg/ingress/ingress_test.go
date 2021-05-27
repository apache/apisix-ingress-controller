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
package ingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
)

func TestIsIngressEffective(t *testing.T) {
	c := &ingressController{
		controller: &Controller{
			cfg: config.NewDefaultConfig(),
		},
	}
	cn := "ingress"
	ingV1 := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "networking/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "v1-ing",
			Annotations: map[string]string{
				_ingressKey: "apisix",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &cn,
		},
	}
	ing, err := kube.NewIngress(ingV1)
	assert.Nil(t, err)
	// Annotations takes precedence.
	assert.Equal(t, c.isIngressEffective(ing), true)

	ingV1 = &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "networking/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "v1-ing",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &cn,
		},
	}
	ing, err = kube.NewIngress(ingV1)
	assert.Nil(t, err)
	// Spec.IngressClassName takes the precedence.
	assert.Equal(t, c.isIngressEffective(ing), false)

	ingV1beta1 := &networkingv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "networking/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "v1beta1-ing",
			Annotations: map[string]string{
				_ingressKey: "apisix",
			},
		},
		Spec: networkingv1beta1.IngressSpec{
			IngressClassName: &cn,
		},
	}
	ing, err = kube.NewIngress(ingV1beta1)
	assert.Nil(t, err)
	// Annotations takes precedence.
	assert.Equal(t, c.isIngressEffective(ing), true)

	ingV1beta1 = &networkingv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "networking/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "v1beta1-ing",
		},
		Spec: networkingv1beta1.IngressSpec{
			IngressClassName: &cn,
		},
	}
	ing, err = kube.NewIngress(ingV1beta1)
	assert.Nil(t, err)
	// Spec.IngressClassName takes the precedence.
	assert.Equal(t, c.isIngressEffective(ing), false)

	ingV1 = &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "networking/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "v1-ing",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &cn,
		},
	}
	ing, err = kube.NewIngress(ingV1)
	assert.Nil(t, err)
	// Spec.IngressClassName takes the precedence.
	assert.Equal(t, c.isIngressEffective(ing), false)

	ingExtV1beta1 := &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "v1extbeta1-ing",
			Annotations: map[string]string{
				_ingressKey: "apisix",
			},
		},
		Spec: extensionsv1beta1.IngressSpec{
			IngressClassName: &cn,
		},
	}
	ing, err = kube.NewIngress(ingExtV1beta1)
	assert.Nil(t, err)
	// Annotations takes precedence.
	assert.Equal(t, c.isIngressEffective(ing), true)

	ingExtV1beta1 = &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "v1extbeta1-ing",
		},
		Spec: extensionsv1beta1.IngressSpec{
			IngressClassName: &cn,
		},
	}
	ing, err = kube.NewIngress(ingExtV1beta1)
	assert.Nil(t, err)
	// Spec.IngressClassName takes the precedence.
	assert.Equal(t, c.isIngressEffective(ing), false)
}
