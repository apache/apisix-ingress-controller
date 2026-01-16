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
	"cmp"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

// CreateResourceFromString creates resource from a loaded yaml string.
func (s *Scaffold) CreateResourceFromString(yaml string) error {
	return k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, yaml)
}

// CreateResourceFromStringAndGetOutput creates resource from a loaded yaml string and returns the output of the command.
func (s *Scaffold) CreateResourceFromStringAndGetOutput(yaml string) (string, error) {
	tmpfile, err := k8s.StoreConfigToTempFileE(s.t, yaml)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.Remove(tmpfile)
	}()
	return k8s.RunKubectlAndGetOutputE(s.t, s.kubectlOptions, "apply", "-f", tmpfile)
}

func (s *Scaffold) DeleteResourceFromString(yaml string) error {
	return k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, yaml)
}

func (s *Scaffold) Exec(podName, containerName string, args ...string) (string, error) {
	cmdArgs := []string{}

	if s.kubectlOptions.ContextName != "" {
		cmdArgs = append(cmdArgs, "--context", s.kubectlOptions.ContextName)
	}
	if s.kubectlOptions.ConfigPath != "" {
		cmdArgs = append(cmdArgs, "--kubeconfig", s.kubectlOptions.ConfigPath)
	}
	if s.kubectlOptions.Namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace", s.kubectlOptions.Namespace)
	}

	cmdArgs = append(cmdArgs, "exec")
	cmdArgs = append(cmdArgs, "-i")
	cmdArgs = append(cmdArgs, podName)
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, containerName)
	cmdArgs = append(cmdArgs, "--", "sh", "-c")
	cmdArgs = append(cmdArgs, args...)

	GinkgoWriter.Printf("running command: kubectl %v\n", strings.Join(cmdArgs, " "))

	output, err := exec.Command("kubectl", cmdArgs...).Output()

	return strings.TrimSuffix(string(output), "\n"), err
}

func (s *Scaffold) GetOutputFromString(shell ...string) (string, error) {
	cmdArgs := []string{}
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, shell...)
	output, err := k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, cmdArgs...)
	return output, err
}

func (s *Scaffold) GetResourceYamlFromNamespace(resourceType, resourceName, namespace string) (string, error) {
	return s.GetOutputFromString(resourceType, resourceName, "-n", namespace, "-o", "yaml")
}

func (s *Scaffold) GetResourceYaml(resourceType, resourceName string) (string, error) {
	return s.GetOutputFromString(resourceType, resourceName, "-o", "yaml")
}

// RemoveResourceByString remove resource from a loaded yaml string.
func (s *Scaffold) RemoveResourceByString(yaml string) error {
	err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, yaml)
	time.Sleep(5 * time.Second)
	return err
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
		_ = s.DeleteResourceFromStringWithNamespace(yaml, namespace)
	})
	return s.CreateResourceFromString(yaml)
}

func (s *Scaffold) DeleteResourceFromStringWithNamespace(yaml, namespace string) error {
	originalNamespace := s.kubectlOptions.Namespace
	s.kubectlOptions.Namespace = namespace
	defer func() {
		s.kubectlOptions.Namespace = originalNamespace
	}()
	return k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, yaml)
}

// Namespace returns the current working namespace.
func (s *Scaffold) Namespace() string {
	return s.kubectlOptions.Namespace
}

func (s *Scaffold) EnsureNumEndpointsReady(t testing.TestingT, endpointsName string, desired int) {
	e, err := k8s.GetKubernetesClientFromOptionsE(t, s.kubectlOptions)
	Expect(err).ToNot(HaveOccurred(), "Getting Kubernetes clientset")

	statusMsg := fmt.Sprintf("Wait for endpoints %s to be ready.", endpointsName)
	message := retry.DoWithRetry(
		t,
		statusMsg,
		20,
		2*time.Second,
		func() (string, error) {
			endpoints, err := e.CoreV1().Endpoints(s.Namespace()).Get(context.Background(), endpointsName, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			readyNum := 0
			for _, subset := range endpoints.Subsets {
				readyNum += len(subset.Addresses)
			}
			if readyNum == desired {
				return "Service is now available", nil
			}
			return "failed", fmt.Errorf("endpoints not ready yet, expect %v, actual %v", desired, readyNum)
		},
	)
	GinkgoT().Log(message)
}

// GetKubernetesClient get kubernetes client use by scaffold
func (s *Scaffold) GetKubernetesClient() *kubernetes.Clientset {
	client, err := k8s.GetKubernetesClientFromOptionsE(s.t, s.kubectlOptions)
	Expect(err).ToNot(HaveOccurred(), "Getting Kubernetes clientset")
	return client
}

func (s *Scaffold) RunKubectlAndGetOutput(args ...string) (string, error) {
	return k8s.RunKubectlAndGetOutputE(GinkgoT(), s.kubectlOptions, args...)
}

func (s *Scaffold) ResourceApplied(resourType, resourceName, resourceRaw string, observedGeneration int) {
	Expect(s.CreateResourceFromString(resourceRaw)).
		NotTo(HaveOccurred(), fmt.Sprintf("creating %s", resourType))

	s.RetryAssertion(func() string {
		hryaml, err := s.GetResourceYaml(resourType, resourceName)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("getting %s yaml", resourType))
		return hryaml
	}).Should(
		SatisfyAll(
			ContainSubstring(`status: "True"`),
			ContainSubstring(fmt.Sprintf("observedGeneration: %d", observedGeneration)),
		),
		fmt.Sprintf("checking %s condition status", resourType),
	)
}

