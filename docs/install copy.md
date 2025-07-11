---
title: Install with Helm
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - kind
description: Guide to install APISIX ingress controller on kind.
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

Helm is a package manager for Kubernetes that automates the release and management of software on Kubernetes.

This document guides you through installing the APISIX ingress controller using Helm.

## Prerequisites

Before installing APISIX ingress controller, ensure you have:

1. A working Kubernetes cluster (version 1.26+)
   - Production: TKE, EKS, AKS, or other cloud-managed clusters
   - Development: minikube, kind, or k3s
2. [kubectl](https://kubernetes.io/docs/tasks/tools/) installed and configured to access your cluster
3. [Helm](https://helm.sh/) (version 3.8+) installed

## Install APISIX Ingress Controller

The APISIX ingress controller can be installed using the Helm chart provided by the Apache APISIX project. The following steps will guide you through the installation process.

```shell
helm repo add apisix https://charts.apiseven.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Set the access address and adminkey for apisix
helm install apisix-ingress-controller \
  --create-namespace \
  -n ingress-apisix \
  --set gatewayProxy.createDefault=true \
  --set gatewayProxy.provider.controlPlane.auth.adminKey.value=edd1c9f034335f136f87ad84b625c8f1 \
  --set apisix.adminService.namespace=apisix-ingress \
  --set apisix.adminService.name=apisix-admin \
  --set apisix.adminService.port=9180 \
  apisix/apisix-ingress-controller
```

## Install APISIX And Ingress Controller

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

This will create the five resources mentioned below:

* `apisix-gateway`: dataplane that processes the traffic.
* `apisix-admin`: control plane that processes all configuration changes.
* `apisix-ingress-controller`: ingress controller which exposes APISIX.
* `apisix-etcd` and `apisix-etcd-headless`: stores configuration and handles internal communication.

You should now be able to use APISIX ingress controller. You can try running this [minimal example](../tutorials/proxy-the-httpbin-service.md) to see if everything is working perfectly.

## Install APISIX And Ingress Controller (Standalone Mode)

### Prerequisites

* APISIX version 3.13+ is required.

```shell
helm repo add apisix https://charts.apiseven.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
helm install apisix apisix/apisix \
  --set service.type=NodePort \
  --set ingress-controller.enabled=true \
  --create-namespace \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix \
  --set ingress-controller.config.provider.type=apisix-standalone
kubectl get service --namespace ingress-apisix
```
