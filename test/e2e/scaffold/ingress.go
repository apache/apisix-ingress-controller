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
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IngressControllerServiceName = "apisix-ingress-controller"
)

var (
	_serviceAccount = "ingress-apisix-e2e-test-service-account"
	_clusterRole    = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %s-apisix-view-clusterrole
rules:
  - apiGroups:
    - ""
    resources:
    - events
    verbs:
      - "*"
  - apiGroups:
      - ""
    resources:
      - configmaps
      - endpoints
      - persistentvolumeclaims
      - pods
      - replicationcontrollers
      - replicationcontrollers/scale
      - serviceaccounts
      - services
      - secrets
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - bindings
      - limitranges
      - namespaces/status
      - pods/log
      - pods/status
      - replicationcontrollers/status
      - resourcequotas
      - resourcequotas/status
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - namespaces
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - apps
    resources:
      - controllerrevisions
      - daemonsets
      - deployments
      - deployments/scale
      - replicasets
      - replicasets/scale
      - statefulsets
      - statefulsets/scale
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - autoscaling
    resources:
      - horizontalpodautoscalers
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - batch
    resources:
      - cronjobs
      - jobs
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - extensions
    resources:
      - daemonsets
      - deployments
      - deployments/scale
      - ingresses
      - networkpolicies
      - replicasets
      - replicasets/scale
      - replicationcontrollers/scale
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - policy
    resources:
      - poddisruptionbudgets
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - networking.k8s.io
    resources:
      - ingresses
      - ingresses/status
      - networkpolicies
    verbs:
      - '*'
  - apiGroups:
      - metrics.k8s.io
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - apisix.apache.org
    resources:
      - apisixroutes
      - apisixroutes/status
      - apisixupstreams
      - apisixupstreams/status
      - apisixtlses
      - apisixtlses/status
      - apisixclusterconfigs
      - apisixclusterconfigs/status
      - apisixconsumers
      - apisixconsumers/status
      - apisixpluginconfigs
      - apisixpluginconfigs/status
      - apisixglobalrules
      - apisixglobalrules/status
    verbs:
      - '*'
  - apiGroups:
    - coordination.k8s.io
    resources:
    - leases
    verbs:
    - '*'
  - apiGroups:
    - discovery.k8s.io
    resources:
    - endpointslices
    verbs:
    - get
    - list
    - watch
  - apiGroups:
    - gateway.networking.k8s.io
    resources:
    - httproutes
    - tlsroutes
    - tcproutes
    - gateways
    - gatewayclasses
    - udproutes
    verbs:
    - get
    - list
    - watch
  - apiGroups:
    - gateway.networking.k8s.io
    resources:
    - gateways/status
    - gatewayclasses/status
    verbs:
    - get
    - update
    - list
`
	_clusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: %s-clusterrolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s-apisix-view-clusterrole
subjects:
- kind: ServiceAccount
  name: ingress-apisix-e2e-test-service-account
  namespace: %s
`
	_ingressAPISIXAdmissionService = `
apiVersion: v1
kind: Service
metadata:
  name: webhook
  namespace: %s
spec:
  ports:
    - name: https
      protocol: TCP
      port: 8443
      targetPort: 8443
  selector:
    app: ingress-apisix-controller-deployment-e2e-test
`

	_ingressAPISIXAdmissionWebhook = `
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: ingress-apisix-webhook-%s
  labels:
    app: ingress-apisix-webhhok
webhooks:
- name: apisix.validator.webhook.kubernetes.io
  admissionReviewVersions: ["v1", "v1beta1"]
  clientConfig:
    service:
      name: webhook
      namespace: %s
      port: 8443
      path: /validate
    caBundle: %s
  rules:
  - apiGroups: 
    - "apisix.apache.org"
    apiVersions:
    - v2
    operations:
    - CREATE
    - UPDATE  
    resources:
    - apisixroutes
    - apisixglobalrules
    - apisixconsumers
    - apisixpluginconfigs
    - apisixclusterconfigs
    - apisixtlses
    - apisixupstreams
  timeoutSeconds: 30
  failurePolicy: Fail
  sideEffects: None
  namespaceSelector:
    matchLabels:
      %s: %s
`
)

