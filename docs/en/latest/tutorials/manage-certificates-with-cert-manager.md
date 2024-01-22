---
title: Manage Certificates With Cert Manager
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

This tutorial will detail how to manage secrets of ApisixTls using cert-manager.

## Prerequisites

* Prepare an available Kubernetes cluster in your workstation, we recommend you to use [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/) to create a local Kubernetes cluster.
* Install Apache APISIX in Kubernetes by [Helm Chart](https://github.com/apache/apisix-helm-chart).
* Install [apisix-ingress-controller](https://github.com/apache/apisix-ingress-controller/blob/master/install.md).
* Install [cert-manager](https://cert-manager.io/docs/installation/#default-static-install).

In this guide, we assume that your APISIX is installed with `ssl` enabled, which is not enabled by default in the Helm Chart. To enable it, you need to set `apisix.ssl.enabled=true` during installation.

For example, you could install APISIX and APISIX ingress controller by running:

```bash
#  We use Apisix 3.0 in this example. If you're using Apisix v2.x, please set to v2
ADMIN_API_VERSION=v3
helm install apisix apisix/apisix \
  --set service.type=NodePort \
  --set ingress-controller.enabled=true \
  --set apisix.ssl.enabled=true \
  --set ingress-controller.config.apisix.serviceNamespace=default \
  --set ingress-controller.config.apisix.adminAPIVersion=$ADMIN_API_VERSION
```

Assume that the SSL port is `9443`.

## Create Issuer

For testing purposes, we will use a simple CA issuer. All required files can be found [here](https://github.com/apache/apisix-ingress-controller/tree/master/docs/en/latest/tutorials/cert-manager).

To create a CA issuer, use the following commands:

```bash
kubectl apply -f ./cert-manager/ca.yaml
kubectl apply -f ./cert-manager/issuer.yaml
```

If the cert-manager is working correctly, we should be able to see the Ready status by running:

```bash
kubectl get issuer
```

It should output:

```text
NAME        READY   AGE
ca-issuer   True    50s
```

## Create Certificate

Before creating ApisixTls, we should create a `Certificate` resource.

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: demo-cert
spec:
  dnsNames:
    - local.httpbin.org
  issuerRef:
    kind: Issuer
    name: ca-issuer
  secretName: example-cert
  usages:
    - digital signature
    - key encipherment
  renewBefore: 0h55m0s
  duration: 1h0m0s
```

Note that we set the parameters `duration` and `renewBefore`. We want to test if the certificate rotation functionality is working well, so a shorter renewal time will help.

Like `Issuer`, we could see its readiness status by running:

```bash
kubectl get certificate
```

It should output:

```text
NAME        READY   SECRET        AGE
demo-cert   True    example-cert  50s
```

Check the secrets by running:

```bash
kubectl get secret
```

It should output:

```text
NAME          TYPE                DATA   AGE
example-cert  kubernetes.io/tls   3      2m20s
```

This means that our cert-manager is working properly.

## Create Test Service

We use [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/) as the service image.

Deploy it by running:

```bash
kubectl run httpbin --image kennethreitz/httpbin --expose --port 80
```

## Route the Service

Create an ApisixRoute to route the service:

```yaml
apiVersion: apisix.apache.org/v2
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
      backends:
        - serviceName: httpbin
          servicePort: 80
```

Run curl command in a APISIX pod to see if the routing configuration works.

```bash
kubectl -n <APISIX_NAMESPACE> exec -it <APISIX_POD_NAME> -- curl http://127.0.0.1:9080/ip -H 'Host: local.httpbin.org'
```

It should output:

```json
{
  "origin": "127.0.0.1"
}
```

## Secure the Route

Create an ApisixTls to secure the route, referring to the secret created by cert-manager:

```yaml
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: example-tls
spec:
  hosts:
    - local.httpbin.org
  secret:
    name: example-cert # the secret created by cert-manager
    namespace: default # secret namespace
```

Run curl command in a APISIX pod to see if the Ingress and TLS configuration are working.

```bash
kubectl -n <APISIX_NAMESPACE> exec -it <APISIX_POD_NAME> -- curl --resolve 'local.httpbin.org:9443:127.0.0.1' "https://local.httpbin.org:9443/ip" -k
```

It should output:

```json
{
  "origin": "127.0.0.1"
}
```

## Test Certificate Rotation

To verify certificate rotation, we can add a verbose parameter `-v` to curl command:

```bash
kubectl -n <APISIX_NAMESPACE> exec -it <APISIX_POD_NAME> -- curl --resolve 'local.httpbin.org:9443:127.0.0.1' "https://local.httpbin.org:9443/ip" -k -v
```

The verbose option will show us the handshake log, which also contains the certificate information.

Example output:

```text
* Added local.httpbin.org:9443:127.0.0.1 to DNS cache
* Hostname local.httpbin.org was found in DNS cache
*   Trying 127.0.0.1:9443...
* Connected to local.httpbin.org (127.0.0.1) port 9443 (#0)
...
...
* Server certificate:
*  subject: [NONE]
*  start date: Sep 16 00:14:55 2021 GMT
*  expire date: Sep 16 01:14:55 2021 GMT
*  issuer: C=CN; ST=Zhejiang; L=Hangzhou; O=APISIX-Test-CA_; OU=APISIX_CA_ROOT_; CN=APISIX.ROOT_; emailAddress=test@test.com
```

We could see the start date and expiration date of the server certificate.

Since the `Certificate` we defined requires the cert-manager to renew the cert every 5 minutes, we should be able to see the changes to the server certificate after 5 minutes.

```text
* Added local.httpbin.org:9443:127.0.0.1 to DNS cache
* Hostname local.httpbin.org was found in DNS cache
*   Trying 127.0.0.1:9443...
* Connected to local.httpbin.org (127.0.0.1) port 9443 (#0)
...
...
* Server certificate:
*  subject: [NONE]
*  start date: Sep 16 00:19:55 2021 GMT
*  expire date: Sep 16 01:19:55 2021 GMT
*  issuer: C=CN; ST=Zhejiang; L=Hangzhou; O=APISIX-Test-CA_; OU=APISIX_CA_ROOT_; CN=APISIX.ROOT_; emailAddress=test@test.com
```

The certificate was rotated as expected.
