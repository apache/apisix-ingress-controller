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

	networkingk8siov1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var ingresslog = logf.Log.WithName("ingress-resource")

// SetupIngressWebhookWithManager registers the webhook for Ingress in the manager.
func SetupIngressWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&networkingk8siov1.Ingress{}).
		WithValidator(&IngressCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-networking-k8s-io-v1-ingress,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.k8s.io,resources=ingresses,verbs=create;update,versions=v1,name=vingress-v1.kb.io,admissionReviewVersions=v1

// IngressCustomValidator struct is responsible for validating the Ingress resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type IngressCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &IngressCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Ingress.
func (v *IngressCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	ingress, ok := obj.(*networkingk8siov1.Ingress)
	if !ok {
		return nil, fmt.Errorf("expected a Ingress object but got %T", obj)
	}
	ingresslog.Info("Validation for Ingress upon creation", "name", ingress.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Ingress.
func (v *IngressCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	ingress, ok := newObj.(*networkingk8siov1.Ingress)
	if !ok {
		return nil, fmt.Errorf("expected a Ingress object for the newObj but got %T", newObj)
	}
	ingresslog.Info("Validation for Ingress upon update", "name", ingress.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Ingress.
func (v *IngressCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	ingress, ok := obj.(*networkingk8siov1.Ingress)
	if !ok {
		return nil, fmt.Errorf("expected a Ingress object but got %T", obj)
	}
	ingresslog.Info("Validation for Ingress upon deletion", "name", ingress.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