func (s *Scaffold) ApplyHTTPRoute(hrNN types.NamespacedName, spec string, until ...wait.ConditionWithContextFunc) {
	err := s.CreateResourceFromString(spec)
	Expect(err).NotTo(HaveOccurred(), "creating HTTPRoute %s", hrNN)
	framework.HTTPRouteMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second,
		types.NamespacedName{},
		types.NamespacedName{Namespace: cmp.Or(hrNN.Namespace, s.Namespace()), Name: hrNN.Name},
		metav1.Condition{
			Type:   string(gatewayv1.RouteConditionAccepted),
			Status: metav1.ConditionTrue,
		},
	)
	for i, f := range until {
		err := wait.PollUntilContextTimeout(context.Background(), time.Second, 20*time.Second, true, f)
		require.NoError(s.GinkgoT, err, "wait for ConditionWithContextFunc[%d] OK", i)
	}
}

func (s *Scaffold) ApplyHTTPRoutePolicy(refNN, hrpNN types.NamespacedName, spec string, conditions ...metav1.Condition) {
	err := s.CreateResourceFromString(spec)
	Expect(err).NotTo(HaveOccurred(), "creating HTTPRoutePolicy %s", hrpNN)
	if len(conditions) == 0 {
		conditions = []metav1.Condition{
			{
				Type:   string(v1alpha2.PolicyConditionAccepted),
				Status: metav1.ConditionTrue,
			},
		}
	}
	for _, condition := range conditions {
		framework.HTTPRoutePolicyMustHaveCondition(s.GinkgoT, s.K8sClient, 8*time.Second, refNN, hrpNN, condition)
	}
}

func (s *Scaffold) WaitUntilDeploymentAvailable(name string) {
	k8s.WaitUntilDeploymentAvailable(s.GinkgoT, s.kubectlOptions, name, 10, 10*time.Second)
}

func (s *Scaffold) RunDigDNSClientFromK8s(args ...string) (string, error) {
	kubectlArgs := []string{
		"run",
		"dig",
		"-i",
		"--rm",
		"--restart=Never",
		"--image-pull-policy=IfNotPresent",
		"--image=toolbelt/dig",
		"--",
	}
	kubectlArgs = append(kubectlArgs, args...)
	return s.RunKubectlAndGetOutput(kubectlArgs...)
}

// RunCurlFromK8s runs a curl command from a temporary pod inside the cluster.
// This is useful for making HTTP requests from within the cluster, avoiding
// port-forward limitations where server_port variables may not work correctly.
func (s *Scaffold) RunCurlFromK8s(args ...string) (string, error) {
	kubectlArgs := []string{
		"run",
		"curl-test",
		"-i",
		"--rm",
		"--restart=Never",
		"--image-pull-policy=IfNotPresent",
		"--image=alpine/curl:flattened",
		"--",
		"curl",
	}
	kubectlArgs = append(kubectlArgs, args...)
	return s.RunKubectlAndGetOutput(kubectlArgs...)
}

func (s *Scaffold) GetGatewayProxySpec() string {
	var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      service:
        name: %s
        port: 9180
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`
	return fmt.Sprintf(gatewayProxyYaml, framework.ProviderType, s.AdminKey())
}

const ingressClassYaml = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: %s
spec:
  controller: %s
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: %s
    scope: Namespace
`

func (s *Scaffold) GetIngressClassYaml() string {
	return fmt.Sprintf(ingressClassYaml, s.Namespace(), s.GetControllerName(), s.Namespace())
}

const gatewayClassYaml = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

func (s *Scaffold) GetGatewayClassYaml() string {
	return fmt.Sprintf(gatewayClassYaml, s.Namespace(), s.GetControllerName())
}

const gatewayYaml = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

func (s *Scaffold) GetGatewayYaml() string {
	return fmt.Sprintf(gatewayYaml, s.Namespace(), s.Namespace())
}

type WebhookData struct {
	Namespace string
	CABundle  string
}

func (s *Scaffold) SetupWebhookResources() error {
	// Generate TLS certificates
	caCert, serverCert, serverKey, _, _ := s.GenerateMACert(s.GinkgoT, []string{fmt.Sprintf("webhook-service.%s.svc", s.Namespace())})

	err := s.NewKubeTlsSecret("webhook-server-certs", serverCert.String(), serverKey.String())
	if err != nil {
		return err
	}

	data := WebhookData{
		Namespace: s.Namespace(),
		CABundle:  base64.StdEncoding.EncodeToString(caCert.Bytes()),
	}

	var buf bytes.Buffer
	err = framework.ValidatingWebhookTpl.Execute(&buf, data)
	if err != nil {
		return err
	}

	return s.CreateResourceFromStringWithNamespace(buf.String(), "")
}

func (s *Scaffold) GetKubeClient() client.Client {
	if s.client == nil {
		scheme := runtime.NewScheme()
		_ = apiv2.AddToScheme(scheme)
		_ = corev1.AddToScheme(scheme)
		_ = gatewayv1.Install(scheme)
		_ = v1alpha2.Install(scheme)
		cfg, err := clientcmd.BuildConfigFromFlags("", s.opts.Kubeconfig)
		Expect(err).NotTo(HaveOccurred(), "building kubeconfig")
		s.client, err = client.New(cfg, client.Options{Scheme: scheme})
		Expect(err).NotTo(HaveOccurred(), "building controller-runtime client")
	}
	return s.client
}
