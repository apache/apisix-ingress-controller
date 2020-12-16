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
	"fmt"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/api7/ingress-controller/pkg/log"
)

const (
	_defaultPollInterval = 2 * time.Second
	_defaultTimeout      = time.Minute
)

type deploymentDesc struct {
	name            string
	namespace       string
	image           string
	ports           []int32
	replica         int32
	command         []string
	volumeMounts    []corev1.VolumeMount
	volumes         []corev1.Volume
	probe           *corev1.Probe
	envVar          []corev1.EnvVar
	imagePullPolicy string
	serviceAccount  string
}

type serviceDesc struct {
	name      string
	namespace string
	selector  map[string]string
	ports     []corev1.ServicePort
}

func newService(desc *serviceDesc) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desc.name,
			Namespace: desc.namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports:    desc.ports,
			Selector: desc.selector,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}

	return svc
}

func ensureService(clientset kubernetes.Interface, svc *corev1.Service) (*corev1.Service, error) {
	condFunc := func() (bool, error) {
		_, err := clientset.CoreV1().Services(svc.Namespace).Create(svc)
		if err == nil {
			return true, nil
		}
		return false, err
	}
	if err := waitExponentialBackoff(condFunc); err != nil {
		return nil, err
	}

	return clientset.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
}

func newDeployment(desc *deploymentDesc) *appsv1.Deployment {
	var (
		containerPorts                []corev1.ContainerPort
		terminationGracePeriodSeconds int64
	)
	replica := desc.replica

	for i, port := range desc.ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          "undefined-" + strconv.Itoa(i),
			ContainerPort: port,
			Protocol:      corev1.ProtocolTCP,
		})
	}

	imagePullPolicy := corev1.PullIfNotPresent
	if desc.imagePullPolicy != "" {
		imagePullPolicy = corev1.PullPolicy(desc.imagePullPolicy)
	}

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desc.name,
			Namespace: desc.namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replica,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": desc.name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": desc.name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            desc.serviceAccount,
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{
						{
							Name:            desc.name,
							Image:           desc.image,
							ImagePullPolicy: imagePullPolicy,
							Env:             desc.envVar,
							Ports:           containerPorts,
							ReadinessProbe:  desc.probe,
							LivenessProbe:   desc.probe,
							VolumeMounts:    desc.volumeMounts,
							Command:         desc.command,
						},
					},
					Volumes: desc.volumes,
				},
			},
		},
	}
	return d
}

func ensureDeployment(clientset kubernetes.Interface, d *appsv1.Deployment) (*appsv1.Deployment, error) {
	condFunc := func() (bool, error) {
		_, err := clientset.AppsV1().Deployments(d.Namespace).Create(d)
		if err == nil {
			return true, nil
		}
		// TODO: Distinguish errors
		return false, err
	}
	if err := waitExponentialBackoff(condFunc); err != nil {
		return nil, err
	}

	// Try to fetch deployment from API Server, so we can get the status.
	return clientset.AppsV1().Deployments(d.Namespace).Get(d.Name, metav1.GetOptions{})
}

func loadConfig(kubeconfig, context string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = kubeconfig
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  context,
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
}

func createNamespace(clientset kubernetes.Interface, scaffoldName string) (string, error) {
	ts := time.Now().UnixNano()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("ingress-apisix-e2e-tests-%s-%d", scaffoldName, ts),
		},
	}

	var got *corev1.Namespace
	var err error

	err = wait.Poll(_defaultPollInterval, _defaultTimeout, func() (bool, error) {
		got, err = clientset.CoreV1().Namespaces().Create(ns)
		if err != nil {
			log.Errorf("unexpected error while creating namespace: %s", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return got.Name, nil
}

func deleteNamespace(clientset kubernetes.Interface, namespace string) error {
	gracePeriod := int64(0)
	pb := metav1.DeletePropagationBackground
	return clientset.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &pb,
	})
}

func waitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Jitter:   0,
		Steps:    6,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}

func createServiceAccount(clientset kubernetes.Interface, name, namespace string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	condFunc := func() (bool, error) {
		_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(sa)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}

func createClusterRoleBinding(clientset kubernetes.Interface, name, namespace, sa, clusterRole string) error {
	if err := clientset.RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      sa,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
	}
	condFunc := func() (bool, error) {
		_, err := clientset.RbacV1().ClusterRoleBindings().Create(crb)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}

func createConfigMap(clientset kubernetes.Interface, name, namespace string, data map[string]string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	condFunc := func() (bool, error) {
		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(cm)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}
