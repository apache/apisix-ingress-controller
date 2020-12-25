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
	"github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"

	"github.com/api7/ingress-controller/pkg/seven/conf"
	"github.com/api7/ingress-controller/pkg/seven/utils"
)

// ListSsl list ssl from etcd , convert to v1.Upstream
func ListSsl(group string) ([]*v1.Ssl, error) {
	baseUrl := conf.FindUrl(group)
	url := baseUrl + "/ssl"
	ret, err := Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed, url: %s, err: %+v", url, err)
	}
	var sslsResponse SslsResponse
	if err := json.Unmarshal(ret, &sslsResponse); err != nil {
		return nil, fmt.Errorf("json transform error")
	} else {
		ssls := make([]*v1.Ssl, 0)
		for _, s := range sslsResponse.SslList.SslNodes {
			id := strings.ReplaceAll(*s.Key, "/apisix/ssl/", "")
			ssl := &v1.Ssl{
				ID:     &id,
				Snis:   s.Ssl.Snis,
				Cert:   s.Ssl.Cert,
				Key:    s.Ssl.Key,
				Status: s.Ssl.Status,
				Group:  &group,
			}
			ssls = append(ssls, ssl)
		}
		return ssls, nil
	}
}

func AddOrUpdateSsl(ssl *v1.Ssl) (*SslResponse, error) {
	baseUrl := conf.FindUrl(*ssl.Group)
	url := fmt.Sprintf("%s/ssl/%s", baseUrl, *ssl.ID)
	glog.V(2).Info(url)
	ur := &v1.Ssl{
		Snis:   ssl.Snis,
		Cert:   ssl.Cert,
		Key:    ssl.Key,
		Status: ssl.Status,
	}
	if b, err := json.Marshal(ur); err != nil {
		return nil, err
	} else {
		if res, err := utils.Put(url, b); err != nil {
			return nil, fmt.Errorf("http put failed, url: %s, err: %+v", url, err)
		} else {
			var uRes SslResponse
			if err = json.Unmarshal(res, &uRes); err != nil {
				glog.Errorf("json Unmarshal error: %s", err.Error())
				return nil, err
			} else {
				glog.V(2).Info(uRes)
				if uRes.Ssl.Key != nil {
					return &uRes, nil
				} else {
					return nil, fmt.Errorf("apisix ssl not expected response")
				}
			}
		}
	}
}

func DeleteSsl(ssl *v1.Ssl) error {
	baseUrl := conf.FindUrl(*ssl.Group)
	url := fmt.Sprintf("%s/ssl/%s", baseUrl, *ssl.ID)
	if _, err := utils.Delete(url); err != nil {
		return fmt.Errorf("http delete failed, url: %s, err: %+v", url, err)
	} else {
		return nil
	}
}

type SslResponse struct {
	Action string  `json:"action"`
	Ssl    SslNode `json:"node"`
}

type SslsResponse struct {
	Action  string  `json:"action"`
	SslList SslList `json:"node"`
}

type SslList struct {
	SslNodes SslSet `json:"nodes"`
}

type SslNode struct {
	Key *string `json:"key"`
	Ssl *v1.Ssl `json:"value"`
}

type SslSet []SslNode

// SslSet.UnmarshalJSON implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (set *SslSet) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var ssls []SslNode
	if err := json.Unmarshal(p, &ssls); err != nil {
		return err
	}
	*set = ssls
	return nil
}
