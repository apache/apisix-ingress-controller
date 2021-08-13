---
title: Install Ingress APISIX on K3S and Rancher RKE
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

This document explains how to install Ingress APISIX on [k3S](https://k3s.io/) and [Rancher RKE](https://rancher.com/products/rke/).

K3S is a certified Kubernetes distribution built for IoT and Edge computing, whilst [Apache APISIX](https://apisix.apache.org) is also good at IoT (See [MQTT plugin](https://github.com/apache/apisix/blob/master/docs/en/latest/plugins/mqtt-proxy.md)) and runs well on ARM architecture.
It's a good choice to use Ingress APISIX as the north-south API gateway in K3S.

## Prerequisites

* Install [K3S](https://rancher.com/docs/k3s/latest/en/installation/) or [Rancher RKE](https://rancher.com/docs/rke/latest/en/installation/).
* Install [Helm](https://helm.sh/).
* Clone [Apache APISIX Charts](https://github.com/apache/apisix-helm-chart).
* Make sure your target namespace exists, kubectl operations through this document will be executed in namespace `ingress-apisix`.

## Install APISIX and apisix-ingress-controller

As the data plane of apisix-ingress-controller, [Apache APISIX](http://apisix.apache.org/) can be deployed at the same time using Helm chart.

```shell
cd /path/to/apisix-helm-chart
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
kubectl create ns ingress-apisix
helm install apisix charts/apisix \
  --set gateway.type=NodePort \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix \
  --kubeconfig /etc/rancher/k3s/k3s.yaml
kubectl get service --namespace ingress-apisix
```

*If you are using K3S, the default kubeconfig file is in /etc/rancher/k3s and root permission may required.*

Five Service resources were created.

* `apisix-gateway`, which processes the real traffic;
* `apisix-admin`, which acts as the control plane to process all the configuration changes.
* `apisix-ingress-controller`, which exposes apisix-ingress-controller's metrics.
* `apisix-etcd` and `apisix-etcd-headless` for etcd service and internal communication.

The gateway service type is set to `NodePort`, so that clients can access Apache APISIX through the Node IPs and the assigned port.
If you are using K3S and you want to expose a `LoadBalancer` service, try to use [Klipper](https://github.com/k3s-io/klipper-lb).

Now try to create some [resources](https://github.com/apache/apisix-ingress-controller/tree/master/docs/en/latest/concepts) to verify the running status. As a minimalist example, see [proxy-the-httpbin-service](../practices/proxy-the-httpbin-service.md) to learn how to apply resources to drive the apisix-ingress-controller.
