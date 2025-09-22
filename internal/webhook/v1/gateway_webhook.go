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
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gatewaynetworkingk8siov1 "sigs.k8s.io/gateway-api/apis/v1"
)

// nolint:unused
// log is for logging in this package.
var gatewaylog = logf.Log.WithName("gateway-resource")

// SetupGatewayWebhookWithManager registers the webhook for Gateway in the manager.
func SetupGatewayWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&gatewaynetworkingk8siov1.Gateway{}).
		WithValidator(&GatewayCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1-gateway,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=gateways,verbs=create;update,versions=v1,name=vgateway-v1.kb.io,admissionReviewVersions=v1

// GatewayCustomValidator struct is responsible for validating the Gateway resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type GatewayCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &GatewayCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	gateway, ok := obj.(*gatewaynetworkingk8siov1.Gateway)
	if !ok {
		return nil, fmt.Errorf("expected a Gateway object but got %T", obj)
	}
	gatewaylog.Info("Validation for Gateway upon creation", "name", gateway.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	gateway, ok := newObj.(*gatewaynetworkingk8siov1.Gateway)
	if !ok {
		return nil, fmt.Errorf("expected a Gateway object for the newObj but got %T", newObj)
	}
	gatewaylog.Info("Validation for Gateway upon update", "name", gateway.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gateway, ok := obj.(*gatewaynetworkingk8siov1.Gateway)
	if !ok {
		return nil, fmt.Errorf("expected a Gateway object but got %T", obj)
	}
	gatewaylog.Info("Validation for Gateway upon deletion", "name", gateway.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
