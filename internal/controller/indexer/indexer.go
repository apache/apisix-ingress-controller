// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package indexer

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
)

const (
	ServiceIndexRef           = "serviceRefs"
	ExtensionRef              = "extensionRef"
	ParametersRef             = "parametersRef"
	ParentRefs                = "parentRefs"
	IngressClass              = "ingressClass"
	SecretIndexRef            = "secretRefs"
	IngressClassRef           = "ingressClassRef"
	IngressClassParametersRef = "ingressClassParametersRef"
	ConsumerGatewayRef        = "consumerGatewayRef"
	PolicyTargetRefs          = "targetRefs"
	GatewayClassIndexRef      = "gatewayClassRef"
)

func SetupIndexer(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		setupGatewayIndexer,
		setupHTTPRouteIndexer,
		setupIngressIndexer,
		setupConsumerIndexer,
		setupBackendTrafficPolicyIndexer,
		setupIngressClassIndexer,
		setupGatewayProxyIndexer,
		setupGatewaySecretIndex,
		setupGatewayClassIndexer,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}

func setupGatewayIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.Gateway{},
		ParametersRef,
		GatewayParametersRefIndexFunc,
	); err != nil {
		return err
	}
	return nil
}

func setupConsumerIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v1alpha1.Consumer{},
		ConsumerGatewayRef,
		ConsumerGatewayRefIndexFunc,
	); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v1alpha1.Consumer{},
		SecretIndexRef,
		ConsumerSecretIndexFunc,
	); err != nil {
		return err
	}
	return nil
}

func ConsumerSecretIndexFunc(rawObj client.Object) []string {
	consumer := rawObj.(*v1alpha1.Consumer)
	secretKeys := make([]string, 0)

	for _, credential := range consumer.Spec.Credentials {
		if credential.SecretRef == nil {
			continue
		}
		ns := consumer.GetNamespace()
		if credential.SecretRef.Namespace != nil {
			ns = *credential.SecretRef.Namespace
		}
		key := GenIndexKey(ns, credential.SecretRef.Name)
		secretKeys = append(secretKeys, key)
	}
	return secretKeys
}

func ConsumerGatewayRefIndexFunc(rawObj client.Object) []string {
	consumer := rawObj.(*v1alpha1.Consumer)

	if consumer.Spec.GatewayRef.Name == "" {
		return nil
	}

	ns := consumer.GetNamespace()
	if consumer.Spec.GatewayRef.Namespace != nil {
		ns = *consumer.Spec.GatewayRef.Namespace
	}
	return []string{GenIndexKey(ns, consumer.Spec.GatewayRef.Name)}
}

func setupHTTPRouteIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.HTTPRoute{},
		ParentRefs,
		HTTPRouteParentRefsIndexFunc,
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.HTTPRoute{},
		ExtensionRef,
		HTTPRouteExtensionIndexFunc,
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.HTTPRoute{},
		ServiceIndexRef,
		HTTPRouteServiceIndexFunc,
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v1alpha1.HTTPRoutePolicy{},
		PolicyTargetRefs,
		HTTPRoutePolicyIndexFunc,
	); err != nil {
		return err
	}

	return nil
}

func setupIngressClassIndexer(mgr ctrl.Manager) error {
	// create IngressClass index
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.IngressClass{},
		IngressClass,
		IngressClassIndexFunc,
	); err != nil {
		return err
	}

	// create IngressClassParametersRef index
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.IngressClass{},
		IngressClassParametersRef,
		IngressClassParametersRefIndexFunc,
	); err != nil {
		return err
	}

	return nil
}

func setupGatewayProxyIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v1alpha1.GatewayProxy{},
		SecretIndexRef,
		GatewayProxySecretIndexFunc,
	); err != nil {
		return err
	}
	return nil
}

func setupGatewaySecretIndex(mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.Gateway{},
		SecretIndexRef,
		GatewaySecretIndexFunc,
	)
}

func setupGatewayClassIndexer(mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&gatewayv1.Gateway{},
		GatewayClassIndexRef,
		func(obj client.Object) (requests []string) {
			return []string{string(obj.(*gatewayv1.Gateway).Spec.GatewayClassName)}
		},
	)
}

