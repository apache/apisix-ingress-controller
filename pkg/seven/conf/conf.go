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
package conf

import (
	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

var (
	BaseUrl  = "http://172.16.20.90:30116/apisix/admin"
	UrlGroup = make(map[string]string)
	Client   apisix.APISIX
)

func SetBaseUrl(url string) {
	BaseUrl = url
}

func AddGroup(group string) {
	if group != "" {
		err := Client.AddCluster(&apisix.ClusterOptions{
			Name:    group,
			BaseURL: "http://" + group + "/apisix/admin",
		})
		if err != nil {
			if err == apisix.ErrDuplicatedCluster {
				log.Errorf("failed to create cluster %s: %s", group, err)
			} else {
				log.Infof("cluster %s already exists", group)
			}
		}
	}
}

func SetAPISIXClient(c apisix.APISIX) {
	Client = c
}
