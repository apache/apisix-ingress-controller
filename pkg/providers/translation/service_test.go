// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestTranslateServiceNoEndpoints(t *testing.T) {
	client := fake.NewSimpleClientset()
	informersFactory := informers.NewSharedInformerFactory(client, 0)
	epLister, _ := kube.NewEndpointListerAndInformer(informersFactory, false)
	auLister2 := v2.NewApisixUpstreamLister(cache.NewIndexer(func(obj interface{}) (out string, err error) { return }, map[string]cache.IndexFunc{}))

	tr := &translator{&TranslatorOptions{
		APIVersion:           config.ApisixV2,
		EndpointLister:       epLister,
		ApisixUpstreamLister: kube.NewApisixUpstreamLister(auLister2),
	}}

	expected := &v1.Upstream{
		Metadata: v1.Metadata{
			Desc:   "Created by apisix-ingress-controller, DO NOT modify it manually",
			Labels: map[string]string{"managed-by": "apisix-ingress-controller"},
		},
		Type:   "roundrobin",
		Nodes:  v1.UpstreamNodes{},
		Scheme: "http",
	}

	upstream, err := tr.TranslateService("test", "svc", "", 9080)
	assert.Nil(t, err)
	assert.Equal(t, expected, upstream)
}
