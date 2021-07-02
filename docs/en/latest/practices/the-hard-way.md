---
title: APISIX Ingress Controller the Hard Way
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

In this tutorial, we will install APISIX and APISIX Ingress Controller in Kubernetes from native yaml.

## Prerequisites

If you don't have a Kubernetes cluster to use, we recommend you to use [KiND](https://kind.sigs.k8s.io/docs/user/quick-start/) to create a local Kubernetes cluster.

```bash
kubectl create ns apisix
```

In this tutorial, all our operations will be performed at namespace `apisix`.

## ETCD Installation

Here, we will deploy a single-node ETCD cluster without authentication inside the Kubernetes cluster.

In this case, we assume you have a storage provisioner. If you are using KiND, a local path provisioner will be created automatically. If you don't have a storage provisioner or don't want to use persistence volume, you could use an `emptyDir` as volume.

```yaml
# etcd-headless.yaml
apiVersion: v1
kind: Service
metadata:
  name: etcd-headless
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
# etcd.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: etcd
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

Apply these two yaml files to Kubernetes, wait few seconds, etcd installation should be successful. We could run a health check to ensure that.

```bash
$ kubectl -n apisix exec -it etcd-0 -- etcdctl endpoint health
127.0.0.1:2379 is healthy: successfully committed proposal: took = 1.741883ms
```

Please notice that this etcd installation is quite simple and lack of many necessary production features, it should only be used for learning case. If you want to deploy a production-ready etcd, please refer to [bitnami/etcd](https://bitnami.com/stack/etcd/helm).

## APISIX Installation

Create a config file for our APISIX. We are going to deploy APISIX version 2.5.

Note that the APISIX ingress controller needs to communicate with the APISIX admin API, so we set `apisix.allow_admin` to `0.0.0.0/0` for test.

```yaml
apisix:
  node_listen: 9080             # APISIX listening port
  enable_heartbeat: true
  enable_admin: true
  enable_admin_cors: true
  enable_debug: false
  enable_dev_mode: false          # Sets nginx worker_processes to 1 if set to true
  enable_reuseport: true          # Enable nginx SO_REUSEPORT switch if set to true.
  enable_ipv6: true
  config_center: etcd             # etcd: use etcd to store the config value

  allow_admin:                  # http://nginx.org/en/docs/http/ngx_http_access_module.html#allow
    - 0.0.0.0/0
  port_admin: 9180

  # Default token when use API to call for Admin API.
  # *NOTE*: Highly recommended to modify this value to protect APISIX's Admin API.
  # Disabling this configuration item means that the Admin API does not
  # require any authentication.
  admin_key:
    # admin: can everything for configuration data
    - name: "admin"
      key: edd1c9f034335f136f87ad84b625c8f1
      role: admin
    # viewer: only can view configuration data
    - name: "viewer"
      key: 4054f7cf07e344346cd3f287985e76a2
      role: viewer
  # dns_resolver:
  #   - 127.0.0.1
  dns_resolver_valid: 30
  resolver_timeout: 5

nginx_config:                     # config for render the template to genarate nginx.conf
  error_log: "/dev/stderr"
  error_log_level: "warn"         # warn,error
  worker_rlimit_nofile: 20480     # the number of files a worker process can open, should be larger than worker_connections
  event:
    worker_connections: 10620
  http:
    access_log: "/dev/stdout"
    keepalive_timeout: 60s         # timeout during which a keep-alive client connection will stay open on the server side.
    client_header_timeout: 60s     # timeout for reading client request header, then 408 (Request Time-out) error is returned to the client
    client_body_timeout: 60s       # timeout for reading client request body, then 408 (Request Time-out) error is returned to the client
    send_timeout: 10s              # timeout for transmitting a response to the client.then the connection is closed
    underscores_in_headers: "on"   # default enables the use of underscores in client request header fields
    real_ip_header: "X-Real-IP"    # http://nginx.org/en/docs/http/ngx_http_realip_module.html#real_ip_header
    real_ip_from:                  # http://nginx.org/en/docs/http/ngx_http_realip_module.html#set_real_ip_from
      - 127.0.0.1
      - 'unix:'

etcd:
  host:
    - "http://etcd-headless.apisix.svc.cluster.local:2379"
  prefix: "/apisix"     # apisix configurations prefix
  timeout: 30   # seconds
plugins:                          # plugin list
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

