// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
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
	"cmp"
	"context"
	"fmt"
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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
)

// CreateResourceFromString creates resource from a loaded yaml string.
func (s *Scaffold) CreateResourceFromString(yaml string) error {
	return k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, yaml)
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

	Eventually(func() string {
		hryaml, err := s.GetResourceYaml(resourType, resourceName)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("getting %s yaml", resourType))
		return hryaml
	}).WithTimeout(8*time.Second).ProbeEvery(2*time.Second).
		Should(
			SatisfyAll(
				ContainSubstring(`status: "True"`),
				ContainSubstring(fmt.Sprintf("observedGeneration: %d", observedGeneration)),
			),
			fmt.Sprintf("checking %s condition status", resourType),
		)
	time.Sleep(3 * time.Second)
}

func (s *Scaffold) ApplyDefaultGatewayResource(
	defaultGatewayProxy string,
	defaultGatewayClass string,
	defaultGateway string,
	defaultHTTPRoute string,
) {
	By("create GatewayProxy")
	gatewayProxy := fmt.Sprintf(defaultGatewayProxy, s.Deployer.GetAdminEndpoint(), s.AdminKey())
	err := s.CreateResourceFromString(gatewayProxy)
	Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
	time.Sleep(5 * time.Second)

	By("create GatewayClass")
	gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
	gatewayString := fmt.Sprintf(defaultGatewayClass, gatewayClassName, s.GetControllerName())
	err = s.CreateResourceFromStringWithNamespace(gatewayString, "")
	Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
	time.Sleep(5 * time.Second)

	By("check GatewayClass condition")
	gcyaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
	Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
	Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
	Expect(gcyaml).To(
		ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"),
		"checking GatewayClass condition message",
	)

	By("create Gateway")
	err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGateway, gatewayClassName), s.Namespace())
	Expect(err).NotTo(HaveOccurred(), "creating Gateway")
	time.Sleep(5 * time.Second)

	By("check Gateway condition")
	gwyaml, err := s.GetResourceYaml("Gateway", "apisix")
	Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
	Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
	Expect(gwyaml).To(
		ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"),
		"checking Gateway condition message",
	)

	s.ResourceApplied("httproute", "httpbin", defaultHTTPRoute, 1)
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
		err := wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, f)
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
