// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package scaffold

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gavv/httpexpect/v2"
	"github.com/google/uuid"
	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_apisixConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: apisix-gw-config.yaml
data:
  config.yaml: |
    apisix:
      enable_control: true
      proxy_mode: http&stream
      stream_proxy:
        tcp:
          - 9100
          - addr: 9110
            tls: true
        udp:
          - 9200
      node_listen:
        - 9080
        - enable_http2: true
          port: 9081
      enable_admin: true
      ssl:
        enabled: true
    nginx_config:
      worker_processes: 1
      error_log_level: debug
    deployment:
      role: traditional
      role_traditional:
        config_provider: etcd
      etcd:
        host:
          - "http://192.168.31.50:7900"
        timeout: 30
        resync_delay: 0
      admin:
        allow_admin:
          - all
    discovery:
      dns:
        servers:
          - "10.96.0.10:53"
    plugin_attr:
      prometheus:
        enable_export_server: false
    plugins:
      - real-ip                        # priority: 23000
      - ai                             # priority: 22900
      - client-control                 # priority: 22000
      - proxy-control                  # priority: 21990
      - request-id                     # priority: 12015
      - zipkin                         # priority: 12011
      #- skywalking                    # priority: 12010
      #- opentelemetry                 # priority: 12009
      - ext-plugin-pre-req             # priority: 12000
      - fault-injection                # priority: 11000
      - mocking                        # priority: 10900
      - serverless-pre-function        # priority: 10000
      #- batch-requests                # priority: 4010
      - cors                           # priority: 4000
      - ip-restriction                 # priority: 3000
      - ua-restriction                 # priority: 2999
      - referer-restriction            # priority: 2990
      - csrf                           # priority: 2980
      - uri-blocker                    # priority: 2900
      - request-validation             # priority: 2800
      - multi-auth                     # priority: 2600
      - openid-connect                 # priority: 2599
      - cas-auth                       # priority: 2597
      - authz-casbin                   # priority: 2560
      - authz-casdoor                  # priority: 2559
      - wolf-rbac                      # priority: 2555
      - ldap-auth                      # priority: 2540
      - hmac-auth                      # priority: 2530
      - basic-auth                     # priority: 2520
      - jwt-auth                       # priority: 2510
      - key-auth                       # priority: 2500
      - consumer-restriction           # priority: 2400
      - forward-auth                   # priority: 2002
      - opa                            # priority: 2001
      - authz-keycloak                 # priority: 2000
      #- error-log-logger              # priority: 1091
      - proxy-cache                    # priority: 1085
      - body-transformer               # priority: 1080
      - proxy-mirror                   # priority: 1010
      - proxy-rewrite                  # priority: 1008
      - workflow                       # priority: 1006
      - api-breaker                    # priority: 1005
      - limit-conn                     # priority: 1003
      - limit-count                    # priority: 1002
      - limit-req                      # priority: 1001
      #- node-status                   # priority: 1000
      #- brotli                        # priority: 996
      - gzip                           # priority: 995
      - server-info                    # priority: 990
      - traffic-split                  # priority: 966
      - redirect                       # priority: 900
      - response-rewrite               # priority: 899
      - degraphql                      # priority: 509
      - kafka-proxy                    # priority: 508
      - grpc-transcode                 # priority: 506
      - grpc-web                       # priority: 505
      - public-api                     # priority: 501
      - prometheus                     # priority: 500
      - datadog                        # priority: 495
      - elasticsearch-logger           # priority: 413
      - echo                           # priority: 412
      - loggly                         # priority: 411
      - http-logger                    # priority: 410
      - splunk-hec-logging             # priority: 409
      - skywalking-logger              # priority: 408
      - google-cloud-logging           # priority: 407
      - sls-logger                     # priority: 406
      - tcp-logger                     # priority: 405
      - kafka-logger                   # priority: 403
      - rocketmq-logger                # priority: 402
      - syslog                         # priority: 401
      - udp-logger                     # priority: 400
      - file-logger                    # priority: 399
      - clickhouse-logger              # priority: 398
      - tencent-cloud-cls              # priority: 397
      - inspect                        # priority: 200
      #- log-rotate                    # priority: 100
      # <- recommend to use priority (0, 100) for your custom plugins
      - example-plugin                 # priority: 0
      #- gm                            # priority: -43
      #- ocsp-stapling                 # priority: -44
      - aws-lambda                     # priority: -1899
      - azure-functions                # priority: -1900
      - openwhisk                      # priority: -1901
      - openfunction                   # priority: -1902
      - serverless-post-function       # priority: -2000
      - ext-plugin-post-req            # priority: -3000
      - ext-plugin-post-resp           # priority: -4000
