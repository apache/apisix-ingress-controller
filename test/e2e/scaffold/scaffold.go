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
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gavv/httpexpect/v2"
	"github.com/gorilla/websocket"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
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

	SkipHooks bool

	EnableWebhook bool

	// NamespaceSelector is rendered into the namespace_selector of the
	// controller configuration.
	NamespaceSelector []string
}

type Scaffold struct {
	*framework.Framework

	// opts holds the original, user-provided options.
	// It is treated as read-only and must not be modified after initialization.
	opts Options

	kubectlOptions   *k8s.KubectlOptions
	namespace        string
	t                testing.TestingT
	dataplaneService *corev1.Service
	httpbinService   *corev1.Service

	finalizers    []func()
	apisixTunnels *Tunnels

	additionalGateways map[string]*GatewayResources

	runtimeOpts Options
	Deployer    Deployer

	client client.Client
}

type Tunnels struct {
	HTTP  Tunnel
	HTTPS Tunnel
	TCP   Tunnel
	TLS   Tunnel
	// TLSPassthrough reaches the stream listen that prereads the SNI and
	// forwards the connection without terminating it.
	TLSPassthrough Tunnel
}

func (t *Tunnels) Close() {
	if t.HTTP != nil {
		t.safeClose(t.HTTP.Close)
		t.HTTP = nil
	}
	if t.HTTPS != nil {
		t.safeClose(t.HTTPS.Close)
		t.HTTPS = nil
	}
	if t.TCP != nil {
		t.safeClose(t.TCP.Close)
		t.TCP = nil
	}
	if t.TLS != nil {
		t.safeClose(t.TLS.Close)
		t.TLS = nil
	}
	if t.TLSPassthrough != nil {
		t.safeClose(t.TLSPassthrough.Close)
		t.TLSPassthrough = nil
	}
}

func (t *Tunnels) safeClose(close func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic when closing tunnel: %v", r)
		}
	}()

	close()
}

// GatewayResources contains resources associated with a specific Gateway group
type GatewayResources struct {
	Namespace        string
	DataplaneService *corev1.Service
	Tunnels          *Tunnels
	AdminAPIKey      string
}

func (g *GatewayResources) GetAdminEndpoint() string {
	return fmt.Sprintf("http://%s.%s:9180", g.DataplaneService.Name, g.DataplaneService.Namespace)
}

func (s *Scaffold) AdminKey() string {
	return s.runtimeOpts.APISIXAdminAPIKey
}

