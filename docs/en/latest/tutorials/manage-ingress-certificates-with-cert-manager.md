---
title: Manage Ingress Certificates With Cert Manager
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

This tutorial will detail how to secure ingress using cert-manager.

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

## Create Test Certificate

To ensure that cert-manager is working properly, we can create a test `Certificate` resource.

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: demo-cert
spec:
  dnsNames:
    - example.com
  issuerRef:
    kind: Issuer
    name: ca-issuer
  secretName: example-cert
  usages:
    - digital signature
    - key encipherment
```

Like `Issuer`, we could see its readiness status by running:

```bash
kubectl get certificate
```

It should output:

```text
NAME        READY   SECRET        AGE
demo-cert   True    example.com   50s
```

Check the secrets by running:

```bash
kubectl get secret
```

It should output:

```text
NAME          TYPE                DATA   AGE
example.com   kubernetes.io/tls   3      2m20s
```

This means that our cert-manager is working properly.

## Create Test Service

We use [kennethreitz/httpbin](https://hub.docker.com/r/kennethreitz/httpbin/) as the service image.

Deploy it by running:

```bash
kubectl run httpbin --image kennethreitz/httpbin --expose --port 80
```

## Secure Ingress

The cert-manager supports several ways to [secure ingress](https://cert-manager.io/docs/usage/ingress/). The easiest way is to use annotations.

By using annotations, we don't need to manage `Certificate` CRD manually.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpserver-ingress
  annotations:
    # add an annotation indicating the issuer to use.
    cert-manager.io/issuer: "ca-issuer"
spec:
  # apisix-ingress-controller is only interested in Ingress
  # resources with the matched ingressClass name, in our case,
  # it's apisix.
  ingressClassName: apisix
  tls:
    - hosts:
        - local.httpbin.org # placing a host in the TLS config will determine what ends up in the cert's subjectAltNames
      secretName: ingress-cert-manager-tls # cert-manager will store the created certificate in this secret.
  rules:
  - host: local.httpbin.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin
            port:
              number: 80
```

The annotation `cert-manager.io/issuer` tells cert-manager which issuer should be used. The Issuer must be in the same namespace as the Ingress resource. Please read [Securing Ingress Resources](https://cert-manager.io/docs/usage/ingress/) for more details.

We should now be able to see the certificate and secret resource created by cert-manager:

```bash
kubectl get certificate
kubectl get secret
```

It should output:

```text
NAME                       READY   SECRET                     AGE
ingress-cert-manager-tls   True    ingress-cert-manager-tls   2m

NAME                       TYPE                DATA   AGE
ingress-cert-manager-tls   kubernetes.io/tls   3      3m
```

## Test

Run curl command in a APISIX pod to see if the Ingress and TLS configuration works.

```bash
kubectl -n <APISIX_NAMESPACE> exec -it <APISIX_POD_NAME> -- curl --resolve 'local.httpbin.org:9443:127.0.0.1' "https://local.httpbin.org:9443/ip" -k
```

It should output:

```json
{
  "origin": "127.0.0.1"
}
```
