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

package framework

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	. "github.com/onsi/gomega" //nolint:staticcheck
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/utils/ptr"

	"github.com/apache/apisix-ingress-controller/pkg/utils"
)

// buildRestConfig builds the rest.Config object from kubeconfig filepath and
// context, if kubeconfig is missing, building from in-cluster configuration.
func buildRestConfig(context string) (*rest.Config, error) {

	// Config loading rules:
	// 1. kubeconfig if it not empty string
	// 2. Config(s) in KUBECONFIG environment variable
	// 3. In cluster config if running in-cluster
	// 4. Use $HOME/.kube/config
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  context,
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return clientConfig.ClientConfig()
}

func (f *Framework) ensureService(name, namespace string, desiredEndpoints int) error {
	return f.ensureServiceWithTimeout(name, namespace, desiredEndpoints, 120)
}

func (f *Framework) ensureServiceWithTimeout(name, namespace string, desiredEndpoints, timeout int) error {
	backoff := wait.Backoff{
		Duration: 6 * time.Second,
		Factor:   1,
		Steps:    timeout / 6,
	}
	var lastErr error
	condFunc := func() (bool, error) {
		ep, err := f.clientset.CoreV1().Endpoints(namespace).Get(f.Context, name, metav1.GetOptions{})
		if err != nil {
			lastErr = err
			log.Errorw("failed to list endpoints",
				zap.String("service", name),
				zap.Error(err),
			)
			return false, nil
		}
		count := 0
		for _, ss := range ep.Subsets {
			count += len(ss.Addresses)
		}
		if count == desiredEndpoints {
			return true, nil
		}
		log.Infow("endpoints count mismatch",
			zap.String("service", name),
			zap.Any("ep", ep),
			zap.Int("expected", desiredEndpoints),
			zap.Int("actual", count),
		)
		lastErr = fmt.Errorf("expected endpoints: %d but seen %d", desiredEndpoints, count)
		return false, nil
	}

	err := wait.ExponentialBackoff(backoff, condFunc)
	if err != nil {
		return lastErr
	}
	return nil
}

func (f *Framework) GetServiceEndpoints(nn types.NamespacedName) ([]string, error) {
	ep, err := f.clientset.CoreV1().Endpoints(cmp.Or(nn.Namespace, _namespace)).Get(f.Context, nn.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var endpoints []string
	for _, ss := range ep.Subsets {
		for _, addr := range ss.Addresses {
			endpoints = append(endpoints, addr.IP)
		}
	}
	return endpoints, nil
}

//nolint:unused
func (f *Framework) deletePods(selector string) {
	podList, err := f.clientset.CoreV1().Pods(_namespace).List(f.Context, metav1.ListOptions{
		LabelSelector: selector,
	})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "list pods")
	for _, pod := range podList.Items {
		_ = f.clientset.CoreV1().
			Pods(_namespace).
			Delete(f.Context, pod.Name, metav1.DeleteOptions{GracePeriodSeconds: ptr.To(int64(30))})
	}
}

func (f *Framework) CreateNamespaceWithTestService(name string) {
	_, err := f.clientset.CoreV1().
		Namespaces().
		Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "create namespace")
		return
	}

	_, err = f.clientset.CoreV1().Services(name).Create(f.Context, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": "httpbin",
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "create service")
	}
}

func (f *Framework) DeleteNamespace(name string) {
	err := f.clientset.CoreV1().Namespaces().Delete(f.Context, name, metav1.DeleteOptions{})
	if err == nil || errors.IsNotFound(err) {
		return
	}
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "delete namespace")
}

func (f *Framework) Scale(name string, replicas int32) {
	scale, err := f.clientset.AppsV1().Deployments(_namespace).GetScale(context.Background(), name, metav1.GetOptions{})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("get deployment %s scale failed", name))
	if scale.Spec.Replicas == replicas {
		return
	}
	scale.Spec.Replicas = replicas
	_, err = f.clientset.AppsV1().
		Deployments(_namespace).
		UpdateScale(context.Background(), name, scale, metav1.UpdateOptions{})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("scale deployment %s to %v failed", name, replicas))

	// FIXME: The service name and the deployment name may not be the same
	err = f.ensureService(name, _namespace, int(replicas))
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(),
		fmt.Sprintf("ensure service %s/%s has %v endpoints failed", _namespace, name, replicas))
}

func (f *Framework) GetPodIP(namespace, selector string) string {
	pods := f.GetPods(namespace, selector)
	f.GomegaT.Expect(pods).ShouldNot(BeEmpty())
	return pods[0].Status.PodIP
}

