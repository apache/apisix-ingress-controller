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
