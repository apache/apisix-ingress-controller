package scaffold

import (
	"fmt"

	"github.com/gruntwork-io/terratest/modules/k8s"
	corev1 "k8s.io/api/core/v1"
)

var (
	_testBackendDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-backend-deployment-e2e-test
spec:
  replicas: %d
  selector:
    matchLabels:
      app: test-backend-deployment-e2e-test
  strategy:
    rollingUpdate:
      maxSurge: 50%%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: test-backend-deployment-e2e-test
    spec:
      terminationGracePeriodSeconds: 0
      containers:
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 2
            periodSeconds: 5
            successThreshold: 1
            tcpSocket:
              port: 80
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 2
            periodSeconds: 5
            successThreshold: 1
            tcpSocket:
              port: 80
            timeoutSeconds: 2
          image: "localhost:5000/apache/test-backend:dev"
          imagePullPolicy: IfNotPresent
          name: test-backend-deployment-e2e-test
          ports:
            - containerPort: 80
              name: "http"
              protocol: "TCP"
            - containerPort: 443
              name: "https"
              protocol: "TCP"
            - containerPort: 8443
              name: "http-mtls"
              protocol: "TCP"
            - containerPort: 50051
              name: "grpc"
              protocol: "TCP"
            - containerPort: 50052
              name: "grpcs"
              protocol: "TCP"
            - containerPort: 50053
              name: "grpc-mtls"
              protocol: "TCP"
`
	_testBackendService = `
apiVersion: v1
kind: Service
metadata:
  name: test-backend-service-e2e-test
spec:
  selector:
    app: test-backend-deployment-e2e-test
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
    - name: https
      port: 443
      protocol: TCP
      targetPort: 443
    - name: http-mtls
      port: 8443
      protocol: TCP
      targetPort: 8443
    - name: grpc
      port: 50051
      protocol: TCP
      targetPort: 50051
    - name: grpcs
      port: 50052
      protocol: TCP
      targetPort: 50052
    - name: grpc-mtls
      port: 50053
      protocol: TCP
      targetPort: 50053
  type: ClusterIP
`
)

func (s *Scaffold) newTestBackend() (*corev1.Service, error) {
	backendDeployment := fmt.Sprintf(_testBackendDeploymentTemplate, 1)
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, backendDeployment); err != nil {
		return nil, err
	}
	if err := k8s.KubectlApplyFromStringE(s.t, s.kubectlOptions, _testBackendService); err != nil {
		return nil, err
	}
	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, "test-backend-service-e2e-test")
	if err != nil {
		return nil, err
	}
	return svc, nil
}
