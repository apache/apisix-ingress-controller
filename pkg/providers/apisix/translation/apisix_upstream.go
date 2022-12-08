// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

// translateUpstreamNotStrictly translates Upstream nodes with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateUpstreamNotStrictly(namespace, svcName, subset string, svcPort int32, resolveGranularity string) (*apisixv1.Upstream, error) {
	ups := &apisixv1.Upstream{}
	ups.Name = apisixv1.ComposeUpstreamName(namespace, svcName, subset, svcPort, resolveGranularity)
	ups.ID = id.GenID(ups.Name)
	return ups, nil
}

func (t *translator) translateService(namespace, svcName, subset, svcResolveGranularity, svcClusterIP string, svcPort int32) (*apisixv1.Upstream, error) {
	ups, err := t.TranslateService(namespace, svcName, subset, svcPort)
	if err != nil {
		return nil, err
	}
	if svcResolveGranularity == types.ResolveGranularity.Service {
		ups.Nodes = apisixv1.UpstreamNodes{
			{
				Host:   svcClusterIP,
				Port:   int(svcPort),
				Weight: translation.DefaultWeight,
			},
		}
	}
	ups.Name = apisixv1.ComposeUpstreamName(namespace, svcName, subset, svcPort, svcResolveGranularity)
	ups.ID = id.GenID(ups.Name)
	return ups, nil
}

func (t *translator) TranslateApisixUpstreamExternalNodes(au *v2.ApisixUpstream) ([]apisixv1.UpstreamNode, error) {
	var nodes []apisixv1.UpstreamNode
	for i, node := range au.Spec.ExternalNodes {
		if node.Type == v2.ExternalTypeDomain {

			weight := translation.DefaultWeight
			if node.Weight != nil {
				weight = *node.Weight
			}
			n := apisixv1.UpstreamNode{
				Host:   node.Name,
				Weight: weight,
			}

			if node.Port != nil {
				n.Port = *node.Port
			} else {
				n.Port = utils.SchemeToPort(au.Spec.Scheme)
			}

			nodes = append(nodes, n)
		} else if node.Type == v2.ExternalTypeService {
			svc, err := t.ServiceLister.Services(au.Namespace).Get(node.Name)
			if err != nil {
				// In theory, ApisixRoute now watches all service add event, a not found error is already handled
				if k8serrors.IsNotFound(err) {
					// TODO: Should retry
					return nil, err
				}
				return nil, err
			}

			if svc.Spec.Type != corev1.ServiceTypeExternalName {
				return nil, fmt.Errorf("ApisixUpstream %s/%s ExternalNodes[%v] must refers to a ExternalName service: %s", au.Namespace, au.Name, i, node.Name)
			}

			weight := translation.DefaultWeight
			if node.Weight != nil {
				weight = *node.Weight
			}
			n := apisixv1.UpstreamNode{
				Host:   svc.Spec.ExternalName,
				Weight: weight,
			}

			if node.Port != nil {
				n.Port = *node.Port
			} else {
				n.Port = utils.SchemeToPort(au.Spec.Scheme)
			}

			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

// TODO: Retry when ApisixUpstream/ExternalName service not found
func (t *translator) translateExternalApisixUpstream(namespace, upstream string) (*apisixv1.Upstream, error) {
	multiVersioned, err := t.ApisixUpstreamLister.V2(namespace, upstream)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// TODO: Should retry
			return nil, err
		}
		return nil, err
	}

	au := multiVersioned.V2()
	if len(au.Spec.ExternalNodes) == 0 && au.Spec.Discovery == nil {
		// should do further resolve
		return nil, fmt.Errorf("%s/%s has empty ExternalNodes or Discovery configuration", namespace, upstream)
	}

	ups, err := t.TranslateUpstreamConfigV2(&au.Spec.ApisixUpstreamConfig)
	if err != nil {
		return nil, err
	}
	ups.Name = apisixv1.ComposeExternalUpstreamName(namespace, upstream)
	ups.ID = id.GenID(ups.Name)

	// APISIX does not allow discovery_type and nodes to exist at the same time.
	// https://github.com/apache/apisix/blob/01b4b49eb2ba642b337f7a1fbe1894a77942910b/apisix/schema_def.lua#L501-L504
	if len(au.Spec.ExternalNodes) != 0 {
		externalNodes, err := t.TranslateApisixUpstreamExternalNodes(au)
		if err != nil {
			return nil, err
		}

		ups.Nodes = append(ups.Nodes, externalNodes...)
	}

	return ups, nil
}
