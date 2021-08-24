---
title: Install Ingress APISIX on Azure AKS
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

This document explains how to install Ingress APISIX on [Azure AKS](https://docs.microsoft.com/en-us/azure/aks/intro-kubernetes#:~:text=Azure%20Kubernetes%20Service%20(AKS)%20makes,managed%20Kubernetes%20cluster%20in%20Azure.&text=The%20Kubernetes%20masters%20are%20managed,clusters%2C%20not%20for%20the%20masters.).

## Prerequisites

* Create an Kubernetes Service on Azure.
* Install [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/#:~:text=The%20Azure%20command%2Dline%20interface,with%20an%20emphasis%20on%20automation.) and download the credentials by running `az aks get-credentials`.
* Install [Helm](https://helm.sh/).
* Clone [Apache APISIX Charts](https://github.com/apache/apisix-helm-chart).
* Make sure your target namespace exists, kubectl operations thorough this document will be executed in namespace `ingress-apisix`.

## Install APISIX and apisix-ingress-controller

As the data plane of apisix-ingress-controller, [Apache APISIX](http://apisix.apache.org/) can be deployed at the same time using Helm chart.

```shell
cd /path/to/apisix-helm-chart
helm repo add apisix https://charts.apisix.com
helm repo update
kubectl create ns ingress-apisix
helm install apisix charts/apisix \
  --set gateway.type=LoadBalancer \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix
kubectl get service --namespace ingress-apisix
```

Five Service resources were created.

* `apisix-gateway`, which processes the real traffic;
* `apisix-admin`, which acts as the control plane to process all the configuration changes.
* `apisix-ingress-controller`, which exposes apisix-ingress-controller's metrics.
* `apisix-etcd` and `apisix-etcd-headless` for etcd service and internal communication.

The gateway service type is set to `LoadBalancer`, so that clients can access Apache APISIX through a load balancer IP. You can find the load balancer IP by running:

```shell
kubectl get service apisix-gateway --namespace ingress-apisix -o jsonpath='{.status.loadBalancer.ingress[].ip}'
```

Now try to create some [resources](https://github.com/apache/apisix-ingress-controller/tree/master/docs/en/latest/concepts) to verify the running status. As a minimalist example, see [proxy-the-httpbin-service](../practices/proxy-the-httpbin-service.md) to learn how to apply resources to drive the apisix-ingress-controller.
