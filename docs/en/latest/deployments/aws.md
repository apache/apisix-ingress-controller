---
title: EKS (Amazon)
keywords:
  - APISIX ingress
  - Apache APISIX
  - Kubernetes ingress
  - Amazon EKS
description: Guide to install APISIX ingress controller on Amazon Elastic Kubernetes Service (EKS).
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

This guide explains how you can install APISIX ingress on [Amazon EKS](https://aws.amazon.com/eks/?whats-new-cards.sort-by=item.additionalFields.postDateTime&whats-new-cards.sort-order=desc&eks-blogs.sort-by=item.additionalFields.createdDate&eks-blogs.sort-order=desc).

## Prerequisites

Before installing APISIX, you need to:

* [Create an EKS cluster](https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html) on AWS.
* Enable kubectl to communicate with your cluster by adding the credentials to your kube config file.
* Install [Helm](https://helm.sh/).

## Install APISIX and ingress controller

The script below installs APISIX and the ingress controller:

```shell
helm repo add apisix https://charts.apiseven.com
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
kubectl create ns ingress-apisix
helm install apisix apisix/apisix \
  --set gateway.type=LoadBalancer \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix
kubectl get service --namespace ingress-apisix
```

:::tip

By default AWS provisions a [Classic LoadBalancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/introduction.html). If you want to use a [Network LoadBalancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/introduction.html) you can set the annotation, `service.beta.kubernetes.io/aws-load-balancer-type: nlb`. The install command would now be:

```shell
helm install apisix apisix/apisix \
  --set gateway.type=LoadBalancer \
  --set ingress-controller.enabled=true \
  --namespace ingress-apisix \
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix \
  --set gateway.tls.enabled=true \
  --set gateway.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-type"=nlb
```

:::

:::note

By default, APISIX ingress controller will watch the apiVersion of `networking.k8s.io/v1`.

If the target Kubernetes version is under `v1.19`, add the flag `--set ingress-controller.config.kubernetes.ingressVersion=networking/v1beta1`.

Else, if your Kubernetes cluster version is under `v1.16`, set the flag `--set ingress-controller.config.kubernetes.ingressVersion=extensions/v1beta1`.

:::

This will create the five resources mentioned below:

* `apisix-gateway`: dataplane the process the traffic.
* `apisix-admin`: control plane that processes all configuration changes.
* `apisix-ingress-controller`: ingress controller which exposes APISIX.
* `apisix-etcd` and `apisix-etcd-headless`: stores configuration and handles internal communication.

The gateway service type will be set to `LoadBalancer`. See [Network Load Balancers](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/network-load-balancers.html) for more details on using it in AWS.

You can find the load balancer IP address by running:

```shell
kubectl get service apisix-gateway --namespace ingress-apisix -o jsonpath='{.status.loadBalancer.ingress[].hostname}'
```

Now, if you open your [EKS console](https://console.aws.amazon.com/eks/home), select your cluster, and click the workloads tag, you will be able to see all APISIX, etcd, and ingress controller pods.

You should now be able to use APISIX ingress controller. You can try running this [minimal example](../tutorials/proxy-the-httpbin-service.md) to see if everything is working perfectly.

## Next steps

### Enable SSL

SSL is disabled by default. You can enable it by adding the flag `--set gateway.tls.enabled=true`.

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
