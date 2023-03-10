---
title: Install APISIX Ingress with Kubernetes manifest files
keywords:
  - APISIX Ingress
  - Apache APISIX
  - Kubernetes Ingress
  - Kubernetes manifest
description: A guide to check the synchronization status of APISIX CRDs.
---
<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

This tutorial will walk you through installing APISIX and APISIX Ingress controller with Kubernetes manifest files.

## Prerequisites

Before you move on, make sure you have access to a Kubernetes cluster. This tutorial uses [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) to create the cluster.

Create a namespace `apisix` in your cluster:

```bash
kubectl create ns apisix
```

## Installing etcd

For this example, we will deploy a single-node etcd cluster without authentication.

This tutorial also assumes that you have a storage provisioner. If you are using kind, it would be created for you automatically. If you don't have a storage provisioner or don't want to use a persistence volume, you could use `emptyDir` as your volume.

The yaml file below will install etcd:

```yaml title="etcd.yaml"
apiVersion: v1
kind: Service
metadata:
  name: etcd-headless
  namespace: apisix
  labels:
    app.kubernetes.io/name: etcd
  annotations:
    service.alpha.kubernetes.io/tolerate-unready-endpoints: "true"
spec:
  type: ClusterIP
  clusterIP: None
  ports:
    - name: "client"
      port: 2379
      targetPort: client
    - name: "peer"
      port: 2380
      targetPort: peer
  selector:
    app.kubernetes.io/name: etcd
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: etcd
  namespace: apisix
  labels:
    app.kubernetes.io/name: etcd
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: etcd
  serviceName: etcd-headless
  podManagementPolicy: Parallel
  replicas: 1
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app.kubernetes.io/name: etcd
    spec:
      securityContext:
        fsGroup: 1001
        runAsUser: 1001
      containers:
        - name: etcd
          image: docker.io/bitnami/etcd:3.4.14-debian-10-r0
          imagePullPolicy: "IfNotPresent"
          # command:
            # - /scripts/setup.sh
          env:
            - name: BITNAMI_DEBUG
              value: "false"
            - name: MY_POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: MY_POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: ETCDCTL_API
              value: "3"
            - name: ETCD_NAME
              value: "$(MY_POD_NAME)"
            - name: ETCD_DATA_DIR
              value: /etcd/data
            - name: ETCD_ADVERTISE_CLIENT_URLS
              value: "http://$(MY_POD_NAME).etcd-headless.apisix.svc.cluster.local:2379"
            - name: ETCD_LISTEN_CLIENT_URLS
              value: "http://0.0.0.0:2379"
            - name: ETCD_INITIAL_ADVERTISE_PEER_URLS
              value: "http://$(MY_POD_NAME).etcd-headless.apisix.svc.cluster.local:2380"
            - name: ETCD_LISTEN_PEER_URLS
              value: "http://0.0.0.0:2380"
            - name: ALLOW_NONE_AUTHENTICATION
              value: "yes"
          ports:
            - name: client
              containerPort: 2379
            - name: peer
              containerPort: 2380
          volumeMounts:
            - name: data
              mountPath: /etcd
      # if you don't have a storage provisioner or don't want to use a persistent volume
      # volumes:
      #   - name: data
      #     emptyDir: {}
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes:
          - "ReadWriteOnce"
        resources:
          requests:
            storage: "8Gi"
```

Once you have applied these files, you can wait for some time and run a health check to ensure everything is running:

```bash
kubectl -n apisix exec -it etcd-0 -- etcdctl endpoint health
```

```text title="output"
127.0.0.1:2379 is healthy: successfully committed proposal: took = 1.741883ms
```

:::info IMPORTANT

