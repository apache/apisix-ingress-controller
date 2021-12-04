---
title: Install Ingress APISIX on Amazon EKS
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

This document explains how to install Ingress APISIX on [Amazon EKS](https://amazonaws-china.com/eks/?whats-new-cards.sort-by=item.additionalFields.postDateTime&whats-new-cards.sort-order=desc&eks-blogs.sort-by=item.additionalFields.createdDate&eks-blogs.sort-order=desc).

## Prerequisites

* Create an EKS Service on AWS.
* Download the kube config for your EKS from [aws cli interface](https://amazonaws-china.com/cli/).
* Install [Helm](https://helm.sh/).
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
  --namespace ingress-apisix
  --set ingress-controller.config.apisix.serviceNamespace=ingress-apisix
kubectl get service --namespace ingress-apisix
```

Five Service resources were created.

* `apisix-gateway`, which processes the real traffic;
* `apisix-admin`, which acts as the control plane to process all the configuration changes.
* `apisix-ingress-controller`, which exposes apisix-ingress-controller's metrics.
* `apisix-etcd` and `apisix-etcd-headless` for etcd service and internal communication.

The gateway service type is set to `LoadBalancer` (See [AWS Network Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/network-load-balancers.html) for more details), so that clients can access Apache APISIX through a load balancer. You can find the load balancer hostname by running:

```shell
kubectl get service apisix-gateway --namespace ingress-apisix -o jsonpath='{.status.loadBalancer.ingress[].hostname}'
```

Now open your [EKS console](https://console.aws.amazon.com/eks/home), choosing your cluster and clicking the Workloads tag, you'll see all pods of Apache APISIX, etcd and apisix-ingress-controller are ready.

Try to create some [resources](https://github.com/apache/apisix-ingress-controller/tree/master/docs/en/latest/concepts) to verify the running status. As a minimalist example, see [proxy-the-httpbin-service](../practices/proxy-the-httpbin-service.md) to learn how to apply resources to drive the apisix-ingress-controller.

### Specify The Ingress Version

apisix-ingress-controller will watch apiVersion of `networking.k8s.io/v1` by default. If the target kubernetes version is under `v1.19`, add `--set ingress-controller.config.kubernetes.ingressVersion=networking/v1beta1` or `--set ingress-controller.config.kubernetes.ingressVersion=extensions/v1beta1` if your kubernetes cluster is under `v1.16`

### Enable SSL

The ssl config is disabled by default, add `--set gateway.tls.enabled=true` to enable tls support.

### Change default apikey

It's Recommended to change the default key by add `--set ingress-controller.config.apisix.adminKey=ADMIN_KEY_GENERATED_BY_YOURSELF`, `--set admin.credentials.admin=ADMIN_KEY_GENERATED_BY_YOURSELF`, `--set admin.credentials.viewer=VIEWER_KEY_GENERATED_BY_YOURSELF`, notice that `ingress-controller.config.apisix.adminKey` and `admin.credentials.admin` must be the same, and should better not same as `admin.credentials.viewer`.
