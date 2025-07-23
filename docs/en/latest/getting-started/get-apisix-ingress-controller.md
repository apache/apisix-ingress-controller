---
title: Get APISIX and APISIX Ingress Controller
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
description: Learn how to quickly install and set up Apache APISIX Ingres Controller.
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

This tutorial series walks you through how to quickly get started with APISIX on a [kind](https://kind.sigs.k8s.io) Kubernetes cluster and use the APISIX Ingress Controller to manage resources.

## Prerequisites

* Install [Docker](https://docs.docker.com/get-docker/) as a dependency of [kind](https://kind.sigs.k8s.io).
* Install [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) to start a local Kubernetes cluster, or use any existing Kubernetes cluster (version 1.26+).
* Install [Helm](https://helm.sh/docs/intro/install/) (version 3.8+).
* Install [kubectl](https://kubernetes.io/docs/tasks/tools/) to run commands against Kubernetes clusters.

## Create a Cluster and Configure Namespace

In this section, you will be creating a kind cluster and configuring the namespace. Skip to [the next section](#install-apisix-and-apisix-ingress-controller-standalone-api-driven-mode) if you already have an existing cluster and a corresponding namspace.

Ensure you have Docker running and start a kind cluster:

```shell
kind create cluster
```

Create a new namespace `ingress-apisix`:

```shell
kubectl create namespace ingress-apisix
```

Set the namespace to `ingress-apisix` to avoid specifying it explicitly in each subsequent command:

```shell
kubectl config set-context --current --namespace=ingress-apisix
```

## Install APISIX and APISIX Ingress Controller (Standalone API-driven mode)

Install the Gateway API CRDs, [APISIX Standalone API-driven mode](https://apisix.apache.org/docs/apisix/deployment-modes/#api-driven-experimental), and APISIX Ingress Controller:

```bash
helm repo add apisix https://charts.apiseven.com
helm repo update

helm install apisix \
  --namespace ingress-apisix \
  --create-namespace \
  --set apisix.deployment.role=traditional \
  --set apisix.deployment.role_traditional.config_provider=yaml \
  --set etcd.enabled=false \
  --set ingress-controller.enabled=true \
  --set ingress-controller.config.provider.type=apisix-standalone \
  --set ingress-controller.apisix.adminService.namespace=ingress-apisix \
  --set ingress-controller.gatewayProxy.createDefault=true \
  apisix/apisix
```

More details on the installation can be found in the [Installation Guide](../install.md).

## Verify Installation

Check the statuses of resources in the current namespace:

```shell
kubectl get all
```

You should wait for all pods to be running before proceeding:

```text
NAME                                             READY   STATUS    RESTARTS   AGE
pod/apisix-7c5fb8d546-gtfqn                      1/1     Running   0          113s
pod/apisix-ingress-controller-56c46fd54f-f8fxt   1/1     Running   0          113s

NAME                             TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
service/apisix-admin             ClusterIP   10.96.174.119   <none>        9180/TCP       113s
service/apisix-gateway           NodePort    10.96.231.33    <none>        80:31321/TCP   113s
service/apisix-metrics-service   ClusterIP   10.96.77.248    <none>        8443/TCP       113s

NAME                                        READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/apisix                      1/1     1            1           113s
deployment.apps/apisix-ingress-controller   1/1     1            1           113s

NAME                                                   DESIRED   CURRENT   READY   AGE
replicaset.apps/apisix-7c5fb8d546                      1         1         1       113s
replicaset.apps/apisix-ingress-controller-56c46fd54f   1         1         1       113s
```

To verify the installed APISIX version, map port `80` of the `apisix-gateway` service to port `8080` on the local machine:

```shell
kubectl port-forward svc/apisix-gateway 9080:80 &
```

Send a request to the gateway:

```shell
curl -sI "http://127.0.0.1:9080" | grep Server
```

If everything is ok, you should see the APISIX version:

```text
Server: APISIX/x.x.x
```
