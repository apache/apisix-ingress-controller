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
      port: 443
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
        path: "/validation/apisixroute/plugin"
      caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURaVENDQWsyZ0F3SUJBZ0lSQVA1VWkwNXJZK0JlVkw3Q2tBNENpL1l3RFFZSktvWklodmNOQVFFTEJRQXcKRlRFVE1CRUdBMVVFQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TVRBNE1qVXhNVFE0TlRKYUZ3MHlNakE0TWpVeApNVFE0TlRKYU1DVXhJekFoQmdOVkJBTVRHbmRsWW1odmIyc3VhVzVuY21WemN5MWhjR2x6YVhndWMzWmpNSUlCCklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0aFNKZUZpc1IyL3E2Tyt1dU5FT3p2c2IKSnJ2ZTVCQ2tnaGVLSDh3TSs3akh0d0ZpRDArRi9LQ1FFN0hYQjlLY3JSRjYxSys3QzRINDBiYmYyTjdoQWtYTAo0bm5KeUMrSkYwRDhwTFp3c1E5bWpWdUtaU2xQdDBRRE13UENLc3NvTFhTdTBLVEZYYlVTVTZucjAzVEdNTmJDCkxHT2t4NTFKakp2bGQvQUZ5ZWp0TFA1cXhyRVpzVjNqN3lvbFNXWTNJV2dsMEsrM0hNU1JMVytPNlR3OEp1MTMKbjRsSDdoY0hLbmhBSHZjL2ExQy9KRTJrOXR3TlJPdDJVeVd2R24ybHJBQjhsMWtwbHhOYUVsM1pSYWRKMlpXQwprTnhneC94NG1ZS2hPRGFZQjVCSDRNeHh4a1Y3eEpITlZFa1pueTBjZXBKd3VuUFZEblh2WmplVnJXS2Npd0lECkFRQUJvNEdmTUlHY01BNEdBMVVkRHdFQi93UUVBd0lGb0RBVEJnTlZIU1VFRERBS0JnZ3JCZ0VGQlFjREFUQU0KQmdOVkhSTUJBZjhFQWpBQU1COEdBMVVkSXdRWU1CYUFGRng5V0JoK3NzSzdZd2FCbDVVT0k1SkczOTVRTUVZRwpBMVVkRVFRL01EMkNCM2RsWW1odmIydUNGbmRsWW1odmIyc3VhVzVuY21WemN5MWhjR2x6YVhpQ0duZGxZbWh2CmIyc3VhVzVuY21WemN5MWhjR2x6YVhndWMzWmpNQTBHQ1NxR1NJYjNEUUVCQ3dVQUE0SUJBUUJSdUcwVXJMa20KRFNFRVBkZWdlekplcE94ckJNaHpYMzJPSVNldndxTlRlWTA3OEQ5eUg1Y0pZNmJvTjBxeURicGpaTVVzdXpmYgp5OXovbFAzbFRiUlpUeUJJYkdaeXNvemRsMGMxVjQralRkcFYyOFlvWDNFVTVQRXErczhoNHpqbTM4elYxQnFDCkVwb01RYk00YUZGbE9jcFZrZXpGOTE0L0JyODNZNGFaUVZERDBFTlQ0ZWVXdVp6eTdxak9RRVFNdVpyckp0aFMKa2hIUkhTaVh6YVdNU0R0QzE4VzJNNjZvVFZwWnpFRzhhbUhZUmpFTmpCa0tYeWJCMUk2VFFnbEdpbHdydVJpdgpVdWpyaGFmL0F1cmRsNEVSTHhRaldnUmJEUzFyTENBR0F3czcyaFhNeTdpYjdtc21mblFzK3pLZEZ1Rk9jMXRCCndoa25BVWk2bFdzdAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    rules:
      - operations: [ "CREATE", "UPDATE" ]
        apiGroups: ["apisix.apache.org"]
        apiVersions: ["*"]
        resources: ["apisixroutes"]
    timeoutSeconds: 30
    failurePolicy: Fail
`
	_webhookCertSecret = "webhook-certs"
	_tlsCert           = `-----BEGIN CERTIFICATE-----
