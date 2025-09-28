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
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
)

var grpcRouteLog = logf.Log.WithName("grpcroute-resource")

func SetupGRPCRouteWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&gatewayv1.GRPCRoute{}).
		WithValidator(NewGRPCRouteCustomValidator(mgr.GetClient())).
		Complete()
}

// +kubebuilder:webhook:path=/validate-gateway-networking-k8s-io-v1-grpcroute,mutating=false,failurePolicy=fail,sideEffects=None,groups=gateway.networking.k8s.io,resources=grpcroutes,verbs=create;update,versions=v1,name=vgrpcroute-v1.kb.io,admissionReviewVersions=v1

type GRPCRouteCustomValidator struct {
	Client  client.Client
	checker reference.Checker
}

var _ webhook.CustomValidator = &GRPCRouteCustomValidator{}

func NewGRPCRouteCustomValidator(c client.Client) *GRPCRouteCustomValidator {
	return &GRPCRouteCustomValidator{
		Client:  c,
		checker: reference.NewChecker(c, grpcRouteLog),
	}
}

func (v *GRPCRouteCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	route, ok := obj.(*gatewayv1.GRPCRoute)
	if !ok {
		return nil, fmt.Errorf("expected a GRPCRoute object but got %T", obj)
	}
	grpcRouteLog.Info("Validation for GRPCRoute upon creation", "name", route.GetName(), "namespace", route.GetNamespace())

	return v.collectWarnings(ctx, route), nil
}

func (v *GRPCRouteCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	route, ok := newObj.(*gatewayv1.GRPCRoute)
	if !ok {
		return nil, fmt.Errorf("expected a GRPCRoute object for the newObj but got %T", newObj)
	}
	grpcRouteLog.Info("Validation for GRPCRoute upon update", "name", route.GetName(), "namespace", route.GetNamespace())

	return v.collectWarnings(ctx, route), nil
}

func (*GRPCRouteCustomValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *GRPCRouteCustomValidator) collectWarnings(ctx context.Context, route *gatewayv1.GRPCRoute) admission.Warnings {
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

	addBackendRef := func(ns string, name string, group *gatewayv1.Group, kind *gatewayv1.Kind) {
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

	processFilters := func(filters []gatewayv1.GRPCRouteFilter) {
		for _, filter := range filters {
			if filter.RequestMirror != nil {
				targetNamespace := namespace
				if filter.RequestMirror.BackendRef.Namespace != nil && *filter.RequestMirror.BackendRef.Namespace != "" {
					targetNamespace = string(*filter.RequestMirror.BackendRef.Namespace)
				}
				addBackendRef(targetNamespace, string(filter.RequestMirror.BackendRef.Name),
					filter.RequestMirror.BackendRef.Group, filter.RequestMirror.BackendRef.Kind)
			}
		}
	}

	for _, rule := range route.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			targetNamespace := namespace
			if backend.Namespace != nil && *backend.Namespace != "" {
				targetNamespace = string(*backend.Namespace)
			}
			addBackendRef(targetNamespace, string(backend.Name), backend.Group, backend.Kind)
			processFilters(backend.Filters)
		}

		processFilters(rule.Filters)
	}

	return warnings
}
