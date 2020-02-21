package endpoint

import (
	"github.com/iresty/ingress-controller/conf"
	"github.com/golang/glog"
	"github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
)

type Endpoint interface{
	BuildEps(ns, name string, port int) []*v1.Node
}

type EndpointRequest struct {}

func (epr *EndpointRequest) BuildEps(ns, name string, port int) []*v1.Node {
	nodes := make([]*v1.Node, 0)
	epInformers := conf.EndpointsInformer
	if ep, err := epInformers.Lister().Endpoints(ns).Get(name); err != nil {
		glog.Errorf("find endpoint %s/%s err", ns, name, err.Error())
	} else {
		for _, s := range ep.Subsets {
			for _, ip := range s.Addresses{
				p := ip.IP
				weight := 100
				node := &v1.Node{IP: &p, Port: &port, Weight: &weight}
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}

// BuildEps build nodes from endpoints for upstream
func BuildEps(ns, name string, port int) []*v1.Node{
	nodes := make([]*v1.Node, 0)
	epInformers := conf.EndpointsInformer
	if ep, err := epInformers.Lister().Endpoints(ns).Get(name); err != nil {
		glog.Errorf("find endpoint %s/%s err", ns, name, err.Error())
	} else {
		for _, s := range ep.Subsets {
			for _, ip := range s.Addresses{
				p := ip.IP
				weight := 100
				node := &v1.Node{IP: &p, Port: &port, Weight: &weight}
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}