Please make sure `etcd.host` matches the headless service we created at first. In our case, it's `http://etcd-headless.apisix.svc.cluster.local:2379`.

In this config, we defined an access key with the `admin` name under the `apisix.admin_key` section. This key is our API key, will be used to control APISIX later. This key is the default API key for APISIX, and it should be changed in production environments.

Save this as `config.yaml`, then run `kubectl -n apisix create cm apisix-conf --from-file ./config.yaml` to create configmap. Later we will mount this configmap into APISIX deployment.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apisix
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
          image: "apache/apisix:2.5-alpine"
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

Now, APISIX should be ready to use. Use `kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name` to list APISIX pod name. Here we assume the pod name is `apisix-7644966c4d-cl4k6`.

Let's have a check:

```bash
kubectl -n apisix exec -it apisix-7644966c4d-cl4k6 -- curl http://127.0.0.1:9080
```

If you are using Linux or macOS, run the command below in bash:

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl http://127.0.0.1:9080
```

If APISIX works properly, it should output: `{"error_msg":"404 Route Not Found"}`. Because we haven't defined any route yet.

## HTTPBIN service

Before configuring the APISIX, we need to create a test service. We use [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/) here. We put this httpbin service in `demo` namespace.

```bash
kubectl create ns demo
kubectl -n demo run httpbin --image-pull-policy=IfNotPresent --image kennethreitz/httpbin --port 80
kubectl -n demo expose pod httpbin --port 80
```

After the httpbin service started, we should be able to access it inside the APISIX pod via service.

```bash
kubectl -n apisix exec -it apisix-7644966c4d-cl4k6 -- curl http://httpbin.demo/get
```

This should output the request's query parameters, for example:

```json
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

