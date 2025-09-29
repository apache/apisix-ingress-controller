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

	apisixv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
)

var apisixTlsLog = logf.Log.WithName("apisixtls-resource")

func SetupApisixTlsWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&apisixv2.ApisixTls{}).
		WithValidator(NewApisixTlsCustomValidator(mgr.GetClient())).
		Complete()
}

// +kubebuilder:webhook:path=/validate-apisix-apache-org-v2-apisixtls,mutating=false,failurePolicy=fail,sideEffects=None,groups=apisix.apache.org,resources=apisixtlses,verbs=create;update,versions=v2,name=vapisixtls-v2.kb.io,admissionReviewVersions=v1

type ApisixTlsCustomValidator struct {
	Client  client.Client
	checker reference.Checker
}

var _ webhook.CustomValidator = &ApisixTlsCustomValidator{}

func NewApisixTlsCustomValidator(c client.Client) *ApisixTlsCustomValidator {
	return &ApisixTlsCustomValidator{
		Client:  c,
		checker: reference.NewChecker(c, apisixTlsLog),
	}
}

func (v *ApisixTlsCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	tls, ok := obj.(*apisixv2.ApisixTls)
	if !ok {
		return nil, fmt.Errorf("expected an ApisixTls object but got %T", obj)
	}
	apisixTlsLog.Info("Validation for ApisixTls upon creation", "name", tls.GetName(), "namespace", tls.GetNamespace())

	return v.collectWarnings(ctx, tls), nil
}

func (v *ApisixTlsCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	tls, ok := newObj.(*apisixv2.ApisixTls)
	if !ok {
		return nil, fmt.Errorf("expected an ApisixTls object for the newObj but got %T", newObj)
	}
	apisixTlsLog.Info("Validation for ApisixTls upon update", "name", tls.GetName(), "namespace", tls.GetNamespace())

	return v.collectWarnings(ctx, tls), nil
}

func (*ApisixTlsCustomValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *ApisixTlsCustomValidator) collectWarnings(ctx context.Context, tls *apisixv2.ApisixTls) admission.Warnings {
	var warnings admission.Warnings

	warnings = append(warnings, v.checker.Secret(ctx, reference.SecretRef{
		Object: tls,
		NamespacedName: types.NamespacedName{
			Namespace: tls.Spec.Secret.Namespace,
			Name:      tls.Spec.Secret.Name,
		},
	})...)

	if client := tls.Spec.Client; client != nil {
		warnings = append(warnings, v.checker.Secret(ctx, reference.SecretRef{
			Object: tls,
			NamespacedName: types.NamespacedName{
				Namespace: client.CASecret.Namespace,
				Name:      client.CASecret.Name,
			},
		})...)
	}

	return warnings
}
