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

package translator

import (
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func (t *Translator) TranslateIngressClass(tctx *provider.TranslateContext, obj *networkingv1.IngressClass) (*TranslateResult, error) {
	result := &TranslateResult{}

	rk := provider.ResourceKind{
		Kind:      obj.Kind,
		Namespace: obj.Namespace,
		Name:      obj.Name,
	}
	gatewayProxy, ok := tctx.GatewayProxies[rk]
	if !ok {
		log.Debugw("no GatewayProxy found for IngressClass", zap.String("ingressclass", obj.Name))
		return result, nil
	}

	globalRules := make(adctypes.GlobalRule)
	pluginMetadata := make(adctypes.PluginMetadata)
	// apply plugins from GatewayProxy to global rules
	t.fillPluginsFromGatewayProxy(globalRules, &gatewayProxy)
	t.fillPluginMetadataFromGatewayProxy(pluginMetadata, &gatewayProxy)

	result.GlobalRules = globalRules
	result.PluginMetadata = pluginMetadata

	return result, nil
}
