---
title: ACK (Alibaba Cloud)
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - Alibaba Cloud
description: Guide to install APISIX ingress controller on Alibaba Cloud Container Service for Kubernetes (ACK).
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

This document explains how you can install APISIX ingress on [Alibaba Cloud Container Service for Kubernetes (ACK)](https://www.alibabacloud.com/product/kubernetes).

## Prerequisites

Setting up APISIX ingress on ACK requires the following:

* [Create an ACK service](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/create-an-ack-dedicated-cluster).
* [Add the cluster credentials](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/connect-to-ack-clusters-by-using-kubectl) to your kube config file.
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
  --set gateway.type=LoadBalancer \
  --set ingress-controller.enabled=true \
  --set etcd.persistence.storageClass="alicloud-disk-ssd" \
  --set etcd.persistence.size="20Gi" \
  --create-namespace \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix \
  --set ingress-controller.config.apisix.adminAPIVersion=$ADMIN_API_VERSION
kubectl get service --namespace ingress-apisix
```

:::note

By default, APISIX ingress controller will watch the apiVersion of `networking.k8s.io/v1`.

If the target Kubernetes version is under `v1.19`, add the flag `--set ingress-controller.config.kubernetes.ingressVersion=networking/v1beta1`.

Else, if your Kubernetes cluster version is under `v1.16`, set the flag `--set ingress-controller.config.kubernetes.ingressVersion=extensions/v1beta1`.

:::

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

The gateway service type will be set to `LoadBalancer`. See [Use an existing SLB instance to expose an application
](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/use-an-existing-slb-instance-to-expose-an-application-2) for details on using a load balancer.

You can find the load balancer IP address by running:

```shell
kubectl get service apisix-gateway --namespace ingress-apisix -o jsonpath='{.status.loadBalancer.ingress[].ip}'
```

ACK PersistentVolume requires the minimum size of `20Gi` using FlexVolume (select `alicloud-disk-ssd`)

`ACK` PV require min_size is `20Gi`,cluster with `flexVolume` component select `alicloud-disk-ssd`. If you are using Helm, you can use this [etcd configuration file](https://hub.kubeapps.com/charts/bitnami/etcd):

```yaml
etcd:
  persistence:
    storageClass: "alicloud-disk-ssd"
    size: 20Gi
```

You should now be able to use APISIX ingress controller. You can try running this [minimal example](../tutorials/proxy-the-httpbin-service.md) to see if everything is working perfectly.

## Next steps

### Enable SSL

SSL is disabled by default. You can enable it by adding the flag `--set apisix.ssl.enabled=true`.

### Change default keys

It is recommended to change the default keys for security:

```shell
--set ingress-controller.config.apisix.adminKey=ADMIN_KEY_GENERATED_BY_YOURSELF
```

```shell
--set admin.credentials.admin=ADMIN_KEY_GENERATED_BY_YOURSELF
```

```shell
--set admin.credentials.viewer=VIEWER_KEY_GENERATED_BY_YOURSELF
```

:::note

The `ingress-controller.config.apisix.adminKey` and `admin.credentials.admin` must be the same. It is better if these are not same as `admin.credentials.viewer`.

:::
