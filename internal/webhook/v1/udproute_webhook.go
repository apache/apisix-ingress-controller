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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
)

var udpRouteLog = logf.Log.WithName("udproute-resource")

func SetupUDPRouteWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&gatewayv1alpha2.UDPRoute{}).
		WithValidator(NewUDPRouteCustomValidator(mgr.GetClient())).
		Complete()
}

// +kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1alpha2-udproute,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=udproutes,verbs=create;update,versions=v1alpha2,name=vudproute-v1alpha2.kb.io,admissionReviewVersions=v1

type UDPRouteCustomValidator struct {
	Client  client.Client
	checker reference.Checker
}

var _ webhook.CustomValidator = &UDPRouteCustomValidator{}

func NewUDPRouteCustomValidator(c client.Client) *UDPRouteCustomValidator {
	return &UDPRouteCustomValidator{
		Client:  c,
		checker: reference.NewChecker(c, udpRouteLog),
	}
}

func (v *UDPRouteCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	route, ok := obj.(*gatewayv1alpha2.UDPRoute)
	if !ok {
		return nil, fmt.Errorf("expected a UDPRoute object but got %T", obj)
	}
	udpRouteLog.Info("Validation for UDPRoute upon creation", "name", route.GetName(), "namespace", route.GetNamespace())
	managed, err := isUDPRouteManaged(ctx, v.Client, route)
	if err != nil {
		udpRouteLog.Error(err, "failed to decide controller ownership", "name", route.GetName(), "namespace", route.GetNamespace())
		return nil, nil
	}
	if !managed {
		return nil, nil
	}

	return v.collectWarnings(ctx, route), nil
}

func (v *UDPRouteCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	route, ok := newObj.(*gatewayv1alpha2.UDPRoute)
	if !ok {
		return nil, fmt.Errorf("expected a UDPRoute object for the newObj but got %T", newObj)
	}
	udpRouteLog.Info("Validation for UDPRoute upon update", "name", route.GetName(), "namespace", route.GetNamespace())
	managed, err := isUDPRouteManaged(ctx, v.Client, route)
	if err != nil {
		udpRouteLog.Error(err, "failed to decide controller ownership", "name", route.GetName(), "namespace", route.GetNamespace())
		return nil, nil
	}
	if !managed {
		return nil, nil
	}

	return v.collectWarnings(ctx, route), nil
}

func (*UDPRouteCustomValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *UDPRouteCustomValidator) collectWarnings(ctx context.Context, route *gatewayv1alpha2.UDPRoute) admission.Warnings {
	serviceVisited := make(map[types.NamespacedName]struct{})
	namespace := route.GetNamespace()

	var warnings admission.Warnings

	addServiceWarning := func(nn types.NamespacedName) {
		if nn.Name == "" || nn.Namespace == "" {
			return
		}
		if _, seen := serviceVisited[nn]; seen {
			return
		}
		serviceVisited[nn] = struct{}{}
		warnings = append(warnings, v.checker.Service(ctx, reference.ServiceRef{
			Object:         route,
			NamespacedName: nn,
		})...)
	}

	addBackendRef := func(ns, name string, group *gatewayv1alpha2.Group, kind *gatewayv1alpha2.Kind) {
		if name == "" {
			return
		}
		if group != nil && string(*group) != corev1.GroupName {
			return
		}
		if kind != nil && *kind != internaltypes.KindService {
			return
		}
		nn := types.NamespacedName{Namespace: ns, Name: name}
		addServiceWarning(nn)
	}

	for _, rule := range route.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			targetNamespace := namespace
			if backend.Namespace != nil && *backend.Namespace != "" {
				targetNamespace = string(*backend.Namespace)
			}
			addBackendRef(targetNamespace, string(backend.Name), backend.Group, backend.Kind)
		}
	}

	return warnings
}