// NewScaffold creates an e2e test scaffold.
func NewScaffold(o Options) *Scaffold {
	if o.Name == "" {
		o.Name = defaultProfileName
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

	if !o.SkipHooks {
		BeforeEach(s.Deployer.BeforeEach)
		AfterEach(s.Deployer.AfterEach)
	}

	return s
}

// NewDefaultScaffold creates a scaffold with some default options.
// apisix-version default v2
func NewDefaultScaffold() *Scaffold {
	return NewScaffold(Options{})
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
		Scheme: apiv2.SchemeHTTP,
		Host:   s.apisixTunnels.HTTP.Endpoint(),
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

func (s *Scaffold) NewAPISIXClientOnTCPPort() *httpexpect.Expect {
	u := url.URL{
		Scheme: "http",
		Host:   s.apisixTCPEndpoint(),
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

func (s *Scaffold) ApisixHTTPEndpoint() string {
	return s.apisixTunnels.HTTP.Endpoint()
}

// GetAPISIXHTTPSEndpoint get apisix https endpoint from tunnel map
func (s *Scaffold) GetAPISIXHTTPSEndpoint() string {
	return s.apisixTunnels.HTTPS.Endpoint()
}

func (s *Scaffold) GetAPISIXTCPEndpoint() string {
	return s.apisixTCPEndpoint()
}

// apisixTCPEndpoint returns the local TCP tunnel address forced to IPv4.
// terratest's Endpoint() yields "localhost:<port>", which resolves to ::1
// first; the client-go port-forward is not guaranteed to bind ::1, so raw TCP
// dials (e.g. the MQTT client, net.Dial) fail with "connection refused" on IPv6.
func (s *Scaffold) apisixTCPEndpoint() string {
	return strings.Replace(s.apisixTunnels.TCP.Endpoint(), "localhost", "127.0.0.1", 1)
}

func (s *Scaffold) UpdateNamespace(ns string) {
	s.kubectlOptions.Namespace = ns
}

// NewAPISIXHttpsClient creates the default HTTPS client.
func (s *Scaffold) NewAPISIXHttpsClient(host string) *httpexpect.Expect {
	u := url.URL{
		Scheme: apiv2.SchemeHTTPS,
		Host:   s.apisixTunnels.HTTPS.Endpoint(),
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

// NewAPISIXClientWithTCPProxy creates the HTTP client but with the TCP proxy of APISIX.
func (s *Scaffold) NewAPISIXClientWithTCPProxy() *httpexpect.Expect {
	u := url.URL{
		Scheme: apiv2.SchemeHTTP,
		Host:   s.apisixTCPEndpoint(),
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
			httpexpect.NewAssertReporter(s.GinkgoT),
		),
	})
}

// tlsPassthroughPortName is the data plane Service port that forwards to the
// APISIX stream listen configured with tls_passthrough.
const tlsPassthroughPortName = "tls-passthrough"

// NewAPISIXClientWithTLSPassthrough dials the data plane's TLS passthrough
// stream listen with sni and verifies the served chain against caCert.
//
// Verifying is the point: the gateway holds no certificate for a passthrough
// listener, so a chain that validates against the backend's own CA can only
// have come from the backend itself.
func (s *Scaffold) NewAPISIXClientWithTLSPassthrough(sni string, caCert []byte) *httpexpect.Expect {
	pool := x509.NewCertPool()
	Expect(pool.AppendCertsFromPEM(caCert)).To(BeTrue(), "parsing CA certificate")

	forward := s.newTLSPassthroughForward(&tls.Config{ServerName: sni, RootCAs: pool})

	u := url.URL{
		Scheme: apiv2.SchemeHTTPS,
		Host:   forward.endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{
				// Every request dials afresh, so a forwarder that died since the
				// last one is noticed and replaced rather than reused.
				DisableKeepAlives: true,
				DialTLSContext:    forward.dialTLSContext,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(s.GinkgoT),
		),
	})
}

// tlsPassthroughForward owns the port-forward to the data plane's
// tls_passthrough stream port and replaces it whenever it stops working.
//
// Two things make this port different from the others the scaffold forwards.
//
// A tls_passthrough listen prereads the ClientHello to pick a stream route, and
// resets a connection no route matches. The port-forward treats that reset as
// fatal and exits, taking the local listener with it - every later dial then
// gets "connection refused" rather than anything describing the cause. A
// handshake that succeeded once does not rule this out: the data plane workers
// load a new stream route independently, so the next connection can still land
// on one that has not. So the forward is verified with a real handshake for the
// SNI, never a bare TCP dial, and any failed dial rebuilds it instead of
// leaving every later retry to hit a forwarder that is already gone.
//
// It is also forwarded on demand rather than alongside the HTTP tunnels when the
// data plane is deployed: APISIX opens its stream listens later than its HTTP
// ones, and only this one spec dials this port, so the rest of the suite would
// pay for a forward it never uses.
type tlsPassthroughForward struct {
	s         *Scaffold
	opts      k8s.KubectlOptions
	service   string
	port      int
	tlsConfig *tls.Config

	mu     sync.Mutex
	tunnel *k8s.Tunnel
}

func (s *Scaffold) newTLSPassthroughForward(tlsConfig *tls.Config) *tlsPassthroughForward {
	svc := s.dataplaneService
	Expect(svc).NotTo(BeNil(), "data plane service")

	var port int
	for _, p := range svc.Spec.Ports {
		if p.Name == tlsPassthroughPortName {
			port = int(p.Port)
		}
	}
	Expect(port).NotTo(BeZero(), "data plane service has no %s port", tlsPassthroughPortName)

	// Copy the scaffold's options rather than building fresh ones: a run that
	// configures a kubeconfig or context would otherwise be ignored here while
	// every other dataplane tunnel honours it.
	f := &tlsPassthroughForward{
		s:         s,
		opts:      *s.kubectlOptions,
		service:   svc.Name,
		port:      port,
		tlsConfig: tlsConfig,
	}
	f.opts.Namespace = svc.Namespace

	Eventually(func() error {
		t, err := f.open()
		if err != nil {
			return err
		}
		f.replace(t)
		return nil
	}).WithTimeout(time.Minute*2).WithPolling(time.Second*2).
		Should(Succeed(), "forwarding the tls passthrough port")
	return f
}

// open forwards the port and proves the forward with a handshake, since
// ForwardPortE returns before the forward has proxied anything.
func (f *tlsPassthroughForward) open() (*k8s.Tunnel, error) {
	t := k8s.NewTunnel(&f.opts, k8s.ResourceTypeService, f.service, 0, f.port)
	if err := t.ForwardPortE(f.s.t); err != nil {
		return nil, err
	}
	conn, err := f.handshake(context.Background(), t.Endpoint())
	if err != nil {
		t.Close()
		return nil, err
	}
	_ = conn.Close()
	return t, nil
}

func (f *tlsPassthroughForward) handshake(ctx context.Context, endpoint string) (*tls.Conn, error) {
	d := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 10 * time.Second},
		Config:    f.tlsConfig.Clone(),
	}
	conn, err := d.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	return conn.(*tls.Conn), nil
}

// replace installs t as the current forward and closes the one it supersedes.
// Tunnels.Close only knows about the tunnel stored there, so a superseded
// forward would otherwise keep running until the process exits.
func (f *tlsPassthroughForward) replace(t *k8s.Tunnel) {
	if f.tunnel != nil {
		f.tunnel.Close()
	} else if previous := f.s.apisixTunnels.TLSPassthrough; previous != nil {
		previous.Close()
	}
	f.tunnel = t
	f.s.apisixTunnels.TLSPassthrough = t
}

func (f *tlsPassthroughForward) endpoint() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tunnel.Endpoint()
}

