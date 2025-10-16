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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
	sslvalidator "github.com/apache/apisix-ingress-controller/internal/webhook/v1/ssl"
)

// nolint:unused
// log is for logging in this package.
var gatewaylog = logf.Log.WithName("gateway-resource")

// SetupGatewayWebhookWithManager registers the webhook for Gateway in the manager.
func SetupGatewayWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&gatewayv1.Gateway{}).
		WithValidator(NewGatewayCustomValidator(mgr.GetClient())).
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
	Client  client.Client
	checker reference.Checker
}

var _ webhook.CustomValidator = &GatewayCustomValidator{}

func NewGatewayCustomValidator(c client.Client) *GatewayCustomValidator {
	return &GatewayCustomValidator{
		Client:  c,
		checker: reference.NewChecker(c, gatewaylog),
	}
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	gateway, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		return nil, fmt.Errorf("expected a Gateway object but got %T", obj)
	}
	gatewaylog.Info("Validation for Gateway upon creation", "name", gateway.GetName())

	managed, err := isGatewayManaged(ctx, v.Client, gateway)
	if err != nil {
		gatewaylog.Error(err, "failed to decide controller ownership", "name", gateway.GetName(), "namespace", gateway.GetNamespace())
		return nil, nil
	}
	if !managed {
		return nil, nil
	}

	detector := sslvalidator.NewConflictDetector(v.Client)
	conflicts := detector.DetectConflicts(ctx, gateway)
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%s", sslvalidator.FormatConflicts(conflicts))
	}

	warnings := v.warnIfMissingGatewayProxyForGateway(ctx, gateway)
	warnings = append(warnings, v.collectReferenceWarnings(ctx, gateway)...)

	return warnings, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	gateway, ok := newObj.(*gatewayv1.Gateway)
	if !ok {
		return nil, fmt.Errorf("expected a Gateway object for the newObj but got %T", newObj)
	}
	gatewaylog.Info("Validation for Gateway upon update", "name", gateway.GetName())

	managed, err := isGatewayManaged(ctx, v.Client, gateway)
	if err != nil {
		gatewaylog.Error(err, "failed to decide controller ownership", "name", gateway.GetName(), "namespace", gateway.GetNamespace())
		return nil, nil
	}
	if !managed {
		return nil, nil
	}

	detector := sslvalidator.NewConflictDetector(v.Client)
	conflicts := detector.DetectConflicts(ctx, gateway)
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%s", sslvalidator.FormatConflicts(conflicts))
	}

	warnings := v.warnIfMissingGatewayProxyForGateway(ctx, gateway)
	warnings = append(warnings, v.collectReferenceWarnings(ctx, gateway)...)

	return warnings, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Gateway.
func (v *GatewayCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *GatewayCustomValidator) collectReferenceWarnings(ctx context.Context, gateway *gatewayv1.Gateway) admission.Warnings {
	if gateway == nil {
		return nil
	}

	var warnings admission.Warnings
	secretVisited := make(map[types.NamespacedName]struct{})

	addSecretWarning := func(nn types.NamespacedName) {
		if nn.Name == "" || nn.Namespace == "" {
			return
		}
		if _, seen := secretVisited[nn]; seen {
			return
		}
		secretVisited[nn] = struct{}{}
		warnings = append(warnings, v.checker.Secret(ctx, reference.SecretRef{
			Object:         gateway,
			NamespacedName: nn,
		})...)
	}

	for _, listener := range gateway.Spec.Listeners {
		if listener.TLS == nil {
			continue
		}
		for _, ref := range listener.TLS.CertificateRefs {
			if ref.Kind != nil && *ref.Kind != internaltypes.KindSecret {
				continue
			}
			if ref.Group != nil && string(*ref.Group) != corev1.GroupName {
				continue
			}
			nn := types.NamespacedName{
				Namespace: gateway.GetNamespace(),
				Name:      string(ref.Name),
			}
			if ref.Namespace != nil && *ref.Namespace != "" {
				nn.Namespace = string(*ref.Namespace)
			}
			addSecretWarning(nn)
		}
	}

	return warnings
}

func (v *GatewayCustomValidator) warnIfMissingGatewayProxyForGateway(ctx context.Context, gateway *gatewayv1.Gateway) admission.Warnings {
	var warnings admission.Warnings

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
