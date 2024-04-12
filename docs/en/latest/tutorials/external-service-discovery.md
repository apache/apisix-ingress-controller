---
title: Using External Services Discovery In ApisixUpstream
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

In this tutorial, we will introduce how to configure external services discovery in the ApisixUpstream resources.

APISIX already supports various service discovery components, such as DNS, consul, nacos, etc.
Please see [Integration service discovery registry](https://apisix.apache.org/docs/apisix/discovery/) for details.

## Prerequisites

- An available Kubernetes cluster
- An available APISIX and APISIX Ingress Controller installation

We assume that your APISIX is installed in the `apisix` namespace.

## Introduction

Integration of APISIX Ingress with service discovery components is configured through the ApisixUpstream resource.
In this case, we don't configure the `backends` field in the ApisixRoute resource.
Instead, we will use the `upstreams` field to refer to an ApisixUpstream resources with the `discovery` field configured.

For example:

```yaml
# httpbin-route.yaml
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - local.httpbin.org
      paths:
      - /*
    # backends:  # We won't use the `backends` field
    #    - serviceName: httpbin
    #      servicePort: 80
    upstreams:
    - name: httpbin-upstream
```

This configuration tells the ingress controller not to resolve upstream hosts through the K8s services, but to use the configuration defined in the referenced ApisixUpstream.
The referenced ApisixUpstream *MUST* have `discovery` field configured. For example:

```yaml
# httpbin-upstream.yaml
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-upstream
spec:
  discovery:
    type: dns
    serviceName: httpbin.default.svc.cluster.local
```

In this yaml example, we configured `httpbin.default.svc.cluster.local` as the backend.
The type of service discovery needs to be pre-configured in APISIX. For example:

```yaml
discovery:
  dns:
    servers:
      - "10.96.0.10:53" # default kube-dns cluster IP.
```

After applying the above configuration, we can try to access `httpbin.default.svc.cluster.local` directly through APISIX.

:::note
The above discovery configuration needs to be configured at the time of installation and cannot be edited later. For example, if you're installing via helm chart, make sure that you use the below configuration to override default helm values.

```yaml
apisix:
  discovery:
    enabled: true
    registry:
      dns:
        servers:
          - "172.17.0.11:53" # replace with your server addresses
```

:::
