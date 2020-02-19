package apisix

import (
	ingress "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	apisix "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"github.com/iresty/ingress-controller/pkg/ingress/endpoint"
	"strconv"
	"github.com/gxthrj/seven/conf"
)

const (
	DefaultLBType      = "roundrobin"
	DefaultGroup       = "apisix.cloud.svc.cluster.local:9180"
	SSLREDIRECT        = "k8s.apisix.apache.org/ssl-redirect"
	WHITELIST          = "k8s.apisix.apache.org/whitelist-source-range"
	ENABLE_CORS        = "k8s.apisix.apache.org/enable-cors"
	CORS_ALLOW_ORIGIN  = "k8s.apisix.apache.org/cors-allow-origin"
	CORS_ALLOW_HEADERS = "k8s.apisix.apache.org/cors-allow-headers"
	CORS_ALLOW_METHODS = "k8s.apisix.apache.org/cors-allow-methods"
	INGRESS_CLASS      = "k8s.apisix.apache.org/ingress.class"
)

type ApisixRoute ingress.ApisixRoute

// Convert convert to  apisix.Route from ingress.ApisixRoute CRD
func (ar *ApisixRoute) Convert() ([]*apisix.Route, []*apisix.Service, []*apisix.Upstream, error) {
	ns := ar.Namespace
	// meta annotation
	plugins, group := BuildAnnotation(ar.Annotations)
	conf.AddGroup(group)
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
			// plugins defined in Route Level
			pls := p.Plugins
			pluginRet := make(apisix.Plugins)
			// 1.add annotation plugins
			for k, v := range plugins {
				pluginRet[k] = v
			}
			// 2.add route plugins
			for _, p := range pls {
				if p.Enable {
					pluginRet[p.Name] = p.Config
				}
			}

			// routes
			route := &apisix.Route{
				Group:           &group,
				ResourceVersion: &rv,
				Name:            &apisixRouteName,
				Host:            &host,
				Path:            &uri,
				ServiceName:     &apisixSvcName,
				UpstreamName:    &apisixUpstreamName,
				Plugins:         &pluginRet,
			}
			routes = append(routes, route)
			// services
			service := &apisix.Service{
				Group:           &group,
				Name:            &apisixSvcName,
				UpstreamName:    &apisixUpstreamName,
				ResourceVersion: &rv,
			}
			services = append(services, service)
			// upstreams
			LBType := DefaultLBType
			port, _ := strconv.Atoi(svcPort)
			nodes := endpoint.BuildEps(ns, svcName, port)
			upstream := &apisix.Upstream{
				Group:           &group,
				ResourceVersion: &rv,
				Name:            &apisixUpstreamName,
				Type:            &LBType,
				Nodes:           nodes,
			}
			upstreams = append(upstreams, upstream)
		}
	}
	return routes, services, upstreams, nil
}
