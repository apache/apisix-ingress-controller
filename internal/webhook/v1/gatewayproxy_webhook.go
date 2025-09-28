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

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
)

var gatewayProxyLog = logf.Log.WithName("gatewayproxy-resource")

func SetupGatewayProxyWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.GatewayProxy{}).
		WithValidator(NewGatewayProxyCustomValidator(mgr.GetClient())).
		Complete()
}

// +kubebuilder:webhook:path=/validate-apisix-apache-org-v1alpha1-gatewayproxy,mutating=false,failurePolicy=fail,sideEffects=None,groups=apisix.apache.org,resources=gatewayproxies,verbs=create;update,versions=v1alpha1,name=vgatewayproxy-v1alpha1.kb.io,admissionReviewVersions=v1

type GatewayProxyCustomValidator struct {
	Client  client.Client
	checker reference.Checker
}

var _ webhook.CustomValidator = &GatewayProxyCustomValidator{}

func NewGatewayProxyCustomValidator(c client.Client) *GatewayProxyCustomValidator {
	return &GatewayProxyCustomValidator{
		Client:  c,
		checker: reference.NewChecker(c, gatewayProxyLog),
	}
}

func (v *GatewayProxyCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gp, ok := obj.(*v1alpha1.GatewayProxy)
	if !ok {
		return nil, fmt.Errorf("expected a GatewayProxy object but got %T", obj)
	}
	gatewayProxyLog.Info("Validation for GatewayProxy upon creation", "name", gp.GetName(), "namespace", gp.GetNamespace())

	return v.collectWarnings(ctx, gp), nil
}

func (v *GatewayProxyCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	gp, ok := newObj.(*v1alpha1.GatewayProxy)
	if !ok {
		return nil, fmt.Errorf("expected a GatewayProxy object for the newObj but got %T", newObj)
	}
	gatewayProxyLog.Info("Validation for GatewayProxy upon update", "name", gp.GetName(), "namespace", gp.GetNamespace())

	return v.collectWarnings(ctx, gp), nil
}

func (v *GatewayProxyCustomValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *GatewayProxyCustomValidator) collectWarnings(ctx context.Context, gp *v1alpha1.GatewayProxy) admission.Warnings {
	var warnings admission.Warnings

	if gp.Spec.Provider != nil && gp.Spec.Provider.ControlPlane != nil {
		if svc := gp.Spec.Provider.ControlPlane.Service; svc != nil {
			warnings = append(warnings, v.checker.Service(ctx, reference.ServiceRef{
				Object: gp,
				NamespacedName: types.NamespacedName{
					Namespace: gp.GetNamespace(),
					Name:      svc.Name,
				},
			})...)
		}

		auth := gp.Spec.Provider.ControlPlane.Auth
		if auth.Type == v1alpha1.AuthTypeAdminKey && auth.AdminKey != nil && auth.AdminKey.ValueFrom != nil && auth.AdminKey.ValueFrom.SecretKeyRef != nil {
			secretRef := auth.AdminKey.ValueFrom.SecretKeyRef
			key := secretRef.Key
			warnings = append(warnings, v.checker.Secret(ctx, reference.SecretRef{
				Object: gp,
				NamespacedName: types.NamespacedName{
					Namespace: gp.GetNamespace(),
					Name:      secretRef.Name,
				},
				Key: &key,
			})...)
		}
	}

	return warnings
}
