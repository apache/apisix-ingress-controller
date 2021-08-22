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
            - containerPort: 443
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
            - :443
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
           secretName: apisix-ingress-controller-webhook-certs
      serviceAccount: ingress-apisix-e2e-test-service-account
`
	_ingressAPISIXAdmissionService = `
apiVersion: v1
kind: Service
metadata:
  name: apisix-admission-server-e2e-test
  namespace: %s
spec:
  ports:
    - name: https
      protocol: TCP
      port: 443
      targetPort: 443
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
        name: apisix-admission-server-e2e-test
        namespace: %s
        path: "/validation/apisixroute/plugin"
      caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUwQ0ZIb1c5NjR6T0dlMjl0WEp3QTRXV3NyVXh5Z2dNQTBHQ1NxR1NJYjNEUUVCQ3dVQU1EOHgKUFRBN0JnTlZCQU1NTkdGd2FYTnBlQzFwYm1keVpYTnpMV052Ym5SeWIyeHNaWEl0ZDJWaWFHOXZheTVwYm1keQpaWE56TFdGd2FYTnBlQzV6ZG1Nd0hoY05NakV3T0RJeE1EZ3hORFF6V2hjTk1qSXdPREl4TURneE5EUXpXakEvCk1UMHdPd1lEVlFRREREUmhjR2x6YVhndGFXNW5jbVZ6Y3kxamIyNTBjbTlzYkdWeUxYZGxZbWh2YjJzdWFXNW4KY21WemN5MWhjR2x6YVhndWMzWmpNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQQoybU1EbkFrSGJwbU1QTWdaSFRoNVZLblVYUnJIS01ZM09FelR5RHM0TXhCU3J4QnNJclJZWGpYQmk2QTc1SVJVClhEOS9XOER5SUVOY2xMUlRyWWRMdDAzT0Q4bjVhMlo2K0RXOFhmQU8wRlowNThRbnlLT285djEvUktxSGtQdFYKUHdiQ2pVdkNDQ2xzZ2loT1N6eGNnY0Yyb0htMngxSmFBVEJpY1dOUzRjemU2THJrbVZTSTJCTC82bGlVOWhTSgoxNU10eU5ScWUxOHNRLzd6NmNXWkJrQWZ3VzlwWTRsQzBKV05IbnRGZG5RSnpQbHcvak05cnpIYW1uQnJNYXM5ClIyVERWcWZnVVJxS21KYVFCZDBsRHRjOVpycDJHOWRDcW1GOFVQM092SDNjQmE4VUtCem80a1BSc2pLRWY1K3IKenpNcndORzdrWDQ3SzgySlpOaEtsd0lEQVFBQk1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQkJKMzg4MWtNRgpEYVlYSjl4bElvMGxXaWp0OHlvRG41YkdYcnZUMFErdExKaGJGbVZoOU1yKy9Od2F5dGhLUE00ZGNYWFdLbHdOCkhhbThPcHFmRlAyQlo5M252K0NYZ1F4cGROQUdRUE5tSjMxNDZvOHNKcGJuTndRQ1Rjb2U5bm02NkRUVzYzNDAKU0NkRHd1d2tOUk1zYzI0RW5UZG13ZTdaMFhCZ3orangwV0dsenhtZVFLSlFWVUNocDd3MXFOaVVmTldqSzNVZApoQ1VqbVV3aXFWcGs5K0k5OTdhOS9ETnU2Q0V0N1NJSkszbmJ1TFdEdVhhNFMzZWJNZ1ZsQ0dYQWFwYjVRZkRlClMzQlRBamd1dXlnd2JwbzRNK1M2aHlPYk1wZE5icjlkVmhGTEdqMDJsekwzYSttTTFDMTlrSkNwYkpnZ3UxWTMKb1hERjRWMlhIYnpKCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
    rules:
      - operations: [ "CREATE", "UPDATE" ]
        apiGroups: ["apisix.apache.org"]
        apiVersions: ["*"]
        resources: ["apisixroutes"]
    timeoutSeconds: 30
    failurePolicy: Fail
`
	_webhookCertSecret = "apisix-ingress-controller-webhook-certs"
	_tlsCert           = `-----BEGIN CERTIFICATE-----
