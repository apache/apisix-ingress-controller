package controller

import (
	"github.com/iresty/ingress-controller/conf"
	"k8s.io/api/core/v1"
	"strconv"
	"github.com/gxthrj/seven/apisix"
	apisixType "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"github.com/gxthrj/seven/state"
	"github.com/golang/glog"
)

const (
	ADD = "ADD"
	UPDATE = "UPDATE"
	DELETE = "DELETE"
	WatchFromKind = "watch"
)

func Watch(){
	c := &controller{
		queue: make(chan interface{}, 100),
	}
	conf.EndpointsInformer.Informer().AddEventHandler(&QueueEventHandler{c:c})
	go c.run()
}

func (c *controller) pop() interface{}{
	e := <- c.queue
	return e
}

func (c *controller) run() {
	for {
		ele := c.pop()
		c.process(ele)
	}
}

func (c *controller) process(obj interface{}) {
	qo, _ := obj.(*queueObj)
	ep, _ := qo.Obj.(*v1.Endpoints)
	if ep.Namespace != "kube-system"{ // todo here is some ignore namespaces
		for _, s := range ep.Subsets{
			// if upstream need to watch
			// ips
			ips := make([]string, 0)
			for _, address := range s.Addresses{
				ips = append(ips, address.IP)
			}
			// ports
			for _, port := range s.Ports{
				upstreamName := ep.Namespace + "_" + ep.Name + "_" + strconv.Itoa(int(port.Port))
				// find upstreamName is in apisix
				upstreams, err :=  apisix.ListUpstream()
				if err == nil {
					for _, upstream := range upstreams {
						if *(upstream.Name) == upstreamName {
							nodes := make([]*apisixType.Node, 0)
							for _, ip := range ips {
								ipAddress := ip
								p := int(port.Port)
								weight := 100
								node := &apisixType.Node{IP: &ipAddress, Port: &p, Weight: &weight}
								nodes = append(nodes, node)
							}
							upstream.Nodes = nodes
							// update upstream nodes
							// add to seven solver queue
							//apisix.UpdateUpstream(upstream)
							fromKind := WatchFromKind
							upstream.FromKind = &fromKind
							upstreams := []*apisixType.Upstream{upstream}
							comb := state.ApisixCombination{Routes: nil, Services: nil, Upstreams: upstreams}
							if _, err = comb.Solver(); err != nil {
								glog.Errorf(err.Error())
							}
						}
					}
				}
			}
		}
	}
}

type controller struct {
	queue chan interface{}
}

type queueObj struct {
	OpeType string `json:"ope_type"`
	Obj interface{} `json:"obj"`
}

type QueueEventHandler struct {
	c *controller
}

func (h *QueueEventHandler) OnAdd(obj interface{}) {
	h.c.queue <- &queueObj{ADD, obj}
}

func (h *QueueEventHandler) OnDelete(obj interface{}) {
	h.c.queue <- &queueObj{DELETE, obj}
}

func (h *QueueEventHandler) OnUpdate(old, update interface{}) {
	h.c.queue <- &queueObj{ UPDATE, update}
}