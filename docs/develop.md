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

## Configuration

### Configure the `kube config` file locally to facilitate local debugging

1. Start minikube.

2. Location: ~/.kube/config

3. Copy the config file to the local development environment, and set the path of configure file to the k8sAuth item ($GOPATH/src/github.com/apache/apisix-ingress-controller/conf/conf.json#conf.k8sAuth).

### Configure Apache APISIX service address

Configure the service address of Apache APISIX to conf/apisix/base_url ($GOPATH/src/github.com/apache/apisix-ingress-controller/conf/conf.json).

## Start ingress-controller locally

* Create CRDs

```yaml
kubectl apply -f - <<EOF
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apisixroutes.apisix.apache.org
spec:
  group: apisix.apache.org
  versions:
    - name: v1
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
  name: apisixservices.apisix.apache.org
spec:
  group: apisix.apache.org
  versions:
    - name: v1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: apisixservices
    singular: apisixservice
    kind: ApisixService
    shortNames:
    - as

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

EOF
```

* Run apisix-ingress-controller

```shell
# go run main.go -logtostderr -v=5
```

Tips: The program may print some error logs, indicating that the resource cannot be found. Continue with the following steps to define the route through CRDs.

### Define ApisixRoute

Take the back-end service `httpserver` as an example (you can choose any upstream service for test).

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
          serviceName: httpserver
          servicePort: 8080
        path: /hello*
EOF
```

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
          serviceName: httpserver
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
  name: httpserver      # default/httpserver
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
