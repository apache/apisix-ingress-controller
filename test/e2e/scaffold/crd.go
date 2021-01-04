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
package scaffold

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gxthrj/seven/apisix"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type counter struct {
	Count string `json:"count"`
}

// ApisixRoute is the ApisixRoute CRD definition.
// We don't use the definition in apisix-ingress-controller,
// since the k8s dependencies in terratest and
// apisix-ingress-controller are conflicted.
type apisixRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              apisixRouteSpec `json:"spec"`
}

type apisixRouteSpec struct {
	Rules []ApisixRouteRule `json:"rules"`
}

// ApisixRouteRule defines the route policies of ApisixRoute.
type ApisixRouteRule struct {
	Host string              `json:"host"`
	HTTP ApisixRouteRuleHTTP `json:"http"`
}

// ApisixRouteRuleHTTP defines the HTTP part of route policies.
type ApisixRouteRuleHTTP struct {
	Paths []ApisixRouteRuleHTTPPath `json:"paths"`
}

// ApisixRouteRuleHTTP defines a route in the HTTP part of ApisixRoute.
type ApisixRouteRuleHTTPPath struct {
	Path    string                     `json:"path"`
	Backend ApisixRouteRuleHTTPBackend `json:"backend"`
}

// ApisixRouteRuleHTTPBackend defines a HTTP backend.
type ApisixRouteRuleHTTPBackend struct {
	ServiceName string `json:"serviceName"`
	ServicePort int32  `json:"servicePort"`
}

// CreateApisixRoute creates an ApisixRoute object.
func (s *Scaffold) CreateApisixRoute(name string, rules []ApisixRouteRule) {
	route := &apisixRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixRoute",
			APIVersion: "apisix.apache.org/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apisixRouteSpec{
			Rules: rules,
		},
	}
	data, err := json.Marshal(route)
	assert.Nil(s.t, err)
	k8s.KubectlApplyFromString(s.t, s.kubectlOptions, string(data))
}

func (s *Scaffold) CreateApisixRouteByString(yaml string) {
	k8s.KubectlApplyFromString(s.t, s.kubectlOptions, yaml)
}

func ensureNumApisixCRDsCreated(url string, desired int) error {
	condFunc := func() (bool, error) {
		resp, err := http.Get(url)
		if err != nil {
			ginkgo.GinkgoT().Logf("failed to get resources from APISIX: %s", err.Error())
			return false, nil
		}
		if resp.StatusCode != http.StatusOK {
			ginkgo.GinkgoT().Logf("got status code %d from APISIX", resp.StatusCode)
			return false, nil
		}
		var c counter
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&c); err != nil {
			return false, err
		}
		// NOTE count field is a string.
		count, err := strconv.Atoi(c.Count)
		if err != nil {
			return false, err
		}
		// 1 for dir.
		if count != desired+1 {
			ginkgo.GinkgoT().Logf("mismatched number of items, expected %d but found %d", desired, count-1)
			return false, nil
		}
		return true, nil
	}
	return wait.Poll(3*time.Second, 35*time.Second, condFunc)
}

// EnsureNumApisixRoutesCreated waits until desired number of Routes are created in
// APISIX cluster.
func (s *Scaffold) EnsureNumApisixRoutesCreated(desired int) error {
	host, err := s.apisixAdminServiceURL()
	if err != nil {
		return err
	}
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/apisix/admin/routes",
	}
	return ensureNumApisixCRDsCreated(u.String(), desired)
}

// EnsureNumApisixUpstreamsCreated waits until desired number of Upstreams are created in
// APISIX cluster.
func (s *Scaffold) EnsureNumApisixUpstreamsCreated(desired int) error {
	host, err := s.apisixAdminServiceURL()
	if err != nil {
		return err
	}
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/apisix/admin/upstreams",
	}
	return ensureNumApisixCRDsCreated(u.String(), desired)
}

// ListApisixUpstreams list all upstream from APISIX
func (s *Scaffold) ListApisixUpstreams() (*apisix.UpstreamsResponse, error) {
	host, err := s.apisixAdminServiceURL()
	if err != nil {
		return nil, err
	}
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/apisix/admin/upstreams",
	}
	resp, err := http.Get(u.String())
	var responses *apisix.UpstreamsResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&responses); err != nil {
		return nil, err
	}
	return responses, nil
}
