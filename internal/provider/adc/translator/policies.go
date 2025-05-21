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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
)

func convertBackendRef(namespace, name, kind string) gatewayv1.BackendRef {
	backendRef := gatewayv1.BackendRef{}
	backendRef.Name = gatewayv1.ObjectName(name)
	backendRef.Namespace = ptr.To(gatewayv1.Namespace(namespace))
	backendRef.Kind = ptr.To(gatewayv1.Kind(kind))
	return backendRef
}

func (t *Translator) AttachBackendTrafficPolicyToUpstream(ref gatewayv1.BackendRef, policies map[types.NamespacedName]*v1alpha1.BackendTrafficPolicy, upstream *adctypes.Upstream) {
	if len(policies) == 0 {
		return
	}
	var policy *v1alpha1.BackendTrafficPolicy
	for _, po := range policies {
		if ref.Namespace != nil && string(*ref.Namespace) != po.Namespace {
			continue
		}
		for _, targetRef := range po.Spec.TargetRefs {
			if ref.Name == targetRef.Name {
				policy = po
				break
			}
		}
	}
	if policy == nil {
		return
	}
	t.attachBackendTrafficPolicyToUpstream(policy, upstream)
}

func (t *Translator) attachBackendTrafficPolicyToUpstream(policy *v1alpha1.BackendTrafficPolicy, upstream *adctypes.Upstream) {
	if policy == nil {
		return
	}
	upstream.PassHost = policy.Spec.PassHost
	upstream.UpstreamHost = string(policy.Spec.Host)
	upstream.Scheme = policy.Spec.Scheme
	if policy.Spec.Retries != nil {
		upstream.Retries = new(int64)
		*upstream.Retries = int64(*policy.Spec.Retries)
	}
	if policy.Spec.Timeout != nil {
		upstream.Timeout = &adctypes.Timeout{
			Connect: int(policy.Spec.Timeout.Connect.Seconds()),
			Read:    int(policy.Spec.Timeout.Read.Seconds()),
			Send:    int(policy.Spec.Timeout.Send.Seconds()),
		}
	}
	if policy.Spec.LoadBalancer != nil {
		upstream.Type = adctypes.UpstreamType(policy.Spec.LoadBalancer.Type)
		upstream.HashOn = policy.Spec.LoadBalancer.HashOn
		upstream.Key = policy.Spec.LoadBalancer.Key
	}
}
