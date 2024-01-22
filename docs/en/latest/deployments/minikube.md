---
title: minikube
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - minikube
description: Guide to install APISIX ingress controller on minikube.
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

This guide explains how you can install APISIX ingress on [minikube](https://minikube.sigs.k8s.io/).

## Prerequisites

* Install [Minikube](https://minikube.sigs.k8s.io/docs/start/).
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
  --set service.type=NodePort \
  --set ingress-controller.enabled=true \
  --create-namespace \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix \
  --set ingress-controller.config.apisix.adminAPIVersion=$ADMIN_API_VERSION
kubectl get service --namespace ingress-apisix
```

:::tip

APISIX Ingress also supports (beta) the new [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/).

If the Gateway API CRDs are not installed in your cluster by default, you can install it by running:

```shell
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v0.5.0/standard-install.yaml
```

You should also enable APISIX Ingress controller to work with the Gateway API. You can do this by adding the flag `--set ingress-controller.config.kubernetes.enableGatewayAPI=true` while installing through Helm.

See [this tutorial](https://apisix.apache.org/docs/ingress-controller/tutorials/configure-ingress-with-gateway-api) for more info.

:::

This will create the five resources mentioned below:

* `apisix-gateway`: dataplane the process the traffic.
* `apisix-admin`: control plane that processes all configuration changes.
* `apisix-ingress-controller`: ingress controller which exposes APISIX.
* `apisix-etcd` and `apisix-etcd-headless`: stores configuration and handles internal communication.

You should now be able to use APISIX ingress controller. You can try running this [minimal example](../tutorials/proxy-the-httpbin-service.md) to see if everything is working perfectly.
