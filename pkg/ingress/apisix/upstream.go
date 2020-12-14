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
	"github.com/api7/ingress-controller/pkg/ingress/endpoint"
	ingress "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	apisix "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"github.com/gxthrj/seven/conf"
	"strconv"
)

const (
	RR             = "roundrobin"
	CHASH          = "chash"
	ApisixUpstream = "ApisixUpstream"
)

//type ApisixUpstreamCRD ingress.ApisixUpstream

type ApisixUpstreamBuilder struct {
	CRD *ingress.ApisixUpstream
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
		port := r.Port
		// apisix route name = namespace_svcName_svcPort = apisix service name
		apisixUpstreamName := ns + "_" + name + "_" + strconv.Itoa(int(port))

		lb := r.Loadbalancer

		//nodes := endpoint.BuildEps(ns, name, int(port))
		nodes := aub.Ep.BuildEps(ns, name, int(port))
		fromKind := ApisixUpstream

		// fullName
		fullName := apisixUpstreamName
		if group != "" {
			fullName = group + "_" + apisixUpstreamName
		}
		upstream := &apisix.Upstream{
			FullName:        &fullName,
			Group:           &group,
			ResourceVersion: &rv,
			Name:            &apisixUpstreamName,
			Nodes:           nodes,
			FromKind:        &fromKind,
		}
		lbType := RR
		if lb != nil {
			lbType = lb["type"].(string)
		}
		switch {
		case lbType == CHASH:
			upstream.Type = &lbType
			hashOn := lb["hashOn"]
			key := lb["key"]
			if hashOn != nil {
				ho := hashOn.(string)
				upstream.HashOn = &ho
			}
			if key != nil {
				k := key.(string)
				upstream.Key = &k
			}
		default:
			lbType = RR
			upstream.Type = &lbType
		}
		upstreams = append(upstreams, upstream)
	}
	return upstreams, nil
}
