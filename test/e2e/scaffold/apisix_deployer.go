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
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/apache/apisix-ingress-controller/internal/provider/adc"
	"github.com/apache/apisix-ingress-controller/pkg/utils"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

type APISIXDeployOptions struct {
	Namespace string
	AdminKey  string

	ServiceName      string
	ServiceType      string
	ServiceHTTPPort  int
	ServiceHTTPSPort int

	ConfigProvider string
	Replicas       *int
}

type APISIXDeployer struct {
	*Scaffold
	adminTunnel *k8s.Tunnel
}

func NewAPISIXDeployer(s *Scaffold) *APISIXDeployer {
	return &APISIXDeployer{
		Scaffold: s,
	}
}

func (s *APISIXDeployer) BeforeEach() {
	s.namespace = fmt.Sprintf("ingress-apisix-e2e-tests-%s-%d", s.opts.Name, time.Now().Nanosecond())
	s.kubectlOptions = &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  s.namespace,
	}
	if s.opts.ControllerName == "" {
		s.opts.ControllerName = fmt.Sprintf("%s/%d", DefaultControllerName, time.Now().Nanosecond())
	}
	s.finalizers = nil
	if s.label == nil {
		s.label = make(map[string]string)
	}
	if s.opts.NamespaceSelectorLabel != nil {
		for k, v := range s.opts.NamespaceSelectorLabel {
			if len(v) > 0 {
				s.label[k] = v[0]
			}
		}
	} else {
		s.label["apisix.ingress.watch"] = s.namespace
	}

	// Initialize additionalGateways map
	s.additionalGateways = make(map[string]*GatewayResources)

	var nsLabel map[string]string
	if !s.opts.DisableNamespaceLabel {
		nsLabel = s.label
	}
	k8s.CreateNamespaceWithMetadata(s.t, s.kubectlOptions, metav1.ObjectMeta{Name: s.namespace, Labels: nsLabel})

	if s.opts.APISIXAdminAPIKey == "" {
		s.opts.APISIXAdminAPIKey = getEnvOrDefault("APISIX_ADMIN_KEY", "edd1c9f034335f136f87ad84b625c8f1")
	}

	s.Logf("apisix admin api key: %s", s.opts.APISIXAdminAPIKey)

	e := utils.ParallelExecutor{}

	e.Add(func() {
		s.DeployDataplane(DeployDataplaneOptions{})
		s.DeployIngress()
		adminTunnel, err := s.createAdminTunnel(s.dataplaneService)
		Expect(err).NotTo(HaveOccurred())
		s.adminTunnel = adminTunnel
	})
	e.Add(s.DeployTestService)
	e.Wait()
}

func (s *APISIXDeployer) AfterEach() {
	if CurrentSpecReport().Failed() {
		if os.Getenv("TEST_ENV") == "CI" {
			_, _ = fmt.Fprintln(GinkgoWriter, "Dumping namespace contents")
			_, _ = k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, "get", "deploy,sts,svc,pods,gatewayproxy")
			_, _ = k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, "describe", "pods")
		}

		output := s.GetDeploymentLogs("apisix-ingress-controller")
		if output != "" {
			_, _ = fmt.Fprintln(GinkgoWriter, output)
		}
	}

	// Delete all additional gateways
	for identifier := range s.additionalGateways {
		err := s.CleanupAdditionalGateway(identifier)
		Expect(err).NotTo(HaveOccurred(), "cleaning up additional gateway")
	}

	// if the test case is successful, just delete namespace
	err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
	Expect(err).NotTo(HaveOccurred(), "deleting namespace "+s.namespace)

	for i := len(s.finalizers) - 1; i >= 0; i-- {
		runWithRecover(s.finalizers[i])
	}

	// Wait for a while to prevent the worker node being overwhelming
	// (new cases will be run).
	time.Sleep(3 * time.Second)
}

