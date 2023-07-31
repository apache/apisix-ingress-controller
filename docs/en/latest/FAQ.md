---
title: FAQ
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - FAQ
description: Answers to frequently asked questions about APISIX Ingress.
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

## How do I bind a Service with an Upstream?

All resources are uniquely identified by the namespace/name/port combination. If this combination is the same, the Service and the Upstream will be binded.

## While modifying a CRD, how does the binded resources perceive it?

This is a cascading update problem. See [Design](./design.md) for more details.

## Can I use both CRDs and the Admin API together to configure Routes?

No. CRDs are declarative and when applied they are translated to APISIX configuration. Configuring APISIX through Admin API would not change the CRDs.

## Why is there an error like "list upstreams failed, err: http get failed, url: httpbin.org, err: status: 401"?

APISIX Ingress controller does not support configuring `admin_key` for APISIX. Removing `admin_key` from both your configuration file (`config.yaml` and `config-default.yaml`) when deploying APISIX will fix this issue.

<!-- ### 5. Failed to create route with `ApisixRoute`

When `apisix-ingress-controller` creates a route with CRD, it checks the `Endpoint` resources in Kubernetes (matched by namespace_name_port). If the corresponding endpoint information is not found, the route will not be created and wait for the next retry.

Tips: The failure caused by empty upstream nodes is a limitation of Apache APISIX, related [issue](https://github.com/apache/apisix/issues/3072) -->

## How does APISIX Ingress controller retry?

If an error occurs while parsing the CRD and translating the configuration to APISIX, a retry will be triggered.

Delays are used while retrying. It retries once per second at first and after five retries, it will be decreased to one retry per minute until it succeeds.

## How do I update the CRDs when updating APISIX Ingress controller?

The Helm chart will skip applying these CRDs if they already exist.

In such cases, you can apply the CRDs manually:

```shell
kubectl apply -k samples/deploy/crd/
```

:::note

With Helm 3, old CRD-install hooks were replaced by a simpler system. You can now create a special directory called `crds` in your charts for holding CRDs.

These CRDs are not templated but will be installed by default when running `helm install`. If the CRD already exists, it will be skipped with a warning. You can skip the CRD installation step by passing the `--skip-crds` flag.

:::

## Why is there an error like "no matches for kind "ApisixRoute" in version "apisix.apache.org/v2beta3"" when I try to create a Route?

Make sure that you have the correct version of the CRDs installed in your cluster (see [updating CRDs](#how-do-i-update-the-crds-when-updating-apisix-ingress-controller)). `ApisixRoute` has two versions: `v2beta3` and `v2`.

Also check your `ApisixRoute` definition for the correct version by running:

```shell
kubectl get crd apisixroutes.apisix.apache.org -o jsonpath='{ .spec.versions[*].name }' -A
```

## How do I modify the Admin API key in APISIX Ingress?

You can change the Admin API key in two ways:

1. Modify the key in both [apisix/values.yaml](https://github.com/apache/apisix-helm-chart/blob/57cdbe461765cd49af2195cc6a1976cc55262e9b/charts/apisix/values.yaml#L181) and [apisix/apisix-ingress-controller/values.yaml](https://github.com/apache/apisix-helm-chart/blob/57cdbe461765cd49af2195cc6a1976cc55262e9b/charts/apisix-ingress-controller/values.yaml#L128) files.
2. You can also set this imperatively by passing the flag `--set ingress-controller.config.apisix.adminKey=<new key> --set admin.credentials.admin=<new key>` to the `helm install` command.

## Why does my Ingress resource not have an address?

1. **Using the External address of LoadBalancer service.**

You will need to get the apisix-gateway service an external IP assigned for it to reflect on the Ingress's status.

* While installing APISIX helm chart make sure to override gateway type with `--set gateway.type=LoadBalancer`.

* Also make sure to pass ingressPublishService while installing Ingress controller with `--set ingress-controller.config.ingressPublishService=<namespace/service-name>`. If namespace is not specified then `default` namespace will be chosen.

Note: External IP is allocated either by cloud provider or some other controller like metallb(if you're using kind or minikube) so if you're deploying Ingress controller on minikube or kind then make sure to install and configure something like metallb with an address pool which can allocate external IP for service of type LoadBalancer.

2. **Using the address given explicitly by `ingress_status_address`**
In case the gateway `service` is not of type LoadBalancer, this field should be configured in the config-default.yaml used by the Ingress controller. This field takes precedence over ExternalIP of the service so if `ingress_status_address` array has non zero elements(addresses) present then it will used over the ExternalIP of the gateway service.
