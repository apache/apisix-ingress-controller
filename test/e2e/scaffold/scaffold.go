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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/testing"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Options struct {
	Name                         string
	Kubeconfig                   string
	APISIXAdminAPIVersion        string
	APISIXConfigPath             string
	IngressAPISIXReplicas        int
	HTTPBinServicePort           int
	APISIXAdminAPIKey            string
	EnableWebhooks               bool
	APISIXPublishAddress         string
	ApisixResourceSyncInterval   string
	ApisixResourceSyncComparison string
	ApisixResourceVersion        string
	DisableStatus                bool
	IngressClass                 string
	EnableEtcdServer             bool

	NamespaceSelectorLabel   map[string]string
	DisableNamespaceSelector bool
	DisableNamespaceLabel    bool
}

type Scaffold struct {
	opts               *Options
	kubectlOptions     *k8s.KubectlOptions
	namespace          string
	t                  testing.TestingT
	nodes              []corev1.Node
	etcdService        *corev1.Service
	apisixService      *corev1.Service
	httpbinService     *corev1.Service
	testBackendService *corev1.Service
	finalizers         []func()
	label              map[string]string

	apisixAdminTunnel      *k8s.Tunnel
	apisixHttpTunnel       *k8s.Tunnel
	apisixHttpsTunnel      *k8s.Tunnel
	apisixTCPTunnel        *k8s.Tunnel
	apisixTLSOverTCPTunnel *k8s.Tunnel
	apisixUDPTunnel        *k8s.Tunnel
	apisixControlTunnel    *k8s.Tunnel
}

type apisixResourceVersionInfo struct {
	V2      string
	Default string
}

var (
	apisixResourceVersion = &apisixResourceVersionInfo{
		V2:      config.ApisixV2,
		Default: config.DefaultAPIVersion,
	}

	createVersionedApisixResourceMap = map[string]struct{}{
		"ApisixRoute":        {},
		"ApisixConsumer":     {},
		"ApisixPluginConfig": {},
		"ApisixUpstream":     {},
	}
)

// GetKubeconfig returns the kubeconfig file path.
// Order:
// env KUBECONFIG;
// ~/.kube/config;
// "" (in case in-cluster configuration will be used).
func GetKubeconfig() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		kubeconfig = filepath.Join(u.HomeDir, ".kube", "config")
		if _, err := os.Stat(kubeconfig); err != nil && !os.IsNotExist(err) {
			kubeconfig = ""
		}
	}
	return kubeconfig
}

// NewScaffold creates an e2e test scaffold.
func NewScaffold(o *Options) *Scaffold {
	if o.Name == "" {
		o.Name = "default"
	}
	if o.IngressAPISIXReplicas <= 0 {
		o.IngressAPISIXReplicas = 1
	}
	if o.ApisixResourceVersion == "" {
		o.ApisixResourceVersion = ApisixResourceVersion().Default
	}
	if o.APISIXAdminAPIKey == "" {
		o.APISIXAdminAPIKey = "edd1c9f034335f136f87ad84b625c8f1"
	}
	if o.ApisixResourceSyncInterval == "" {
		o.ApisixResourceSyncInterval = "1h"
	}
	if o.ApisixResourceSyncComparison == "" {
		o.ApisixResourceSyncComparison = "true"
	}
	if o.Kubeconfig == "" {
		o.Kubeconfig = GetKubeconfig()
	}
	if o.APISIXAdminAPIVersion == "" {
		adminVersion := os.Getenv("APISIX_ADMIN_API_VERSION")
		if adminVersion != "" {
			o.APISIXAdminAPIVersion = adminVersion
		} else {
			o.APISIXAdminAPIVersion = "v3"
		}
	}
	if enabled := os.Getenv("ENABLED_ETCD_SERVER"); enabled == "true" {
		o.EnableEtcdServer = true
	}

	if o.APISIXConfigPath == "" {
		if o.EnableEtcdServer {
			o.APISIXConfigPath = "testdata/apisix-gw-config-v3-etcd-server.yaml"
		} else if o.APISIXAdminAPIVersion == "v3" {
			o.APISIXConfigPath = "testdata/apisix-gw-config-v3.yaml"
		} else {
			o.APISIXConfigPath = "testdata/apisix-gw-config.yaml"
		}
	}
	if o.HTTPBinServicePort == 0 {
		o.HTTPBinServicePort = 80
	}
	if o.IngressClass == "" {
		// Env acts on ci and will be deleted after the release of 1.17
		ingClass := os.Getenv("INGRESS_CLASS")
		if ingClass != "" {
			o.IngressClass = ingClass
		} else {
			o.IngressClass = config.IngressClass
		}
	}
	defer ginkgo.GinkgoRecover()

	s := &Scaffold{
		opts: o,
		t:    ginkgo.GinkgoT(),
	}
	// Disable logging of terratest library.
	logger.Default = logger.Discard
	logger.Global = logger.Discard
	logger.Terratest = logger.Discard

	ginkgo.BeforeEach(s.beforeEach)
	ginkgo.AfterEach(s.afterEach)

	return s
}