`
	_apisixDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apisix-deployment-e2e-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: apisix-deployment-e2e-test
  strategy:
    rollingUpdate:
      maxSurge: 50%%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: apisix-deployment-e2e-test
    spec:
      terminationGracePeriodSeconds: 0
      containers:
        - livenessProbe:
            failureThreshold: 3
            initialDelaySeconds: 1
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 9080
            timeoutSeconds: 2
          readinessProbe:
            failureThreshold: 3
            initialDelaySeconds: 1
            periodSeconds: 2
            successThreshold: 1
            tcpSocket:
              port: 9080
            timeoutSeconds: 2
          image: "api7-dev/api7-ee-3-gateway:dev"
          imagePullPolicy: IfNotPresent
          name: apisix-deployment-e2e-test
          env:
            - name: API7_CONTROL_PLANE_TOKEN
              value: %s
          securityContext:
            runAsNonRoot: false
            runAsUser: 0
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
      volumes:
        - configMap:
            name: apisix-gw-config.yaml
          name: apisix-config-yaml-configmap
`
	_apisixService = `
apiVersion: v1
kind: Service
metadata:
  name: apisix-service-e2e-test
spec:
  selector:
    app: apisix-deployment-e2e-test
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
)

type APISIXConfig struct {
	// Used for template rendering.
	EtcdServiceFQDN string
}

func (s *Scaffold) newAPISIXConfigMap(cm *APISIXConfig) error {
	if err := s.CreateResourceFromString(_apisixConfigMap); err != nil {
		return err
	}
	return nil
}

type requestFactory struct{}

func (requestFactory) NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	r.SetBasicAuth("admin", "admin")
	return r, nil
}

func (s *Scaffold) newAPISIX() (*corev1.Service, error) {
	t := ginkgo.GinkgoT()
	e := httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  "http://192.168.31.50:7080",
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewCompactPrinter(t),
		},
		RequestFactory: requestFactory{},
	})
	id := e.POST("/api/gateway_groups").WithJSON(map[string]interface{}{
		"name": uuid.New().String(),
	}).Expect().Status(http.StatusOK).JSON().Path("$.value.id").String().Raw()
	token := e.POST("/api/gateway_groups/" + id + "/instance_token").Expect().Status(http.StatusOK).JSON().Path("$.value.token_plain_text").String().Raw()
	println("token: ", token)
	deployment := fmt.Sprintf(_apisixDeployment, token)
	if err := s.CreateResourceFromString(deployment); err != nil {
		return nil, err
	}
	if err := s.CreateResourceFromString(_apisixService); err != nil {
		return nil, err
	}

	svc, err := k8s.GetServiceE(s.t, s.kubectlOptions, "apisix-service-e2e-test")
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func indent(data string) string {
	list := strings.Split(data, "\n")
	for i := 0; i < len(list); i++ {
		list[i] = "    " + list[i]
	}
	return strings.Join(list, "\n")
}

func (s *Scaffold) waitAllAPISIXPodsAvailable() error {
	opts := metav1.ListOptions{
		LabelSelector: "app=apisix-deployment-e2e-test",
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(s.t, s.kubectlOptions, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			ginkgo.GinkgoT().Log("no apisix pods created")
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