// dialTLSContext ignores addr: the request URL names whichever local port the
// forward had when the client was built, and that port changes on a rebuild.
func (f *tlsPassthroughForward) dialTLSContext(ctx context.Context, _, _ string) (net.Conn, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	conn, err := f.handshake(ctx, f.tunnel.Endpoint())
	if err == nil {
		return conn, nil
	}
	t, rebuildErr := f.open()
	if rebuildErr != nil {
		return nil, fmt.Errorf("%w; rebuilding the tls passthrough forward: %v", err, rebuildErr)
	}
	f.replace(t)
	return f.handshake(ctx, t.Endpoint())
}

func (s *Scaffold) NewAPISIXClientWithTLSProxy(host string) *httpexpect.Expect {
	u := url.URL{
		Scheme: apiv2.SchemeHTTPS,
		Host:   s.apisixTunnels.TLS.Endpoint(),
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
			httpexpect.NewAssertReporter(s.GinkgoT),
		),
	})
}

// NewAPISIXClientForPort creates an HTTP client for a specific APISIX port.
// For built-in ports (80, 443, 9100), it reuses the existing helpers/tunnels.
// For any other port, it creates a new tunnel for that call.
func (s *Scaffold) NewAPISIXClientForPort(port int) (*httpexpect.Expect, error) {
	// Check if we can reuse existing tunnels
	switch port {
	case 80:
		return s.NewAPISIXClient(), nil
	case 443:
		return s.NewAPISIXHttpsClient(""), nil
	case 9100:
		return s.NewAPISIXClientOnTCPPort(), nil
	}

	// Create new tunnel for custom port
	serviceName := s.dataplaneService.Name
	tunnel := k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, serviceName, 0, port)
	if err := tunnel.ForwardPortE(s.t); err != nil {
		return nil, fmt.Errorf("failed to create tunnel for port %d: %w", port, err)
	}
	s.addFinalizers(tunnel.Close)

	u := url.URL{
		Scheme: "http",
		Host:   tunnel.Endpoint(),
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
			Timeout:   3 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(GinkgoT()),
		),
	}), nil
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
		for _, c := range pod.Spec.Containers {
			buf.WriteString(fmt.Sprintf("--- container: %s ---\n", c.Name))
			logs, err := cli.CoreV1().RESTClient().Get().
				Resource("pods").
				Namespace(s.namespace).
				Name(pod.Name).
				SubResource("log").
				Param("container", c.Name).
				Do(context.TODO()).
				Raw()
			if err == nil {
				buf.Write(logs)
			} else {
				buf.WriteString(fmt.Sprintf("Error getting logs: %v\n", err))
			}
			buf.WriteByte('\n')
		}
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

