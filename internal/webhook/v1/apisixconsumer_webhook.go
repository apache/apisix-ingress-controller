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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	apisixv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
)

var apisixConsumerLog = logf.Log.WithName("apisixconsumer-resource")

func SetupApisixConsumerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&apisixv2.ApisixConsumer{}).
		WithValidator(NewApisixConsumerCustomValidator(mgr.GetClient())).
		Complete()
}

// +kubebuilder:webhook:path=/validate-apisix-apache-org-v2-apisixconsumer,mutating=false,failurePolicy=fail,sideEffects=None,groups=apisix.apache.org,resources=apisixconsumers,verbs=create;update,versions=v2,name=vapisixconsumer-v2.kb.io,admissionReviewVersions=v1

type ApisixConsumerCustomValidator struct {
	Client  client.Client
	checker reference.Checker
}

var _ webhook.CustomValidator = &ApisixConsumerCustomValidator{}

func NewApisixConsumerCustomValidator(c client.Client) *ApisixConsumerCustomValidator {
	return &ApisixConsumerCustomValidator{
		Client:  c,
		checker: reference.NewChecker(c, apisixConsumerLog),
	}
}

func (v *ApisixConsumerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	consumer, ok := obj.(*apisixv2.ApisixConsumer)
	if !ok {
		return nil, fmt.Errorf("expected an ApisixConsumer object but got %T", obj)
	}
	apisixConsumerLog.Info("Validation for ApisixConsumer upon creation", "name", consumer.GetName(), "namespace", consumer.GetNamespace())

	if !controller.MatchesIngressClass(v.Client, apisixConsumerLog, consumer) {
		return nil, nil
	}

	return v.collectWarnings(ctx, consumer), nil
}

func (v *ApisixConsumerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	consumer, ok := newObj.(*apisixv2.ApisixConsumer)
	if !ok {
		return nil, fmt.Errorf("expected an ApisixConsumer object for the newObj but got %T", newObj)
	}
	apisixConsumerLog.Info("Validation for ApisixConsumer upon update", "name", consumer.GetName(), "namespace", consumer.GetNamespace())
	if !controller.MatchesIngressClass(v.Client, apisixConsumerLog, consumer) {
		return nil, nil
	}

	return v.collectWarnings(ctx, consumer), nil
}

func (*ApisixConsumerCustomValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *ApisixConsumerCustomValidator) collectWarnings(ctx context.Context, consumer *apisixv2.ApisixConsumer) admission.Warnings {
	namespace := consumer.GetNamespace()
	var warnings admission.Warnings

	addSecretWarning := func(ref *corev1.LocalObjectReference) {
		if ref == nil || ref.Name == "" {
			return
		}

		warnings = append(warnings, v.checker.Secret(ctx, reference.SecretRef{
			Object: consumer,
			NamespacedName: types.NamespacedName{
				Namespace: namespace,
				Name:      ref.Name,
			},
		})...)
	}

	params := consumer.Spec.AuthParameter
	if params.BasicAuth != nil {
		addSecretWarning(params.BasicAuth.SecretRef)
	}
	if params.KeyAuth != nil {
		addSecretWarning(params.KeyAuth.SecretRef)
	}
	if params.WolfRBAC != nil {
		addSecretWarning(params.WolfRBAC.SecretRef)
	}
	if params.JwtAuth != nil {
		addSecretWarning(params.JwtAuth.SecretRef)
	}
	if params.HMACAuth != nil {
		addSecretWarning(params.HMACAuth.SecretRef)
	}
	if params.LDAPAuth != nil {
		addSecretWarning(params.LDAPAuth.SecretRef)
	}

	return warnings
}
