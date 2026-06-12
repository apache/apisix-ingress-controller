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
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

var nsCounter int64

// isPoolable reports whether an environment with these options can be served
// from the prewarm pool. Only the default profile is pooled; webhook-enabled
// and custom-keyed environments fall back to synchronous deployment.
func isPoolable(o Options) bool {
	return !o.SkipHooks &&
		!o.EnableWebhook &&
		o.ControllerName == "" &&
		o.APISIXAdminAPIKey == ""
}

// profileKey identifies the pool an environment belongs to. Within a process
// all default scaffolds share one pool.
func profileKey(o Options) string {
	name := o.Name
	if name == "" {
		name = "default"
	}
	return "name=" + name
}

func formatRegistry(workloadTemplate string) string {
	if customRegistry, ok := os.LookupEnv("REGISTRY"); ok {
		return strings.ReplaceAll(workloadTemplate, "127.0.0.1:5000", customRegistry)
	}
	return workloadTemplate
}

// provisionAPISIXEnv builds a complete default-profile environment using
// error-returning primitives only, so it is safe to run in a background
// goroutine. Any failure is captured in pooledEnv.err for the caller to handle.
func provisionAPISIXEnv(fw *framework.Framework, opts Options) *pooledEnv {
	t := &bgTestingT{}
	env := &pooledEnv{}

	name := opts.Name
	if name == "" {
		name = "default"
	}
	ns := fmt.Sprintf("ingress-apisix-e2e-tests-%s-p%d-%d",
		name, GinkgoParallelProcess(), atomic.AddInt64(&nsCounter, 1))
	env.namespace = ns
	env.kubectlOptions = &k8s.KubectlOptions{
		ConfigPath: GetKubeconfig(),
		Namespace:  ns,
	}
	env.adminKey = getEnvOrDefault("APISIX_ADMIN_KEY", "edd1c9f034335f136f87ad84b625c8f1")
	env.controllerName = fmt.Sprintf("%s/%s", DefaultControllerName, ns)

	if err := k8s.CreateNamespaceE(t, env.kubectlOptions, ns); err != nil {
		env.err = fmt.Errorf("creating namespace: %w", err)
		return env
	}

	// 1) Data plane (APISIX, plus etcd when the provider needs it).
	svc, err := provisionDataplane(t, env, opts)
	if err != nil {
		env.err = err
		return env
	}
	env.dataplaneService = svc

	// 2) Tunnels: 4 dataplane port-forwards + 1 admin port-forward.
	if err := provisionTunnels(t, env); err != nil {
		env.err = err
		return env
	}

	// 3) Ingress controller.
	if err := provisionIngress(t, env); err != nil {
		env.err = err
		return env
	}

	// 4) httpbin test backend.
	httpbinSvc, err := provisionHTTPBIN(fw, t, env)
	if err != nil {
		env.err = err
		return env
	}
	env.httpbinService = httpbinSvc

	return env
}

func provisionDataplane(t *bgTestingT, env *pooledEnv, _ Options) (*corev1.Service, error) {
	serviceName := framework.ProviderType
	configProvider := framework.ConfigProviderTypeYaml
	if framework.ProviderType == framework.ProviderTypeAPISIX {
		configProvider = framework.ConfigProviderTypeEtcd
	}

	if configProvider == framework.ConfigProviderTypeEtcd {
		if err := k8s.KubectlApplyFromStringE(t, env.kubectlOptions, framework.EtcdSpec); err != nil {
			return nil, fmt.Errorf("applying etcd: %w", err)
		}
		if err := framework.WaitPodsAvailable(t, env.kubectlOptions, metav1.ListOptions{
			LabelSelector: "app=etcd",
		}); err != nil {
			return nil, fmt.Errorf("waiting for etcd pod: %w", err)
		}
	}

	deployOpts := APISIXDeployOptions{
		Namespace:        env.namespace,
		AdminKey:         env.adminKey,
		ServiceName:      serviceName,
		ServiceHTTPPort:  9080,
		ServiceHTTPSPort: 9443,
		ConfigProvider:   configProvider,
		Replicas:         ptr.To(1),
	}
	buf := bytes.NewBuffer(nil)
	if err := framework.APISIXStandaloneTpl.Execute(buf, &deployOpts); err != nil {
		return nil, fmt.Errorf("rendering apisix template: %w", err)
	}
	if err := k8s.KubectlApplyFromStringE(t, env.kubectlOptions, buf.String()); err != nil {
		return nil, fmt.Errorf("applying apisix: %w", err)
	}
	if err := framework.WaitPodsAvailable(t, env.kubectlOptions, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=apisix",
	}); err != nil {
		return nil, fmt.Errorf("waiting for apisix pod: %w", err)
	}

	svc, err := k8s.GetServiceE(t, env.kubectlOptions, serviceName)
	if err != nil {
		return nil, fmt.Errorf("getting dataplane service: %w", err)
	}
	return svc, nil
}