func (s *APISIXDeployer) DeployDataplane(deployOpts DeployDataplaneOptions) {
	opts := APISIXDeployOptions{
		Namespace:        s.namespace,
		AdminKey:         s.opts.APISIXAdminAPIKey,
		ServiceHTTPPort:  9080,
		ServiceHTTPSPort: 9443,
		Replicas:         ptr.To(1),
	}

	if deployOpts.Namespace != "" {
		opts.Namespace = deployOpts.Namespace
	}
	if deployOpts.ServiceType != "" {
		opts.ServiceType = deployOpts.ServiceType
	}
	if deployOpts.ServiceHTTPPort != 0 {
		opts.ServiceHTTPPort = deployOpts.ServiceHTTPPort
	}
	if deployOpts.ServiceHTTPSPort != 0 {
		opts.ServiceHTTPSPort = deployOpts.ServiceHTTPSPort
	}
	if deployOpts.Replicas != nil {
		opts.Replicas = deployOpts.Replicas
	}

	for _, close := range []func(){
		s.closeAdminTunnel,
		s.closeApisixHttpTunnel,
		s.closeApisixHttpsTunnel,
	} {
		close()
	}

	svc := s.deployDataplane(&opts)
	s.dataplaneService = svc

	if !deployOpts.SkipCreateTunnels {
		err := s.newAPISIXTunnels(opts.ServiceName)
		Expect(err).ToNot(HaveOccurred(), "creating apisix tunnels")
	}
}

func (s *APISIXDeployer) newAPISIXTunnels(serviceName string) error {
	httpTunnel, httpsTunnel, err := s.createDataplaneTunnels(s.dataplaneService, s.kubectlOptions, serviceName)
	if err != nil {
		return err
	}

	s.apisixHttpTunnel = httpTunnel
	s.apisixHttpsTunnel = httpsTunnel
	return nil
}

func (s *APISIXDeployer) deployDataplane(opts *APISIXDeployOptions) *corev1.Service {
	if opts.ServiceName == "" {
		opts.ServiceName = framework.ProviderType
	}

	if opts.ServiceHTTPPort == 0 {
		opts.ServiceHTTPPort = 80
	}

	if opts.ServiceHTTPSPort == 0 {
		opts.ServiceHTTPSPort = 443
	}
	opts.ConfigProvider = "yaml"

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)

	if framework.ProviderType == adc.BackendModeAPISIX {
		opts.ConfigProvider = "etcd"
		// deploy etcd
		k8s.KubectlApplyFromString(s.GinkgoT, kubectlOpts, framework.EtcdSpec)
		err := framework.WaitPodsAvailable(s.GinkgoT, kubectlOpts, metav1.ListOptions{
			LabelSelector: "app=etcd",
		})
		Expect(err).ToNot(HaveOccurred(), "waiting for etcd pod ready")
	}

	buf := bytes.NewBuffer(nil)
	err := framework.APISIXStandaloneTpl.Execute(buf, opts)
	Expect(err).ToNot(HaveOccurred(), "executing template")

	k8s.KubectlApplyFromString(s.GinkgoT, kubectlOpts, buf.String())

	err = framework.WaitPodsAvailable(s.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=apisix",
	})
	Expect(err).ToNot(HaveOccurred(), "waiting for gateway pod ready")

	Eventually(func() bool {
		svc, err := k8s.GetServiceE(s.GinkgoT, kubectlOpts, opts.ServiceName)
		if err != nil {
			s.Logf("failed to get service %s: %v", opts.ServiceName, err)
			return false
		}
		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
			return len(svc.Status.LoadBalancer.Ingress) > 0
		}
		return true
	}, "20s", "4s").Should(BeTrue(), "waiting for LoadBalancer IP")

	svc, err := k8s.GetServiceE(s.GinkgoT, kubectlOpts, opts.ServiceName)
	Expect(err).ToNot(HaveOccurred(), "failed to get service %s: %v", opts.ServiceName, err)
	return svc
}

func (s *APISIXDeployer) DeployIngress() {
	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ControllerName:     s.opts.ControllerName,
		ProviderType:       framework.ProviderType,
		ProviderSyncPeriod: 200 * time.Millisecond,
		Namespace:          s.namespace,
		Replicas:           1,
	})
}

