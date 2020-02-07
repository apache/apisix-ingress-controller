package controller

import (
	"github.com/iresty/ingress-controller/conf"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

const ADD = "ADD"
const UPDATE = "UPDATE"
const DELETE = "DELETE"

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
	if ep.Namespace != "kube-system"{
		glog.Info(ep.Name)
		glog.Info(qo.OpeType)
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