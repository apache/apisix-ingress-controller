---
title: How to proxy the gRPC service
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

In this practice, we will introduce how to proxy the gRPC service.

## Prerequisites

* Prepare an available Kubernetes cluster in your workstation, we recommend you to use [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/) to create a local Kubernetes cluster.
* Install [Apache APISIX](https://github.com/apache/apisix) in Kubernetes by [Helm Chart](https://github.com/apache/apisix-helm-chart).
* Install [apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller/blob/master/install.md).

Please note that in this practice, all components will be installed in the `ingress-apisix` namespace. If your Kubernetes cluster does not have such namespace, please create it first.

```bash
kubectl create ns ingress-apisix
```

You could install APISIX and APISIX ingress controller by running:

```bash
#  We use Apisix 3.0 in this example. If you're using Apisix v2.x, please set to v2
ADMIN_API_VERSION=v3
helm install apisix apisix/apisix -n ingress-apisix \
  --set service.type=NodePort \
  --set ingress-controller.enabled=true \
  --set apisix.ssl.enabled=true \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix \
  --set ingress-controller.config.apisix.adminAPIVersion=$ADMIN_API_VERSION
```

Check that all related components have been installed successfully, including ETCD cluster / APISIX / apisix-ingress-controller.

```shell
$ kubectl get pod -n ingress-apisix
NAME                                        READY   STATUS    RESTARTS   AGE
apisix-569f94b7b6-qt5jj                     1/1     Running   0          101m
apisix-etcd-0                               1/1     Running   0          101m
apisix-etcd-1                               1/1     Running   0          101m
apisix-etcd-2                               1/1     Running   0          101m
apisix-ingress-controller-b5f5d49db-r9cxb   1/1     Running   0          101m
```

## Prepare a gRPC service

Using [yages](https://github.com/mhausenblas/yages) as the gRPC server.

Declare the deployment configuration of yapes, exposing port `9000`.

```bash
kubectl run yages -n ingress-apisix --image smirl/yages:0.1.3 --expose --port 9000
```

Use the service that includes `grpcurl` to test gRPC connectivity.

```shell
$ kubectl run -it -n ingress-apisix --rm grpcurl --restart=Never --image=fullstorydev/grpcurl:v1.8.7 --command -- \
  /bin/grpcurl -plaintext yages:9000 yages.Echo.Ping
# It should output:
{
  "text": "pong"
}
```

**If you encounter a timeout error, you can first download `quay.io/mhausenblas/gump:0.1` to the local.**

## Declare gRPC proxy configuration

### Create a route and tell APISIX proxy rules

```bash
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: grpc-proxy-route
  namespace: ingress-apisix
spec:
  http:
    - name: grpc-route
      match:
        hosts:
          - grpc-proxy
        paths:
          - "/*"
      backends:
      - serviceName: yages
        servicePort: 9000
        weight: 10
EOF
```

### Inform APISIX the yages is a gRPC server through ApisixUpstream

```bash
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: yages
  namespace: ingress-apisix
spec:
  scheme: grpc
EOF
```

### Configure certificates for gRPC

Common Name should be `grpc-proxy`, which needs to be consistent with the hosts declared in ApisixRoute.

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=grpc-proxy/O=grpc-proxy"
```

Store key and crt in secret.

```bash
kubectl create secret tls grpc-secret -n ingress-apisix --cert=tls.crt --key=tls.key
```

Inform APISIX SSL configuration through ApisixTls.

```bash
kubectl apply -f - <<EOF
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: grpc-secret
  namespace: ingress-apisix
spec:
  hosts:
    - "grpc-proxy"
  secret:
    name: grpc-secret
    namespace: ingress-apisix
EOF
```

### Test

OK, the configuration is complete, continue to verify through `grpcurl`, this time we visit the `yages` service through the Apache APISIX proxy.

Check the APISIX DP (Data Plane) service, which is apisix-gateway in this example.

```bash
kubectl get svc -n ingress-apisix
NAME                        TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)                      AGE
apisix-admin                ClusterIP   10.96.49.113   <none>        9180/TCP                     98m
apisix-etcd                 ClusterIP   10.96.81.162   <none>        2379/TCP,2380/TCP            98m
apisix-etcd-headless        ClusterIP   None           <none>        2379/TCP,2380/TCP            98m
apisix-gateway              NodePort    10.96.74.145   <none>        80:32600/TCP,443:32103/TCP   98m
apisix-ingress-controller   ClusterIP   10.96.78.108   <none>        80/TCP                       98m
yages                       ClusterIP   10.96.37.236   <none>        9000/TCP                     94m
```

```shell
$ kubectl run -it -n ingress-apisix --rm grpcurl --restart=Never --image=fullstorydev/grpcurl:v1.8.7 --command -- \
  /bin/grpcurl -insecure -servername grpc-proxy apisix-gateway:443 yages.Echo.Ping
# It should output:
{
  "text": "pong"
}
```

APISIX proxy gRPC server succeeded.

### Cleanup

```bash
kubectl delete ns ingress-apisix
```
