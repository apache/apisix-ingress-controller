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

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gatewaynetworkingk8siov1 "sigs.k8s.io/gateway-api/apis/v1"

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

// nolint:unused
// log is for logging in this package.
var gatewaylog = logf.Log.WithName("gateway-resource")

// SetupGatewayWebhookWithManager registers the webhook for Gateway in the manager.
func SetupGatewayWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&gatewaynetworkingk8siov1.Gateway{}).
		WithValidator(&GatewayCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1-gateway,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=gateways,verbs=create;update,versions=v1,name=vgateway-v1.kb.io,admissionReviewVersions=v1

// GatewayCustomValidator struct is responsible for validating the Gateway resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type GatewayCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &GatewayCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gateway, ok := obj.(*gatewaynetworkingk8siov1.Gateway)
	if !ok {
		return nil, fmt.Errorf("expected a Gateway object but got %T", obj)
	}
	gatewaylog.Info("Validation for Gateway upon creation", "name", gateway.GetName())

	warnings := v.warnIfMissingGatewayProxyForGateway(ctx, gateway)

	return warnings, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	gateway, ok := newObj.(*gatewaynetworkingk8siov1.Gateway)
	if !ok {
		return nil, fmt.Errorf("expected a Gateway object for the newObj but got %T", newObj)
	}
	gatewaylog.Info("Validation for Gateway upon update", "name", gateway.GetName())

	warnings := v.warnIfMissingGatewayProxyForGateway(ctx, gateway)

	return warnings, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *GatewayCustomValidator) warnIfMissingGatewayProxyForGateway(ctx context.Context, gateway *gatewaynetworkingk8siov1.Gateway) admission.Warnings {
	var warnings admission.Warnings

	// get gateway class
	gatewayClass := &gatewaynetworkingk8siov1.GatewayClass{}
	if err := v.Client.Get(ctx, client.ObjectKey{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		gatewaylog.Error(err, "failed to get gateway class", "gateway", gateway.GetName(), "gatewayclass", gateway.Spec.GatewayClassName)
		return nil
	}
	// match controller
	if string(gatewayClass.Spec.ControllerName) != config.ControllerConfig.ControllerName {
		return nil
	}

	infra := gateway.Spec.Infrastructure
	if infra == nil || infra.ParametersRef == nil {
		return nil
	}
	ref := infra.ParametersRef
	if string(ref.Group) != v1alpha1.GroupVersion.Group || string(ref.Kind) != internaltypes.KindGatewayProxy {
		return nil
	}

	ns := gateway.GetNamespace()
	name := ref.Name

	var gp v1alpha1.GatewayProxy
	if err := v.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &gp); err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Referenced GatewayProxy '%s/%s' not found.", ns, name)
			warnings = append(warnings, msg)
			gatewaylog.Info("Gateway references missing GatewayProxy", "gateway", gateway.GetName(), "namespace", ns, "gatewayproxy", name)
		} else {
			gatewaylog.Error(err, "failed to resolve GatewayProxy for Gateway", "gateway", gateway.GetName(), "namespace", ns, "gatewayproxy", name)
		}
	}
	return warnings
}
