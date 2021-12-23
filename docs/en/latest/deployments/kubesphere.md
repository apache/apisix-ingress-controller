---
title: Install Ingress APISIX on KubeSphere
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

This document explains how to install Ingress APISIX on [KubeSphere](https://kubesphere.io/).

KubeSphere is a distributed operating system managing cloud native applications with Kubernetes as its kernel, and provides plug-and-play architecture for the seamless integration of third-party applications to boost its ecosystem.

## Prerequisites

* Install [KubeSphere](https://kubesphere.io/docs/quick-start/), you can choose [All-in-one Installation on Linux](https://kubesphere.io/docs/quick-start/all-in-one-on-linux/) or [Minimal KubeSphere on Kubernetes](https://kubesphere.io/docs/quick-start/minimal-kubesphere-on-k8s/).
* Install [Helm](https://helm.sh/).
* Clone [Apache APISIX Charts](https://github.com/apache/apisix-helm-chart).
* Make sure your target namespace exists, kubectl operations of this document will be executed in namespace `ingress-apisix`.

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
  --namespace ingress-apisix
kubectl get service --namespace ingress-apisix
```

Five Service resources were created.

* `apisix-gateway`, which processes the real traffic;
* `apisix-admin`, which acts as the control plane to process all the configuration changes.
* `apisix-ingress-controller`, which exposes apisix-ingress-controller's metrics.
* `apisix-etcd` and `apisix-etcd-headless` for etcd service and internal communication.

The gateway service type is set to `NodePort`, so that clients can access Apache APISIX through the Node IPs and the assigned port.
If you want to expose a `LoadBalancer` service, try to use [Porter](https://github.com/kubesphere/porter).

Now try to create some [resources](https://github.com/apache/apisix-ingress-controller/tree/master/docs/en/latest/concepts) to verify the running status. As a minimalist example, see [proxy-the-httpbin-service](../practices/proxy-the-httpbin-service.md) to learn how to apply resources to drive the apisix-ingress-controller.
