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

package provider

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

type Provider interface {
	Update(context.Context, *TranslateContext, client.Object) error
	Delete(context.Context, client.Object) error
	Sync(context.Context) error
	Start(context.Context) error
}

type TranslateContext struct {
	context.Context
	RouteParentRefs  []gatewayv1.ParentReference
	BackendRefs      []gatewayv1.BackendRef
	GatewayTLSConfig []gatewayv1.GatewayTLSConfig
	Credentials      []v1alpha1.Credential

	EndpointSlices         map[k8stypes.NamespacedName][]discoveryv1.EndpointSlice
	Secrets                map[k8stypes.NamespacedName]*corev1.Secret
	PluginConfigs          map[k8stypes.NamespacedName]*v1alpha1.PluginConfig
	Services               map[k8stypes.NamespacedName]*corev1.Service
	BackendTrafficPolicies map[k8stypes.NamespacedName]*v1alpha1.BackendTrafficPolicy
	GatewayProxies         map[types.NamespacedNameKind]v1alpha1.GatewayProxy
	ResourceParentRefs     map[types.NamespacedNameKind][]types.NamespacedNameKind
	HTTPRoutePolicies      []v1alpha1.HTTPRoutePolicy

	StatusUpdaters []status.Update
}

func NewDefaultTranslateContext(ctx context.Context) *TranslateContext {
	return &TranslateContext{
		Context:                ctx,
		EndpointSlices:         make(map[k8stypes.NamespacedName][]discoveryv1.EndpointSlice),
		Secrets:                make(map[k8stypes.NamespacedName]*corev1.Secret),
		PluginConfigs:          make(map[k8stypes.NamespacedName]*v1alpha1.PluginConfig),
		Services:               make(map[k8stypes.NamespacedName]*corev1.Service),
		BackendTrafficPolicies: make(map[k8stypes.NamespacedName]*v1alpha1.BackendTrafficPolicy),
		GatewayProxies:         make(map[types.NamespacedNameKind]v1alpha1.GatewayProxy),
		ResourceParentRefs:     make(map[types.NamespacedNameKind][]types.NamespacedNameKind),
	}
}