func (s *Scaffold) labelSelector(label string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: label,
	}
}

func (s *Scaffold) GetControllerName() string {
	return s.runtimeOpts.ControllerName
}

// createDataplaneTunnels creates HTTP and HTTPS tunnels for a dataplane service.
// It's extracted from newAPISIXTunnels to be reusable for additional gateway groups.
func (s *Scaffold) createDataplaneTunnels(
	svc *corev1.Service,
	kubectlOpts *k8s.KubectlOptions,
	serviceName string,
) (*Tunnels, error) {
	var (
		httpPort  int
		httpsPort int
		tcpPort   int
		tlsPort   int
	)

	for _, port := range svc.Spec.Ports {
		switch port.Name {
		case apiv2.SchemeHTTP:
			httpPort = int(port.Port)
		case apiv2.SchemeHTTPS:
			httpsPort = int(port.Port)
		case apiv2.SchemeTCP:
			tcpPort = int(port.Port)
		case apiv2.SchemeTLS:
			tlsPort = int(port.Port)
		}
	}

	tunnels := &Tunnels{}
	s.addFinalizers(tunnels.Close)

	httpTunnel := k8s.NewTunnel(kubectlOpts, k8s.ResourceTypeService, serviceName,
		0, httpPort)
	httpsTunnel := k8s.NewTunnel(kubectlOpts, k8s.ResourceTypeService, serviceName,
		0, httpsPort)
	tlsTunnel := k8s.NewTunnel(kubectlOpts, k8s.ResourceTypeService, serviceName,
		0, tlsPort)

	if err := httpTunnel.ForwardPortE(s.t); err != nil {
		return nil, err
	}
	tunnels.HTTP = httpTunnel

	if err := httpsTunnel.ForwardPortE(s.t); err != nil {
		return nil, err
	}
	tunnels.HTTPS = httpsTunnel

	// The raw-TCP stream port uses a WebSocket-capable tunnel; see wsTunnel.
	tcpTunnel, err := newWebsocketServiceTunnel(s.t, kubectlOpts, serviceName, tcpPort)
	if err != nil {
		return nil, err
	}
	tunnels.TCP = tcpTunnel
	if err := tlsTunnel.ForwardPortE(s.t); err != nil {
		return nil, err
	}
	tunnels.TLS = tlsTunnel

	// The TLS passthrough tunnel is deliberately not forwarded here. See
	// tlsPassthroughForward.

	return tunnels, nil
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
		Scheme: apiv2.SchemeHTTP,
		Host:   resources.Tunnels.HTTP.Endpoint(),
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
		Scheme: apiv2.SchemeHTTPS,
		Host:   resources.Tunnels.HTTPS.Endpoint(),
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

	return resources.Tunnels.HTTP.Endpoint(), nil
}