// NewV2Scaffold creates a scaffold with some default options.
func NewV2Scaffold(o *Options) *Scaffold {
	o.ApisixResourceVersion = ApisixResourceVersion().V2
	return NewScaffold(o)
}

// NewDefaultScaffold creates a scaffold with some default options.
// apisix-version default v2
func NewDefaultScaffold() *Scaffold {
	return NewScaffold(&Options{})
}

// NewDefaultV2Scaffold creates a scaffold with some default options.
func NewDefaultV2Scaffold() *Scaffold {
	opts := &Options{
		ApisixResourceVersion: ApisixResourceVersion().V2,
	}
	return NewScaffold(opts)
}

// Skip case if  is not supported etcdserver
func (s *Scaffold) IsEtcdServer() bool {
	return s.opts.EnableEtcdServer
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
	var ports []int32
	for _, p := range s.httpbinService.Spec.Ports {
		ports = append(ports, p.Port)
	}
	return s.httpbinService.Name, ports
}

// ApisixAdminServiceAndPort returns the apisix service name and
// it's admin port.
func (s *Scaffold) ApisixAdminServiceAndPort() (string, int32) {
	return "apisix-service-e2e-test", 9180
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
			httpexpect.NewAssertReporter(ginkgo.GinkgoT()),
		),
	})
}

// GetAPISIXHTTPSEndpoint get apisix https endpoint from tunnel map
func (s *Scaffold) GetAPISIXHTTPSEndpoint() string {
	return s.apisixHttpsTunnel.Endpoint()
}

// NewAPISIXClientWithTCPProxy creates the HTTP client but with the TCP proxy of APISIX.
func (s *Scaffold) NewAPISIXClientWithTCPProxy() *httpexpect.Expect {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixTCPTunnel.Endpoint(),
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
			httpexpect.NewAssertReporter(ginkgo.GinkgoT()),
		),
	})
}

// NewAPISIXClientWithTLSOverTCP creates a TSL over TCP client
func (s *Scaffold) NewAPISIXClientWithTLSOverTCP(host string) *httpexpect.Expect {
	u := url.URL{
		Scheme: "https",
		Host:   s.apisixTLSOverTCPTunnel.Endpoint(),
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
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(ginkgo.GinkgoT()),
		),
	})
}

func (s *Scaffold) NewMQTTClient() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", s.apisixTCPTunnel.Endpoint()))
	client := mqtt.NewClient(opts)
	return client
}

func (s *Scaffold) DNSResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: false,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, "udp", s.apisixUDPTunnel.Endpoint())
		},
	}
}

