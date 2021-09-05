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

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
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
      - events
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
	_ingressAPISIXDeploymentTemplate = `
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
      initContainers:
      - name: wait-apisix-admin
        image: localhost:5000/busybox:1.28
        command: ['sh', '-c', "until nc -z apisix-service-e2e-test.%s.svc.cluster.local 9180 ; do echo waiting for apisix-admin; sleep 2; done;"]
      containers:
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 5
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
          image: "localhost:5000/apache/apisix-ingress-controller:dev"
          imagePullPolicy: Always
          name: ingress-apisix-controller-deployment-e2e-test
          ports:
            - containerPort: 8080
              name: "http"
              protocol: "TCP"
            - containerPort: 8443
              name: "https"
              protocol: "TCP"
          command:
            - /ingress-apisix/apisix-ingress-controller
            - ingress
            - --log-level
            - debug
            - --log-output
            - stdout
            - --http-listen
            - :8080
            - --https-listen
            - :8443
            - --default-apisix-cluster-name
            - default
            - --default-apisix-cluster-base-url
            - http://apisix-service-e2e-test:9180/apisix/admin
            - --default-apisix-cluster-admin-key
            - edd1c9f034335f136f87ad84b625c8f1
            - --app-namespace
            - %s,kube-system
            - --apisix-route-version
            - %s
            - --watch-endpointslices
          volumeMounts:
           - name: webhook-certs
             mountPath: /etc/webhook/certs
             readOnly: true
      volumes:
       - name: webhook-certs
         secret:
           secretName: %s
      serviceAccount: ingress-apisix-e2e-test-service-account
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
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: apisix-validation-webhooks-e2e-test
webhooks:
  - name: apisixroute-plugin-validator-webhook.apisix.apache.org
    clientConfig:
      service:
        name: webhook
        namespace: %s
        port: 8443
        path: "/validation/apisixroutes/plugin"
      caBundle: %s
    rules:
      - operations: [ "CREATE", "UPDATE" ]
        apiGroups: ["apisix.apache.org"]
        apiVersions: ["*"]
        resources: ["apisixroutes"]
    timeoutSeconds: 30
    failurePolicy: Fail
`
	_webhookCertSecret = "webhook-certs"
)

func (s *Scaffold) newIngressAPISIXController() error {
	err := k8s.CreateServiceAccountE(s.t, s.kubectlOptions, _serviceAccount)
	assert.Nil(s.t, err, "create service account")

	cr := fmt.Sprintf(_clusterRole, s.namespace)
	err = k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, cr)
	assert.Nil(s.t, err, "create cluster role")

	crb := fmt.Sprintf(_clusterRoleBinding, s.namespace, s.namespace, s.namespace)
	err = k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, crb)
	assert.Nil(s.t, err, "create cluster role binding")

	s.addFinalizers(func() {
		err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, crb)
		assert.Nil(s.t, err, "deleting ClusterRoleBinding")
	})
	s.addFinalizers(func() {
		err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, cr)
		assert.Nil(s.t, err, "deleting ClusterRole")
	})

	ingressAPISIXDeployment := fmt.Sprintf(_ingressAPISIXDeploymentTemplate, s.opts.IngressAPISIXReplicas, s.namespace, s.namespace, s.opts.APISIXRouteVersion, _webhookCertSecret)
	err = k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, ingressAPISIXDeployment)
	assert.Nil(s.t, err, "create deployment")

	if s.opts.EnableWebhooks {
		admissionSvc := fmt.Sprintf(_ingressAPISIXAdmissionService, s.namespace)
		err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, admissionSvc)
		assert.Nil(s.t, err, "create admission webhook service")

		// get caBundle from the secret
		secret, err := k8s.GetSecretE(s.t, s.kubectlOptions, _webhookCertSecret)
		assert.Nil(s.t, err, "get webhook secret")
		cert, ok := secret.Data["cert.pem"]
		assert.True(s.t, ok, "get cert.pem from the secret")
		caBundle := base64.StdEncoding.EncodeToString(cert)

		webhookReg := fmt.Sprintf(_ingressAPISIXAdmissionWebhook, s.namespace, caBundle)
		err = k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, webhookReg)
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

	return nil
}

func (s *Scaffold) waitAllIngressControllerPodsAvailable() error {
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
func (s *Scaffold) GetIngressPodDetails() ([]v1.Pod, error) {
	return k8s.ListPodsE(s.t, s.kubectlOptions, metav1.ListOptions{
		LabelSelector: "app=ingress-apisix-controller-deployment-e2e-test",
	})
}
