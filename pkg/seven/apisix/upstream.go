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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"

	"github.com/api7/ingress-controller/pkg/seven/conf"
	"github.com/api7/ingress-controller/pkg/seven/db"
	"github.com/api7/ingress-controller/pkg/seven/utils"
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
		if upstreams, err := ListUpstream(group); err != nil {
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

// ListUpstream list upstream from etcd , convert to v1.Upstream
func ListUpstream(group string) ([]*v1.Upstream, error) {
	baseUrl := conf.FindUrl(group)
	url := baseUrl + "/upstreams"
	ret, err := Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed, url: %s, err: %+v", url, err)
	}
	var upstreamsResponse UpstreamsResponse
	if err := json.Unmarshal(ret, &upstreamsResponse); err != nil {
		return nil, fmt.Errorf("json转换失败")
	} else {
		upstreams := make([]*v1.Upstream, 0)
		for _, u := range upstreamsResponse.Upstreams.Upstreams {
			if n, err := u.convert(group); err == nil {
				upstreams = append(upstreams, n)
			} else {
				return nil, fmt.Errorf("upstream: %s 转换失败, %s", *u.UpstreamNodes.Desc, err.Error())
			}
		}
		return upstreams, nil
	}
}

//func IsExist(name string) (bool, error) {
//	if upstreams, err := ListUpstream(); err != nil {
//		return false, err
//	} else {
//		for _, upstream := range upstreams {
//			if *upstream.Name == name {
//				return true, nil
//			}
//		}
//		return false, nil
//	}
//}

func AddUpstream(upstream *v1.Upstream) (*UpstreamResponse, error) {
	baseUrl := conf.FindUrl(*upstream.Group)
	url := fmt.Sprintf("%s/upstreams", baseUrl)
	glog.V(2).Info(url)
	ur := convert2UpstreamRequest(upstream)
	if b, err := json.Marshal(ur); err != nil {
		return nil, err
	} else {
		if res, err := utils.Post(url, b); err != nil {
			return nil, fmt.Errorf("http post failed, url: %s, err: %+v", url, err)
		} else {
			var uRes UpstreamResponse
			if err = json.Unmarshal(res, &uRes); err != nil {
				glog.Errorf("json Unmarshal error: %s", err.Error())
				return nil, err
			} else {
				glog.V(2).Info(uRes)
				if uRes.Upstream.Key != nil {
					return &uRes, nil
				} else {
					return nil, fmt.Errorf("apisix upstream not expected response")
				}
			}
		}
	}
}

func UpdateUpstream(upstream *v1.Upstream) error {
	baseUrl := conf.FindUrl(*upstream.Group)
	url := fmt.Sprintf("%s/upstreams/%s", baseUrl, *upstream.ID)
	ur := convert2UpstreamRequest(upstream)
	if b, err := json.Marshal(ur); err != nil {
		return err
	} else {
		if _, err := utils.Patch(url, b); err != nil {
			return fmt.Errorf("http patch failed, url: %s, err: %+v", url, err)
		} else {
			return nil
		}
	}
}

func PatchNodes(upstream *v1.Upstream, nodes []*v1.Node) error {
	baseUrl := conf.FindUrl(*upstream.Group)
	url := fmt.Sprintf("%s/upstreams/%s/nodes", baseUrl, *upstream.ID)
	nodeMap := convertNodes(nodes)
	if b, err := json.Marshal(nodeMap); err != nil {
		return err
	} else {
		if _, err := utils.Patch(url, b); err != nil {
			return fmt.Errorf("http patch failed, url: %s, err: %+v", url, err)
		} else {
			return nil
		}
	}
}

func DeleteUpstream(upstream *v1.Upstream) error {
	baseUrl := conf.FindUrl(*upstream.Group)
	url := fmt.Sprintf("%s/upstreams/%s", baseUrl, *upstream.ID)
	if _, err := utils.Delete(url); err != nil {
		return fmt.Errorf("http delete failed, url: %s, err: %+v", url, err)
	} else {
		return nil
	}
}

func convert2UpstreamRequest(upstream *v1.Upstream) *UpstreamRequest {
	nodes := convertNodes(upstream.Nodes)
	return &UpstreamRequest{
		LBType: *upstream.Type,
		HashOn: upstream.HashOn,
		Key:    upstream.Key,
		Desc:   *upstream.Name,
		Nodes:  nodes,
	}
}

func convertNodes(nodes []*v1.Node) map[string]int64 {
	result := make(map[string]int64)
	for _, u := range nodes {
		result[*u.IP+":"+strconv.Itoa(*u.Port)] = int64(*u.Weight)
	}
	return result
}

// convert convert Upstream from etcd to v1.Upstream
func (u *Upstream) convert(group string) (*v1.Upstream, error) {
	// id
	keys := strings.Split(*u.Key, "/")
	id := keys[len(keys)-1]
	// Name
	name := u.UpstreamNodes.Desc
	// type
	LBType := u.UpstreamNodes.LBType
	// key
	key := u.Key
	// nodes
	nodes := make([]*v1.Node, 0)
	for k, v := range u.UpstreamNodes.Nodes {
		ks := strings.Split(k, ":")
		ip := ks[0]
		port := 80
		if len(ks) > 1 {
			port, _ = strconv.Atoi(ks[1])
		}
		weight := int(v)
		node := &v1.Node{IP: &ip, Port: &port, Weight: &weight}
		nodes = append(nodes, node)
	}
	// fullName
	fullName := *name
	if group != "" {
		fullName = group + "_" + *name
	}
	return &v1.Upstream{ID: &id, FullName: &fullName, Group: &group, Name: name, Type: LBType, Key: key, Nodes: nodes}, nil
}

type UpstreamsResponse struct {
	Upstreams Upstreams `json:"node"`
}

type UpstreamResponse struct {
	Action   string   `json:"action"`
	Upstream Upstream `json:"node"`
}

type Upstreams struct {
	Key       string      `json:"key"` // 用来定位upstreams 列表
	Upstreams UpstreamSet `json:"nodes"`
}

type UpstreamSet []Upstream

// UpstreamSet.UnmarshalJSON implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (set *UpstreamSet) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var ups []Upstream
	if err := json.Unmarshal(p, &ups); err != nil {
		return err
	}
	*set = ups
	return nil
}

type Upstream struct {
	Key           *string       `json:"key"` // upstream key
	UpstreamNodes UpstreamNodes `json:"value"`
}

type UpstreamNodes struct {
	Nodes  map[string]int64 `json:"nodes"`
	Desc   *string          `json:"desc"` // upstream name  = k8s svc
	LBType *string          `json:"type"` // 负载均衡类型
}

//{"type":"roundrobin","nodes":{"10.244.10.11:8080":100},"desc":"somesvc"}
type UpstreamRequest struct {
	LBType string           `json:"type"`
	HashOn *string          `json:"hash_on,omitempty"`
	Key    *string          `json:"key,omitempty"`
	Nodes  map[string]int64 `json:"nodes"`
	Desc   string           `json:"desc"`
}