func (s *Scaffold) DialTLSOverTcp(serverName string) (*tls.Conn, error) {
	return tls.Dial("tcp", s.apisixTLSOverTCPTunnel.Endpoint(), &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         serverName,
	})
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
			httpexpect.NewAssertReporter(ginkgo.GinkgoT()),
		),
	})
}

// NewAPISIXHttpsClientWithCertificates creates the default HTTPS client with giving trusted CA and client certs.
func (s *Scaffold) NewAPISIXHttpsClientWithCertificates(host string, insecure bool, ca *x509.CertPool, certs []tls.Certificate) *httpexpect.Expect {
	u := url.URL{
		Scheme: "https",
		Host:   s.apisixHttpsTunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecure,
					ServerName:         host,
					RootCAs:            ca,
					Certificates:       certs,
				},
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(ginkgo.GinkgoT()),
		),
	})
}

// APISIXGatewayServiceEndpoint returns the apisix http gateway endpoint.
func (s *Scaffold) APISIXGatewayServiceEndpoint() string {
	return s.apisixHttpTunnel.Endpoint()
}

// RestartAPISIXDeploy delete apisix pod and wait new pod be ready
func (s *Scaffold) RestartAPISIXDeploy() {
	s.shutdownApisixTunnel()
	pods, err := k8s.ListPodsE(s.t, s.kubectlOptions, metav1.ListOptions{
		LabelSelector: "app=apisix-deployment-e2e-test",
	})
	assert.NoError(s.t, err, "list apisix pod")
	for _, pod := range pods {
		err = s.KillPod(pod.Name)
		assert.NoError(s.t, err, "killing apisix pod")
	}
	err = s.waitAllAPISIXPodsAvailable()
	assert.NoError(s.t, err, "waiting for new apisix instance ready")
	err = s.newAPISIXTunnels()
	assert.NoError(s.t, err, "renew apisix tunnels")
}

func (s *Scaffold) RestartIngressControllerDeploy() {
	if s.IsEtcdServer() {
		s.shutdownApisixTunnel()
	}
	pods, err := k8s.ListPodsE(s.t, s.kubectlOptions, metav1.ListOptions{
		LabelSelector: "app=ingress-apisix-controller-deployment-e2e-test",
	})
	assert.NoError(s.t, err, "list ingress-controller pod")
	for _, pod := range pods {
		err = s.KillPod(pod.Name)
		assert.NoError(s.t, err, "killing ingress-controller pod")
	}

	err = s.WaitAllIngressControllerPodsAvailable()
	assert.NoError(s.t, err, "waiting for new ingress-controller instance ready")

	if s.IsEtcdServer() {
		err = s.newAPISIXTunnels()
		assert.NoError(s.t, err, "renew apisix tunnels")
	}
}

func (s *Scaffold) beforeEach() {
	var err error
	s.namespace = fmt.Sprintf("ingress-apisix-e2e-tests-%s-%d", s.opts.Name, time.Now().Nanosecond())
	s.kubectlOptions = &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  s.namespace,
	}
	s.finalizers = nil

	if s.opts.NamespaceSelectorLabel != nil {
		s.label = s.opts.NamespaceSelectorLabel
	} else {
		s.label = map[string]string{"apisix.ingress.watch": s.namespace}
	}

	var nsLabel map[string]string
	if !s.opts.DisableNamespaceLabel {
		nsLabel = s.label
	}
	k8s.CreateNamespaceWithMetadata(s.t, s.kubectlOptions, metav1.ObjectMeta{Name: s.namespace, Labels: nsLabel})

	s.nodes, err = k8s.GetReadyNodesE(s.t, s.kubectlOptions)
	assert.Nil(s.t, err, "querying ready nodes")

	if s.opts.EnableEtcdServer {
		s.DeployCompositeMode()
	} else {
		s.DeployAdminaAPIMode()
	}
	s.DeployTestService()
	s.DeployRetryTimeout()
}

