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
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type counter struct {
	Count apisix.IntOrString `json:"count"`
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

// CreateResourceFromString creates resource from a loaded yaml string.
func (s *Scaffold) CreateResourceFromString(yaml string) error {
	return k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, yaml)
}

// RemoveResourceByString remove resource from a loaded yaml string.
func (s *Scaffold) RemoveResourceByString(yaml string) error {
	return k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, yaml)
}

func (s *Scaffold) GetServiceByName(name string) (*corev1.Service, error) {
	return k8s.GetServiceE(s.t, s.kubectlOptions, name)
}

// ListPodsByLabels lists all pods which matching the label selector.
func (s *Scaffold) ListPodsByLabels(labels string) ([]corev1.Pod, error) {
	return k8s.ListPodsE(s.t, s.kubectlOptions, metav1.ListOptions{
		LabelSelector: labels,
	})
}

// CreateResourceFromStringWithNamespace creates resource from a loaded yaml string
// and sets its namespace to the specified one.
func (s *Scaffold) CreateResourceFromStringWithNamespace(yaml, namespace string) error {
	originalNamespace := s.kubectlOptions.Namespace
	s.kubectlOptions.Namespace = namespace
	defer func() {
		s.kubectlOptions.Namespace = originalNamespace
	}()
	s.addFinalizers(func() {
		originalNamespace := s.kubectlOptions.Namespace
		s.kubectlOptions.Namespace = namespace
		defer func() {
			s.kubectlOptions.Namespace = originalNamespace
		}()
		assert.Nil(s.t, k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, yaml))
	})
	return k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, yaml)
}

func (s *Scaffold) ensureNumApisixCRDsCreated(url string, desired int) error {
	condFunc := func() (bool, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return false, err
		}
		if s.opts.APISIXAdminAPIKey != "" {
			req.Header.Set("X-API-Key", s.opts.APISIXAdminAPIKey)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			ginkgo.GinkgoT().Logf("failed to get resources from APISIX: %s", err.Error())
			return false, nil
		}
		if resp.StatusCode != http.StatusOK {
			ginkgo.GinkgoT().Logf("got status code %d from APISIX", resp.StatusCode)
			return false, nil
		}
		var c counter
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		err = json.Unmarshal(b, &c)
		if err != nil {
			return false, err
		}
		count := c.Count.IntValue
		if count != desired {
			ginkgo.GinkgoT().Logf("mismatched number of items, expected %d but found %d", desired, count)
			return false, nil
		}
		return true, nil
	}
	return wait.Poll(3*time.Second, 35*time.Second, condFunc)
}

// EnsureNumApisixRoutesCreated waits until desired number of Routes are created in
// APISIX cluster.
func (s *Scaffold) EnsureNumApisixRoutesCreated(desired int) error {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "/apisix/admin/routes",
	}
	return s.ensureNumApisixCRDsCreated(u.String(), desired)
}

// EnsureNumApisixStreamRoutesCreated waits until desired number of Stream Routes are created in
// APISIX cluster.
func (s *Scaffold) EnsureNumApisixStreamRoutesCreated(desired int) error {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "/apisix/admin/stream_routes",
	}
	return s.ensureNumApisixCRDsCreated(u.String(), desired)
}

// EnsureNumApisixUpstreamsCreated waits until desired number of Upstreams are created in
// APISIX cluster.
func (s *Scaffold) EnsureNumApisixUpstreamsCreated(desired int) error {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "/apisix/admin/upstreams",
	}
	return s.ensureNumApisixCRDsCreated(u.String(), desired)
}

// GetServerInfo collect server info from "/v1/server_info" (Control API) exposed by server-info plugin
func (s *Scaffold) GetServerInfo() (map[string]interface{}, error) {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixControlTunnel.Endpoint(),
		Path:   "/v1/server_info",
	}
	resp, err := http.Get(u.String())
	if err != nil {
		ginkgo.GinkgoT().Logf("failed to get response from Control API: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		ginkgo.GinkgoT().Logf("got status code %d from Control API", resp.StatusCode)
		return nil, err
	}
	var ret map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// ListApisixUpstreams list all upstreams from APISIX
func (s *Scaffold) ListApisixUpstreams() ([]*v1.Upstream, error) {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "/apisix/admin",
	}
	cli, err := apisix.NewClient()
	if err != nil {
		return nil, err
	}
	err = cli.AddCluster(&apisix.ClusterOptions{
		BaseURL:  u.String(),
		AdminKey: s.opts.APISIXAdminAPIKey,
	})
	if err != nil {
		return nil, err
	}
	return cli.Cluster("").Upstream().List(context.TODO())
}

// ListApisixGlobalRules list all global_rules from APISIX
func (s *Scaffold) ListApisixGlobalRules() ([]*v1.GlobalRule, error) {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "/apisix/admin",
	}
	cli, err := apisix.NewClient()
	if err != nil {
		return nil, err
	}
	err = cli.AddCluster(&apisix.ClusterOptions{
		BaseURL:  u.String(),
		AdminKey: s.opts.APISIXAdminAPIKey,
	})
	if err != nil {
		return nil, err
	}
	return cli.Cluster("").GlobalRule().List(context.TODO())
}

