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
	"strings"

	"github.com/golang/glog"

	"github.com/api7/ingress-controller/pkg/seven/conf"
	sevendb "github.com/api7/ingress-controller/pkg/seven/db"
	"github.com/api7/ingress-controller/pkg/seven/utils"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

// FindCurrentService find service from memDB,
// if Not Found, find service from apisix
func FindCurrentService(group, name, fullName string) (*v1.Service, error) {
	db := sevendb.ServiceRequest{Group: group, Name: name, FullName: fullName}
	currentService, _ := db.FindByName()
	if currentService != nil {
		return currentService, nil
	} else {
		// find service from apisix
		if services, err := ListService(group); err != nil {
			glog.Errorf("list services in etcd failed, group: %s, err: %+v", group, err)
			return nil, fmt.Errorf("list services failed, err: %+v", err)
		} else {
			for _, s := range services {
				if s.Name != nil && *(s.Name) == name {
					// and save to memDB
					db := &sevendb.ServiceDB{Services: []*v1.Service{s}}
					db.Insert()
					// return
					return s, nil
				}
			}
		}
	}
	return nil, nil
}

// ListUpstream list upstream from etcd , convert to v1.Upstream
func ListService(group string) ([]*v1.Service, error) {
	baseUrl := conf.FindUrl(group)
	url := baseUrl + "/services"
	ret, err := Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed, url: %s, err: %+v", url, err)
	}
	var servicesResponse ServicesResponse
	if err := json.Unmarshal(ret, &servicesResponse); err != nil {
		return nil, fmt.Errorf("json unmarshal failed, err: %+v", err)
	} else {
		result := make([]*v1.Service, 0)
		for _, u := range servicesResponse.Services.Services {
			if n, err := u.convert(group); err == nil {
				result = append(result, n)
			} else {
				return nil, fmt.Errorf("service : %+v 转换失败, %s", u.ServiceValue, err.Error())
			}
		}
		return result, nil
	}
}

// convert convert Service from etcd to v1.Service
func (u *Service) convert(group string) (*v1.Service, error) {
	// id
	keys := strings.Split(*u.Key, "/")
	id := keys[len(keys)-1]
	// Name
	name := u.ServiceValue.Desc
	// upstreamId
	upstreamId := u.ServiceValue.UpstreamId
	// plugins
	plugins := &v1.Plugins{}
	for k, v := range u.ServiceValue.Plugins {
		(*plugins)[k] = v
	}
	fullName := *name
	if group != "" {
		fullName = group + "_" + *name
	}
	return &v1.Service{ID: &id, FullName: &fullName, Group: &group, Name: name, UpstreamId: upstreamId, Plugins: plugins}, nil
}

func AddService(service *v1.Service) (*ServiceResponse, error) {
	baseUrl := conf.FindUrl(*service.Group)
	url := fmt.Sprintf("%s/services", baseUrl)
	ur := convert2ServiceRequest(service)
	if b, err := json.Marshal(ur); err != nil {
		return nil, err
	} else {
		if res, err := utils.Post(url, b); err != nil {
			return nil, fmt.Errorf("http post failed, err: %+v", err)
		} else {
			var uRes ServiceResponse
			if err = json.Unmarshal(res, &uRes); err != nil {
				return nil, err
			} else {
				if uRes.Service.Key != nil {
					return &uRes, nil
				} else {
					return nil, fmt.Errorf("apisix service not expected response")
				}

			}
		}
	}
}

func UpdateService(service *v1.Service) (*ServiceResponse, error) {
	baseUrl := conf.FindUrl(*service.Group)
	url := fmt.Sprintf("%s/services/%s", baseUrl, *service.ID)
	ur := convert2ServiceRequest(service)
	if b, err := json.Marshal(ur); err != nil {
		return nil, err
	} else {
		if res, err := utils.Patch(url, b); err != nil {
			return nil, err
		} else {
			var uRes ServiceResponse
			if err = json.Unmarshal(res, &uRes); err != nil {
				return nil, err
			} else {
				if uRes.Service.Key != nil {
					return &uRes, nil
				} else {
					var errResp ErrorResponse
					json.Unmarshal(res, &errResp)
					glog.Error(errResp.Message)
					return nil, fmt.Errorf("apisix service not expected response %s", errResp.Message)
				}
			}
		}
	}
}

func convert2ServiceRequest(service *v1.Service) *ServiceRequest {
	request := &ServiceRequest{
		Desc:       service.Name,
		UpstreamId: service.UpstreamId,
		Plugins:    service.Plugins,
	}
	glog.V(2).Info(*request.Desc)
	return request
}

type ServiceRequest struct {
	Desc       *string     `json:"desc,omitempty"`
	UpstreamId *string     `json:"upstream_id"`
	Plugins    *v1.Plugins `json:"plugins,omitempty"`
}

type ServicesResponse struct {
	Services Services `json:"node"`
}

type Services struct {
	Key      string     `json:"key"` // 用来定位upstreams 列表
	Services ServiceSet `json:"nodes"`
}

type ServiceSet []Service

// UpstreamSet.UnmarshalJSON implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (set *ServiceSet) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var svcs []Service
	if err := json.Unmarshal(p, &svcs); err != nil {
		return err
	}
	*set = svcs
	return nil
}

type ServiceResponse struct {
	Action  string  `json:"action"`
	Service Service `json:"node"`
}

type Service struct {
	Key          *string      `json:"key"` // service key
	ServiceValue ServiceValue `json:"value,omitempty"`
}

type ServiceValue struct {
	UpstreamId *string                `json:"upstream_id,omitempty"`
	Plugins    map[string]interface{} `json:"plugins"`
	Desc       *string                `json:"desc,omitempty"`
}
