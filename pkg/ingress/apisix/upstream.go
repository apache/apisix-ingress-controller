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
package apisix

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/apache/apisix-ingress-controller/pkg/ingress/endpoint"
	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	"github.com/apache/apisix-ingress-controller/pkg/seven/conf"
	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	ApisixUpstream = "ApisixUpstream"
)

//type ApisixUpstreamCRD ingress.ApisixUpstream

type ApisixUpstreamBuilder struct {
	CRD *configv1.ApisixUpstream
	Ep  endpoint.Endpoint
}

// Convert convert to  apisix.Route from ingress.ApisixRoute CRD
func (aub *ApisixUpstreamBuilder) Convert() ([]*apisix.Upstream, error) {
	ar := aub.CRD
	ns := ar.Namespace
	name := ar.Name
	// meta annotation
	_, group := BuildAnnotation(ar.Annotations)
	conf.AddGroup(group)

	upstreams := make([]*apisix.Upstream, 0)
	rv := ar.ObjectMeta.ResourceVersion
	Ports := ar.Spec.Ports
	for _, r := range Ports {
		if r.Scheme != "" && r.Scheme != configv1.SchemeHTTP && r.Scheme != configv1.SchemeGRPC {
			return nil, fmt.Errorf("bad scheme %s", r.Scheme)
		}

		port := r.Port
		// apisix route name = namespace_svcName_svcPort = apisix service name
		apisixUpstreamName := ns + "_" + name + "_" + strconv.Itoa(int(port))

		lb := r.LoadBalancer

		//nodes := endpoint.BuildEps(ns, name, int(port))
		nodes := aub.Ep.BuildEps(ns, name, port)
		fromKind := ApisixUpstream

		// fullName
		fullName := apisixUpstreamName
		if group != "" {
			fullName = group + "_" + apisixUpstreamName
		}
		upstream := &apisix.Upstream{
			Metadata: apisix.Metadata{
				FullName:        fullName,
				Group:           group,
				ResourceVersion: rv,
				Name:            apisixUpstreamName,
			},
			Nodes:    nodes,
			FromKind: fromKind,
		}
		if r.Scheme != "" {
			upstream.Scheme = r.Scheme
		}
		if lb == nil || lb.Type == "" {
			upstream.Type = apisix.LbRoundRobin
		} else {
			switch lb.Type {
			case apisix.LbRoundRobin, apisix.LbLeastConn, apisix.LbEwma:
				upstream.Type = lb.Type
			case apisix.LbConsistentHash:
				upstream.Type = lb.Type
				upstream.Key = lb.Key
				switch lb.HashOn {
				case apisix.HashOnVars:
					fallthrough
				case apisix.HashOnHeader:
					fallthrough
				case apisix.HashOnCookie:
					fallthrough
				case apisix.HashOnConsumer:
					fallthrough
				case apisix.HashOnVarsCombination:
					upstream.HashOn = lb.HashOn
				default:
					return nil, errors.New("invalid hashOn value")
				}
			default:
				return nil, errors.New("invalid load balancer type")
			}
		}
		upstreams = append(upstreams, upstream)
	}
	return upstreams, nil
}
