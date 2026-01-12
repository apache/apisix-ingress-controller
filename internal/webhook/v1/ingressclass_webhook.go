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

	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// nolint:unused
// log is for logging in this package.
var ingressclasslog = logf.Log.WithName("ingressclass-resource")

// SetupIngressClassWebhookWithManager registers the webhook for IngressClass in the manager.
func SetupIngressClassWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&networkingv1.IngressClass{}).
		WithValidator(&IngressClassCustomValidator{Client: mgr.GetClient()}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-networking-k8s-io-v1-ingressclass,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.k8s.io,resources=ingressclasses,verbs=create;update,versions=v1,name=vingressclass-v1.kb.io,admissionReviewVersions=v1,failurePolicy=Ignore

// IngressClassCustomValidator struct is responsible for validating the IngressClass resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type IngressClassCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &IngressClassCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type IngressClass.
func (v *IngressClassCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	ingressclass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		return nil, fmt.Errorf("expected a IngressClass object but got %T", obj)
	}
	ingressclasslog.Info("Validation for IngressClass upon creation", "name", ingressclass.GetName())

	warnings := v.warnIfMissingGatewayProxyForIngressClass(ctx, ingressclass)

	return warnings, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type IngressClass.
func (v *IngressClassCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	ingressclass, ok := newObj.(*networkingv1.IngressClass)
	if !ok {
		return nil, fmt.Errorf("expected a IngressClass object for the newObj but got %T", newObj)
	}
	ingressclasslog.Info("Validation for IngressClass upon update", "name", ingressclass.GetName())

	warnings := v.warnIfMissingGatewayProxyForIngressClass(ctx, ingressclass)

	return warnings, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type IngressClass.
func (v *IngressClassCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *IngressClassCustomValidator) warnIfMissingGatewayProxyForIngressClass(ctx context.Context, ingressClass *networkingv1.IngressClass) admission.Warnings {
	var warnings admission.Warnings

	// match controller
	if ingressClass.Spec.Controller != config.ControllerConfig.ControllerName {
		return nil
	}

	params := ingressClass.Spec.Parameters
	if params == nil || params.APIGroup == nil {
		return nil
	}
	if *params.APIGroup != v1alpha1.GroupVersion.Group || params.Kind != internaltypes.KindGatewayProxy {
		return nil
	}

	ns := utils.GetIngressClassParametersNamespace(*ingressClass)
	name := params.Name

	var gp v1alpha1.GatewayProxy
	if err := v.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &gp); err != nil {
		if k8serrors.IsNotFound(err) {
			msg := fmt.Sprintf("Referenced GatewayProxy '%s/%s' not found.", ns, name)
			warnings = append(warnings, msg)
			ingressclasslog.Info("IngressClass references missing GatewayProxy", "ingressclass", ingressClass.GetName(), "namespace", ns, "gatewayproxy", name)
		} else {
			ingressclasslog.Error(err, "failed to resolve GatewayProxy for IngressClass", "ingressclass", ingressClass.GetName(), "namespace", ns, "gatewayproxy", name)
		}
	}
	return warnings
}
