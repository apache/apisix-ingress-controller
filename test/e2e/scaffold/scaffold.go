// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package scaffold

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

const (
	DefaultControllerName = "apisix.apache.org/apisix-ingress-controller"
)

type Options struct {
	Name              string
	Kubeconfig        string
	APISIXAdminAPIKey string
	ControllerName    string

	NamespaceSelectorLabel map[string][]string
	DisableNamespaceLabel  bool
	SkipHooks              bool
}

type Scaffold struct {
	*framework.Framework

	opts             *Options
	kubectlOptions   *k8s.KubectlOptions
	namespace        string
	t                testing.TestingT
	dataplaneService *corev1.Service
	httpbinService   *corev1.Service

	finalizers []func()
	label      map[string]string

	apisixHttpTunnel  *k8s.Tunnel
	apisixHttpsTunnel *k8s.Tunnel

	additionalGateways map[string]*GatewayResources

	Deployer Deployer
}

// GatewayResources contains resources associated with a specific Gateway group
type GatewayResources struct {
	Namespace        string
	DataplaneService *corev1.Service
	HttpTunnel       *k8s.Tunnel
	HttpsTunnel      *k8s.Tunnel
	AdminAPIKey      string
}

func (g *GatewayResources) GetAdminEndpoint() string {
	return fmt.Sprintf("http://%s.%s:9180", g.DataplaneService.Name, g.DataplaneService.Namespace)
}

func (s *Scaffold) AdminKey() string {
	return s.opts.APISIXAdminAPIKey
}

// NewScaffold creates an e2e test scaffold.
func NewScaffold(o *Options) *Scaffold {
	if o.Name == "" {
		o.Name = "default"
	}
	if o.Kubeconfig == "" {
		o.Kubeconfig = GetKubeconfig()
	}

	defer GinkgoRecover()

	s := &Scaffold{
		Framework: framework.GetFramework(),
		opts:      o,
		t:         GinkgoT(),
	}

	s.Deployer = NewDeployer(s)

	if !s.opts.SkipHooks {
		BeforeEach(s.Deployer.BeforeEach)
		AfterEach(s.Deployer.AfterEach)
	}

	return s
}

// NewDefaultScaffold creates a scaffold with some default options.
// apisix-version default v2
func NewDefaultScaffold() *Scaffold {
	return NewScaffold(&Options{})
}

// KillPod kill the pod which name is podName.
func (s *Scaffold) KillPod(podName string) error {
	cli, err := k8s.GetKubernetesClientE(s.t)
	if err != nil {
		return err
	}
	return cli.CoreV1().Pods(s.namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
}

// DefaultHTTPBackend returns the service name and service ports
// of the default http backend.
func (s *Scaffold) DefaultHTTPBackend() (string, []int32) {
	ports := make([]int32, 0, len(s.httpbinService.Spec.Ports))
	for _, p := range s.httpbinService.Spec.Ports {
		ports = append(ports, p.Port)
	}
	return s.httpbinService.Name, ports
}

// NewAPISIXClient creates the default HTTP client.
func (s *Scaffold) NewAPISIXClient() *httpexpect.Expect {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixHttpTunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}

// GetAPISIXHTTPSEndpoint get apisix https endpoint from tunnel map
func (s *Scaffold) GetAPISIXHTTPSEndpoint() string {
	return s.apisixHttpsTunnel.Endpoint()
}

func (s *Scaffold) UpdateNamespace(ns string) {
	s.kubectlOptions.Namespace = ns
}

// NewAPISIXHttpsClient creates the default HTTPS client.
func (s *Scaffold) NewAPISIXHttpsClient(host string) *httpexpect.Expect {
	u := url.URL{
		Scheme: "https",
		Host:   s.apisixHttpsTunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// accept any certificate; for testing only!
					InsecureSkipVerify: true,
					ServerName:         host,
				},
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}

func (s *Scaffold) DefaultDataplaneResource() DataplaneResource {
	return s.Deployer.DefaultDataplaneResource()
}

func (s *Scaffold) DeployTestService() {
	var err error

	s.httpbinService, err = s.newHTTPBIN()
	Expect(err).NotTo(HaveOccurred(), "creating httpbin service")
	s.EnsureNumEndpointsReady(s.t, s.httpbinService.Name, 1)
}

func (s *Scaffold) GetDeploymentLogs(name string) string {
	cli, err := k8s.GetKubernetesClientE(s.t)
	Expect(err).NotTo(HaveOccurred(), "getting k8s client")

	pods, err := cli.CoreV1().Pods(s.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=" + name,
	})
	if err != nil {
		return ""
	}
	var buf strings.Builder
	for _, pod := range pods.Items {
		buf.WriteString(fmt.Sprintf("=== pod: %s ===\n", pod.Name))
		logs, err := cli.CoreV1().RESTClient().Get().
			Resource("pods").
			Namespace(s.namespace).
			Name(pod.Name).SubResource("log").
			Do(context.TODO()).
			Raw()
		if err == nil {
			buf.Write(logs)
		}
		buf.WriteByte('\n')
	}
	return buf.String()
}

func (s *Scaffold) addFinalizers(f func()) {
	s.finalizers = append(s.finalizers, f)
}

// FormatRegistry replace default registry to custom registry if exist
func (s *Scaffold) FormatRegistry(workloadTemplate string) string {
	customRegistry, isExist := os.LookupEnv("REGISTRY")
	if isExist {
		return strings.ReplaceAll(workloadTemplate, "127.0.0.1:5000", customRegistry)
	} else {
		return workloadTemplate
	}
}

