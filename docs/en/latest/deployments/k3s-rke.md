---
title: K3s and RKE (Rancher)
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - K3s
  - Rancher RKE
description: Guide to install APISIX ingress controller on K3s and Rancher Kubernetes Engine(RKE).
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

This document explains how you can install APISIX ingress on [k3S](https://k3s.io/) and [Rancher RKE](https://rancher.com/products/rke/).

:::tip

K3s is built for IoT and edge computing applications. Apache APISIX also supports an MQTT Plugin and runs well on ARM processors. APISIX ingress is therefore a good choice to handle North-South traffic in K3s. 

:::

## Prerequisites

* Install [K3S](https://rancher.com/docs/k3s/latest/en/installation/) or [Rancher RKE](https://rancher.com/docs/rke/latest/en/installation/).
* Install [Helm](https://helm.sh/).

## Install APISIX and ingress controller

The script below installs APISIX and the ingress controller:

```shell
helm repo add apisix https://charts.apiseven.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
kubectl create ns ingress-apisix
helm install apisix apisix/apisix \
  --set gateway.type=NodePort \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix \
  --kubeconfig /etc/rancher/k3s/k3s.yaml
kubectl get service --namespace ingress-apisix
```

:::info IMPORTANT

If you are using K3s, the default kube config file is located in `/etc/rancher/k3s/` and you make require root permission.

:::

This will create the five resources mentioned below:

* `apisix-gateway`: dataplane the process the traffic.
* `apisix-admin`: control plane that processes all configuration changes.
* `apisix-ingress-controller`: ingress controller which exposes APISIX.
* `apisix-etcd` and `apisix-etcd-headless`: stores configuration and handles internal communication.

The gateway service type is set to `NodePort`. Clients can access APISIX through the Node IPs and the assigned port. To use a service of type `LoadBalancer` with K3s, use a bare-metal load balancer implementation like [Klipper](https://github.com/k3s-io/klipper-lb).

You should now be able to use APISIX ingress controller. You can try running this [minimal example](../tutorials/proxy-the-httpbin-service.md) to see if everything is working perfectly.