var _initContainers = `
      initContainers:
      - name: wait-apisix-admin
        image: localhost:5000/busybox:dev
        imagePullPolicy: IfNotPresent
        command: ['sh', '-c', "until nc -z apisix-service-e2e-test 9180 ; do echo waiting for apisix-admin; sleep 2; done;"]
`

var _apisixContainer = `
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 10
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 9080
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 8
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 9080
            timeoutSeconds: 1
          name: apisix-container
          image: "localhost:5000/apisix:dev"
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 9080
              name: "http"
              protocol: "TCP"
            - containerPort: 9180
              name: "http-admin"
              protocol: "TCP"
            - containerPort: 9443
              name: "https"
              protocol: "TCP"
          volumeMounts:
            - mountPath: /usr/local/apisix/conf/config.yaml
              name: apisix-config-yaml-configmap
              subPath: config.yaml
`

var _ingressAPISIXService = `
apiVersion: v1
kind: Service
metadata:
  name: apisix-service-e2e-test
spec:
  selector:
    app: ingress-apisix-controller-deployment-e2e-test
  ports:
    - name: http
      port: 9080
      protocol: TCP
      targetPort: 9080
    - name: http-admin
      port: 9180
      protocol: TCP
      targetPort: 9180
    - name: https
      port: 9443
      protocol: TCP
      targetPort: 9443
    - name: tcp
      port: 9100
      protocol: TCP
      targetPort: 9100
    - name: tcp-tls
      port: 9110
      protocol: TCP
      targetPort: 9110
    - name: udp
      port: 9200
      protocol: UDP
      targetPort: 9200
    - name: http-control
      port: 9090
      protocol: TCP
      targetPort: 9090
  type: NodePort
`

var _ingressAPISIXDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ingress-apisix-controller-deployment-e2e-test
spec:
  replicas: %d
  selector:
    matchLabels:
      app: ingress-apisix-controller-deployment-e2e-test
  strategy:
    rollingUpdate:
      maxSurge: 50%%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ingress-apisix-controller-deployment-e2e-test
    spec:
      terminationGracePeriodSeconds: 0
      %s
      containers:
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 7
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 8080
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 5
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 8080
            timeoutSeconds: 2
          env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
          image: "localhost:5000/apisix-ingress-controller:dev"
          imagePullPolicy: IfNotPresent
          name: ingress-apisix-controller-deployment-e2e-test
          ports:
            - containerPort: 8080
              name: "http"
              protocol: "TCP"
            - containerPort: 8443
              name: "https"
              protocol: "TCP"
            - containerPort: 2379
              name: "etcd-server"
              protocol: "TCP"
          command:
            - /ingress-apisix/apisix-ingress-controller
            - ingress
            - --log-level
            - debug
            - --log-output
            - stdout
            - --apisix-admin-api-version
            - %s
            - --apisix-resource-sync-interval
            - %s
            - --apisix-resource-sync-comparison=%s
            - --http-listen
            - :8080
            - --https-listen
            - :8443
            - --default-apisix-cluster-name
            - default
            - --default-apisix-cluster-base-url
            - %s
            - --default-apisix-cluster-admin-key
            - edd1c9f034335f136f87ad84b625c8f1
            - --namespace-selector
            - %s
            - --api-version
            - %s
            - --ingress-status-address
            - "%s"
            - --enable-admission=%t
            - --enable-gateway-api=true
            - --ingress-class
            - %s
            - --disable-status-updates=%t
            - --etcd-server-enabled=%t
            - --etcd-server-listen-address
            - ":2379"
          volumeMounts:
            - name: admission-webhook
              mountPath: /etc/webhook/certs
              readOnly: true
        %s
      volumes:
       - name: admission-webhook
         secret:
           secretName: webhook-certs
           optional: true
       - configMap:
           name: apisix-gw-config.yaml
         name: apisix-config-yaml-configmap
      serviceAccount: ingress-apisix-e2e-test-service-account