// GetGatewayHTTPSEndpoint returns the HTTPS endpoint for a specific gateway
func (s *Scaffold) GetGatewayHTTPSEndpoint(identifier string) (string, error) {
	resources, exists := s.additionalGateways[identifier]
	if !exists {
		return "", fmt.Errorf("gateway %s not found", identifier)
	}

	return resources.Tunnels.HTTPS.Endpoint(), nil
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

func (s *Scaffold) GetMetricsEndpoint() string {
	tunnel := k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix-ingress-controller-manager-metrics-service", 8080, 8080)
	if err := tunnel.ForwardPortE(s.t); err != nil {
		return ""
	}
	s.addFinalizers(tunnel.Close)
	return fmt.Sprintf("http://%s/metrics", tunnel.Endpoint())
}

func (s *Scaffold) NewWebsocketClient(tls *tls.Config, path string, headers http.Header) *websocket.Conn {
	var host = s.ApisixHTTPEndpoint()
	var scheme = "ws"
	if tls != nil {
		scheme = "wss"
		host = s.GetAPISIXHTTPSEndpoint()
	}

	dialer := websocket.Dialer{
		TLSClientConfig: tls,
	}

	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}
	var conn *websocket.Conn

	s.RetryAssertion(func() error {
		c, resp, err := dialer.Dial(u.String(), headers)
		if err != nil {
			return err
		}
		if resp == nil || resp.StatusCode != http.StatusSwitchingProtocols {
			_ = c.Close()
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		conn = c
		return nil
	}).ShouldNot(HaveOccurred(), "establishing websocket connection")
	return conn
}

func (s *Scaffold) ControlAPIClient() (ControlAPIClient, error) {
	tunnel := k8s.NewTunnel(s.kubectlOptions, k8s.ResourceTypeService, "apisix-control-api", 9090, 9090)
	if err := tunnel.ForwardPortE(s.t); err != nil {
		return nil, err
	}
	s.addFinalizers(tunnel.Close)

	return &controlAPI{
		client: NewClient("http", tunnel.Endpoint()),
	}, nil
}

func (s *Scaffold) EnsureNumService(controlAPIClient ControlAPIClient, matcher func(result int) bool) error {
	times := 0
	return wait.PollUntilContextTimeout(context.Background(), 100*time.Millisecond, 10*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		times++
		results, _, err := controlAPIClient.ListServices()
		if err != nil {
			log.Errorw("failed to ListServices", zap.Error(err))
			return false, nil
		}
		if !matcher(len(results)) {
			log.Debugw("number of effective services", zap.Int("number", len(results)), zap.Int("times", times))
			return false, nil
		}
		return true, nil
	})
}

func (s *Scaffold) ExpectUpstream(controlAPIClient ControlAPIClient, name string, matcher func(upstream adctypes.Upstream) bool) error {
	times := 0
	return wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 10*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		times++
		upstreams, _, err := controlAPIClient.ListUpstreams()
		if err != nil {
			log.Errorw("failed to ListServices", zap.Error(err))
			return false, nil
		}
		for _, upstream := range upstreams {
			upsValue := upstream.(map[string]any)
			data, err := json.Marshal(upsValue["value"])
			if err != nil {
				return false, fmt.Errorf("failed to marshal upstream: %v", err)
			}

			var ups adctypes.Upstream
			if err := json.Unmarshal(data, &ups); err != nil {
				return false, fmt.Errorf("failed to unmarshal upstream: %v", err)
			}
			if name != "" && ups.Name != name {
				continue
			}
			if ok := matcher(ups); !ok {
				return false, nil
			}
		}
		return true, nil
	})
}

func (s *Scaffold) EnsureNumUpstreamNodes(controlAPIClient ControlAPIClient, name string, number int) error {
	return s.ExpectUpstream(controlAPIClient, name, func(upstream adctypes.Upstream) bool {
		if len(upstream.Nodes) != number {
			log.Warnf("expect upstream: [%s] nodes num to be %d, but got %d", upstream.Name, number, len(upstream.Nodes))
			return false
		}
		return true
	})
}

type ControlAPIClient interface {
	ListServices() ([]any, int64, error)
	ListUpstreams() ([]any, int64, error)
}

type controlAPI struct {
	client *httpexpect.Expect
}

func (c *controlAPI) ListUpstreams() (result []any, total int64, err error) {
	resp := c.client.Request(http.MethodGet, "/v1/upstreams").Expect()
	if resp.Raw().StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status code: %v, message: %s", resp.Raw().StatusCode, resp.Body().Raw())
	}
	if err = json.Unmarshal([]byte(resp.Body().Raw()), &result); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return result, int64(len(result)), err
}

func (c *controlAPI) ListServices() (result []any, total int64, err error) {
	resp := c.client.Request(http.MethodGet, "/v1/services").Expect()
	if resp.Raw().StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status code: %v, message: %s", resp.Raw().StatusCode, resp.Body().Raw())
	}
	if err = json.Unmarshal([]byte(resp.Body().Raw()), &result); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return result, int64(len(result)), err
}
