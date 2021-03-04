---
title: Getting Started
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

## What is apisix-ingress-controller

apisix-ingress-controller is yet another Ingress controller for Kubernetes using [Apache APISIX](https://apisix.apache.org) as the high performance reverse proxy.

It's configured by using the declarative configurations like [ApisixRoute](./concepts/apisix_route.md), [ApisixUpstream](./concepts/apisix), [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/).
All these resources are watched and converted to corresponding resources in Apache APISIX.

Service Discovery are also supported through [Kubernetes Service](https://kubernetes.io/docs/concepts/services-networking/service/),
and will be reflected to nodes in APISIX Upstream.

![scene](../../assets/images/scene.png)

## Features

* Declarative configuration
* Full dynamic capabilities to delivery configurations.
* Native Kubernetes Ingress (both v1 and v1beta1) support.
* Service Discovery based on Kubernetes Service.
* Out of box support for node health check.
* Support load balancing based on Pod (upstream nodes).
* Rich plugins support.
* Easy to deploy and use.

## Installation on Cloud

apisix-ingress-controller supports to be installed on some clouds such as AWS, GCP.

* [Install Ingress APISIX on Azure AKS](./docs/en/latest/deployments/azure.md)
* [Install Ingress APISIX on AWS EKS](./docs/en/latest/deployments/aws.md)
* [Install Ingress APISIX on ACK](./docs/en/latest/deployments/ack.md)
* [Install Ingress APISIX on Google Cloud GKE](./docs/en/latest/deployments/gke.md)
* [Install Ingress APISIX on Minikube](./docs/en/latest/deployments/minikube.md)
* [Install Ingress APISIX on KubeSphere](./docs/en/latest/deployments/kubesphere.md)
* [Install Ingress APISIX on K3S and RKE](./docs/en/latest/deployments/k3s-rke.md)

## Installation on Prem

If you want to deploy apisix-ingress-controller on Prem, we recommend you to use [Helm](https://helm.io). Just a few steps

## Get Involved to Contribute

First, your supports and cooperations to make this project better are appreciated.
But before you start, please read [How to Contribute](./contribute.md) and [How to Develop](./development.md).