This etcd installation is simple and not meant for production scenarios. If you want to deploy a production ready etcd cluster, see [bitnami/etcd](https://bitnami.com/stack/etcd/helm).

:::

## Installing APISIX

Before deploying APISIX, we will first create a configuration file.

APISIX Ingress controller will need to communicate with the APISIX Admin API, so we need to set `apisix.allow_admin` to `0.0.0.0/0`.

```yaml title="config.yaml"
apiVersion: v1
kind: ConfigMap
metadata:
  name: apisix-conf
  namespace: apisix
data:
  config.yaml: |-
    apisix:
      node_listen: 9080             # APISIX listening port
      enable_heartbeat: true
      enable_admin: true
      enable_admin_cors: true
      enable_debug: false
      enable_dev_mode: false          # when set to true, sets Nginx worker_processes to 1
      enable_reuseport: true          # when set to true, enables nginx SO_REUSEPORT switch
      enable_ipv6: true
      config_center: etcd             # use etcd to store configuration

      allow_admin:                  # see: http://nginx.org/en/docs/http/ngx_http_access_module.html#allow
        - 0.0.0.0/0
      port_admin: 9180

      # default token used when calling the Admin API
      # it is recommended to modify this value in production
      # when disabled, Admin API won't require any authentication
      admin_key:
        # admin: full access to configuration data
        - name: "admin"
          key: edd1c9f034335f136f87ad84b625c8f1
          role: admin
        # viewer: can only view the configuration data
        - name: "viewer"
          key: 4054f7cf07e344346cd3f287985e76a2
          role: viewer
      # dns_resolver:
      #   - 127.0.0.1
      dns_resolver_valid: 30
      resolver_timeout: 5

    nginx_config:                     # template configuration to generate nginx.conf
      error_log: "/dev/stderr"
      error_log_level: "warn"         # warn, error
      worker_rlimit_nofile: 20480     # number of files a worker process can open. Should be larger than worker_connections
      event:
        worker_connections: 10620
      http:
        access_log: "/dev/stdout"
        keepalive_timeout: 60s         # timeout for which a keep-alive client connection will stay open on the server side
        client_header_timeout: 60s     # timeout for reading client request header, then 408 (Request Time-out) error is returned to the client
        client_body_timeout: 60s       # timeout for reading client request body, then 408 (Request Time-out) error is returned to the client
        send_timeout: 10s              # timeout for transmitting a response to the client, then the connection is closed
        underscores_in_headers: "on"   # enables the use of underscores in client request header fields
        real_ip_header: "X-Real-IP"    # see: http://nginx.org/en/docs/http/ngx_http_realip_module.html#real_ip_header
        real_ip_from:                  # see: http://nginx.org/en/docs/http/ngx_http_realip_module.html#set_real_ip_from
          - 127.0.0.1
          - 'unix:'

    etcd:
      host:
        - "http://etcd-headless.apisix.svc.cluster.local:2379"
      prefix: "/apisix"     # APISIX configurations prefix
      timeout: 30   # in seconds
    plugins:                          # list of APISIX Plugins
      - api-breaker
      - authz-keycloak
      - basic-auth
      - batch-requests
      - consumer-restriction
      - cors
      - echo
      - fault-injection
      - grpc-transcode
      - hmac-auth
      - http-logger
      - ip-restriction
      - jwt-auth
      - kafka-logger
      - key-auth
      - limit-conn
      - limit-count
      - limit-req
      - node-status
      - openid-connect
      - prometheus
      - proxy-cache
      - proxy-mirror
      - proxy-rewrite
      - redirect
      - referer-restriction
      - request-id
      - request-validation
      - response-rewrite
      - serverless-post-function
      - serverless-pre-function
      - sls-logger
      - syslog
      - tcp-logger
      - udp-logger
      - uri-blocker
      - wolf-rbac
      - zipkin
      - traffic-split
    stream_plugins:
      - mqtt-proxy
```

:::note

Make sure that `etcd.host` matches the headless etcd service we created first. In this case, it is `http://etcd-headless.apisix.svc.cluster.local:2379`.

:::

The Admin API key (`apisix.admin_key` in `config.yaml`) will be used to configure APISIX later.

:::danger

The key used in the example above is the default key and should be changed in production environments.

:::

We can now create a ConfigMap from this configuration file. To do this, run:

```shell
kubectl -n apisix apply -f ./apisix-config.yaml
```

We can mount this ConfigMap to the APISIX deployment.

The yaml file below will deploy APISIX:

```yaml title="apisix-dep.yaml"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apisix
  namespace: apisix
  labels:
    app.kubernetes.io/name: apisix
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: apisix
  template:
    metadata:
      labels:
        app.kubernetes.io/name: apisix
    spec:
      containers:
        - name: apisix
          image: "apache/apisix:2.15.0-alpine"
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 9080
              protocol: TCP
            - name: tls
              containerPort: 9443
              protocol: TCP
            - name: admin
              containerPort: 9180
              protocol: TCP
          readinessProbe:
            failureThreshold: 6
            initialDelaySeconds: 10
            periodSeconds: 10
            successThreshold: 1
            tcpSocket:
              port: 9080
            timeoutSeconds: 1
          lifecycle:
            preStop:
              exec:
                command:
                - /bin/sh
                - -c
                - "sleep 30"
          volumeMounts:
            - mountPath: /usr/local/apisix/conf/config.yaml
              name: apisix-config
              subPath: config.yaml
          resources: {}
      volumes:
        - configMap:
            name: apisix-conf
          name: apisix-config
```

APISIX will be ready in some time. You can check the pod name of APISIX by running:

```shell
kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name
```

The examples below use the pod name `apisix-7644966c4d-cl4k6`.

You can check if APISIX is deployed correctly by running:

```bash
kubectl -n apisix exec -it apisix-7644966c4d-cl4k6 -- curl http://127.0.0.1:9080
```

If you are on Linux or macOS, you can run the command below instead:

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl http://127.0.0.1:9080
```

APISIX should show a "Route not found" message as we haven't configured it yet:

```json
{"error_msg":"404 Route Not Found"}
```

## Deploying httpbin

We will deploy a sample application to test APISIX. We are using [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/) and we will deploy it to the `demo` namespace:

```bash
kubectl create ns demo
kubectl -n demo run httpbin --image-pull-policy=IfNotPresent --image kennethreitz/httpbin --port 80
kubectl -n demo expose pod httpbin --port 80
```

Once httpbin is running, we can access it in the APISIX pod using the created service:

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl http://httpbin.demo/get
```

```json title="output"
{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "httpbin.demo",
    "User-Agent": "curl/7.67.0"
  },
  "origin": "172.17.0.1",
  "url": "http://httpbin.demo/get"
}
```

## Configuring a Route

Now, we will create a Route in APISIX to forward traffic to the httpbin service.

The below command will configure APISIX to route all requests with the Header `Host: httpbin.org`:

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl "http://127.0.0.1:9180/apisix/admin/routes/1" -H "X-API-KEY: edd1c9f034335f136f87ad84b625c8f1" -X PUT -d '
{
  "uri": "/*",
  "host": "httpbin.org",
  "upstream": {
    "type": "roundrobin",
    "nodes": {
      "httpbin.demo:80": 1
    }
  }
}'
```

This will create a Route and will give back a response as shown below:

```json title="output"
{
   "action":"set",
   "node":{
      "key":"\/apisix\/routes\/1",
      "value":{
         "status":1,
         "create_time":1621408897,
         "upstream":{
            "pass_host":"pass",
            "type":"roundrobin",
            "hash_on":"vars",
            "nodes":{
               "httpbin.demo:80":1
            },
            "scheme":"http"
         },
         "update_time":1621408897,
         "priority":0,
         "host":"httpbin.org",
         "id":"1",
         "uri":"\/*"
      }
   }
}
```

Now we can test the created Route:

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl "http://127.0.0.1:9080/get" -H 'Host: httpbin.org'
```

This will give back a response from httpbin:

```json title="output"
{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "httpbin.org",
    "User-Agent": "curl/7.67.0",
    "X-Forwarded-Host": "httpbin.org"
  },
  "origin": "127.0.0.1",
  "url": "http://httpbin.org/get"
}
```

## Installing APISIX Ingress controller

Till now, we manually sent requests to the Admin API to configure APISIX. Installing APISIX Ingress controller will allow you to configure APISIX using Kubernetes resources.

APISIX Ingress controller supports the Kubernetes Ingress API, Gateway API, and APISIX custom CRDs for configuration.

First we will create a ServiceAccount and a corresponding ClusterRole to ensure that the Ingress controller has sufficient permissions to access the required resources:

```bash
git clone https://github.com/apache/apisix-ingress-controller.git --depth 1
cd apisix-ingress-controller/
kubectl apply -k samples/deploy/rbac/apisix_view_clusterrole.yaml # apply cluster role
kubectl -n apisix create serviceaccount apisix-ingress-controller # create service account
# bind cluster role and service account
kubectl create clusterrolebinding apisix-viewer --clusterrole=apisix-view-clusterrole --serviceaccount=apisix:apisix-ingress-controller
```

Once you apply it to your cluster, you have to create the [ApisixRoute](https://apisix.apache.org/docs/ingress-controller/concepts/apisix_route) CRD:

```bash
# Under apisix-ingress-controller git repo
kubectl apply -k samples/deploy/crd
```

See [samples](http://github.com/apache/apisix-ingress-controller/blob/master/samples/deploy/crd) for details.

For the Ingress controller to work with APISIX, you need to create a config file containing the APISIX Admin API URL and key. You can do this by creating a ConfigMap:

```yaml title="apisix-config.yaml"
apiVersion: v1
data:
  config.yaml: |
    # log options
    log_level: "debug"
    log_output: "stderr"
    http_listen: ":8080"
    enable_profiling: true
    kubernetes:
      kubeconfig: ""
      resync_interval: "30s"
      namespace_selector:
      - ""
      ingress_class: "apisix"
      ingress_version: "networking/v1"
      apisix_route_version: "apisix.apache.org/v2"
    apisix:
      default_cluster_base_url: "http://apisix-admin.apisix:9180/apisix/admin"
      default_cluster_admin_key: "edd1c9f034335f136f87ad84b625c8f1"
kind: ConfigMap
metadata:
  name: apisix-ingress-conf
  namespace: apisix
  labels:
    app.kubernetes.io/name: ingress-controller
```

See [conf/config-default.yaml](http://github.com/apache/apisix-ingress-controller/blob/master/conf/config-default.yaml) for a list of all the available configurations.

Now we will create a Service for the Ingress controller to access the Admin API:

```yaml title="ingress-service.yaml"
apiVersion: v1
kind: Service
metadata:
  name: apisix-admin
  namespace: apisix
  labels:
    app.kubernetes.io/name: apisix
spec:
  type: ClusterIP
  ports:
  - name: apisix-admin
    port: 9180
    targetPort: 9180
    protocol: TCP
  selector:
    app.kubernetes.io/name: apisix
```

We can delete the existing Route in APISIX through the Admin API before we create a new Route. This is to prevent any error due to data structure mismatches which will be fixed in the future:

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl "http://127.0.0.1:9180/apisix/admin/routes/1" -X DELETE -H "X-API-KEY: edd1c9f034335f136f87ad84b625c8f1"
```

Now we can create a Deployment to install the Ingress controller in our cluster:

```yaml title="ingress-deployment.yaml"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apisix-ingress-controller
  namespace: apisix
  labels:
    app.kubernetes.io/name: ingress-controller
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: ingress-controller
  template:
    metadata:
      labels:
        app.kubernetes.io/name: ingress-controller
    spec:
      serviceAccountName: apisix-ingress-controller
      volumes:
        - name: configuration
          configMap:
            name: apisix-ingress-conf
            items:
              - key: config.yaml
                path: config.yaml
      initContainers:
        - name: wait-apisix-admin
          image: busybox:1.28
          command: ['sh', '-c', "until nc -z apisix-admin.apisix.svc.cluster.local 9180 ; do echo waiting for apisix-admin; sleep 2; done;"]
      containers:
        - name: ingress-controller
          command:
            - /ingress-apisix/apisix-ingress-controller
            - ingress
            - --config-path
            - /ingress-apisix/conf/config.yaml
          image: "apache/apisix-ingress-controller:1.6.0"
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8080
          resources:
            {}
          volumeMounts:
            - mountPath: /ingress-apisix/conf
              name: configuration
```

Once the Ingress controller is in the `Running` state, you can create a Route using the ApisixRoute resource:

```yaml title="httpbin-route.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpserver-route
  namespace: demo
spec:
  http:
  - name: httpbin
    match:
      hosts:
      - local.httpbin.org
      paths:
      - /*
    backends:
      - serviceName: httpbin
        servicePort: 80
```

The `apiVersion` field should match the created ConfigMap and the `serviceName` here is the `httpbin` service.

The ApisixRoute should be created in the same namespace as the service. In our example, this is the `demo` namespace.

Now if you send requests to APISIX, it will be routed to the httpbin service:

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl "http://127.0.0.1:9080/get" -H "Host: local.httpbin.org"
```

```json title="output"
{
  "args": {},
  "headers": {
    "Accept": "*/*",
    "Host": "local.httpbin.org",
    "User-Agent": "curl/7.67.0",
    "X-Forwarded-Host": "local.httpbin.org"
  },
  "origin": "127.0.0.1",
  "url": "http://local2.httpbin.org/get"
}
```

See [Installation](https://apisix.apache.org/docs/ingress-controller/deployments/minikube) for more installation methods.
