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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apisixv1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
)

var consumerLog = logf.Log.WithName("consumer-resource")

func SetupConsumerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&apisixv1alpha1.Consumer{}).
		WithValidator(NewConsumerCustomValidator(mgr.GetClient())).
		Complete()
}

// +kubebuilder:webhook:path=/validate-apisix-apache-org-v1alpha1-consumer,mutating=false,failurePolicy=fail,sideEffects=None,groups=apisix.apache.org,resources=consumers,verbs=create;update,versions=v1alpha1,name=vconsumer-v1alpha1.kb.io,admissionReviewVersions=v1

type ConsumerCustomValidator struct {
	Client  client.Client
	checker reference.Checker
}

var _ webhook.CustomValidator = &ConsumerCustomValidator{}

func NewConsumerCustomValidator(c client.Client) *ConsumerCustomValidator {
	return &ConsumerCustomValidator{
		Client:  c,
		checker: reference.NewChecker(c, consumerLog),
	}
}

func (v *ConsumerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	consumer, ok := obj.(*apisixv1alpha1.Consumer)
	if !ok {
		return nil, fmt.Errorf("expected a Consumer object but got %T", obj)
	}
	consumerLog.Info("Validation for Consumer upon creation", "name", consumer.GetName(), "namespace", consumer.GetNamespace())

	return v.collectWarnings(ctx, consumer), nil
}

func (v *ConsumerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	consumer, ok := newObj.(*apisixv1alpha1.Consumer)
	if !ok {
		return nil, fmt.Errorf("expected a Consumer object for the newObj but got %T", newObj)
	}
	consumerLog.Info("Validation for Consumer upon update", "name", consumer.GetName(), "namespace", consumer.GetNamespace())

	return v.collectWarnings(ctx, consumer), nil
}

func (*ConsumerCustomValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *ConsumerCustomValidator) collectWarnings(ctx context.Context, consumer *apisixv1alpha1.Consumer) admission.Warnings {
	defaultNamespace := consumer.GetNamespace()

	visited := make(map[types.NamespacedName]struct{})
	var warnings admission.Warnings

	for _, credential := range consumer.Spec.Credentials {
		if credential.SecretRef == nil || credential.SecretRef.Name == "" {
			continue
		}

		namespace := defaultNamespace
		if credential.SecretRef.Namespace != nil && *credential.SecretRef.Namespace != "" {
			namespace = *credential.SecretRef.Namespace
		}

		nn := types.NamespacedName{Namespace: namespace, Name: credential.SecretRef.Name}
		if _, ok := visited[nn]; ok {
			continue
		}
		visited[nn] = struct{}{}

		warnings = append(warnings, v.checker.Secret(ctx, reference.SecretRef{
			Object:         consumer,
			NamespacedName: nn,
		})...)
	}

	return warnings
}
