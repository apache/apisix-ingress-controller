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

# Document for developer

## Dependencies

1. Kubernetes
2. Apache APISIX
3. golang >= 1.13 (go mod)

## Prepare development environment

### 1. Kubernetes

[Install minikube](https://kubernetes.io/docs/tasks/tools/install-minikube)

Tips: The Kubernetes cluster deployment method is recommended for production and test environments.

### 2. Apache APISIX

[Install Apache APISIX in Kubernetes](https://github.com/apache/apisix/tree/master/kubernetes)

### 3. httpbin service

Deploy [httpbin](https://github.com/postmanlabs/httpbin) to your Kubernetes cluster and expose it as a Service.

## Configuration

### Configure the `kube config` file locally to facilitate local debugging

1. Start minikube.

2. Location: ~/.kube/config

3. Copy the config file to your local development environment, the path should be configured in apisix-ingress-controller by specifying `--kuebconfig` option.

### Configure APISIX service address

Your APISIX service address should be configured in apisix-ingress-controller by specifying `--apisix-base-url` option.

## Start ingress-controller locally

* Build and run apisix-ingress-controller

```shell
$ make build
$ ./apisix-ingress-controller ingress \
  --kubeconfig /path/to/kubeconfig \
  --http-listen :8080 \
  --log-output stderr \
  --apisix-base-url http://apisix-service:port/apisix
```

Tips: The program may print some error logs, indicating that the resource cannot be found. Continue with the following steps to define the route through CRDs.

### Define ApisixRoute

Take the backend service `httpbin` as an example (you can choose any other upstream services for test).

In fact, in order to reduce the trouble caused by ingress migration, we try to keep the structure of ApisixRoute consistent with the original ingress.

The configuration differences with nginx-ingress are

1. `apiVersion` and `kind` are different.
2. Path uses wildcards for prefix matching, such as `/hello*`.

```yaml
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  name: httpserver-route
spec:
  rules:
  - host: test.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: httpbin.default.svc.cluster.local
          servicePort: 8080
        path: /hello*
EOF
```

Here we use the FQDN `httpbin.default.svc.cluster.local` as the `serviceName`, and the service port is 8080, change them if your `httpbin` service has different name, namespace or port.

In addition, `ApisixRoute` also continues to support the definition with annotation, you can also define as below.

```yaml
apiVersion: apisix.apache.org/v1
kind: ApisixRoute
metadata:
  annotations:
    k8s.apisix.apache.org/cors-allow-headers: DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization,openID,audiotoken
    k8s.apisix.apache.org/cors-allow-methods: HEAD,GET,POST,PUT,PATCH,DELETE
    k8s.apisix.apache.org/cors-allow-origin: '*'
    k8s.apisix.apache.org/enable-cors: "true"
    k8s.apisix.apache.org/ssl-redirect: "false"
    k8s.apisix.apache.org/whitelist-source-range: 1.2.3.4,2.2.0.0/16
  name: httpserver-route
spec:
  rules:
  - host: test1.apisix.apache.org
    http:
      paths:
      - backend:
          serviceName: httpbin.default.svc.cluster.local
          servicePort: 8080
        path: /hello*
        plugins:
        - enable: true
          name: proxy-rewrite
          config:
            regex_uri:
            - '^/(.*)'
            - '/voice-copy-outer-service/$1'
            scheme: http
            host: internalalpha.talkinggenie.com
            enable_websocket: true
```

The definition in config needs to follow the schema definition of the plugin [proxy-rewrite](https://github.com/apache/apisix/blob/master/doc/plugins/proxy-rewrite.md).

If the plug-in schema is an array, you need to change the `config` field to `config_set`.

### Define ApisixService

```yaml
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v1
kind: ApisixService                   # apisix service
metadata:
  name: httpserver
spec:
  upstream: httpserver          # upstream = default/httpserver (namespace/upstreamName)
  port: 8080                        # set port on service
  plugins:
  - name: aispeech-chash
    enable: true
    config:
      uri_args:
        - "userId"
        - "productId|deviceName"
      key: "apisix-chash-key"
EOF
```

`ApisixService` supports plug-ins similar to `ApisixRoute`.

### Define ApisixUpstream

```yaml
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream                  # apisix upstream
metadata:
  name: httpbin      # default/httpbin
spec:
  ports:
  - port: 8080
    loadbalancer:
      type: chash
      hashOn: header
      key: hello
EOF
```

Now, you can try to modify these yaml according to the CRD format and see if it will be synchronized to Apache APISIX.

Enjoy! If you have any questions, please report to the [issue](https://github.com/apache/apisix-ingress-controller/issues).
