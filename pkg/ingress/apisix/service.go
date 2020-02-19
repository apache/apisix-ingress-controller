package apisix

import (
	ingress "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	apisix "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"github.com/iresty/ingress-controller/pkg/ingress/endpoint"
	"strconv"
)

const (
	ApisixService = "ApisixService"
)

type ApisixServiceCRD ingress.ApisixService

// Convert convert to  apisix.Service from ingress.ApisixService CRD
func (as *ApisixServiceCRD) Convert() ([]*apisix.Service, []*apisix.Upstream, error) {
	ns := as.Namespace
	name := as.Name
	// meta annotation
	pluginsInAnnotation, group := BuildAnnotation(as.Annotations)

	services := make([]*apisix.Service, 0)
	upstreams := make([]*apisix.Upstream, 0)
	rv := as.ObjectMeta.ResourceVersion
	port := as.Spec.Port
	upstreamName := as.Spec.Upstream
	// apisix upstream name = namespace_upstreamName_svcPort
	apisixUpstreamName := ns + "_" + upstreamName + "_" + strconv.Itoa(int(port))
	apisixServiceName := ns + "_" + name + "_" + strconv.Itoa(int(port))
	fromKind := ApisixService
	// plugins
	plugins := as.Spec.Plugins
	pluginRet := &apisix.Plugins{}
	// 1.from annotations
	for k, v := range pluginsInAnnotation {
		(*pluginRet)[k] = v
	}
	// 2.from service plugins
	for _, p := range plugins {
		if p.Enable {
			(*pluginRet)[p.Name] = p.Config
		}
	}

	service := &apisix.Service{
		Group:           &group,
		ResourceVersion: &rv,
		Name:            &apisixServiceName,
		UpstreamName:    &apisixUpstreamName,
		FromKind:        &fromKind,
		Plugins:         pluginRet,
	}
	services = append(services, service)
	// upstream
	LBType := DefaultLBType
	nodes := endpoint.BuildEps(ns, upstreamName, int(port))
	upstream := &apisix.Upstream{
		Group:           &group,
		ResourceVersion: &rv,
		Name:            &apisixUpstreamName,
		Type:            &LBType,
		Nodes:           nodes,
		FromKind:        &fromKind,
	}
	upstreams = append(upstreams, upstream)
	return services, upstreams, nil
}
