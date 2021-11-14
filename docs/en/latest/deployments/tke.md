---
title: Install Ingress APISIX on Tencent TKE
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

This document explains how to install Ingress APISIX on [Tencent TKE](https://cloud.tencent.com/product/tke).

## Prerequisites

* Create a TKE Service on Tencent Cloud and make sure the API Server is accessible from your workspace.
* Install [Helm](https://helm.sh/).
* Download the kube config for your TKE Console.
* Make sure your target namespace exists, kubectl operations thorough this document will be executed in namespace `ingress-apisix`.

## Install APISIX and apisix-ingress-controller

As the data plane of apisix-ingress-controller, [Apache APISIX](http://apisix.apache.org/) can be deployed at the same time using Helm chart.

```shell
helm repo add apisix https://charts.apiseven.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
kubectl create ns ingress-apisix
helm install apisix apisix/apisix \
  --set gateway.type=LoadBalancer \
  --set ingress-controller.enabled=true \
  --set etcd.persistence.size="10Gi" \
  --namespace ingress-apisix
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix
kubectl get service --namespace ingress-apisix
```

Please be careful you must configure the `etcd.persistence.size` to multiples of 10Gi (it's a limitation on TKE), otherwise the [PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) creation will fail.

Five Service resources were created.

* `apisix-gateway`, which processes the real traffic;
* `apisix-admin`, which acts as the control plane to process all the configuration changes.
* `apisix-ingress-controller`, which exposes apisix-ingress-controller's metrics.
* `apisix-etcd` and `apisix-etcd-headless` for etcd service and internal communication.

Two Service resources were created, one is `apisix-gateway`, which processes the real traffic; another is `apisix-admin`, which acts as the control plane to process all the configuration changes.

The gateway service type is set to `LoadBalancer` (see [TKE Service Management](https://cloud.tencent.com/document/product/457/45487?from=10680) for more details), so that clients can access Apache APISIX through a load balancer. You can find the load balancer ip by running:

```shell
kubectl get service apisix-gateway --namespace ingress-apisix -o jsonpath='{.status.loadBalancer.ingress[].ip}'
```

Now open your [TKE console](https://console.cloud.tencent.com/tke2/overview), choosing your cluster and clicking the Workloads tag, you'll see all pods of Apache APISIX, etcd and apisix-ingress-controller are ready.

Try to create some [resources](https://github.com/apache/apisix-ingress-controller/tree/master/docs/en/latest/concepts) to verify the running status. As a minimalist example, see [proxy-the-httpbin-service](../practices/proxy-the-httpbin-service.md) to learn how to apply resources to drive the apisix-ingress-controller.
