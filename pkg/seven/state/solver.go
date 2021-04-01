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
package state

import (
	"context"

	"github.com/apache/apisix-ingress-controller/pkg/seven/conf"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type CRDStatus struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Err    error  `json:"err"`
}

func SyncSsl(ssl *v1.Ssl, method string) error {
	var cluster string
	if ssl.Group != "" {
		cluster = ssl.Group
	}
	switch method {
	case Create:
		_, err := conf.Client.Cluster(cluster).SSL().Create(context.TODO(), ssl)
		return err
	case Update:
		_, err := conf.Client.Cluster(cluster).SSL().Update(context.TODO(), ssl)
		return err
	case Delete:
		return conf.Client.Cluster(cluster).SSL().Delete(context.TODO(), ssl)
	}
	return nil
}