`

func init() {
	if os.Getenv("E2E_ENV") != "ci" {
		_ingressAPISIXDeploymentTemplate = strings.Replace(_ingressAPISIXDeploymentTemplate, "imagePullPolicy: IfNotPresent", "imagePullPolicy: Always", -1)
	}
}

func (s *Scaffold) genIngressDeployment(replicas int, adminAPIVersion,
	syncInterval, syncComparison, label, resourceVersion, publishAddr string, webhooks bool,
	ingressClass string, disableStatus bool, etcdserverEnabled bool) string {
	var (
		initContainers  = _initContainers
		apisixBaseURL   = "http://apisix-service-e2e-test:9180/apisix/admin"
		apisixContainer string
	)

	if etcdserverEnabled {
		initContainers = ""
		apisixBaseURL = "http://127.0.0.1:9180/apisix/admin"
		apisixContainer = _apisixContainer
	}

	return s.FormatRegistry(fmt.Sprintf(_ingressAPISIXDeploymentTemplate, replicas, initContainers, adminAPIVersion, syncInterval, syncComparison,
		apisixBaseURL, label, resourceVersion, publishAddr, webhooks, ingressClass, disableStatus, etcdserverEnabled, apisixContainer))

}

func (s *Scaffold) newIngressAPISIXController() error {
	err := k8s.CreateServiceAccountE(s.t, s.kubectlOptions, _serviceAccount)
	assert.Nil(s.t, err, "create service account")

	cr := fmt.Sprintf(_clusterRole, s.namespace)
	err = s.CreateResourceFromString(cr)
	assert.Nil(s.t, err, "create cluster role")

	crb := fmt.Sprintf(_clusterRoleBinding, s.namespace, s.namespace, s.namespace)
	err = s.CreateResourceFromString(crb)
	assert.Nil(s.t, err, "create cluster role binding")

	s.addFinalizers(func() {
		err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, crb)
		assert.Nil(s.t, err, "deleting ClusterRoleBinding")
	})
	s.addFinalizers(func() {
		err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, cr)
		assert.Nil(s.t, err, "deleting ClusterRole")
	})

	label := `""`
	if labels := s.NamespaceSelectorLabelStrings(); labels != nil && !s.opts.DisableNamespaceSelector {
		label = strings.Join(labels, ",")
	}
	if s.opts.EnableWebhooks {
		s.createAdmissionWebhook()
	}

	if s.opts.EnableEtcdServer {
		err = s.CreateResourceFromString(_ingressAPISIXService)
		assert.Nil(s.t, err, "create ingress-apisix service")

		s.apisixService, err = k8s.GetServiceE(s.t, s.kubectlOptions, "apisix-service-e2e-test")
		assert.Nil(s.t, err, "get ingress-apisix service")
	}

	ingressAPISIXDeployment := s.genIngressDeployment(s.opts.IngressAPISIXReplicas, s.opts.APISIXAdminAPIVersion,
		s.opts.ApisixResourceSyncInterval, s.opts.ApisixResourceSyncComparison, label, s.opts.ApisixResourceVersion, s.opts.APISIXPublishAddress,
		s.opts.EnableWebhooks, s.opts.IngressClass, s.opts.DisableStatus, s.opts.EnableEtcdServer)

	err = s.CreateResourceFromString(ingressAPISIXDeployment)
	assert.Nil(s.t, err, "create deployment")

	return nil
}

func (s *Scaffold) createAdmissionWebhook() {
	err := generateWebhookCert(s.namespace)
	assert.Nil(s.t, err, "generate certs and create webhook secret")
	admissionSvc := fmt.Sprintf(_ingressAPISIXAdmissionService, s.namespace)
	err = k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, admissionSvc)
	assert.Nil(s.t, err, "create admission webhook service")

	// get caBundle from the secret
	secret, err := k8s.GetSecretE(s.t, s.kubectlOptions, "webhook-certs")
	assert.Nil(s.t, err, "get webhook secret")
	cert, ok := secret.Data["cert.pem"]
	assert.True(s.t, ok, "get cert.pem from the secret")
	caBundle := base64.StdEncoding.EncodeToString(cert)
	s.NamespaceSelectorLabel()
	webhookReg := fmt.Sprintf(_ingressAPISIXAdmissionWebhook, s.namespace, s.namespace, caBundle, "apisix.ingress.watch", s.namespace)
	ginkgo.GinkgoT().Log(webhookReg)
	err = s.CreateResourceFromString(webhookReg)
	assert.Nil(s.t, err, "create webhook registration")

	s.addFinalizers(func() {
		err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, admissionSvc)
		assert.Nil(s.t, err, "deleting admission service")
	})
	s.addFinalizers(func() {
		err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, webhookReg)
		assert.Nil(s.t, err, "deleting webhook registration")
	})
}

func (s *Scaffold) WaitAllIngressControllerPodsAvailable() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=ingress-apisix-controller-deployment-e2e-test",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(s.t, s.kubectlOptions, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no ingress-apisix-controller pods created")
			return false, nil
		}
		for _, item := range items {
			foundPodReady := false
			foundContainerReady := false
			for _, cond := range item.Status.Conditions {
				if cond.Type == corev1.PodReady {
					foundPodReady = true
					if cond.Status != "True" {
						return false, nil
					}
				}

				if cond.Type == corev1.ContainersReady {
					foundContainerReady = true
					if cond.Status != "True" {
						return false, nil
					}
				}
			}
			if !foundPodReady || !foundContainerReady {
				return false, nil
			}
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}

// WaitGetLeaderLease waits the lease to be created and returns it.
func (s *Scaffold) WaitGetLeaderLease() (*coordinationv1.Lease, error) {
	cli, err := k8s.GetKubernetesClientE(s.t)
	if err != nil {
		return nil, err
	}
	var lease *coordinationv1.Lease
	condFunc := func() (bool, error) {
		l, err := cli.CoordinationV1().Leases(s.namespace).Get(context.TODO(), "ingress-apisix-leader", metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		lease = l
		return true, nil
	}
	if err := waitExponentialBackoff(condFunc); err != nil {
		return nil, err
	}
	return lease, nil
}

// GetIngressPodDetails returns a batch of pod description
// about apisix-ingress-controller.
func (s *Scaffold) GetIngressPodDetails() ([]corev1.Pod, error) {
	return k8s.ListPodsE(s.t, s.kubectlOptions, metav1.ListOptions{
		LabelSelector: "app=ingress-apisix-controller-deployment-e2e-test",
	})
}

// ScaleIngressController scales the number of Ingress Controller pods to desired.
func (s *Scaffold) ScaleIngressController(desired int) error {
	label := `""`
	if labels := s.NamespaceSelectorLabelStrings(); labels != nil {
		label = strings.Join(labels, ",")
	}

	ingressDeployment := s.genIngressDeployment(desired, s.opts.APISIXAdminAPIVersion,
		s.opts.ApisixResourceSyncInterval, s.opts.ApisixResourceSyncComparison, label, s.opts.ApisixResourceVersion, s.opts.APISIXPublishAddress,
		s.opts.EnableWebhooks, s.opts.IngressClass, s.opts.DisableStatus, s.opts.EnableEtcdServer)

	if err := s.CreateResourceFromString(ingressDeployment); err != nil {
		return err
	}
	if err := k8s.WaitUntilNumPodsCreatedE(s.t, s.kubectlOptions, s.labelSelector("app=ingress-apisix-controller-deployment-e2e-test"), desired, 5, 5*time.Second); err != nil {
		return err
	}
	return nil
}

// generateWebhookCert generates signed certs of webhook and create the corresponding secret by running a script.
func generateWebhookCert(ns string) error {
	commandTemplate := `testdata/webhook-create-cert.sh`
	os.Setenv("namespace", ns)
	cmd := exec.Command("/bin/sh", commandTemplate, "--namespace", ns)

	output, err := cmd.Output()
	if err != nil {
		ginkgo.GinkgoT().Errorf("%s", output)
		return fmt.Errorf("failed to execute the script: %v", err)
	}
	return nil
}
