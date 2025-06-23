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

package manager

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/apache/apisix-ingress-controller/internal/controller"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// K8s
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="discovery.k8s.io",resources=endpointslices,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// CustomResourceDefinition v2
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixconsumers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixglobalrules,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixpluginconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixroutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixtls,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixupstreams,verbs=get;list;watch

// CustomResourceDefinition v2 status
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixconsumers/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixglobalrules/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixpluginconfigs/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixroutes/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixtls/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=apisixupstreams/status,verbs=get;update

// CustomResourceDefinition
// +kubebuilder:rbac:groups=apisix.apache.org,resources=pluginconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=gatewayproxies,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=consumers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=consumers/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=backendtrafficpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=backendtrafficpolicies/status,verbs=get;update
// +kubebuilder:rbac:groups=apisix.apache.org,resources=httproutepolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=apisix.apache.org,resources=httproutepolicies/status,verbs=get;update

// GatewayAPI
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes/status,verbs=get;update

// Networking
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch

type Controller interface {
	SetupWithManager(mgr manager.Manager) error
}

func setupControllers(ctx context.Context, mgr manager.Manager, pro provider.Provider, updater status.Updater) ([]Controller, error) {
	if err := indexer.SetupIndexer(mgr); err != nil {
		return nil, err
	}
	return []Controller{
		&controller.GatewayClassReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName("GatewayClass"),
			Updater: updater,
		},
		&controller.GatewayReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Gateway"),
			Provider: pro,
			Updater:  updater,
		},
		&controller.HTTPRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("HTTPRoute"),
			Provider: pro,
			Updater:  updater,
		},
		&controller.IngressReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Ingress"),
			Provider: pro,
			Updater:  updater,
		},
		&controller.ConsumerReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("Consumer"),
			Provider: pro,
			Updater:  updater,
		},
		&controller.IngressClassReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("IngressClass"),
			Provider: pro,
		},
		&controller.ApisixGlobalRuleReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixGlobalRule"),
			Provider: pro,
			Updater:  updater,
		},
		&controller.ApisixRouteReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixRoute"),
			Provider: pro,
			Updater:  updater,
		},
		&controller.ApisixConsumerReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixConsumer"),
			Provider: pro,
			Updater:  updater,
		},
		&controller.ApisixPluginConfigReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixPluginConfig"),
			Updater: updater,
		},
		&controller.ApisixTlsReconciler{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Log:      ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixTls"),
			Provider: pro,
			Updater:  updater,
		},
		&controller.ApisixUpstreamReconciler{
			Client:  mgr.GetClient(),
			Scheme:  mgr.GetScheme(),
			Log:     ctrl.LoggerFrom(ctx).WithName("controllers").WithName("ApisixUpstream"),
			Updater: updater,
		},
	}, nil
}