func GatewayProxySecretIndexFunc(rawObj client.Object) []string {
	gatewayProxy := rawObj.(*v1alpha1.GatewayProxy)
	secretKeys := make([]string, 0)

	// Check if provider is ControlPlane type and has AdminKey auth
	if gatewayProxy.Spec.Provider != nil &&
		gatewayProxy.Spec.Provider.Type == v1alpha1.ProviderTypeControlPlane &&
		gatewayProxy.Spec.Provider.ControlPlane != nil &&
		gatewayProxy.Spec.Provider.ControlPlane.Auth.Type == v1alpha1.AuthTypeAdminKey &&
		gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey != nil &&
		gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom != nil &&
		gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef != nil {

		ref := gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef
		ns := gatewayProxy.GetNamespace()
		key := GenIndexKey(ns, ref.Name)
		secretKeys = append(secretKeys, key)
	}
	return secretKeys
}

func setupIngressIndexer(mgr ctrl.Manager) error {
	// create IngressClass index
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.Ingress{},
		IngressClassRef,
		IngressClassRefIndexFunc,
	); err != nil {
		return err
	}

	// create Service index for quick lookup of Ingresses using specific services
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.Ingress{},
		ServiceIndexRef,
		IngressServiceIndexFunc,
	); err != nil {
		return err
	}

	// create secret index for TLS
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&networkingv1.Ingress{},
		SecretIndexRef,
		IngressSecretIndexFunc,
	); err != nil {
		return err
	}

	return nil
}

func setupBackendTrafficPolicyIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&v1alpha1.BackendTrafficPolicy{},
		PolicyTargetRefs,
		BackendTrafficPolicyIndexFunc,
	); err != nil {
		return err
	}
	return nil
}

func IngressClassIndexFunc(rawObj client.Object) []string {
	ingressClass := rawObj.(*networkingv1.IngressClass)
	if ingressClass.Spec.Controller == "" {
		return nil
	}
	controllerName := ingressClass.Spec.Controller
	return []string{controllerName}
}

func IngressClassRefIndexFunc(rawObj client.Object) []string {
	ingress := rawObj.(*networkingv1.Ingress)
	if ingress.Spec.IngressClassName == nil {
		return nil
	}
	return []string{*ingress.Spec.IngressClassName}
}

func IngressServiceIndexFunc(rawObj client.Object) []string {
	ingress := rawObj.(*networkingv1.Ingress)
	var services []string

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}
			key := GenIndexKey(ingress.Namespace, path.Backend.Service.Name)
			services = append(services, key)
		}
	}
	return services
}

func IngressSecretIndexFunc(rawObj client.Object) []string {
	ingress := rawObj.(*networkingv1.Ingress)
	secrets := make([]string, 0)

	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		key := GenIndexKey(ingress.Namespace, tls.SecretName)
		secrets = append(secrets, key)
	}
	return secrets
}

