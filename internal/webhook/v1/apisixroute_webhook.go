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
	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/webhook/v1/reference"
)

var apisixRouteLog = logf.Log.WithName("apisixroute-resource")

func SetupApisixRouteWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&apisixv2.ApisixRoute{}).
		WithValidator(NewApisixRouteCustomValidator(mgr.GetClient())).
		Complete()
}

// +kubebuilder:webhook:path=/validate-apisix-apache-org-v2-apisixroute,mutating=false,failurePolicy=fail,sideEffects=None,groups=apisix.apache.org,resources=apisixroutes,verbs=create;update,versions=v2,name=vapisixroute-v2.kb.io,admissionReviewVersions=v1

type ApisixRouteCustomValidator struct {
	Client  client.Client
	checker reference.Checker
}

var _ webhook.CustomValidator = &ApisixRouteCustomValidator{}

func NewApisixRouteCustomValidator(c client.Client) *ApisixRouteCustomValidator {
	return &ApisixRouteCustomValidator{
		Client:  c,
		checker: reference.NewChecker(c, apisixRouteLog),
	}
}

func (v *ApisixRouteCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	route, ok := obj.(*apisixv2.ApisixRoute)
	if !ok {
		return nil, fmt.Errorf("expected an ApisixRoute object but got %T", obj)
	}
	apisixRouteLog.Info("Validation for ApisixRoute upon creation", "name", route.GetName(), "namespace", route.GetNamespace())
	if !controller.MatchesIngressClass(v.Client, apisixRouteLog, route) {
		return nil, nil
	}

	return v.collectWarnings(ctx, route), nil
}

func (v *ApisixRouteCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	route, ok := newObj.(*apisixv2.ApisixRoute)
	if !ok {
		return nil, fmt.Errorf("expected an ApisixRoute object for the newObj but got %T", newObj)
	}
	apisixRouteLog.Info("Validation for ApisixRoute upon update", "name", route.GetName(), "namespace", route.GetNamespace())
	if !controller.MatchesIngressClass(v.Client, apisixRouteLog, route) {
		return nil, nil
	}

	return v.collectWarnings(ctx, route), nil
}

func (*ApisixRouteCustomValidator) ValidateDelete(context.Context, runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (v *ApisixRouteCustomValidator) collectWarnings(ctx context.Context, route *apisixv2.ApisixRoute) admission.Warnings {
	namespace := route.GetNamespace()

	serviceVisited := make(map[types.NamespacedName]struct{})
	secretVisited := make(map[types.NamespacedName]struct{})

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

	addSecretWarning := func(nn types.NamespacedName) {
		if nn.Name == "" || nn.Namespace == "" {
			return
		}
		if _, seen := secretVisited[nn]; seen {
			return
		}
		secretVisited[nn] = struct{}{}
		warnings = append(warnings, v.checker.Secret(ctx, reference.SecretRef{
			Object:         route,
			NamespacedName: nn,
		})...)
	}

	for _, rule := range route.Spec.HTTP {
		for _, backend := range rule.Backends {
			addServiceWarning(types.NamespacedName{Namespace: namespace, Name: backend.ServiceName})
		}
		for _, plugin := range rule.Plugins {
			if !plugin.Enable {
				continue
			}
			if plugin.SecretRef != "" {
				addSecretWarning(types.NamespacedName{Namespace: namespace, Name: plugin.SecretRef})
			}
		}
	}

	for _, rule := range route.Spec.Stream {
		addServiceWarning(types.NamespacedName{Namespace: namespace, Name: rule.Backend.ServiceName})
		for _, plugin := range rule.Plugins {
			if !plugin.Enable {
				continue
			}
			if plugin.SecretRef != "" {
				addSecretWarning(types.NamespacedName{Namespace: namespace, Name: plugin.SecretRef})
			}
		}
	}

	return warnings
}