To read more, please refer to [Getting Started](https://apisix.apache.org/docs/apisix/getting-started).

## Define Route

Now, we can define the route for proxying HTTPBIN service traffic through APISIX.

Assuming we want to route all traffic which URI has `/httpbin` prefix and the request contains `Host: httpbin.org` header.

Please notice that the admin port is `9180`.

```bash
kubectl -n apisix exec -it apisix-7644966c4d-cl4k6 -- curl "http://127.0.0.1:9180/apisix/admin/routes/1" -H "X-API-KEY: edd1c9f034335f136f87ad84b625c8f1" -X PUT -d '
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

The output would be like this:

```json
{"action":"set","node":{"key":"\/apisix\/routes\/1","value":{"status":1,"create_time":1621408897,"upstream":{"pass_host":"pass","type":"roundrobin","hash_on":"vars","nodes":{"httpbin.demo:80":1},"scheme":"http"},"update_time":1621408897,"priority":0,"host":"httpbin.org","id":"1","uri":"\/*"}}}
```

We could check route rules by `GET /apisix/admin/routes`:

```bash
kubectl -n apisix exec -it apisix-7644966c4d-cl4k6 -- curl "http://127.0.0.1:9180/apisix/admin/routes/1" -H "X-API-KEY: edd1c9f034335f136f87ad84b625c8f1"
```

It should output like this:

```json
{"action":"get","node":{"key":"\/apisix\/routes\/1","value":{"upstream":{"pass_host":"pass","type":"roundrobin","scheme":"http","hash_on":"vars","nodes":{"httpbin.demo:80":1}},"id":"1","create_time":1621408897,"update_time":1621408897,"host":"httpbin.org","priority":0,"status":1,"uri":"\/*"}},"count":"1"}
```

Now, we can test the routing rule:

```bash
kubectl -n apisix exec -it apisix-7644966c4d-cl4k6 -- curl "http://127.0.0.1:9080/get" -H 'Host: httpbin.org'
```

It will output like:

```json
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

## Install APISIX Ingress Controller

APISIX ingress controller can help you manage your configurations declaratively by using Kubernetes resources. Here we will install version 0.5.0.

Currently, the APISIX ingress controller supports both official Ingress resource or APISIX's CustomResourceDefinitions, which includes ApisixRoute and ApisixUpstream.

Before installing the APISIX controller, we need to create a service account and the corresponding ClusterRole to ensure that the APISIX ingress controller has sufficient permissions to access required resources.

Here is an example config from [apisix-helm-chart](https://github.com/apache/apisix-helm-chart):

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: apisix-ingress-controller
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: apisix-clusterrole
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
      - networkpolicies
    verbs:
      - get
      - list
      - watch
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
      - get
      - list
      - watch
  - apiGroups:
      - coordination.k8s.io
    resources:
      - leases
    verbs:
      - '*'
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: apisix-clusterrolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: apisix-clusterrole
subjects:
  - kind: ServiceAccount
    name: apisix-ingress-controller
    namespace: apisix
```

Then, we need to create ApisixRoute CRD:

```yaml

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apisixroutes.apisix.apache.org
spec:
  group: apisix.apache.org
  versions:
    - name: v1
      served: true
      storage: false
    - name: v2alpha1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: apisixroutes
    singular: apisixroute
    kind: ApisixRoute
    shortNames:
      - ar
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apisixtlses.apisix.apache.org
spec:
  group: apisix.apache.org
  versions:
    - name: v1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: apisixtlses
    singular: apisixtls
    kind: ApisixTls
    shortNames:
      - atls
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apisixupstreams.apisix.apache.org
spec:
  group: apisix.apache.org
  versions:
    - name: v1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: apisixupstreams
    singular: apisixupstream
    kind: ApisixUpstream
    shortNames:
      - au
```

This yaml doesn't contain all the CRDs for APISIX Ingress Controller. Please refer to [samples](http://github.com/apache/apisix-ingress-controller/blob/master/samples/deploy/crd) for details.

To make the ingress controller works properly with APISIX, we need to create a config file containing the APISIX admin API URL and API key as below:

```yaml
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
      app_namespaces:
      - "*"
      ingress_class: "apisix"
      ingress_version: "networking/v1"
      apisix_route_version: "apisix.apache.org/v2alpha1"
    apisix:
      default_cluster_base_url: "http://apisix-admin.apisix:9180/apisix/admin"
      default_cluster_admin_key: "edd1c9f034335f136f87ad84b625c8f1"
kind: ConfigMap
metadata:
  name: apisix-configmap
  labels:
    app.kubernetes.io/name: ingress-controller
```

If you want to learn all the configuration items, see [conf/config-default.yaml](http://github.com/apache/apisix-ingress-controller/blob/master/conf/config-default.yaml) for details.

Because the ingress controller needs to access APISIX admin API, we need to create a service for APISIX.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: apisix-admin
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

Because currently APISIX ingress controller doesn't 100% compatible with APISIX, we need to delete the previously created route in case of some data structure mismatch.

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl "http://127.0.0.1:9180/apisix/admin/routes/1" -X DELETE -H "X-API-KEY: edd1c9f034335f136f87ad84b625c8f1"
```

After these configurations, we could deploy the ingress controller now.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apisix-ingress-controller
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
            name: apisix-configmap
            items:
              - key: config.yaml
                path: config.yaml
      containers:
        - name: ingress-controller
          command:
            - /ingress-apisix/apisix-ingress-controller
            - ingress
            - --config-path
            - /ingress-apisix/conf/config.yaml
          image: "apache/apisix-ingress-controller:0.5.0"
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

In this deployment, we mount the configmap created above as a config file, and tell Kubernetes to use the service account `apisix-ingress-controller`.

After the ingress controller status is converted to `Running`, we could create an ApisixRoute resource and observe its behaviors.

Here is an example ApisixRoute:

```yaml
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpserver-route
spec:
  http:
    - name: httpbin
      match:
        hosts:
          - local.httpbin.org
        paths:
          - "/*"
      backend:
        serviceName: httpbin
        servicePort: 80
```

Note that the apiVersion field should match the configmap above. And the serviceName should match the exposed service name, it's `httpbin` here.

Before create it, let's ensure requests with header `Host: local.http.demo` will returns 404:

```bash
kubectl -n apisix exec -it apisix-7644966c4d-cl4k6 -- curl "http://127.0.0.1:9080/get" -H 'Host: local.httpbin.org'
```

It will return:

```json
{"error_msg":"404 Route Not Found"}
```

The ApisixRoute should be applied in the same namespace with the target service, in this case is `demo`. After applying it, let's check if it works.

```bash
kubectl -n apisix exec -it $(kubectl get pods -n apisix -l app.kubernetes.io/name=apisix -o name) -- curl "http://127.0.0.1:9080/get" -H "Host: local.httpbin.org"
```

It should return:

```json
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

That's all! Enjoy your journey with APISIX and APISIX ingress controller!