func (s *APISIXDeployer) ScaleIngress(replicas int) {
	s.Framework.DeployIngress(framework.IngressDeployOpts{
		ControllerName:     s.opts.ControllerName,
		ProviderType:       framework.ProviderType,
		ProviderSyncPeriod: 200 * time.Millisecond,
		Namespace:          s.namespace,
		Replicas:           replicas,
	})
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (s *APISIXDeployer) createAdminTunnel(svc *corev1.Service) (*k8s.Tunnel, error) {
	var (
		adminNodePort int
		adminPort     int
	)

	for _, port := range svc.Spec.Ports {
		switch port.Name {
		case "admin":
			adminNodePort = int(port.NodePort)
			adminPort = int(port.Port)
		}
	}

	kubectlOpts := k8s.NewKubectlOptions("", "", svc.Namespace)

	adminTunnel := k8s.NewTunnel(kubectlOpts, k8s.ResourceTypeService, svc.Name,
		adminNodePort, adminPort)

	if err := adminTunnel.ForwardPortE(s.t); err != nil {
		return nil, err
	}
	s.addFinalizers(s.closeAdminTunnel)

	return adminTunnel, nil
}

func (s *APISIXDeployer) closeAdminTunnel() {
	if s.adminTunnel != nil {
		s.adminTunnel.Close()
		s.adminTunnel = nil
	}
}

func (s *APISIXDeployer) CreateAdditionalGateway(namePrefix string) (string, *corev1.Service, error) {
	// Create a new namespace for this additional gateway
	additionalNS := fmt.Sprintf("%s-%d", namePrefix, time.Now().Unix())

	// Create namespace with the same labels
	var nsLabel map[string]string
	if !s.opts.DisableNamespaceLabel {
		nsLabel = s.label
	}
	k8s.CreateNamespaceWithMetadata(s.t, s.kubectlOptions, metav1.ObjectMeta{Name: additionalNS, Labels: nsLabel})

	// Create new kubectl options for the new namespace
	kubectlOpts := &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  additionalNS,
	}

	s.Logf("additional gateway in namespace %s", additionalNS)

	// Use the same admin key as the main gateway
	adminKey := s.opts.APISIXAdminAPIKey
	s.Logf("additional gateway admin api key: %s", adminKey)

	// Store gateway resources info
	resources := &GatewayResources{
		Namespace:   additionalNS,
		AdminAPIKey: adminKey,
	}

	// Deploy dataplane for this additional gateway
	opts := APISIXDeployOptions{
		Namespace:        additionalNS,
		AdminKey:         adminKey,
		ServiceHTTPPort:  9080,
		ServiceHTTPSPort: 9443,
	}
	svc := s.deployDataplane(&opts)

	resources.DataplaneService = svc

	// Create tunnels for the dataplane
	httpTunnel, httpsTunnel, err := s.createDataplaneTunnels(svc, kubectlOpts, svc.Name)
	if err != nil {
		return "", nil, err
	}

	resources.HttpTunnel = httpTunnel
	resources.HttpsTunnel = httpsTunnel

	// Use namespace as identifier for APISIX deployments
	identifier := additionalNS

	// Store in the map
	s.additionalGateways[identifier] = resources

	return identifier, svc, nil
}

func (s *APISIXDeployer) CleanupAdditionalGateway(identifier string) error {
	resources, exists := s.additionalGateways[identifier]
	if !exists {
		return fmt.Errorf("gateway %s not found", identifier)
	}

	// Close tunnels if they exist
	if resources.HttpTunnel != nil {
		resources.HttpTunnel.Close()
	}
	if resources.HttpsTunnel != nil {
		resources.HttpsTunnel.Close()
	}

	// Delete the namespace
	err := k8s.DeleteNamespaceE(s.t, &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  resources.Namespace,
	}, resources.Namespace)

	// Remove from the map
	delete(s.additionalGateways, identifier)

	return err
}

func (s *APISIXDeployer) GetAdminEndpoint(svc ...*corev1.Service) string {
	if len(svc) == 0 {
		return fmt.Sprintf("http://%s.%s:9180", s.dataplaneService.Name, s.dataplaneService.Namespace)
	}
	return fmt.Sprintf("http://%s.%s:9180", svc[0].Name, svc[0].Namespace)
}

func (s *APISIXDeployer) DefaultDataplaneResource() DataplaneResource {
	return newADCDataplaneResource(
		framework.ProviderType,
		fmt.Sprintf("http://%s", s.adminTunnel.Endpoint()),
		s.AdminKey(),
		false, // tlsVerify
	)
}
