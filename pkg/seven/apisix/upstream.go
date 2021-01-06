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
	"context"
	"fmt"

	"github.com/golang/glog"

	"github.com/api7/ingress-controller/pkg/seven/conf"
	"github.com/api7/ingress-controller/pkg/seven/db"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

// FindCurrentUpstream find upstream from memDB,
// if Not Found, find upstream from apisix
func FindCurrentUpstream(group, name, fullName string) (*v1.Upstream, error) {
	ur := &db.UpstreamRequest{Group: group, Name: name, FullName: fullName}
	currentUpstream, _ := ur.FindByName()
	if currentUpstream != nil {
		return currentUpstream, nil
	} else {
		// find upstream from apisix
		if upstreams, err := conf.Client.Upstream().List(context.TODO(), group); err != nil {
			glog.Errorf("list upstreams in etcd failed, group: %s, err: %+v", group, err)
			return nil, fmt.Errorf("list upstreams failed, err: %+v", err)
		} else {
			for _, upstream := range upstreams {
				if upstream.Name != nil && *(upstream.Name) == name {
					// and save to memDB
					upstreamDB := &db.UpstreamDB{Upstreams: []*v1.Upstream{upstream}}
					upstreamDB.InsertUpstreams()
					//InsertUpstreams([]*v1.Upstream{upstream})
					// return
					return upstream, nil
				}
			}
		}

	}
	return nil, nil
}

func PatchNodes(upstream *v1.Upstream, nodes []*v1.Node) error {
	oldNodes := upstream.Nodes
	upstream.Nodes = nodes
	defer func() {
		// Restore it
		upstream.Nodes = oldNodes
	}()
	_, err := conf.Client.Upstream().Update(context.TODO(), upstream)
	return err
}
