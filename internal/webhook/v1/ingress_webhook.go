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
	"slices"

	networkingk8siov1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var ingresslog = logf.Log.WithName("ingress-resource")

// unsupportedAnnotations contains all the APISIX Ingress annotations that are not supported in 2.0.0
// ref: https://apisix.apache.org/docs/ingress-controller/upgrade-guide/#limited-support-for-ingress-annotations
var unsupportedAnnotations = []string{
	"k8s.apisix.apache.org/use-regex",
	"k8s.apisix.apache.org/enable-websocket",
	"k8s.apisix.apache.org/plugin-config-name",
	"k8s.apisix.apache.org/upstream-scheme",
	"k8s.apisix.apache.org/upstream-retries",
	"k8s.apisix.apache.org/upstream-connect-timeout",
	"k8s.apisix.apache.org/upstream-read-timeout",
	"k8s.apisix.apache.org/upstream-send-timeout",
	"k8s.apisix.apache.org/enable-cors",
	"k8s.apisix.apache.org/cors-allow-origin",
	"k8s.apisix.apache.org/cors-allow-headers",
	"k8s.apisix.apache.org/cors-allow-methods",
	"k8s.apisix.apache.org/enable-csrf",
	"k8s.apisix.apache.org/csrf-key",
	"k8s.apisix.apache.org/http-to-https",
	"k8s.apisix.apache.org/http-redirect",
	"k8s.apisix.apache.org/http-redirect-code",
	"k8s.apisix.apache.org/rewrite-target",
	"k8s.apisix.apache.org/rewrite-target-regex",
	"k8s.apisix.apache.org/rewrite-target-regex-template",
	"k8s.apisix.apache.org/enable-response-rewrite",
	"k8s.apisix.apache.org/response-rewrite-status-code",
	"k8s.apisix.apache.org/response-rewrite-body",
	"k8s.apisix.apache.org/response-rewrite-body-base64",
	"k8s.apisix.apache.org/response-rewrite-add-header",
	"k8s.apisix.apache.org/response-rewrite-set-header",
	"k8s.apisix.apache.org/response-rewrite-remove-header",
	"k8s.apisix.apache.org/auth-uri",
	"k8s.apisix.apache.org/auth-ssl-verify",
	"k8s.apisix.apache.org/auth-request-headers",
	"k8s.apisix.apache.org/auth-upstream-headers",
	"k8s.apisix.apache.org/auth-client-headers",
	"k8s.apisix.apache.org/allowlist-source-range",
	"k8s.apisix.apache.org/blocklist-source-range",
	"k8s.apisix.apache.org/http-allow-methods",
	"k8s.apisix.apache.org/http-block-methods",
	"k8s.apisix.apache.org/auth-type",
	"k8s.apisix.apache.org/svc-namespace",
}

// checkUnsupportedAnnotations checks if the Ingress contains any unsupported annotations
// and returns appropriate warnings
func checkUnsupportedAnnotations(ingress *networkingk8siov1.Ingress) admission.Warnings {
	var warnings admission.Warnings

	if len(ingress.Annotations) == 0 {
		return warnings
	}

	for annotation := range ingress.Annotations {
		if slices.Contains(unsupportedAnnotations, annotation) {
			warningMsg := fmt.Sprintf("Annotation '%s' is not supported in APISIX Ingress Controller 2.0.0.", annotation)
			warnings = append(warnings, warningMsg)
			ingresslog.Info("Detected unsupported annotation",
				"ingress", ingress.GetName(),
				"namespace", ingress.GetNamespace(),
				"annotation", annotation)
		}
	}

	return warnings
}

// SetupIngressWebhookWithManager registers the webhook for Ingress in the manager.
func SetupIngressWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&networkingk8siov1.Ingress{}).
		WithValidator(&IngressCustomValidator{}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-networking-k8s-io-v1-ingress,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.k8s.io,resources=ingresses,verbs=create;update,versions=v1,name=vingress-v1.kb.io,admissionReviewVersions=v1

// IngressCustomValidator struct is responsible for validating the Ingress resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type IngressCustomValidator struct{}

var _ webhook.CustomValidator = &IngressCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Ingress.
func (v *IngressCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	ingress, ok := obj.(*networkingk8siov1.Ingress)
	if !ok {
		return nil, fmt.Errorf("expected a Ingress object but got %T", obj)
	}
	ingresslog.Info("Validation for Ingress upon creation", "name", ingress.GetName(), "namespace", ingress.GetNamespace())

	// Check for unsupported annotations and generate warnings
	warnings := checkUnsupportedAnnotations(ingress)

	return warnings, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Ingress.
func (v *IngressCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	ingress, ok := newObj.(*networkingk8siov1.Ingress)
	if !ok {
		return nil, fmt.Errorf("expected a Ingress object for the newObj but got %T", newObj)
	}
	ingresslog.Info("Validation for Ingress upon update", "name", ingress.GetName(), "namespace", ingress.GetNamespace())

	// Check for unsupported annotations and generate warnings
	warnings := checkUnsupportedAnnotations(ingress)

	return warnings, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Ingress.
func (v *IngressCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
