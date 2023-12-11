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

APISIX ingress controller is a [Kubernetes ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) using [Apache APISIX](https://apisix.apache.org) as the high performance reverse proxy.

APISIX ingress controller can be configured using the native Kubernetes Ingress or Gateway API as well as with the declarative and easy to use custom resources provided by APISIX. The APISIX ingress controller converts these resources to APISIX configuration.

The examples below show how these differ. Both the examples configure a Route in APISIX that routes to an httpbin service as the Upstream.

<Tabs
groupId="resources"
defaultValue="apisix"
values={[
{label: 'APISIX Ingress CRD', value: 'apisix'},
{label: 'Kubernetes Ingress API', value: 'ingress'},
{label: 'Kubernetes Gateway API', value: 'gateway'},
]}>

<TabItem value="apisix">

```yaml title="httpbin-route.yaml"
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
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

<TabItem value="gateway">

```yaml title="httpbin-route.yaml"
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: HTTPRoute
metadata:
  name: httpbin-route
spec:
  hostnames:
  - local.httpbin.org
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: httpbin
      port: 80
```

</TabItem>
</Tabs>

APISIX ingress controller defines the CRDs [ApisixRoute](./concepts/apisix_route.md), [ApisixUpstream](./concepts/apisix_upstream.md), [ApisixTls](concepts/apisix_tls.md), and [ApisixClusterConfig](./concepts/apisix_cluster_config.md), [ApisixConsumer](./references/v2/#apisix.apache.org/v2.ApisixConsumer), [ApisixPluginConfig](./references/v2/#apisix.apache.org/v2.ApisixPluginConfig).

APISIX also supports [service discovery](https://apisix.apache.org/docs/apisix/next/discovery/kubernetes/) through [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/) abstraction.

![scene](../../assets/images/scene.png)

See [Design](./design.md) to learn more about how APISIX ingress controller works under the hood.

## Features

To summarize, APISIX ingress controller has the following features:

- Declarative configuration with CRDs.
- Fully dynamic configuration.
- Supports native Kubernetes Ingress resource (both v1 and v1beta1).
- Supports service discovery through Kubernetes Service.
- Out-of-the-box node health check support.
- Supports load balancing based on pods (Upstream nodes).
- Rich [Plugins](https://apisix.apache.org/docs/apisix/next/plugins/batch-requests/) with [custom Plugin](https://apisix.apache.org/docs/apisix/next/plugin-develop/) support.

## Get involved

You can contribute to the development of APISIX ingress controller. See [Development guide](./contribute.md) for instructions on setting up the project locally.

See the [Contribute to APISIX](https://apisix.apache.org/docs/general/contributor-guide/) section for details on the contributing flow.

## Compatibility with APISIX

The table below shows the compatibility between APISIX ingress controller and the APISIX proxy.

| APISIX ingress controller | Supported APISIX versions | Recommended APISIX version |
| ------------------------- | ------------------------- | -------------------------- |
| `master`                  | `>= 2.15`, `>=3.0`        | `3.1`                      |
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
