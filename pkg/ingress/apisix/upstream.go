package apisix

import (
	ingress "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	apisix "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"github.com/iresty/ingress-controller/pkg/ingress/endpoint"
	"strconv"
)

const (
	RR             = "roundrobin"
	CHASH          = "chash"
	ApisixUpstream = "ApisixUpstream"
)

type ApisixUpstreamCRD ingress.ApisixUpstream

// Convert convert to  apisix.Route from ingress.ApisixRoute CRD
func (ar *ApisixUpstreamCRD) Convert() ([]*apisix.Upstream, error) {
	ns := ar.Namespace
	name := ar.Name
	upstreams := make([]*apisix.Upstream, 0)
	rv := ar.ObjectMeta.ResourceVersion
	Ports := ar.Spec.Ports
	for _, r := range Ports {
		port := r.Port
		// apisix route name = namespace_svcName_svcPort = apisix service name
		apisixUpstreamName := ns + "_" + name + "_" + strconv.Itoa(int(port))

		lb := r.Loadbalancer

		nodes := endpoint.BuildEps(ns, name, int(port))
		fromKind := ApisixUpstream
		upstream := &apisix.Upstream{
			ResourceVersion: &rv,
			Name:            &apisixUpstreamName,
			Nodes:           nodes,
			FromKind:        &fromKind,
		}
		lbType := lb["type"].(string)
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