// ListApisixRoutes list all routes from APISIX.
func (s *Scaffold) ListApisixRoutes() ([]*v1.Route, error) {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "/apisix/admin",
	}
	cli, err := apisix.NewClient()
	if err != nil {
		return nil, err
	}
	err = cli.AddCluster(&apisix.ClusterOptions{
		BaseURL:  u.String(),
		AdminKey: s.opts.APISIXAdminAPIKey,
	})
	if err != nil {
		return nil, err
	}
	return cli.Cluster("").Route().List(context.TODO())
}

// ListApisixConsumers list all consumers from APISIX.
func (s *Scaffold) ListApisixConsumers() ([]*v1.Consumer, error) {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "apisix/admin",
	}
	cli, err := apisix.NewClient()
	if err != nil {
		return nil, err
	}
	err = cli.AddCluster(&apisix.ClusterOptions{
		BaseURL:  u.String(),
		AdminKey: s.opts.APISIXAdminAPIKey,
	})
	if err != nil {
		return nil, err
	}
	return cli.Cluster("").Consumer().List(context.TODO())
}

// ListApisixStreamRoutes list all stream_routes from APISIX.
func (s *Scaffold) ListApisixStreamRoutes() ([]*v1.StreamRoute, error) {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "/apisix/admin",
	}
	cli, err := apisix.NewClient()
	if err != nil {
		return nil, err
	}
	err = cli.AddCluster(&apisix.ClusterOptions{
		BaseURL:  u.String(),
		AdminKey: s.opts.APISIXAdminAPIKey,
	})
	if err != nil {
		return nil, err
	}
	return cli.Cluster("").StreamRoute().List(context.TODO())
}

// ListApisixSsl list all ssl from APISIX
func (s *Scaffold) ListApisixSsl() ([]*v1.Ssl, error) {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixAdminTunnel.Endpoint(),
		Path:   "/apisix/admin",
	}
	cli, err := apisix.NewClient()
	if err != nil {
		return nil, err
	}
	err = cli.AddCluster(&apisix.ClusterOptions{
		BaseURL:  u.String(),
		AdminKey: s.opts.APISIXAdminAPIKey,
	})
	if err != nil {
		return nil, err
	}
	return cli.Cluster("").SSL().List(context.TODO())
}

func (s *Scaffold) newAPISIXTunnels() error {
	var (
		adminNodePort   int
		httpNodePort    int
		httpsNodePort   int
		tcpNodePort     int
		udpNodePort     int
		controlNodePort int
		adminPort       int
		httpPort        int
		httpsPort       int
		tcpPort         int
		udpPort         int
		controlPort     int
	)
	for _, port := range s.apisixService.Spec.Ports {
		if port.Name == "http" {
			httpNodePort = int(port.NodePort)
			httpPort = int(port.Port)
		} else if port.Name == "https" {
			httpsNodePort = int(port.NodePort)
			httpsPort = int(port.Port)
		} else if port.Name == "http-admin" {
			adminNodePort = int(port.NodePort)
			adminPort = int(port.Port)
		} else if port.Name == "tcp" {
			tcpNodePort = int(port.NodePort)
			tcpPort = int(port.Port)
		} else if port.Name == "udp" {
			udpNodePort = int(port.NodePort)
			udpPort = int(port.Port)
		} else if port.Name == "http-control" {
			controlNodePort = int(port.NodePort)
			controlPort = int(port.Port)
		}
	}

	s.apisixAdminTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix-service-e2e-test",
		adminNodePort, adminPort)
	s.apisixHttpTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix-service-e2e-test",
		httpNodePort, httpPort)
	s.apisixHttpsTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix-service-e2e-test",
		httpsNodePort, httpsPort)
	s.apisixTCPTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix-service-e2e-test",
		tcpNodePort, tcpPort)
	s.apisixUDPTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix-service-e2e-test",
		udpNodePort, udpPort)
	s.apisixControlTunnel = k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix-service-e2e-test",
		controlNodePort, controlPort)

	if err := s.apisixAdminTunnel.ForwardPortE(s.t); err != nil {
		return err
	}
	s.addFinalizers(s.apisixAdminTunnel.Close)
	if err := s.apisixHttpTunnel.ForwardPortE(s.t); err != nil {
		return err
	}
	s.addFinalizers(s.apisixHttpTunnel.Close)
	if err := s.apisixHttpsTunnel.ForwardPortE(s.t); err != nil {
		return err
	}
	s.addFinalizers(s.apisixHttpsTunnel.Close)
	if err := s.apisixTCPTunnel.ForwardPortE(s.t); err != nil {
		return err
	}
	s.addFinalizers(s.apisixTCPTunnel.Close)
	if err := s.apisixUDPTunnel.ForwardPortE(s.t); err != nil {
		return err
	}
	s.addFinalizers(s.apisixUDPTunnel.Close)
	if err := s.apisixControlTunnel.ForwardPortE(s.t); err != nil {
		return err
	}
	s.addFinalizers(s.apisixControlTunnel.Close)
	return nil
}

// Namespace returns the current working namespace.
func (s *Scaffold) Namespace() string {
	return s.kubectlOptions.Namespace
}