func (s *Scaffold) DeployAdminaAPIMode() {
	err := s.newAPISIXConfigMap(&APISIXConfig{
		EtcdServiceFQDN: EtcdServiceName,
	})
	assert.Nil(s.t, err, "creating apisix configmap")

	s.etcdService, err = s.newEtcd()
	assert.Nil(s.t, err, "initializing etcd")

	err = s.waitAllEtcdPodsAvailable()
	assert.Nil(s.t, err, "waiting for etcd ready")

	s.apisixService, err = s.newAPISIX()
	assert.Nil(s.t, err, "initializing Apache APISIX")

	err = s.waitAllAPISIXPodsAvailable()
	assert.Nil(s.t, err, "waiting for apisix ready")

	err = s.newIngressAPISIXController()
	assert.Nil(s.t, err, "initializing ingress apisix controller")

	err = s.WaitAllIngressControllerPodsAvailable()
	assert.Nil(s.t, err, "waiting for ingress apisix controller ready")

	err = s.newAPISIXTunnels()
	assert.Nil(s.t, err, "creating apisix tunnels")
}

func (s *Scaffold) DeployCompositeMode() {
	err := s.newAPISIXConfigMap(&APISIXConfig{
		EtcdServiceFQDN: "127.0.0.1",
	})
	assert.Nil(s.t, err, "creating apisix configmap")

	err = s.newIngressAPISIXController()
	assert.Nil(s.t, err, "initializing ingress apisix controller")

	err = s.WaitAllIngressControllerPodsAvailable()
	assert.Nil(s.t, err, "waiting for ingress apisix controller ready")

	err = s.newAPISIXTunnels()
	assert.Nil(s.t, err, "creating apisix tunnels")
}

func (s *Scaffold) DeployRetryTimeout() {
	//Two endpoints are blocking(10 second) and one is non blocking
	//Testing timeout
	//With 1 retry and a timeout of 5 sec, it should return 504(timeout)
	//With 1 retry and a timeout of 15 sec, it should success

	//Testing retry
	//With 1 retry and a timeout of 5 sec, it should return 504(timeout)
	//With 2 retry and a timeout of 5 sec, it should success
	err := s.NewDeploymentForRetryTimeoutTest()
	assert.Nil(s.t, err, "error creating deployments for retry and timeout")
	err = s.NewServiceForRetryTimeoutTest()
	assert.Nil(s.t, err, "error creating services for retry and timeout")
}

func (s *Scaffold) DeployTestService() {
	var err error

	s.httpbinService, err = s.newHTTPBIN()
	assert.Nil(s.t, err, "initializing httpbin")
	s.EnsureNumEndpointsReady(s.t, s.httpbinService.Name, 1)

	s.testBackendService, err = s.newTestBackend()
	assert.Nil(s.t, err, "initializing test backend")
	s.EnsureNumEndpointsReady(s.t, s.testBackendService.Name, 1)
}

func (s *Scaffold) afterEach() {
	defer ginkgo.GinkgoRecover()

	if ginkgo.CurrentSpecReport().Failed() {
		// dump and delete related resource
		env := os.Getenv("E2E_ENV")
		if env == "ci" {
			_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, "Dumping namespace contents")
			output, _ := k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), s.kubectlOptions, "get", "deploy,sts,svc,pods")
			if output != "" {
				_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, output)
			}
			output, _ = k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), s.kubectlOptions, "describe", "pods")
			if output != "" {
				_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, output)
			}
			// Get the logs of apisix
			output = s.GetDeploymentLogs("apisix-deployment-e2e-test")
			if output != "" {
				_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, output)
			}
			// Get the logs of ingress
			output = s.GetDeploymentLogs("ingress-apisix-controller-deployment-e2e-test")
			if output != "" {
				_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, output)
			}
			if s.opts.EnableWebhooks {
				output, _ = k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), s.kubectlOptions, "get", "validatingwebhookconfigurations", "-o", "yaml")
				if output != "" {
					_, _ = fmt.Fprintln(ginkgo.GinkgoWriter, output)
				}
			}
		}
		if env != "debug" {
			err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
			assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", s.namespace)
		}
	} else {
		// if the test case is successful, just delete namespace
		err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
		assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", s.namespace)
	}

	for _, f := range s.finalizers {
		runWithRecover(f)
	}

	// Wait for a while to prevent the worker node being overwhelming
	// (new cases will be run).
	time.Sleep(3 * time.Second)
}

