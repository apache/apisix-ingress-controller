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
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

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