func (f *Framework) GetPods(namespace, selector string, filters ...func(pod corev1.Pod) bool) []corev1.Pod {
	podList, err := f.clientset.CoreV1().Pods(cmp.Or(namespace, _namespace)).List(f.Context, metav1.ListOptions{
		LabelSelector: selector,
	})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred())
	for _, filter := range filters {
		podList.Items = utils.Filter(podList.Items, filter)
	}
	return podList.Items
}

//nolint:unused
func (f *Framework) applySSLSecret(namespace, name string, cert, pkey, caCert []byte) {
	kind := "Secret"
	apiVersion := "v1"
	secretType := corev1.SecretTypeTLS
	secret := applycorev1.SecretApplyConfiguration{
		TypeMetaApplyConfiguration: applymetav1.TypeMetaApplyConfiguration{
			Kind:       &kind,
			APIVersion: &apiVersion,
		},
		ObjectMetaApplyConfiguration: &applymetav1.ObjectMetaApplyConfiguration{
			Name: &name,
		},
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": pkey,
			"ca.crt":  caCert,
		},
		Type: &secretType,
	}

	cli, err := k8s.GetKubernetesClientE(f.GinkgoT)
	Expect(err).ToNot(HaveOccurred())

	_, err = cli.CoreV1().Secrets(namespace).Apply(context.TODO(), &secret, metav1.ApplyOptions{
		FieldManager: "e2e",
	})
	Expect(err).ToNot(HaveOccurred(), "apply secret")
}

func WaitPodsAvailable(t testing.TestingT, kubeOps *k8s.KubectlOptions, opts metav1.ListOptions) error {
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(t, kubeOps, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			return false, nil
		}
		for _, item := range items {
			foundPodReady := false
			for _, cond := range item.Status.Conditions {
				if cond.Type != corev1.PodReady {
					continue
				}
				foundPodReady = true
				if cond.Status != "True" {
					return false, nil
				}
			}
			if !foundPodReady {
				return false, nil
			}
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}

func waitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Steps:    8,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}

func (f *Framework) NewExpectResponse(httpBody any) *httpexpect.Response {
	body, err := json.Marshal(httpBody)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred())

	return httpexpect.NewResponse(f.GinkgoT, &http.Response{
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewBuffer(body)),
	})
}

func (f *Framework) ListRunningPods(namespace, selector string) []corev1.Pod {
	return f.GetPods(namespace, selector, func(pod corev1.Pod) bool {
		return pod.Status.Phase == corev1.PodRunning && pod.DeletionTimestamp == nil
	})
}

// ExecCommandInPod exec cmd in specify pod and return the output from stdout and stderr
func (f *Framework) ExecCommandInPod(podName string, cmd ...string) (string, string) {
	req := f.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(_namespace).SubResource("exec")
	req.VersionedParams(
		&corev1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		},
		scheme.ParameterCodec,
	)

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(f.restConfig, "POST", req.URL())
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "request kubernetes exec api")
	_ = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String())
}

func (f *Framework) GetPodLogs(name string, previous bool) string {
	reader, err := f.clientset.CoreV1().
		Pods(_namespace).
		GetLogs(name, &corev1.PodLogOptions{Previous: previous}).
		Stream(context.Background())
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "get logs")
	defer func() {
		_ = reader.Close()
	}()

	logs, err := io.ReadAll(reader)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "read all logs")

	return string(logs)
}

func (f *Framework) WaitControllerManagerLog(namespace, keyword string, sinceSeconds int64, timeout time.Duration) {
	f.WaitPodsLog(namespace, "control-plane=controller-manager", keyword, sinceSeconds, timeout)
}

func (f *Framework) WaitPodsLog(namespace, selector, keyword string, sinceSeconds int64, timeout time.Duration) {
	pods := f.ListRunningPods(namespace, selector)
	f.GomegaT.Expect(pods).ToNot(BeEmpty())
	wg := sync.WaitGroup{}
	for _, p := range pods {
		wg.Add(1)
		go func(p corev1.Pod) {
			defer wg.Done()
			opts := corev1.PodLogOptions{Follow: true}
			if sinceSeconds > 0 {
				opts.SinceSeconds = ptr.To(sinceSeconds)
			} else {
				opts.TailLines = ptr.To(int64(0))
			}
			logStream, err := f.clientset.CoreV1().Pods(p.Namespace).GetLogs(p.Name, &opts).Stream(context.Background())
			f.GomegaT.Expect(err).Should(BeNil())
			scanner := bufio.NewScanner(logStream)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, keyword) {
					return
				}
			}
		}(p)
	}
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return
	case <-time.After(timeout):
		f.GinkgoT.Error("wait log timeout")
	}
}