MIIDBTCCAe0CFHoW964zOGe29tXJwA4WWsrUxyggMA0GCSqGSIb3DQEBCwUAMD8x
PTA7BgNVBAMMNGFwaXNpeC1pbmdyZXNzLWNvbnRyb2xsZXItd2ViaG9vay5pbmdy
ZXNzLWFwaXNpeC5zdmMwHhcNMjEwODIxMDgxNDQzWhcNMjIwODIxMDgxNDQzWjA/
MT0wOwYDVQQDDDRhcGlzaXgtaW5ncmVzcy1jb250cm9sbGVyLXdlYmhvb2suaW5n
cmVzcy1hcGlzaXguc3ZjMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
2mMDnAkHbpmMPMgZHTh5VKnUXRrHKMY3OEzTyDs4MxBSrxBsIrRYXjXBi6A75IRU
XD9/W8DyIENclLRTrYdLt03OD8n5a2Z6+DW8XfAO0FZ058QnyKOo9v1/RKqHkPtV
PwbCjUvCCClsgihOSzxcgcF2oHm2x1JaATBicWNS4cze6LrkmVSI2BL/6liU9hSJ
15MtyNRqe18sQ/7z6cWZBkAfwW9pY4lC0JWNHntFdnQJzPlw/jM9rzHamnBrMas9
R2TDVqfgURqKmJaQBd0lDtc9Zrp2G9dCqmF8UP3OvH3cBa8UKBzo4kPRsjKEf5+r
zzMrwNG7kX47K82JZNhKlwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBBJ3881kMF
DaYXJ9xlIo0lWijt8yoDn5bGXrvT0Q+tLJhbFmVh9Mr+/NwaythKPM4dcXXWKlwN
Ham8OpqfFP2BZ93nv+CXgQxpdNAGQPNmJ3146o8sJpbnNwQCTcoe9nm66DTW6340
SCdDwuwkNRMsc24EnTdmwe7Z0XBgz+jx0WGlzxmeQKJQVUChp7w1qNiUfNWjK3Ud
hCUjmUwiqVpk9+I997a9/DNu6CEt7SIJK3nbuLWDuXa4S3ebMgVlCGXAapb5QfDe
S3BTAjguuygwbpo4M+S6hyObMpdNbr9dVhFLGj02lzL3a+mM1C19kJCpbJggu1Y3
oXDF4V2XHbzJ
-----END CERTIFICATE-----
`
	_tlsKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEA2mMDnAkHbpmMPMgZHTh5VKnUXRrHKMY3OEzTyDs4MxBSrxBs
IrRYXjXBi6A75IRUXD9/W8DyIENclLRTrYdLt03OD8n5a2Z6+DW8XfAO0FZ058Qn
yKOo9v1/RKqHkPtVPwbCjUvCCClsgihOSzxcgcF2oHm2x1JaATBicWNS4cze6Lrk
mVSI2BL/6liU9hSJ15MtyNRqe18sQ/7z6cWZBkAfwW9pY4lC0JWNHntFdnQJzPlw
/jM9rzHamnBrMas9R2TDVqfgURqKmJaQBd0lDtc9Zrp2G9dCqmF8UP3OvH3cBa8U
KBzo4kPRsjKEf5+rzzMrwNG7kX47K82JZNhKlwIDAQABAoIBAQDQJY9LKU/sGm2P
gShusWTzTOsb0moAcuwuvQsdzVPDV8t3EDAA4+NV5+aRLifnpjjBs8OvsDcWiR20
nisjOdDw5TeB1P/lXcfWy2C+KA/2gnDqdgt1MIfa4cJrsB2GEgcuC0NjaNGG9fR2
GfSFwQJqqfpm+Zs8X0Fp4LPzXregfd//sgnNi5dorWxZ142lJvAStC/inEzLFBLW
hC+tDq9zIXUmAhlMzfmJ3cf8gU7z+RMOYkNFaz7EGM6wWZSppiWBk9A7BiknV5AJ
cQRv2woGy2ZgP7MXZVg8RNaX5w6P6GFEK5NbdoyHkGL2olvf8tN7f9oNLdv9apQf
6F3l7OABAoGBAP6sX+tSqs/oAouyZQ4v9NnrnhBKgPgnMwcKaohg4jo58TMJ5ldQ
U10AkZyfVcQ/gE7531N+6D/fzEYSwiiZdsOFVEMHQitIXIZMDeyU+EPoZawyHCpn
h6NuaStkXqowtEdkscJgiCRBNncnKwvCuLu8copoglfwPaaLMzrBilzNAoGBANuG
P6f3XLfvyDyVDM6oAbLVQGIfEBrSueyoLIackSe1a1mJ7pTmMnY9S/9W+i3ZR6Kp
tAKUnEkoN90l8R/1V0x7AobOhMWicblo23eAw9r6jXKZtUxlhbjNKYzfQRVetbT4
ix/qKdme1dXLAeM4YgF1CKxO1ccf6fOJArWpSwTzAoGBAOoux+U0ly2nQvACkzqA
jr71EtwYJpAKO7n1shDGRkEUlt8/8zfG/WE/7KYBPnS/j9UPoHS+9gIGYWjuRuve
cn9IUztvqUDzwWEc/pDWS5TmVtgJHC1CFlAKb1sfaI1HS/96cJs0+Pudm9/lfIfL
/uNjXlA32ePTXl2PEwSsg/bhAoGBAIthmss/8LvM4BsvG9merK1qXx2t0WDmiSws
v1Cc2kEXHFjWjgg2fLW8R6ORCvnPan9qNqQozW5ZvdaJP6bl9I7Xz4veVkjR0llB
rY8bz78atHKeC5G9KAFlKkuKeN1jrAWChXs3B2loQyciZUlqxDdeoqocx/lNVxLM
3E6RddNnAoGBAMCjs0qKwT5ENMsaQxFlwPEKuC5Sl0ejKgUnoHsVl9VuhAMcwE70
hMJMGXv2p1BbBuuW35jH92LBSBjS/Zv4b86DG2VQsDWNI4u3lPFd1zif6dhE8yvU
bKS1uxKukPFp6zxFwR7YZIiwo3tGkcudpHdTNurNMQiSTN97LTo8KL8y
-----END RSA PRIVATE KEY-----
`
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

	// create tls secret
	err = s.NewSecret(_webhookCertSecret, _tlsCert, _tlsKey)
	assert.Nil(ginkgo.GinkgoT(), err, "create tls cert secret error")

	ingressAPISIXDeployment := fmt.Sprintf(_ingressAPISIXDeploymentTemplate, s.opts.IngressAPISIXReplicas, s.namespace, s.opts.APISIXRouteVersion)
	err = k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, ingressAPISIXDeployment)
	assert.Nil(s.t, err, "create deployment")

	if s.opts.EnableWebhooks {
		admissionSvc := fmt.Sprintf(_ingressAPISIXAdmissionService, s.namespace)
		err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, admissionSvc)
		assert.Nil(s.t, err, "create admission webhook service")

		webhookReg := fmt.Sprintf(_ingressAPISIXAdmissionWebhook, s.namespace)
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
