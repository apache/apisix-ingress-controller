---
title: Install Ingress APISIX on Kind
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

This document explains how to install Ingress APISIX on [kind](https://kind.sigs.k8s.io/).

## Prerequisites

* Install [kind](https://kind.sigs.k8s.io/docs/user/quick-start/).
* Install [Helm](https://helm.sh/).
* Install [kubectl](https://kubernetes.io/docs/tasks/tools/).

If you encounter some strange problems, please consider whether it is a version compatibility issue.

The versions of kind(==v0.12.0), kubectl(==v1.23.5) and helm(==v3.8.1) used in this document are confirmed feasible.

## Create Cluster

The quickest way to get a taste is to run command as follows and then go to the next section.

```shell
kind create cluster
```

You can click this [link](https://kind.sigs.k8s.io/docs/user/ingress/#create-cluster) for more information.

## Install APISIX and apisix-ingress-controller

As the data plane of apisix-ingress-controller, [Apache APISIX](http://apisix.apache.org/) can be deployed at the same time using Helm chart.

```shell
helm repo add apisix https://charts.apiseven.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
kubectl create ns ingress-apisix
helm install apisix apisix/apisix \
  --set gateway.type=NodePort \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix
kubectl get service --namespace ingress-apisix
```

Five Service resources were created.

* `apisix-gateway`, which processes the real traffic;
* `apisix-admin`, which acts as the control plane to process all the configuration changes.
* `apisix-ingress-controller`, which exposes apisix-ingress-controller's metrics.
* `apisix-etcd` and `apisix-etcd-headless` for etcd service and internal communication.

Now try to create some [resources](https://github.com/apache/apisix-ingress-controller/tree/master/docs/en/latest/concepts) to verify the running of Ingress APISIX. As a minimalist example, see [proxy-the-httpbin-service](../practices/proxy-the-httpbin-service.md) to learn how to apply resources to drive the apisix-ingress-controller.