func GatewaySecretIndexFunc(rawObj client.Object) (keys []string) {
	gateway := rawObj.(*gatewayv1.Gateway)
	var m = make(map[string]struct{})
	for _, listener := range gateway.Spec.Listeners {
		if listener.TLS == nil || len(listener.TLS.CertificateRefs) == 0 {
			continue
		}
		for _, ref := range listener.TLS.CertificateRefs {
			if ref.Kind == nil || *ref.Kind != "Secret" {
				continue
			}
			namespace := gateway.GetNamespace()
			if ref.Namespace != nil {
				namespace = string(*ref.Namespace)
			}
			key := GenIndexKey(namespace, string(ref.Name))
			if _, ok := m[key]; !ok {
				m[key] = struct{}{}
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func GenIndexKeyWithGK(group, kind, namespace, name string) string {
	gvk := schema.GroupKind{
		Group: group,
		Kind:  kind,
	}
	nsName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	return gvk.String() + "/" + nsName.String()
}

func GenIndexKey(namespace, name string) string {
	return client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}.String()
}

func HTTPRouteParentRefsIndexFunc(rawObj client.Object) []string {
	hr := rawObj.(*gatewayv1.HTTPRoute)
	keys := make([]string, 0, len(hr.Spec.ParentRefs))
	for _, ref := range hr.Spec.ParentRefs {
		ns := hr.GetNamespace()
		if ref.Namespace != nil {
			ns = string(*ref.Namespace)
		}
		keys = append(keys, GenIndexKey(ns, string(ref.Name)))
	}
	return keys
}

func HTTPRouteServiceIndexFunc(rawObj client.Object) []string {
	hr := rawObj.(*gatewayv1.HTTPRoute)
	keys := make([]string, 0, len(hr.Spec.Rules))
	for _, rule := range hr.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			namespace := hr.GetNamespace()
			if backend.Kind != nil && *backend.Kind != "Service" {
				continue
			}
			if backend.Namespace != nil {
				namespace = string(*backend.Namespace)
			}
			keys = append(keys, GenIndexKey(namespace, string(backend.Name)))
		}
	}
	return keys
}

func HTTPRouteExtensionIndexFunc(rawObj client.Object) []string {
	hr := rawObj.(*gatewayv1.HTTPRoute)
	keys := make([]string, 0, len(hr.Spec.Rules))
	for _, rule := range hr.Spec.Rules {
		for _, filter := range rule.Filters {
			if filter.Type != gatewayv1.HTTPRouteFilterExtensionRef || filter.ExtensionRef == nil {
				continue
			}
			if filter.ExtensionRef.Kind == "PluginConfig" {
				keys = append(keys, GenIndexKey(hr.GetNamespace(), string(filter.ExtensionRef.Name)))
			}
		}
	}
	return keys
}

func HTTPRoutePolicyIndexFunc(rawObj client.Object) []string {
	hrp := rawObj.(*v1alpha1.HTTPRoutePolicy)
	var keys = make([]string, 0, len(hrp.Spec.TargetRefs))
	var m = make(map[string]struct{})
	for _, ref := range hrp.Spec.TargetRefs {
		key := GenIndexKeyWithGK(string(ref.Group), string(ref.Kind), hrp.GetNamespace(), string(ref.Name))
		if _, ok := m[key]; !ok {
			m[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	return keys
}

func GatewayParametersRefIndexFunc(rawObj client.Object) []string {
	gw := rawObj.(*gatewayv1.Gateway)
	if gw.Spec.Infrastructure != nil && gw.Spec.Infrastructure.ParametersRef != nil {
		// now we only care about kind: GatewayProxy
		if gw.Spec.Infrastructure.ParametersRef.Kind == "GatewayProxy" {
			name := gw.Spec.Infrastructure.ParametersRef.Name
			return []string{GenIndexKey(gw.GetNamespace(), name)}
		}
	}
	return nil
}

func BackendTrafficPolicyIndexFunc(rawObj client.Object) []string {
	btp := rawObj.(*v1alpha1.BackendTrafficPolicy)
	keys := make([]string, 0, len(btp.Spec.TargetRefs))
	for _, ref := range btp.Spec.TargetRefs {
		keys = append(keys,
			GenIndexKeyWithGK(
				string(ref.Group),
				string(ref.Kind),
				btp.GetNamespace(),
				string(ref.Name),
			),
		)
	}
	return keys
}

func IngressClassParametersRefIndexFunc(rawObj client.Object) []string {
	ingressClass := rawObj.(*networkingv1.IngressClass)
	// check if the IngressClass references this gateway proxy
	if ingressClass.Spec.Parameters != nil &&
		ingressClass.Spec.Parameters.APIGroup != nil &&
		*ingressClass.Spec.Parameters.APIGroup == v1alpha1.GroupVersion.Group &&
		ingressClass.Spec.Parameters.Kind == "GatewayProxy" {
		ns := ingressClass.GetNamespace()
		if ingressClass.Spec.Parameters.Namespace != nil {
			ns = *ingressClass.Spec.Parameters.Namespace
		}
		return []string{GenIndexKey(ns, ingressClass.Spec.Parameters.Name)}
	}
	return nil
}
