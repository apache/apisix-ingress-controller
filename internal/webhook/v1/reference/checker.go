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

package reference

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// ServiceRef captures the information needed to validate a Service reference.
type ServiceRef struct {
	Object         client.Object
	NamespacedName types.NamespacedName
}

// SecretRef captures the information needed to validate a Secret reference.
type SecretRef struct {
	Object         client.Object
	NamespacedName types.NamespacedName
	Key            *string
}

// Checker performs reference lookups and returns admission warnings on failure.
type Checker struct {
	client client.Client
	log    logr.Logger
}

// NewChecker constructs a Checker instance.
func NewChecker(c client.Client, log logr.Logger) Checker {
	return Checker{client: c, log: log}
}

// Service ensures the referenced Service exists and returns warnings when it does not.
func (c Checker) Service(ctx context.Context, ref ServiceRef) admission.Warnings {
	if ref.NamespacedName.Name == "" || ref.NamespacedName.Namespace == "" {
		return nil
	}

	var svc corev1.Service
	if err := c.client.Get(ctx, ref.NamespacedName, &svc); err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Referenced Service '%s/%s' not found", ref.NamespacedName.Namespace, ref.NamespacedName.Name)
			return admission.Warnings{msg}
		}
		c.log.Error(err, "Failed to get Service",
			"ownerKind", ref.Object.GetObjectKind().GroupVersionKind().Kind,
			"ownerNamespace", ref.Object.GetNamespace(),
			"ownerName", ref.Object.GetName(),
			"serviceNamespace", ref.NamespacedName.Namespace,
			"serviceName", ref.NamespacedName.Name,
		)
	}
	return nil
}

// Secret ensures the referenced Secret (and optional key) exists and returns warnings when missing.
func (c Checker) Secret(ctx context.Context, ref SecretRef) admission.Warnings {
	if ref.NamespacedName.Name == "" || ref.NamespacedName.Namespace == "" {
		return nil
	}

	var secret corev1.Secret
	if err := c.client.Get(ctx, ref.NamespacedName, &secret); err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Referenced Secret '%s/%s' not found", ref.NamespacedName.Namespace, ref.NamespacedName.Name)
			return admission.Warnings{msg}
		}
		c.log.Error(err, "Failed to get Secret",
			"ownerKind", ref.Object.GetObjectKind().GroupVersionKind().Kind,
			"ownerNamespace", ref.Object.GetNamespace(),
			"ownerName", ref.Object.GetName(),
			"secretNamespace", ref.NamespacedName.Namespace,
			"secretName", ref.NamespacedName.Name,
		)
		return nil
	}

	if ref.Key != nil {
		if _, ok := secret.Data[*ref.Key]; !ok {
			msg := fmt.Sprintf("Secret key '%s' not found in Secret '%s/%s'", *ref.Key, ref.NamespacedName.Namespace, ref.NamespacedName.Name)
			return admission.Warnings{msg}
		}
	}

	return nil
}
