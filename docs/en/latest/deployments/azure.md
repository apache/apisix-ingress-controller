---
title: AKS (Azure)
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - Microsoft Azure
description: Guide to install APISIX ingress controller on Azure Kubernetes Service (AKS).
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

This document explains how you can install APISIX ingress on [Azure AKS](https://docs.microsoft.com/en-us/azure/aks/intro-kubernetes).

## Prerequisites

Setting up APISIX ingress on AKS requires the following:

* [Create an AKS cluster](https://docs.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-portal?tabs=azure-cli).
* Install [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/#:~:text=The%20Azure%20command%2Dline%20interface,with%20an%20emphasis%20on%20automation.) and update your kube config file by running:

    ```shell
    az aks get-credentials
    ```

* Install [Helm](https://helm.sh/).

## Install APISIX and ingress controller

The script below installs APISIX and the ingress controller:

```shell
helm repo add apisix https://charts.apiseven.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
#  We use Apisix 3.0 in this example. If you're using Apisix v2.x, please set to v2
ADMIN_API_VERSION=v3
helm install apisix apisix/apisix \
  --set gateway.type=LoadBalancer \
  --set ingress-controller.enabled=true \
  --create-namespace \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix \
  --set ingress-controller.config.apisix.adminAPIVersion=$ADMIN_API_VERSION
kubectl get service --namespace ingress-apisix
```

:::note

By default, APISIX ingress controller will watch the apiVersion of `networking.k8s.io/v1`.

If the target Kubernetes version is under `v1.19`, add the flag `--set ingress-controller.config.kubernetes.ingressVersion=networking/v1beta1`.

Else, if your Kubernetes cluster version is under `v1.16`, set the flag `--set ingress-controller.config.kubernetes.ingressVersion=extensions/v1beta1`.

:::

This will create the five resources mentioned below:

* `apisix-gateway`: dataplane the process the traffic.
* `apisix-admin`: control plane that processes all configuration changes.
* `apisix-ingress-controller`: ingress controller which exposes APISIX.
* `apisix-etcd` and `apisix-etcd-headless`: stores configuration and handles internal communication.

The gateway service type will be set to `LoadBalancer`. You can find the load balancer IP address by running:

```shell
kubectl get service apisix-gateway --namespace ingress-apisix -o jsonpath='{.status.loadBalancer.ingress[].ip}'
```

You should now be able to use APISIX ingress controller. You can try running this [minimal example](../tutorials/proxy-the-httpbin-service.md) to see if everything is working perfectly.

## Next steps

### Enable SSL

SSL is disabled by default. You can enable it by adding the flag `--set gateway.tls.enabled=true`.

### Change default keys

It is recommended to change the default keys for security:

```shell
--set ingress-controller.config.apisix.adminKey=ADMIN_KEY_GENERATED_BY_YOURSELF
```

```shell
--set admin.credentials.admin=ADMIN_KEY_GENERATED_BY_YOURSELF
```

```shell
--set admin.credentials.viewer=VIEWER_KEY_GENERATED_BY_YOURSELF
```

:::note

The `ingress-controller.config.apisix.adminKey` and `admin.credentials.admin` must be the same. It is better if these are not same as `admin.credentials.viewer`.

:::