func provisionTunnels(t *bgTestingT, env *pooledEnv) error {
	svc := env.dataplaneService
	var httpPort, httpsPort, tcpPort, tlsPort, adminPort int
	for _, port := range svc.Spec.Ports {
		switch port.Name {
		case "http":
			httpPort = int(port.Port)
		case "https":
			httpsPort = int(port.Port)
		case "tcp":
			tcpPort = int(port.Port)
		case "tls":
			tlsPort = int(port.Port)
		case "admin":
			adminPort = int(port.Port)
		}
	}

	tunnels := &Tunnels{}
	env.finalizers = append(env.finalizers, tunnels.Close)

	for _, spec := range []struct {
		dst  **k8s.Tunnel
		port int
	}{
		{&tunnels.HTTP, httpPort},
		{&tunnels.HTTPS, httpsPort},
		{&tunnels.TCP, tcpPort},
		{&tunnels.TLS, tlsPort},
	} {
		tunnel := k8s.NewTunnel(env.kubectlOptions, k8s.ResourceTypeService, svc.Name, 0, spec.port)
		if err := tunnel.ForwardPortE(t); err != nil {
			return fmt.Errorf("forwarding dataplane port %d: %w", spec.port, err)
		}
		*spec.dst = tunnel
	}
	env.apisixTunnels = tunnels

	adminTunnel := k8s.NewTunnel(env.kubectlOptions, k8s.ResourceTypeService, svc.Name, 0, adminPort)
	if err := adminTunnel.ForwardPortE(t); err != nil {
		return fmt.Errorf("forwarding admin port %d: %w", adminPort, err)
	}
	env.adminTunnel = adminTunnel
	env.finalizers = append(env.finalizers, adminTunnel.Close)

	return nil
}

func provisionIngress(t *bgTestingT, env *pooledEnv) error {
	opts := framework.IngressDeployOpts{
		ControllerName:     env.controllerName,
		ProviderType:       framework.ProviderType,
		ProviderSyncPeriod: 1 * time.Hour,
		Namespace:          env.namespace,
		Replicas:           ptr.To(1),
		WebhookEnable:      false,
		DisableGatewayAPI:  framework.DisableGatewayAPI,
	}
	buf := bytes.NewBuffer(nil)
	if err := framework.IngressSpecTpl.Execute(buf, opts); err != nil {
		return fmt.Errorf("rendering ingress template: %w", err)
	}
	if err := k8s.KubectlApplyFromStringE(t, env.kubectlOptions, buf.String()); err != nil {
		return fmt.Errorf("applying ingress controller: %w", err)
	}
	if err := framework.WaitPodsAvailable(t, env.kubectlOptions, metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	}); err != nil {
		return fmt.Errorf("waiting for controller pod: %w", err)
	}
	return nil
}

func provisionHTTPBIN(fw *framework.Framework, t *bgTestingT, env *pooledEnv) (*corev1.Service, error) {
	deployment := fmt.Sprintf(formatRegistry(_httpbinDeploymentTemplate), 1)
	if err := k8s.KubectlApplyFromStringE(t, env.kubectlOptions, deployment); err != nil {
		return nil, fmt.Errorf("applying httpbin deployment: %w", err)
	}
	if err := k8s.KubectlApplyFromStringE(t, env.kubectlOptions, _httpService); err != nil {
		return nil, fmt.Errorf("applying httpbin service: %w", err)
	}
	if err := fw.EnsureServiceReadyE(env.namespace, HTTPBinServiceName, 1); err != nil {
		return nil, fmt.Errorf("waiting for httpbin endpoints: %w", err)
	}
	svc, err := k8s.GetServiceE(t, env.kubectlOptions, HTTPBinServiceName)
	if err != nil {
		return nil, fmt.Errorf("getting httpbin service: %w", err)
	}
	return svc, nil
}

// loadPooledEnv installs a prewarmed environment onto the deployer's scaffold so
// the rest of the spec behaves exactly as if it had been deployed synchronously.
func (s *APISIXDeployer) loadPooledEnv(env *pooledEnv) {
	s.runtimeOpts = s.opts
	s.namespace = env.namespace
	s.kubectlOptions = env.kubectlOptions
	s.runtimeOpts.ControllerName = env.controllerName
	s.runtimeOpts.APISIXAdminAPIKey = env.adminKey
	s.dataplaneService = env.dataplaneService
	s.httpbinService = env.httpbinService
	s.apisixTunnels = env.apisixTunnels
	s.adminTunnel = env.adminTunnel
	s.finalizers = env.finalizers
	s.additionalGateways = make(map[string]*GatewayResources)
}
