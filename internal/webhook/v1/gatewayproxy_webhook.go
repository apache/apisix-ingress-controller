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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
)

var gatewayProxyLog = logf.Log.WithName("gatewayproxy-resource")

func SetupGatewayProxyWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.GatewayProxy{}).
		WithValidator(&GatewayProxyCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-apisix-apache-org-v1alpha1-gatewayproxy,mutating=false,failurePolicy=fail,sideEffects=None,groups=apisix.apache.org,resources=gatewayproxies,verbs=create;update,versions=v1alpha1,name=vgatewayproxy-v1alpha1.kb.io,admissionReviewVersions=v1

type GatewayProxyCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &GatewayProxyCustomValidator{}

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

	warnings = append(warnings, v.warnIfProviderServiceMissing(ctx, gp)...)
	warnings = append(warnings, v.warnIfAdminKeySecretMissing(ctx, gp)...)

	return warnings
}

func (v *GatewayProxyCustomValidator) warnIfProviderServiceMissing(ctx context.Context, gp *v1alpha1.GatewayProxy) admission.Warnings {
	if gp.Spec.Provider == nil || gp.Spec.Provider.ControlPlane == nil || gp.Spec.Provider.ControlPlane.Service == nil {
		return nil
	}

	svcRef := gp.Spec.Provider.ControlPlane.Service
	key := client.ObjectKey{Namespace: gp.GetNamespace(), Name: svcRef.Name}
	var svc corev1.Service
	if err := v.Client.Get(ctx, key, &svc); err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Referenced Service '%s/%s' not found at spec.provider.controlPlane.service", key.Namespace, key.Name)
			gatewayProxyLog.Info("GatewayProxy references missing Service", "gatewayproxy", gp.GetName(), "namespace", key.Namespace, "service", key.Name)
			return admission.Warnings{msg}
		}
		gatewayProxyLog.Error(err, "failed to resolve Service for GatewayProxy", "gatewayproxy", gp.GetName(), "namespace", key.Namespace, "service", key.Name)
	}
	return nil
}

func (v *GatewayProxyCustomValidator) warnIfAdminKeySecretMissing(ctx context.Context, gp *v1alpha1.GatewayProxy) admission.Warnings {
	if gp.Spec.Provider == nil || gp.Spec.Provider.ControlPlane == nil {
		return nil
	}

	auth := gp.Spec.Provider.ControlPlane.Auth
	if auth.Type != v1alpha1.AuthTypeAdminKey || auth.AdminKey == nil || auth.AdminKey.ValueFrom == nil || auth.AdminKey.ValueFrom.SecretKeyRef == nil {
		return nil
	}

	ref := auth.AdminKey.ValueFrom.SecretKeyRef
	key := client.ObjectKey{Namespace: gp.GetNamespace(), Name: ref.Name}
	var secret corev1.Secret
	if err := v.Client.Get(ctx, key, &secret); err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Referenced Secret '%s/%s' not found at spec.provider.controlPlane.auth.adminKey.valueFrom.secretKeyRef", key.Namespace, key.Name)
			gatewayProxyLog.Info("GatewayProxy references missing Secret", "gatewayproxy", gp.GetName(), "namespace", key.Namespace, "secret", key.Name)
			return admission.Warnings{msg}
		}
		gatewayProxyLog.Error(err, "failed to resolve Secret for GatewayProxy", "gatewayproxy", gp.GetName(), "namespace", key.Namespace, "secret", key.Name)
		return nil
	}

	if _, ok := secret.Data[ref.Key]; !ok {
		msg := fmt.Sprintf("Secret key '%s' not found in Secret '%s/%s' at spec.provider.controlPlane.auth.adminKey.valueFrom.secretKeyRef", ref.Key, key.Namespace, key.Name)
		gatewayProxyLog.Info("GatewayProxy references Secret without required key", "gatewayproxy", gp.GetName(), "namespace", key.Namespace, "secret", key.Name, "key", ref.Key)
		return admission.Warnings{msg}
	}

	return nil
}