func (s *Scaffold) DeleteResource(resourceType, name string) error {
	return k8s.RunKubectlE(s.t, s.kubectlOptions, "delete", resourceType, name)
}

func (s *Scaffold) NamespaceSelectorLabelStrings() []string {
	var labels []string
	if s.opts.NamespaceSelectorLabel != nil {
		for k, v := range s.opts.NamespaceSelectorLabel {
			for _, v0 := range v {
				labels = append(labels, fmt.Sprintf("%s=%s", k, v0))
			}
		}
	} else {
		for k, v := range s.label {
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return labels
}

func (s *Scaffold) NamespaceSelectorLabel() map[string][]string {
	return s.opts.NamespaceSelectorLabel
}
func (s *Scaffold) labelSelector(label string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: label,
	}
}

func (s *Scaffold) GetControllerName() string {
	return s.opts.ControllerName
}

// createDataplaneTunnels creates HTTP and HTTPS tunnels for a dataplane service.
// It's extracted from newAPISIXTunnels to be reusable for additional gateway groups.
func (s *Scaffold) createDataplaneTunnels(
	svc *corev1.Service,
	kubectlOpts *k8s.KubectlOptions,
	serviceName string,
) (*k8s.Tunnel, *k8s.Tunnel, error) {
	var (
		httpNodePort  int
		httpsNodePort int
		httpPort      int
		httpsPort     int
	)

	for _, port := range svc.Spec.Ports {
		switch port.Name {
		case "http":
			httpNodePort = int(port.NodePort)
			httpPort = int(port.Port)
		case "https":
			httpsNodePort = int(port.NodePort)
			httpsPort = int(port.Port)
		}
	}

	httpTunnel := k8s.NewTunnel(kubectlOpts, k8s.ResourceTypeService, serviceName,
		httpNodePort, httpPort)
	httpsTunnel := k8s.NewTunnel(kubectlOpts, k8s.ResourceTypeService, serviceName,
		httpsNodePort, httpsPort)

	if err := httpTunnel.ForwardPortE(s.t); err != nil {
		return nil, nil, err
	}
	s.addFinalizers(s.closeApisixHttpTunnel)

	if err := httpsTunnel.ForwardPortE(s.t); err != nil {
		httpTunnel.Close()
		return nil, nil, err
	}
	s.addFinalizers(s.closeApisixHttpsTunnel)

	return httpTunnel, httpsTunnel, nil
}

func (s *Scaffold) closeApisixHttpTunnel() {
	if s.apisixHttpTunnel != nil {
		s.apisixHttpTunnel.Close()
		s.apisixHttpTunnel = nil
	}
}

func (s *Scaffold) closeApisixHttpsTunnel() {
	if s.apisixHttpsTunnel != nil {
		s.apisixHttpsTunnel.Close()
		s.apisixHttpsTunnel = nil
	}
}

// GetAdditionalGateway returns resources associated with a specific gateway
func (s *Scaffold) GetAdditionalGateway(identifier string) (*GatewayResources, bool) {
	resources, exists := s.additionalGateways[identifier]
	return resources, exists
}

// NewAPISIXClientForGateway creates an HTTP client for a specific gateway
func (s *Scaffold) NewAPISIXClientForGateway(identifier string) (*httpexpect.Expect, error) {
	resources, exists := s.additionalGateways[identifier]
	if !exists {
		return nil, fmt.Errorf("gateway %s not found", identifier)
	}

	u := url.URL{
		Scheme: "http",
		Host:   resources.HttpTunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	}), nil
}

// NewAPISIXHttpsClientForGateway creates an HTTPS client for a specific gateway
func (s *Scaffold) NewAPISIXHttpsClientForGateway(identifier string, host string) (*httpexpect.Expect, error) {
	resources, exists := s.additionalGateways[identifier]
	if !exists {
		return nil, fmt.Errorf("gateway %s not found", identifier)
	}

	u := url.URL{
		Scheme: "https",
		Host:   resources.HttpsTunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// accept any certificate; for testing only!
					InsecureSkipVerify: true,
					ServerName:         host,
				},
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	}), nil
}

// GetGatewayHTTPEndpoint returns the HTTP endpoint for a specific gateway
func (s *Scaffold) GetGatewayHTTPEndpoint(identifier string) (string, error) {
	resources, exists := s.additionalGateways[identifier]
	if !exists {
		return "", fmt.Errorf("gateway %s not found", identifier)
	}

	return resources.HttpTunnel.Endpoint(), nil
}

// GetGatewayHTTPSEndpoint returns the HTTPS endpoint for a specific gateway
func (s *Scaffold) GetGatewayHTTPSEndpoint(identifier string) (string, error) {
	resources, exists := s.additionalGateways[identifier]
	if !exists {
		return "", fmt.Errorf("gateway %s not found", identifier)
	}

	return resources.HttpsTunnel.Endpoint(), nil
}

func (s *Scaffold) GetDataplaneService() *corev1.Service {
	return s.dataplaneService
}

func (s *Scaffold) KubeOpts() *k8s.KubectlOptions {
	return s.kubectlOptions
}

func NewClient(scheme, host string) *httpexpect.Expect {
	u := url.URL{
		Scheme: scheme,
		Host:   host,
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	})
}
