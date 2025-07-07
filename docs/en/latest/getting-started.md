---
title: Getting started
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
description: Guide to get started with Apache APISIX ingress controller.
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

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

APISIX Ingress Controller is a [Kubernetes ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) using [Apache APISIX](https://apisix.apache.org) as the high performance reverse proxy.

APISIX Ingress Controller can be configured using the native Kubernetes Ingress or Gateway API, as well as with APISIX’s own declarative and easy-to-use custom resources. The controller translates these resources into APISIX configuration.

## Quick Start

Get started with APISIX Ingress Controller in a few simple steps.

### Prerequisites

Before installing APISIX Ingress Controller, ensure you have:

1. A working Kubernetes cluster (version 1.26+)
2. [Helm](https://helm.sh/) (version 3.8+) installed

### Install APISIX and APISIX Ingress Controller

Install the Gateway API CRDs, APISIX, and APISIX Ingress Controller using the following commands:

```bash
helm repo add apisix https://charts.apiseven.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

helm install apisix \
  --namespace ingress-apisix \
  --create-namespace \
  --set ingress-controller.enabled=true \
  --set ingress-controller.apisix.adminService.namespace=ingress-apisix \
  --set ingress-controller.gatewayProxy.createDefault=true \
  apisix/apisix
```

### Set Up a Sample Upstream

Install the httpbin example application to test the configuration:

```bash
https://raw.githubusercontent.com/apache/apisix-ingress-controller/refs/heads/v2.0.0/examples/httpbin/deployment.yaml
```

### Configure a Route

Install an ApisixRoute or Ingress resource to route traffic to httpbin:

> The examples below show how these differ. Both the examples configure a Route in APISIX that routes to an httpbin service as the Upstream.

<Tabs
groupId="resources"
defaultValue="apisix"
values={[
{label: 'APISIX Ingress CRD', value: 'apisix'},
{label: 'Kubernetes Ingress API', value: 'ingress'},
]}>

<TabItem value="apisix">

```yaml title="httpbin-route.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  http:
    - name: route-1
      match:
        hosts:
          - local.httpbin.org
        paths:
          - /*
      backends:
        - serviceName: httpbin
          servicePort: 80
```

</TabItem>

<TabItem value="ingress">

```yaml title="httpbin-route.yaml"
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
    - host: local.httpbin.org
      http:
        paths:
          - backend:
              service:
                name: httpbin
                port:
                  number: 80
            path: /
            pathType: Prefix
```

</TabItem>
</Tabs>

:::note

More details on the installation can be found in the [Installation guide](./install.md).

:::

### Verify Route Configuration

Let's verify the configuration. In order to access APISIX locally, we can use `kubectl port-forward` command to forward traffic from the specified port at your local machine to the specified port on the specified service.

```bash
kubectl port-forward -n ingress-apisix svc/apisix-gateway 9080:80
```

Run curl command in a APISIX pod to see if the routing configuration works.

```bash
curl http://127.0.0.1:9080/headers -H 'Host: local.httpbin.org'
```

## Features

To summarize, APISIX ingress controller has the following features:

- Declarative configuration with CRDs.
- Supports native Kubernetes Ingress v1 and Gateway API.
- Supports service discovery through Kubernetes Service.
- Supports load balancing based on pods (Upstream nodes).
- Rich [Plugins](https://apisix.apache.org/docs/apisix/next/plugins/batch-requests/) with [custom Plugin](https://apisix.apache.org/docs/apisix/next/plugin-develop/) support.

## Get involved

You can contribute to the development of APISIX ingress controller. See [Development guide](./developer-guide.md) for instructions on setting up the project locally.

See the [Contribute to APISIX](https://apisix.apache.org/docs/general/contributor-guide/) section for details on the contributing flow.

## Compatibility with APISIX

The table below shows the compatibility between APISIX ingress controller and the APISIX proxy.

:::note

APISIX Ingress Controller 2.0.0+ support the [APISIX Standalone API-driven Mode](https://apisix.apache.org/docs/apisix/deployment-modes/#api-driven-experimental), but require APISIX 3.13+.

:::

| APISIX ingress controller | Supported APISIX versions | Recommended APISIX version |
| ------------------------- | ------------------------- | -------------------------- |
| `master`                  | `>=3.0`                   | `3.13`                     |
| `2.0.0`                   | `>=3.0`                   | `3.13`                     |
| `1.6.0`                   | `>= 2.15`, `>=3.0`        | `2.15`, `3.0`              |
| `1.5.0`                   | `>= 2.7`                  | `2.15`                     |
| `1.4.0`                   | `>= 2.7`                  | `2.11`                     |
| `1.3.0`                   | `>= 2.7`                  | `2.10`                     |
| `1.2.0`                   | `>= 2.7`                  | `2.8`                      |
| `1.1.0`                   | `>= 2.7`                  | `2.7`                      |
| `1.1.0`                   | `>= 2.7`                  | `2.7`                      |
| `1.0.0`                   | `>= 2.7`                  | `2.7`                      |
| `0.6`                     | `>= 2.6`                  | `2.6`                      |
| `0.5`                     | `>= 2.4`                  | `2.5`                      |
| `0.4`                     | `>= 2.4`                  |                            |