MIIDZTCCAk2gAwIBAgIRAP5Ui05rY+BeVL7CkA4Ci/YwDQYJKoZIhvcNAQELBQAw
FTETMBEGA1UEAxMKa3ViZXJuZXRlczAeFw0yMTA4MjUxMTQ4NTJaFw0yMjA4MjUx
MTQ4NTJaMCUxIzAhBgNVBAMTGndlYmhvb2suaW5ncmVzcy1hcGlzaXguc3ZjMIIB
IjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4hSJeFisR2/q6O+uuNEOzvsb
Jrve5BCkgheKH8wM+7jHtwFiD0+F/KCQE7HXB9KcrRF61K+7C4H40bbf2N7hAkXL
4nnJyC+JF0D8pLZwsQ9mjVuKZSlPt0QDMwPCKssoLXSu0KTFXbUSU6nr03TGMNbC
LGOkx51JjJvld/AFyejtLP5qxrEZsV3j7yolSWY3IWgl0K+3HMSRLW+O6Tw8Ju13
n4lH7hcHKnhAHvc/a1C/JE2k9twNROt2UyWvGn2lrAB8l1kplxNaEl3ZRadJ2ZWC
kNxgx/x4mYKhODaYB5BH4MxxxkV7xJHNVEkZny0cepJwunPVDnXvZjeVrWKciwID
AQABo4GfMIGcMA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAM
BgNVHRMBAf8EAjAAMB8GA1UdIwQYMBaAFFx9WBh+ssK7YwaBl5UOI5JG395QMEYG
A1UdEQQ/MD2CB3dlYmhvb2uCFndlYmhvb2suaW5ncmVzcy1hcGlzaXiCGndlYmhv
b2suaW5ncmVzcy1hcGlzaXguc3ZjMA0GCSqGSIb3DQEBCwUAA4IBAQBRuG0UrLkm
DSEEPdegezJepOxrBMhzX32OISevwqNTeY078D9yH5cJY6boN0qyDbpjZMUsuzfb
y9z/lP3lTbRZTyBIbGZysozdl0c1V4+jTdpV28YoX3EU5PEq+s8h4zjm38zV1BqC
EpoMQbM4aFFlOcpVkezF914/Br83Y4aZQVDD0ENT4eeWuZzy7qjOQEQMuZrrJthS
khHRHSiXzaWMSDtC18W2M66oTVpZzEG8amHYRjENjBkKXybB1I6TQglGilwruRiv
Uujrhaf/Aurdl4ERLxQjWgRbDS1rLCAGAws72hXMy7ib7msmfnQs+zKdFuFOc1tB
whknAUi6lWst
-----END CERTIFICATE-----
`
	_tlsKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA4hSJeFisR2/q6O+uuNEOzvsbJrve5BCkgheKH8wM+7jHtwFi
D0+F/KCQE7HXB9KcrRF61K+7C4H40bbf2N7hAkXL4nnJyC+JF0D8pLZwsQ9mjVuK
ZSlPt0QDMwPCKssoLXSu0KTFXbUSU6nr03TGMNbCLGOkx51JjJvld/AFyejtLP5q
xrEZsV3j7yolSWY3IWgl0K+3HMSRLW+O6Tw8Ju13n4lH7hcHKnhAHvc/a1C/JE2k
9twNROt2UyWvGn2lrAB8l1kplxNaEl3ZRadJ2ZWCkNxgx/x4mYKhODaYB5BH4Mxx
xkV7xJHNVEkZny0cepJwunPVDnXvZjeVrWKciwIDAQABAoIBAQC0wHCsXFDZCJzK
wZ5yuwpY56BslnX852VvcTyIcY7Lzo82PI/W5+Ca+xBV/rCJ25RSNpB67UjhSXfS
y6Aqdv903rLEjlSKjZ7Qja+wTQDKPyLhz5dVi/Lk9iaMqeuaZTTpKsn9nE8DvZo6
c7dNJ6axM3KpJL2Arrs4BQgwnSEzrFr84Qqf52O9eE7wo+uSX1e9/jk4Rbyo4vuw
odCX5iNBzB9lFLk5NiYZVsn/eMdHbPXN1zszB3Zc/hb7W3QiiOtL8La9WziJjTML
DSIRbuK2to7558SjeA1zaLFX3SaO2RZChPTeNJF1VYXQPXW6ow45HpowjkRFDogY
I+zHWT5JAoGBAPTOmrATnmQbdm99nabKbJcwZdGqZl1js0fRI+SJ3ngnNnuTAPsT
qZRKMNwS/rXrsIlebohWX0KPlRS0mPUdd7lLWv9/+Yh0Wkp5pNa/lI6+UHCGAdhz
MJt6+f1zUDpMnABPss1qX3DGiMCjhZ+9xao3jXSRKg2eZFO+zdEht7J9AoGBAOxq
vZxGqtvnx5t8bNuRj54WjTNyVrah9uVKvQ7ylzY5SDLbScGC32NuBUocnibGiqwk
P/jd2/6AD4rm5qqWpXWcIXY9uOIgmU9sUeUdDHsz2UvyQ8i+gGnBCzGZuguSMuUn
j7v2HHFcDWvvWVow3M2ovf+AnARs4lyhCYDzFHGnAoGBANaRp9+gsnmH4J0D+wRP
9DHoB7ZnpmVAl8jgtJcBiG7D3+scBAYNS9tf08dxFrOZKxicHkF9gu0yMDb/u/lL
pL5SICZFow9I/EK+sA5RyQH8KUEXE9MF05rThP3y7mTK9QkI0e1dyN1uBjrimKJU
kUYKfv+mpLdfFwyX9onRBdN5AoGAb9m4R11vrIal+zwMzHy7c9G7kCGCQPmzs5t+
grnnLHJBZD43UOQ4B/SfcAbGFBZOuU6VLYrZcDjqIY9IhmCre08YzbY56FH/9oGK
5Viu9QL8xV+jDjCC1IXOY/MVADB0/9GNwSGZJ1Cj0PL2VSNU87/n1B/msHlLRwOx
WV6nx3UCgYAuMe4h0LlvMUA0g1H2tBNc4+ihyqr51PSDHyv8Zd7Tw7OvuuY5HPzR
yXDxPlI/HDXUAVIgFc1Ux4af/U21M0TI7QsCTBNkFcspn9TD+Z4yHaZNDb3U2Hr1
o2mKYK6kGOWeBpP5laq+Fics6BJBOkCl7qZLaQg9LC3iSLg+yzzwTQ==
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

	ingressAPISIXDeployment := fmt.Sprintf(_ingressAPISIXDeploymentTemplate, s.opts.IngressAPISIXReplicas, s.namespace, s.opts.APISIXRouteVersion, _webhookCertSecret)
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
