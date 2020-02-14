package apisix

import (
	ingress "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	apisix "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"strconv"
	"github.com/iresty/ingress-controller/pkg/ingress/endpoint"
	seven "github.com/gxthrj/seven/apisix"
)

const (
	DefaultLBType = "roundrobin"
	SSLREDIRECT = "k8s.apisix.apache.org/ssl-redirect"
	WHITELIST = "k8s.apisix.apache.org/whitelist-source-range"
	ENABLE_CORS = "k8s.apisix.apache.org/enable-cors"
	CORS_ALLOW_ORIGIN = "k8s.apisix.apache.org/cors-allow-origin"
	CORS_ALLOW_HEADERS = "k8s.apisix.apache.org/cors-allow-headers"
	CORS_ALLOW_METHODS = "k8s.apisix.apache.org/cors-allow-methods"
)

type ApisixRoute ingress.ApisixRoute

// Convert convert to  apisix.Route from ingress.ApisixRoute CRD
func (ar *ApisixRoute) Convert() ([]*apisix.Route, []*apisix.Service, []*apisix.Upstream, error) {
	ns := ar.Namespace
	// meta
	// annotation
	plugins := make(apisix.Plugins)
	cors := &CorsYaml{}
	for k, v := range ar.Annotations{
		switch{
		case k == SSLREDIRECT:
			if b, err := strconv.ParseBool(v); err == nil && b {
				// todo add ssl-redirect plugin
			}
		case k == WHITELIST:
			ipRestriction := seven.BuildIpRestriction(&v, nil)
			plugins["ip-restriction"] = ipRestriction
		case k == ENABLE_CORS:
			cors.SetEnable(v)
		case k == CORS_ALLOW_ORIGIN:
			cors.SetOrigin(v)
		case k == CORS_ALLOW_HEADERS:
			cors.SetHeaders(v)
		case k == CORS_ALLOW_METHODS:
			cors.SetMethods(v)
		default:
			// do nothing
		}
	}
	// build CORS plugin
	plugins["aispeech-cors"] = cors.Build()
	// Host
	rules := ar.Spec.Rules
	routes := make([]*apisix.Route, 0)
	services := make([]*apisix.Service, 0)
	upstreams := make([]*apisix.Upstream, 0)
	rv := ar.ObjectMeta.ResourceVersion
	for _, r := range rules {
		host := r.Host
		paths := r.Http.Paths
		for _, p := range paths {
			uri := p.Path
			svcName := p.Backend.ServiceName
			svcPort := strconv.FormatInt(p.Backend.ServicePort, 10)
			// apisix route name = host + path
			apisixRouteName := host + uri
			// apisix service name = namespace_svcName_svcPort
			apisixSvcName := ns + "_" + svcName + "_" + svcPort
			// apisix route name = namespace_svcName_svcPort = apisix service name
			apisixUpstreamName := ns + "_" + svcName + "_" + svcPort
			// todo plugins in the level of route

			// routes
			route := &apisix.Route{
				ResourceVersion: &rv,
				Name: &apisixRouteName,
				Host: &host,
				Path: &uri,
				ServiceName: &apisixSvcName,
				UpstreamName: &apisixUpstreamName,
				Plugins: &plugins,
			}
			routes = append(routes, route)
			// services
			service := &apisix.Service{
				Name: &apisixSvcName,
				UpstreamName:  &apisixUpstreamName,
				ResourceVersion: &rv,
			}
			services = append(services, service)
			// upstreams
			LBType := DefaultLBType
			port, _:= strconv.Atoi(svcPort)
			nodes := endpoint.BuildEps(ns, svcName, port)
			upstream := &apisix.Upstream{
				ResourceVersion: &rv,
				Name: &apisixUpstreamName,
				Type: &LBType,
				Nodes: nodes,
			}
			upstreams = append(upstreams, upstream)
		}
	}
	return routes, services, upstreams,nil
}
