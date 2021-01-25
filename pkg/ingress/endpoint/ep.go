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
package endpoint

import (
	"github.com/golang/glog"

	"github.com/api7/ingress-controller/pkg/kube"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type Endpoint interface {
	BuildEps(ns, name string, port int) []v1.Node
}

type EndpointRequest struct{}

func (epr *EndpointRequest) BuildEps(ns, name string, port int) []v1.Node {
	nodes := make([]v1.Node, 0)
	epInformers := kube.EndpointsInformer
	if ep, err := epInformers.Lister().Endpoints(ns).Get(name); err != nil {
		glog.Errorf("find endpoint %s/%s err: %s", ns, name, err.Error())
	} else {
		for _, s := range ep.Subsets {
			for _, ip := range s.Addresses {
				node := v1.Node{IP: ip.IP, Port: port, Weight: 100}
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}

// BuildEps build nodes from endpoints for upstream
func BuildEps(ns, name string, port int) []v1.Node {
	nodes := make([]v1.Node, 0)
	epInformers := kube.EndpointsInformer
	if ep, err := epInformers.Lister().Endpoints(ns).Get(name); err != nil {
		glog.Errorf("find endpoint %s/%s err: %s", ns, name, err.Error())
	} else {
		for _, s := range ep.Subsets {
			for _, ip := range s.Addresses {
				node := v1.Node{IP: ip.IP, Port: port, Weight: 100}
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}