func runWithRecover(f func()) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err, ok := r.(error)
		if ok {
			// just ignore already closed channel
			if strings.Contains(err.Error(), "close of closed channel") {
				return
			}
		}
		panic(r)
	}()
	f()
}

func (s *Scaffold) GetDeploymentLogs(name string) string {
	cli, err := k8s.GetKubernetesClientE(s.t)
	if err != nil {
		assert.Nilf(ginkgo.GinkgoT(), err, "get client error: %s", err.Error())
	}
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
			Param("container", name).
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

func (s *Scaffold) renderConfig(path string, config any) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	t := template.Must(template.New(path).Parse(string(data)))
	if err := t.Execute(&buf, config); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// FormatRegistry replace default registry to custom registry if exist
func (s *Scaffold) FormatRegistry(workloadTemplate string) string {
	customRegistry, isExist := os.LookupEnv("REGISTRY")
	if isExist {
		return strings.Replace(workloadTemplate, "localhost:5000", customRegistry, -1)
	} else {
		return workloadTemplate
	}
}

var (
	versionRegex = regexp.MustCompile(`apiVersion: apisix.apache.org/v.*?\n`)
	kindRegex    = regexp.MustCompile(`kind: (.*?)\n`)
)

func (s *Scaffold) replaceApiVersion(yml, ver string) string {
	return versionRegex.ReplaceAllString(yml, "apiVersion: "+ver+"\n")
}

func (s *Scaffold) getKindValue(yml string) string {
	subStr := kindRegex.FindStringSubmatch(yml)
	if len(subStr) < 2 {
		return ""
	}
	return subStr[1]
}

func waitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Steps:    8,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}

func (s *Scaffold) CreateVersionedApisixResource(yml string) error {
	kindValue := s.getKindValue(yml)
	if _, ok := createVersionedApisixResourceMap[kindValue]; ok {
		resource := s.replaceApiVersion(yml, s.opts.ApisixResourceVersion)
		return s.CreateResourceFromString(resource)
	}
	return fmt.Errorf("the resource %s does not support", kindValue)
}

func (s *Scaffold) CreateVersionedApisixResourceWithNamespace(yml, namespace string) error {
	kindValue := s.getKindValue(yml)
	if _, ok := createVersionedApisixResourceMap[kindValue]; ok {
		resource := s.replaceApiVersion(yml, s.opts.ApisixResourceVersion)
		return s.CreateResourceFromStringWithNamespace(resource, namespace)

	}
	return fmt.Errorf("the resource %s does not support", kindValue)
}

func (s *Scaffold) ApisixResourceVersion() string {
	return s.opts.ApisixResourceVersion
}

func ApisixResourceVersion() *apisixResourceVersionInfo {
	return apisixResourceVersion
}

func (s *Scaffold) DeleteResource(resourceType, name string) error {
	err := k8s.RunKubectlE(s.t, s.kubectlOptions, "delete", resourceType, name)
	if err != nil {
		log.Errorw("delete resource failed",
			zap.Error(err),
			zap.String("resource", resourceType),
			zap.String("name", name),
		)
	}
	return err
}

func (s *Scaffold) NamespaceSelectorLabelStrings() []string {
	var labels []string
	for k, v := range s.label {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	return labels
}

func (s *Scaffold) NamespaceSelectorLabel() map[string]string {
	return s.label
}
